package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Driver PostgreSQL
)

var db *sql.DB

// InitDB menginisialisasi koneksi ke database PostgreSQL
func InitDB() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback lokal agar backward-compatible di environment dev lama.
		connStr = "postgres://postgres:SuTa@localhost:5432/koperasi?sslmode=disable"
		log.Println("⚠️ DATABASE_URL tidak diset, menggunakan fallback koneksi lokal")
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi ke database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}

	fmt.Println("Berhasil terhubung ke database!")

	// Pastikan tabel angsuran ada agar aplikasi tidak mengalami error saat insert
	if err := ensureAngsuranTable(); err != nil {
		// Jangan fatal; cukup beri peringatan agar pengembang tahu ada masalah migrasi ringan
		log.Printf("Peringatan: gagal memastikan tabel angsuran ada: %v", err)
	}

	// Pastikan tabel import_history ada untuk fitur import anggota
	if err := ensureImportHistoryTable(); err != nil {
		log.Printf("Peringatan: gagal memastikan tabel import_history ada: %v", err)
	}

	// Pastikan tabel referensi pendaftaran ada untuk validasi calon anggota saat register.
	if err := ensureReferensiPendaftaranTable(); err != nil {
		log.Printf("Peringatan: gagal memastikan tabel referensi_pendaftaran ada: %v", err)
	}

	// Pastikan tabel konfigurasi simpanan wajib ada untuk fitur pemotongan otomatis
	if err := ensureSimpananWajibTables(); err != nil {
		log.Printf("Peringatan: gagal memastikan tabel simpanan wajib ada: %v", err)
	}

	// Update anggota status constraint to include 'keluar'
	if err := updateAnggotaStatusConstraint(); err != nil {
		log.Printf("Peringatan: gagal memperbarui constraint status anggota: %v", err)
	}

	// Pastikan kolom metode_pembayaran ada di tabel detail
	if err := ensureDetailMetodePembayaranColumn(); err != nil {
		log.Printf("Peringatan: gagal memastikan kolom metode_pembayaran: %v", err)
	}

	// Bedakan status verifikasi bendahara dengan keputusan final ketua.
	if err := updateTransactionStatusConstraints(); err != nil {
		log.Printf("Peringatan: gagal memperbarui constraint status transaksi: %v", err)
	}
}

// ensureAngsuranTable membuat tabel angsuran jika belum ada.
// Ini migrasi ringan non-destruktif yang akan membantu menghindari
// error `pq: relation "angsuran" does not exist` saat runtime.
func ensureAngsuranTable() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}
	angsuranSQL := `
	CREATE TABLE IF NOT EXISTS angsuran (
	  id_angsuran SERIAL PRIMARY KEY,
	  id_pinjaman INT REFERENCES pinjaman(id_pinjaman) ON DELETE CASCADE,
	  id_pengelola INT REFERENCES pengelola(id_pengelola) ON DELETE SET NULL,
	  tgl_bayar TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	  sisa_pinjaman NUMERIC(15,2),
	  status_angsuran VARCHAR(25) CHECK (status_angsuran IN ('belum_lunas', 'lunas', 'terlambat')),
	  bukti_angsuran VARCHAR(255),
	  status VARCHAR(25) DEFAULT 'valid' CHECK (status IN ('valid', 'invalid'))
	);
	CREATE INDEX IF NOT EXISTS idx_angsuran_pinjaman ON angsuran(id_pinjaman);
	`

	_, err := db.Exec(angsuranSQL)
	return err
}

// ensureImportHistoryTable membuat tabel import_history jika belum ada.
// Tabel ini digunakan untuk menyimpan riwayat import data anggota.
func ensureImportHistoryTable() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}

	importHistorySQL := `
	CREATE TABLE IF NOT EXISTS import_history (
		id_import VARCHAR(36) PRIMARY KEY,
		id_pengelola INT NOT NULL REFERENCES pengelola(id_pengelola) ON DELETE CASCADE,
		username VARCHAR(100) NOT NULL,
		file_name VARCHAR(255) NOT NULL,
		total_data INT NOT NULL DEFAULT 0,
		success_count INT NOT NULL DEFAULT 0,
		failed_count INT NOT NULL DEFAULT 0,
		imported_data TEXT,
		parse_errors TEXT,
		tanggal_import TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_import_history_pengelola ON import_history(id_pengelola);
	CREATE INDEX IF NOT EXISTS idx_import_history_tanggal ON import_history(tanggal_import);
	`

	_, err := db.Exec(importHistorySQL)
	if err != nil {
		return fmt.Errorf("gagal membuat tabel import_history: %v", err)
	}

	log.Println("✓ Tabel import_history siap digunakan")
	return nil
}

// ensureReferensiPendaftaranTable membuat tabel master referensi pendaftaran jika belum ada.
func ensureReferensiPendaftaranTable() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}

	referensiSQL := `
	CREATE TABLE IF NOT EXISTS referensi_pendaftaran (
		id SERIAL PRIMARY KEY,
		nama_lengkap VARCHAR(255) NOT NULL,
		nomor_identitas VARCHAR(32) DEFAULT '',
		gaji_bulanan INT NOT NULL DEFAULT 0,
		status_keanggotaan VARCHAR(32) NOT NULL DEFAULT 'belum_anggota',
		sumber_file VARCHAR(255) DEFAULT '',
		imported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'referensi_pendaftaran' AND column_name = 'nik_ktp'
		) AND NOT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'referensi_pendaftaran' AND column_name = 'nomor_identitas'
		) THEN
			ALTER TABLE referensi_pendaftaran RENAME COLUMN nik_ktp TO nomor_identitas;
		END IF;
	END $$;

	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS no_telepon;
	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS tgl_lahir;
	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS jenis_kelamin;
	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS status_anggota;
	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS fakultas;
	ALTER TABLE referensi_pendaftaran DROP COLUMN IF EXISTS alamat;

	DROP INDEX IF EXISTS idx_referensi_pendaftaran_nik;
	CREATE INDEX IF NOT EXISTS idx_referensi_pendaftaran_nomor_identitas ON referensi_pendaftaran(nomor_identitas);
	DROP INDEX IF EXISTS idx_referensi_pendaftaran_telepon;
	CREATE INDEX IF NOT EXISTS idx_referensi_pendaftaran_nama ON referensi_pendaftaran(nama_lengkap);
	`

	_, err := db.Exec(referensiSQL)
	if err != nil {
		return fmt.Errorf("gagal membuat tabel referensi_pendaftaran: %v", err)
	}

	log.Println("Tabel referensi_pendaftaran siap digunakan")
	return nil
}

// ensureSimpananWajibTables membuat tabel konfigurasi dan log simpanan wajib jika belum ada.
// Tabel ini digunakan untuk fitur pemotongan simpanan wajib otomatis.
func ensureSimpananWajibTables() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}

	// Create konfigurasi_simpanan_wajib table
	configSQL := `
	CREATE TABLE IF NOT EXISTS konfigurasi_simpanan_wajib (
		id SERIAL PRIMARY KEY,
		tanggal_potong INT NOT NULL CHECK (tanggal_potong >= 1 AND tanggal_potong <= 31),
		persentase_potong DECIMAL(15,2) NOT NULL DEFAULT 100000.00 CHECK (persentase_potong >= 0),
		nominal_tetap DECIMAL(15,2) DEFAULT 0,
		tipe_pemotongan VARCHAR(20) DEFAULT 'persentase' CHECK (tipe_pemotongan IN ('persentase', 'nominal_tetap')),
		status_aktif BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := db.Exec(configSQL)
	if err != nil {
		return fmt.Errorf("gagal membuat tabel konfigurasi_simpanan_wajib: %v", err)
	}

	// Migrate existing table: Alter persentase_potong column if it exists with wrong type
	alterSQL := `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'konfigurasi_simpanan_wajib' 
			AND column_name = 'persentase_potong'
			AND numeric_precision = 5
		) THEN
			ALTER TABLE konfigurasi_simpanan_wajib 
			ALTER COLUMN persentase_potong TYPE DECIMAL(15,2);
			
			ALTER TABLE konfigurasi_simpanan_wajib 
			DROP CONSTRAINT IF EXISTS konfigurasi_simpanan_wajib_persentase_potong_check;
			
			ALTER TABLE konfigurasi_simpanan_wajib 
			ADD CONSTRAINT konfigurasi_simpanan_wajib_persentase_potong_check 
			CHECK (persentase_potong >= 0);
			
			RAISE NOTICE 'Kolom persentase_potong berhasil diubah ke DECIMAL(15,2)';
		END IF;
	END $$;
	`
	_, err = db.Exec(alterSQL)
	if err != nil {
		log.Printf("⚠️ Warning: Gagal melakukan migrasi kolom persentase_potong: %v", err)
	} else {
		log.Printf("✓ Migrasi kolom persentase_potong berhasil")
	}

	// Create log_pemotongan_simpanan table
	logSQL := `
	CREATE TABLE IF NOT EXISTS log_pemotongan_simpanan (
		id_log SERIAL PRIMARY KEY,
		id_anggota VARCHAR(50) REFERENCES anggota(id_anggota) ON DELETE CASCADE,
		bulan INT NOT NULL,
		tahun INT NOT NULL,
		gaji_bulanan DECIMAL(15,2) NOT NULL,
		jumlah_potong DECIMAL(15,2) NOT NULL,
		tgl_proses TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status VARCHAR(20) DEFAULT 'berhasil' CHECK (status IN ('berhasil', 'gagal')),
		keterangan TEXT,
		UNIQUE(id_anggota, bulan, tahun)
	);
	
	CREATE INDEX IF NOT EXISTS idx_log_pemotongan_anggota ON log_pemotongan_simpanan(id_anggota);
	CREATE INDEX IF NOT EXISTS idx_log_pemotongan_periode ON log_pemotongan_simpanan(bulan, tahun);
	`

	_, err = db.Exec(logSQL)
	if err != nil {
		return fmt.Errorf("gagal membuat tabel log_pemotongan_simpanan: %v", err)
	}

	// Insert default configuration if table is empty
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM konfigurasi_simpanan_wajib").Scan(&count)
	if err != nil {
		return fmt.Errorf("gagal memeriksa data konfigurasi: %v", err)
	}

	if count == 0 {
		defaultSQL := `
		INSERT INTO konfigurasi_simpanan_wajib (tanggal_potong, persentase_potong, nominal_tetap, tipe_pemotongan, status_aktif)
		VALUES (1, 100000.00, 0, 'persentase', false)
		`
		_, err = db.Exec(defaultSQL)
		if err != nil {
			return fmt.Errorf("gagal menambahkan data default konfigurasi: %v", err)
		}
		log.Println("✓ Data default konfigurasi simpanan wajib ditambahkan")
	}

	log.Println("✓ Tabel simpanan wajib siap digunakan")
	return nil
}

// updateAnggotaStatusConstraint memperbarui constraint status anggota agar mengizinkan 'keluar'
func updateAnggotaStatusConstraint() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}
	alterSQL := `
	ALTER TABLE anggota DROP CONSTRAINT IF EXISTS anggota_status_check;
	ALTER TABLE anggota ADD CONSTRAINT anggota_status_check CHECK (status IN ('aktif', 'nonaktif', 'pending', 'keluar'));
	`
	_, err := db.Exec(alterSQL)
	return err
}

// ensureDetailMetodePembayaranColumn memastikan kolom metode_pembayaran ada di tabel detail
func ensureDetailMetodePembayaranColumn() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}
	alterSQL := `
	ALTER TABLE detail ADD COLUMN IF NOT EXISTS metode_pembayaran VARCHAR(25) DEFAULT 'transfer_bank';
	`
	_, err := db.Exec(alterSQL)
	if err != nil {
		return fmt.Errorf("gagal menambahkan kolom metode_pembayaran: %v", err)
	}
	log.Println("✓ Kolom metode_pembayaran siap digunakan")
	return nil
}

// updateTransactionStatusConstraints memastikan transaksi bisa memakai status final `diterima`.
func updateTransactionStatusConstraints() error {
	if db == nil {
		return fmt.Errorf("koneksi database belum diinisialisasi")
	}
	alterSQL := `
	ALTER TABLE detail DROP CONSTRAINT IF EXISTS detail_status_check;
	ALTER TABLE detail ADD CONSTRAINT detail_status_check CHECK (status IN ('pending', 'confirmed', 'diterima', 'rejected'));

	ALTER TABLE angsuran DROP CONSTRAINT IF EXISTS angsuran_status_check;
	ALTER TABLE angsuran ADD CONSTRAINT angsuran_status_check CHECK (status IN ('pending', 'confirmed', 'diterima', 'rejected'));
	`
	_, err := db.Exec(alterSQL)
	return err
}

// GetDB mengembalikan instance koneksi database yang sudah ada
func GetDB() *sql.DB {
	return db
}
