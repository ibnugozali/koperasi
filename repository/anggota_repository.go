package repository

import (
	"database/sql"
	"fmt"
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"log"
	"strings"
	"time"
)

type simpananStore interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func getOrCreateSimpananIDByJenis(store simpananStore, jenis string) (int, error) {
	jenis = strings.ToLower(strings.TrimSpace(jenis))
	if jenis == "" {
		return 0, fmt.Errorf("jenis simpanan kosong")
	}

	var idSimpanan int
	err := store.QueryRow(`
		SELECT id_simpanan
		FROM simpanan
		WHERE LOWER(TRIM(jenis_simpanan)) = $1
		LIMIT 1
	`, jenis).Scan(&idSimpanan)
	if err == nil {
		return idSimpanan, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	err = store.QueryRow(`
		INSERT INTO simpanan (jenis_simpanan)
		VALUES ($1)
		RETURNING id_simpanan
	`, jenis).Scan(&idSimpanan)
	if err == nil {
		return idSimpanan, nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		err = store.QueryRow(`
			SELECT id_simpanan
			FROM simpanan
			WHERE LOWER(TRIM(jenis_simpanan)) = $1
			LIMIT 1
		`, jenis).Scan(&idSimpanan)
		if err == nil {
			return idSimpanan, nil
		}
	}

	return 0, err
}

func EnsureSimpananPokokPotongGaji(store simpananStore, idAnggota string, nominalSimpananPokok float64) error {
	idAnggota = strings.TrimSpace(idAnggota)
	if idAnggota == "" {
		return fmt.Errorf("id anggota kosong")
	}
	if nominalSimpananPokok <= 0 {
		nominalSimpananPokok = 100000
	}

	idSimpananPokok, err := getOrCreateSimpananIDByJenis(store, "pokok")
	if err != nil {
		return fmt.Errorf("gagal memastikan jenis simpanan pokok: %w", err)
	}

	var sudahAda bool
	err = store.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM detail
			WHERE id_anggota = $1
			  AND id_simpanan = $2
			  AND COALESCE(status, 'confirmed') IN ('confirmed', 'diterima', 'lunas')
		)
	`, idAnggota, idSimpananPokok).Scan(&sudahAda)
	if err != nil {
		return fmt.Errorf("gagal cek simpanan pokok: %w", err)
	}
	if sudahAda {
		return nil
	}

	_, err = store.Exec(`
		INSERT INTO detail (
			id_anggota, id_simpanan, id_pengelola, tgl_transaksi,
			jumlah_simpanan, total_simpanan, status, bukti_pembayaran, metode_pembayaran
		) VALUES ($1, $2, NULL, CURRENT_TIMESTAMP, $3, $3, 'confirmed', 'POTONG_GAJI', 'potong_gaji')
	`, idAnggota, idSimpananPokok, nominalSimpananPokok)
	if err != nil {
		return fmt.Errorf("gagal insert simpanan pokok: %w", err)
	}

	return nil
}

func RepairMissingSimpananPokokPotongGaji(db *sql.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("koneksi database belum diinisialisasi")
	}

	var nominalSimpananPokok float64
	err := db.QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
	if err != nil {
		nominalSimpananPokok = 100000
	}

	idSimpananPokok, err := getOrCreateSimpananIDByJenis(db, "pokok")
	if err != nil {
		return 0, err
	}

	rows, err := db.Query(`
		SELECT a.id_anggota
		FROM anggota a
		WHERE a.status = 'aktif'
		  AND UPPER(TRIM(COALESCE(a.bukti_transfer, ''))) = 'POTONG_GAJI'
		  AND NOT EXISTS (
			SELECT 1
			FROM detail d
			WHERE d.id_anggota = a.id_anggota
			  AND d.id_simpanan = $1
			  AND COALESCE(d.status, 'confirmed') IN ('confirmed', 'diterima', 'lunas')
		  )
	`, idSimpananPokok)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	repaired := 0
	for rows.Next() {
		var idAnggota string
		if err := rows.Scan(&idAnggota); err != nil {
			return repaired, err
		}
		if err := EnsureSimpananPokokPotongGaji(db, idAnggota, nominalSimpananPokok); err != nil {
			return repaired, err
		}
		repaired++
	}
	if err := rows.Err(); err != nil {
		return repaired, err
	}

	return repaired, nil
}

// GetRiwayatTotalAnggotaPerHari mengambil jumlah anggota aktif per hari selama 30 hari terakhir
func GetRiwayatTotalAnggotaPerHari(db *sql.DB) ([]map[string]interface{}, error) {
	query := `SELECT DATE(tgl_gabung) as tanggal, COUNT(*) as total FROM anggota WHERE status = 'aktif' AND tgl_gabung >= CURRENT_DATE - INTERVAL '30 days' GROUP BY DATE(tgl_gabung) ORDER BY tanggal ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var riwayat []map[string]interface{}
	for rows.Next() {
		var tanggal time.Time
		var total int
		if err := rows.Scan(&tanggal, &total); err != nil {
			return nil, err
		}
		riwayat = append(riwayat, map[string]interface{}{
			"Tanggal": tanggal,
			"Jumlah":  total,
		})
	}
	return riwayat, nil
}

// GetAnggotaByStatus mengambil semua anggota dengan status tertentu
func GetAnggotaByStatus(status string) ([]models.Anggota, error) {
	db := config.GetDB()
	var anggotas []models.Anggota

	// Normalisasi input
	status = strings.TrimSpace(strings.ToLower(status))

	query := `
	       SELECT
		       id_anggota,
		       nama_anggota,
		       username,
		       password,
		       tgl_lahir,
		       no_telepon,
		       tgl_gabung,
		       tgl_keluar,
		       alamat,
		       jenis_kelamin,
		       status,
		       status_anggota,
		       unit_kerja,
		       fakultas,
		       COALESCE(gaji_bulanan, 0)
	       FROM anggota
	       WHERE LOWER(TRIM(status)) LIKE '%' || $1 || '%'
	       ORDER BY CAST(nomor_urut AS INTEGER) DESC
	   `

	rows, err := db.Query(query, status)
	if err != nil {
		log.Printf("[ERROR] Query gagal: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		err := rows.Scan(
			&a.IDAnggota,
			&a.NamaAnggota,
			&a.Username,
			&a.Password,
			&a.TglLahir,
			&a.NoTelepon,
			&a.TglGabung,
			&a.TglKeluar,
			&a.Alamat,
			&a.JenisKelamin,
			&a.Status,
			&a.StatusAnggota,
			&a.UnitKerja,
			&a.Fakultas,
			&a.GajiBulanan,
		)
		if err != nil {
			log.Printf("[ERROR] Scan gagal: %v", err)
			return nil, err
		}

		anggotas = append(anggotas, a)
	}
	return anggotas, nil
}

// Mengambil semua anggota dengan status pending
func GetPendingAnggota() ([]models.Anggota, error) {
	db := config.GetDB() // Fungsi untuk mendapatkan koneksi DB
	var anggotas []models.Anggota

	rows, err := db.Query("SELECT id_anggota, nama_anggota, username, no_telepon, tgl_gabung, unit_kerja, fakultas_code, COALESCE(fakultas, '') FROM anggota WHERE status = 'pending' ORDER BY LPAD(nomor_urut, 4, '0') DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota, &a.Username, &a.NoTelepon, &a.TglGabung, &a.UnitKerja, &a.FakultasCode, &a.Fakultas); err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// GetPendingAnggotaKeluar mengambil anggota yang mengajukan keluar (status_anggota = 'pending_keluar')
func GetPendingAnggotaKeluar() ([]models.Anggota, error) {
	db := config.GetDB()
	var anggotas []models.Anggota

	rows, err := db.Query(`SELECT id_anggota, nama_anggota, username, no_telepon,
		tgl_gabung, unit_kerja, fakultas_code, COALESCE(fakultas, ''), status, status_anggota 
		FROM anggota WHERE status_anggota = 'pending_keluar' ORDER BY tgl_gabung DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota, &a.Username, &a.NoTelepon,
			&a.TglGabung, &a.UnitKerja, &a.FakultasCode, &a.Fakultas, &a.Status, &a.StatusAnggota); err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// Update status anggota dan tambahkan kode anggota
func UpdateAnggotaStatusWithCode(id string, newStatus string, memberCode string) error {
	db := config.GetDB()
	_, err := db.Exec("UPDATE anggota SET status = $1, kode_anggota = $2 WHERE id_anggota = $3", newStatus, memberCode, id)
	return err
}

// ... (fungsi lainnya)
// Membuat anggota baru (registrasi)
func CreateAnggota(anggota models.Anggota) error {
	db := config.GetDB()
	// Ambil tahun konfirmasi (tahun sekarang, 2 digit)
	tahun := time.Now().Format("06")
	anggota.Tahun = tahun

	// Generate temporary ID for pending members: TEMP{timestamp}
	tempID := fmt.Sprintf("TEMP%d", time.Now().UnixNano())
	anggota.IDAnggota = tempID

	// nomor_urut will be set to NULL during registration and assigned during confirmation
	anggota.NomorUrut = "NULL"

	// Pastikan waktu gabung menyimpan jam-menit-detik
	anggota.TglGabung = time.Now()
	query := `
	INSERT INTO anggota (id_anggota, nama_anggota, username, password, tgl_lahir, no_telepon, alamat, jenis_kelamin, status, status_anggota, fakultas, tgl_gabung, unit_kerja, fakultas_code, bukti_transfer, gaji_bulanan, tahun, nomor_urut)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	var nomorUrut interface{}
	if anggota.NomorUrut == "NULL" {
		nomorUrut = nil
	} else {
		nomorUrut = anggota.NomorUrut
	}
	_, err := db.Exec(query,
		anggota.IDAnggota,
		anggota.NamaAnggota,
		anggota.Username,
		anggota.Password,
		anggota.TglLahir,
		anggota.NoTelepon,
		anggota.Alamat,
		anggota.JenisKelamin,
		"pending",
		anggota.StatusAnggota,
		anggota.Fakultas,
		anggota.TglGabung,
		anggota.UnitKerja,
		anggota.FakultasCode,
		anggota.BuktiTransfer,
		anggota.GajiBulanan,
		anggota.Tahun,
		nomorUrut,
	)
	return err
}

// Di file: repository/anggota_repository.go

// ... (fungsi-fungsi lain yang sudah ada)

func GetAnggotaByID(id string) (models.Anggota, error) {
	db := config.GetDB()
	var a models.Anggota
	var encryptedPassword string
	query := `
			       SELECT
			       id_anggota, nama_anggota, username, password,
			       tgl_lahir, no_telepon, tgl_gabung,
			       alamat, jenis_kelamin, status,
			       unit_kerja, fakultas_code, COALESCE(tahun, ''), COALESCE(CAST(nomor_urut AS TEXT), '0'), COALESCE(bukti_transfer, ''), COALESCE(status_anggota, ''), COALESCE(fakultas, ''), COALESCE(gaji_bulanan, 0)
		       FROM anggota
		       WHERE id_anggota = $1`

	err := db.QueryRow(query, id).Scan(
		&a.IDAnggota, &a.NamaAnggota, &a.Username, &encryptedPassword,
		&a.TglLahir, &a.NoTelepon, &a.TglGabung,
		&a.Alamat, &a.JenisKelamin, &a.Status,
		&a.UnitKerja, &a.FakultasCode, &a.Tahun, &a.NomorUrut, &a.BuktiTransfer, &a.StatusAnggota, &a.Fakultas, &a.GajiBulanan,
	)
	if err != nil {
		return a, err
	}

	// Password sudah dalam bentuk plain text dari database
	a.Password = encryptedPassword

	return a, err
}

// GetAllAnggota mengambil semua anggota aktif
func GetAllAnggota() ([]models.Anggota, error) {
	db := config.GetDB()
	var anggotas []models.Anggota

	query := `
		SELECT
		 id_anggota, nama_anggota, username, password,
		 tgl_lahir, no_telepon, tgl_gabung,
		 alamat, jenis_kelamin, status, unit_kerja, fakultas, COALESCE(gaji_bulanan, 0)
		FROM anggota
		WHERE status = 'aktif'
		ORDER BY CAST(nomor_urut AS INTEGER) DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		err := rows.Scan(
			&a.IDAnggota, &a.NamaAnggota, &a.Username, &a.Password,
			&a.TglLahir, &a.NoTelepon, &a.TglGabung,
			&a.Alamat, &a.JenisKelamin, &a.Status, &a.UnitKerja, &a.Fakultas, &a.GajiBulanan,
		)
		if err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// DeleteAnggota menghapus anggota berdasarkan ID (soft delete atau hard delete tergantung kebutuhan)
// DeleteAnggota melakukan soft delete dengan mengubah status menjadi 'keluar'
func DeleteAnggota(id string) error {
	db := config.GetDB()
	res, err := db.Exec("DELETE FROM anggota WHERE id_anggota = $1", id)
	if err != nil {
		log.Printf("Gagal hard delete anggota: %v", err)
		return err
	}
	rows, _ := res.RowsAffected()
	log.Printf("Hard delete anggota: id=%s, rowsAffected=%d", id, rows)
	return nil
}

// UpdateAnggotaPassword memperbarui password anggota berdasarkan ID
func UpdateAnggotaPassword(id string, newPassword string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET password = $1 WHERE id_anggota = $2"
	_, err := db.Exec(query, newPassword, id)
	return err
}

// UpdateAnggotaUsernamePassword memperbarui username dan password anggota berdasarkan ID
func UpdateAnggotaUsernamePassword(id string, username, password string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET username = $1, password = $2 WHERE id_anggota = $3"
	_, err := db.Exec(query, username, password, id)
	return err
}

// GetSaldoAnggota mengambil total simpanan, total pinjaman, dan saldo bersih anggota
// Total simpanan termasuk simpanan pokok dari registrasi
func GetSaldoAnggota(id string) (totalSimpanan, totalPinjaman, saldoBersih float64, err error) {
	db := config.GetDB()

	// Ambil simpanan pokok dari nominal registrasi jika anggota aktif
	var simpananPokok float64
	var status string
	var buktiTransfer string
	err = db.QueryRow("SELECT status, COALESCE(bukti_transfer, '') FROM anggota WHERE id_anggota = $1", id).Scan(&status, &buktiTransfer)
	if err == nil && status == "aktif" && buktiTransfer != "" {
		err = db.QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&simpananPokok)
		if err != nil {
			simpananPokok = 100000 // Default
		}
	}

	// Hitung total simpanan lainnya dari detail (ambil total_simpanan terbaru untuk setiap jenis, kecuali pokok)
	var totalSimpananLainnya float64
	querySimpanan := `
		SELECT COALESCE(SUM(latest.total_simpanan), 0)
		FROM (
			SELECT DISTINCT ON (d.id_simpanan) d.id_simpanan, d.total_simpanan
			FROM detail d
			JOIN simpanan s ON d.id_simpanan = s.id_simpanan
			WHERE d.id_anggota = $1 AND COALESCE(LOWER(d.status), '') IN ('diterima', 'lunas')
				AND s.jenis_simpanan != 'pokok'
			ORDER BY d.id_simpanan, d.tgl_transaksi DESC, d.id_detail DESC
		) as latest
	`
	err = db.QueryRow(querySimpanan, id).Scan(&totalSimpananLainnya)
	if err != nil {
		return 0, 0, 0, err
	}

	totalSimpanan = simpananPokok + totalSimpananLainnya

	// Hitung total sisa pinjaman aktif (berdasarkan sisa_pinjaman terakhir dari angsuran yang sudah dikonfirmasi)
	var totalSisaPinjaman float64
	rows, err := db.Query(`SELECT id_pinjaman, jumlah_pinjaman FROM pinjaman WHERE id_anggota = $1 AND status = 'aktif'`, id)
	if err != nil {
		return totalSimpanan, 0, totalSimpanan, err
	}
	defer rows.Close()
	for rows.Next() {
		var idPinjaman int
		var jumlahPinjaman float64
		err := rows.Scan(&idPinjaman, &jumlahPinjaman)
		if err != nil {
			continue
		}
		var sisaPinjaman float64
		err = db.QueryRow(`SELECT sisa_pinjaman FROM angsuran WHERE id_pinjaman = $1 AND status IN ('lunas','diterima') ORDER BY tgl_bayar DESC, id_angsuran DESC LIMIT 1`, idPinjaman).Scan(&sisaPinjaman)
		if err != nil {
			sisaPinjaman = jumlahPinjaman // Jika belum ada angsuran, gunakan jumlah pinjaman
		}
		totalSisaPinjaman += sisaPinjaman
	}
	saldoBersih = totalSimpanan - totalSisaPinjaman

	return totalSimpanan, totalSisaPinjaman, saldoBersih, nil
}

// GetDetailSimpananByJenis mengambil total simpanan per jenis (menggunakan total_simpanan terbaru)
// Simpanan pokok diambil dari nominal registrasi jika anggota sudah aktif
func GetDetailSimpananByJenis(id string) (map[string]float64, error) {
	db := config.GetDB()
	simpananByJenis := make(map[string]float64)

	// Cek status anggota dan ambil simpanan pokok dari nominal registrasi jika aktif
	var status string
	var buktiTransfer string
	err := db.QueryRow("SELECT status, COALESCE(bukti_transfer, '') FROM anggota WHERE id_anggota = $1", id).Scan(&status, &buktiTransfer)
	if err != nil {
		return nil, err
	}

	// Jika anggota sudah aktif atau keluar dan punya bukti transfer, simpanan pokok diambil dari pengaturan
	if (status == "aktif" || status == "keluar") && buktiTransfer != "" {
		var nominalSimpananPokok float64
		err = db.QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
		if err != nil {
			nominalSimpananPokok = 100000 // Default
		}
		simpananByJenis["pokok"] = nominalSimpananPokok
	} else {
		simpananByJenis["pokok"] = 0
	}

	// SIMPANAN WAJIB: Ambil dari log_pemotongan_simpanan (pemotongan otomatis) + detail (konfirmasi manual)
	// 1. Dari log pemotongan otomatis
	simpananWajibQuery := `SELECT COALESCE(SUM(jumlah_potong), 0) as total_simpanan_wajib
	                       FROM log_pemotongan_simpanan 
	                       WHERE status = 'berhasil' AND id_anggota = $1`
	var totalSimpananWajib float64
	err = db.QueryRow(simpananWajibQuery, id).Scan(&totalSimpananWajib)
	if err != nil {
		totalSimpananWajib = 0
	}

	// 2. Tambahkan dari detail (konfirmasi manual) - id_simpanan = 2 adalah simpanan wajib
	// Menggunakan SUM dari jumlah_simpanan, bukan total_simpanan terbaru
	var totalSimpananWajibDetail float64
	detailWajibQuery := `SELECT COALESCE(SUM(jumlah_simpanan), 0) 
	                     FROM detail 
	                     WHERE id_anggota = $1 AND id_simpanan = 2
	                       AND COALESCE(LOWER(status), '') IN ('diterima', 'lunas')`
	err = db.QueryRow(detailWajibQuery, id).Scan(&totalSimpananWajibDetail)
	if err != nil {
		totalSimpananWajibDetail = 0
	}

	simpananByJenis["wajib"] = totalSimpananWajib + totalSimpananWajibDetail

	// SIMPANAN SUKARELA, HARI RAYA, UMROH/HAJI, DAN QURBAN: Ambil dari SUM semua transaksi di detail (sama seperti di anggota_riwayat)
	// Tidak menggunakan total_simpanan terbaru, tapi SUM dari jumlah_simpanan semua transaksi
	// id_simpanan: 3 = sukarela, 4 = hari_raya, 5 = umroh_haji, 6 = qurban
	querySukarelaDanHariRaya := `
		SELECT s.jenis_simpanan, COALESCE(SUM(d.jumlah_simpanan), 0) as total
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1
		  AND COALESCE(LOWER(d.status), '') IN ('diterima', 'lunas')
		  AND s.jenis_simpanan IN ('sukarela', 'hari_raya', 'umroh_haji', 'qurban')
		GROUP BY s.jenis_simpanan
	`

	rows, err := db.Query(querySukarelaDanHariRaya, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var jenis string
			var total float64
			if err := rows.Scan(&jenis, &total); err != nil {
				continue
			}
			simpananByJenis[jenis] = total
		}
	}

	// Kurangi dengan pengambilan simpanan yang disetujui
	// Ambil total pengambilan simpanan yang approved per jenis
	queryPengambilan := `
		SELECT s.jenis_simpanan, COALESCE(SUM(ps.jumlah), 0) as total_pengambilan
		FROM pengambilan_simpanan ps
		JOIN simpanan s ON ps.id_simpanan = s.id_simpanan
		WHERE ps.id_anggota = $1 AND ps.status = 'approved'
		GROUP BY s.jenis_simpanan
	`

	rowsPengambilan, err := db.Query(queryPengambilan, id)
	if err == nil {
		defer rowsPengambilan.Close()
		for rowsPengambilan.Next() {
			var jenis string
			var totalPengambilan float64
			if err := rowsPengambilan.Scan(&jenis, &totalPengambilan); err != nil {
				continue
			}
			// Kurangi saldo simpanan dengan jumlah pengambilan yang disetujui
			if currentSaldo, exists := simpananByJenis[jenis]; exists {
				simpananByJenis[jenis] = currentSaldo - totalPengambilan
				// Pastikan tidak negatif
				if simpananByJenis[jenis] < 0 {
					simpananByJenis[jenis] = 0
				}
			}
		}
	}

	// Pastikan semua jenis simpanan ada dengan nilai 0 jika tidak ada data (kecuali wajib yang sudah diset)
	jenisList := []string{"sukarela", "hari_raya", "umroh_haji", "qurban"}
	for _, jenis := range jenisList {
		if _, exists := simpananByJenis[jenis]; !exists {
			simpananByJenis[jenis] = 0
		}
	}

	return simpananByJenis, nil
}

// GetSimpananWajibAllAnggota mengambil total simpanan wajib dari log pemotongan DAN detail manual untuk semua anggota
func GetSimpananWajibAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	simpananWajib := make(map[string]float64)

	// Ambil total simpanan wajib dari log_pemotongan_simpanan (pemotongan otomatis)
	queryLog := `SELECT id_anggota, COALESCE(SUM(jumlah_potong), 0) as total_simpanan_wajib
	             FROM log_pemotongan_simpanan 
	             WHERE status = 'berhasil' AND id_anggota != 'SYSTEM'
	             GROUP BY id_anggota`

	rows, err := db.Query(queryLog)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var idAnggota string
			var totalSimpanan float64
			if err := rows.Scan(&idAnggota, &totalSimpanan); err != nil {
				continue
			}
			simpananWajib[idAnggota] = totalSimpanan
		}
	}

	// Tambahkan simpanan wajib dari detail (konfirmasi manual oleh bendahara/ketua)
	// id_simpanan = 2 adalah simpanan wajib
	// Menggunakan SUM dari jumlah_simpanan, bukan total_simpanan terbaru
	// Status yang valid: 'confirmed' (bendahara), 'diterima' (ketua), 'lunas', atau NULL (default confirmed)
	queryDetail := `SELECT id_anggota, COALESCE(SUM(jumlah_simpanan), 0) as total_simpanan_wajib
	                FROM detail
	                WHERE id_simpanan = 2
	                  AND COALESCE(LOWER(status), 'confirmed') IN ('confirmed', 'diterima', 'lunas')
	                GROUP BY id_anggota`

	rows2, err := db.Query(queryDetail)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var idAnggota string
			var totalSimpananDetail float64
			if err := rows2.Scan(&idAnggota, &totalSimpananDetail); err != nil {
				continue
			}
			// Tambahkan ke total yang sudah ada dari log
			simpananWajib[idAnggota] += totalSimpananDetail
		}
	}

	return simpananWajib, nil
}

// GetPotonganBulanIniAllAnggota mengambil nominal simpanan wajib yang dikonfigurasi untuk semua anggota
func GetPotonganBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganBulanIni := make(map[string]float64)
	simpananWajib := make(map[string]float64)

	// Ambil konfigurasi simpanan wajib
	config, err := GetKonfigurasiSimpananWajib()
	if err != nil {
		return potonganBulanIni, nil // Return map kosong jika error
	}

	if data, err := GetSimpananWajibAllAnggota(); err == nil {
		simpananWajib = data
	}

	statusAktif, _ := config["StatusAktif"].(bool)
	if !statusAktif {
		return potonganBulanIni, nil
	}

	// Get current month and year
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()
	tanggalSekarang := now.Day()

	// Ambil data dari log pemotongan bulan ini jika setting simpanan wajib aktif.
	logQuery := `SELECT id_anggota, jumlah_potong FROM log_pemotongan_simpanan 
	             WHERE bulan = $1 AND tahun = $2 AND status = 'berhasil' AND id_anggota != 'SYSTEM'`
	rows, err := db.Query(logQuery, bulan, tahun)

	if err == nil {
		defer rows.Close()

		// Ambil data dari log yang sudah ada
		for rows.Next() {
			var idAnggota string
			var jumlahPotong float64
			if err := rows.Scan(&idAnggota, &jumlahPotong); err != nil {
				continue
			}
			potonganBulanIni[idAnggota] += jumlahPotong
		}
	}

	// Jika sudah ada data dari log, gunakan nilai aktual yang sudah diproses.
	if len(potonganBulanIni) > 0 {
		return potonganBulanIni, nil
	}

	nominalSimpananWajib, ok := config["PersentasePotong"].(float64)
	if !ok {
		nominalSimpananWajib = 0
	}
	tanggalPotong, ok := config["TanggalPotong"].(int)
	if !ok || tanggalPotong <= 0 {
		tanggalPotong = 1
	}

	// Sebelum tanggal pemotongan, belum ada potongan bulan ini yang perlu dipreview.
	if tanggalSekarang < tanggalPotong {
		return potonganBulanIni, nil
	}

	// Query anggota aktif dengan gaji bulanan
	queryAnggota := `SELECT id_anggota, gaji_bulanan FROM anggota WHERE status = 'aktif' AND gaji_bulanan > 0`
	rows, err = db.Query(queryAnggota)
	if err != nil {
		return potonganBulanIni, nil
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var gajiBulanan int
		if err := rows.Scan(&idAnggota, &gajiBulanan); err != nil {
			continue
		}

		// Samakan dengan halaman bendahara:
		// estimasi potongan bulan ini mengikuti kekurangan menuju target simpanan wajib.
		if nominalSimpananWajib > 0 {
			kekurangan := nominalSimpananWajib - simpananWajib[idAnggota]
			if kekurangan > 0 {
				potonganBulanIni[idAnggota] = kekurangan
			} else {
				potonganBulanIni[idAnggota] = 0
			}
		}
	}

	return potonganBulanIni, nil
}

// GetPotonganRegisterPotongGajiBulanIniAllAnggota mengambil potongan simpanan pokok
// dari pendaftaran metode potong gaji pada bulan berjalan.
// Nilai ini dipakai untuk mengurangi sisa gaji, tetapi tidak ditampilkan
// sebagai "Potongan Bulan Ini" karena tidak mengikuti jadwal simpanan wajib bulanan.
func GetPotonganRegisterPotongGajiBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganRegister := make(map[string]float64)

	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	query := `
		SELECT d.id_anggota, COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE LOWER(TRIM(COALESCE(s.jenis_simpanan, ''))) = 'pokok'
		  AND COALESCE(d.status, 'confirmed') IN ('confirmed', 'diterima', 'lunas')
		  AND REPLACE(REPLACE(REPLACE(LOWER(TRIM(COALESCE(d.metode_pembayaran, ''))), ' ', ''), '_', ''), '-', '') = 'potonggaji'
		  AND EXTRACT(MONTH FROM d.tgl_transaksi) = $1
		  AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
		GROUP BY d.id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return potonganRegister, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var jumlahPotong float64
		if err := rows.Scan(&idAnggota, &jumlahPotong); err != nil {
			continue
		}
		potonganRegister[idAnggota] += jumlahPotong
	}

	return potonganRegister, nil
}

// GetPotonganSimpananPotongGajiBulanIniAllAnggota mengambil simpanan non-wajib
// metode potong gaji pada bulan berjalan. Simpanan wajib tidak dihitung di sini
// karena sudah ditangani oleh GetPotonganBulanIniAllAnggota agar tidak dobel.
func GetPotonganSimpananPotongGajiBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganSimpanan := make(map[string]float64)

	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	query := `
		SELECT d.id_anggota, COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE d.id_simpanan <> 2
		  AND COALESCE(LOWER(d.status), 'pending') IN ('pending', 'confirmed', 'diterima', 'lunas')
		  AND REPLACE(REPLACE(REPLACE(LOWER(TRIM(COALESCE(d.metode_pembayaran, ''))), ' ', ''), '_', ''), '-', '') = 'potonggaji'
		  AND EXTRACT(MONTH FROM d.tgl_transaksi) = $1
		  AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
		GROUP BY d.id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return potonganSimpanan, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var jumlahPotong float64
		if err := rows.Scan(&idAnggota, &jumlahPotong); err != nil {
			continue
		}
		potonganSimpanan[idAnggota] += jumlahPotong
	}

	return potonganSimpanan, nil
}

// GetPotonganSimpananWajibPotongGajiBulanIniAllAnggota mengambil simpanan wajib
// aktual metode potong gaji pada bulan berjalan. Nilai ini dipakai untuk
// menggantikan estimasi wajib agar sisa gaji tidak dobel.
func GetPotonganSimpananWajibPotongGajiBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganWajib := make(map[string]float64)

	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	query := `
		SELECT d.id_anggota, COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE d.id_simpanan = 2
		  AND COALESCE(LOWER(d.status), 'pending') IN ('pending', 'confirmed', 'diterima', 'lunas')
		  AND REPLACE(REPLACE(REPLACE(LOWER(TRIM(COALESCE(d.metode_pembayaran, ''))), ' ', ''), '_', ''), '-', '') = 'potonggaji'
		  AND EXTRACT(MONTH FROM d.tgl_transaksi) = $1
		  AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
		GROUP BY d.id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return potonganWajib, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var jumlahPotong float64
		if err := rows.Scan(&idAnggota, &jumlahPotong); err != nil {
			continue
		}
		potonganWajib[idAnggota] += jumlahPotong
	}

	return potonganWajib, nil
}

// GetPotonganAngsuranPotongGajiBulanIniAllAnggota mengambil cicilan pinjaman
// metode potong gaji pada bulan berjalan, termasuk yang masih pending setelah
// pinjaman disetujui ketua.
func GetPotonganAngsuranPotongGajiBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganAngsuran := make(map[string]float64)

	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	query := `
		SELECT p.id_anggota, COALESCE(SUM(a.jumlah_angsuran), 0)
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		WHERE REPLACE(REPLACE(REPLACE(LOWER(TRIM(COALESCE(p.metode_angsuran, ''))), ' ', ''), '_', ''), '-', '') = 'potonggaji'
		  AND COALESCE(LOWER(a.status), 'pending') IN ('pending', 'confirmed', 'diterima', 'lunas')
		  AND EXTRACT(MONTH FROM a.tgl_bayar) = $1
		  AND EXTRACT(YEAR FROM a.tgl_bayar) = $2
		GROUP BY p.id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return potonganAngsuran, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var jumlahPotong float64
		if err := rows.Scan(&idAnggota, &jumlahPotong); err != nil {
			continue
		}
		potonganAngsuran[idAnggota] += jumlahPotong
	}

	fallbackQuery := `
		SELECT p.id_anggota,
		       COALESCE(SUM(ROUND((p.jumlah_pinjaman + (p.jumlah_pinjaman * COALESCE(p.bunga, 0) / 100)) / NULLIF(p.jangka_waktu, 0))), 0)
		FROM pinjaman p
		WHERE REPLACE(REPLACE(REPLACE(LOWER(TRIM(COALESCE(p.metode_angsuran, ''))), ' ', ''), '_', ''), '-', '') = 'potonggaji'
		  AND COALESCE(LOWER(p.status), '') IN ('aktif', 'proses')
		  AND COALESCE(p.jangka_waktu, 0) > 0
		  AND NOT EXISTS (
		      SELECT 1
		      FROM angsuran a
		      WHERE a.id_pinjaman = p.id_pinjaman
		        AND COALESCE(LOWER(a.status), 'pending') IN ('pending', 'confirmed', 'diterima', 'lunas')
		        AND EXTRACT(MONTH FROM a.tgl_bayar) = $1
		        AND EXTRACT(YEAR FROM a.tgl_bayar) = $2
		  )
		GROUP BY p.id_anggota
	`
	rowsFallback, err := db.Query(fallbackQuery, bulan, tahun)
	if err != nil {
		return potonganAngsuran, err
	}
	defer rowsFallback.Close()

	for rowsFallback.Next() {
		var idAnggota string
		var jumlahPotong float64
		if err := rowsFallback.Scan(&idAnggota, &jumlahPotong); err != nil {
			continue
		}
		potonganAngsuran[idAnggota] += jumlahPotong
	}

	return potonganAngsuran, nil
}

// GetKonfigurasiSimpananWajib mengambil konfigurasi pemotongan simpanan wajib
func GetKonfigurasiSimpananWajib() (map[string]interface{}, error) {
	db := config.GetDB()
	config := make(map[string]interface{})

	query := `SELECT tanggal_potong, persentase_potong, nominal_tetap, tipe_pemotongan, status_aktif 
	          FROM konfigurasi_simpanan_wajib ORDER BY id DESC LIMIT 1`

	var tanggalPotong int
	var persentasePotong, nominalTetap float64
	var tipePemotongan string
	var statusAktif bool

	err := db.QueryRow(query).Scan(&tanggalPotong, &persentasePotong, &nominalTetap, &tipePemotongan, &statusAktif)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default values
			return map[string]interface{}{
				"TanggalPotong":    1,
				"PersentasePotong": 100000.0,
				"NominalTetap":     0.0,
				"TipePemotongan":   "persentase",
				"StatusAktif":      false,
			}, nil
		}
		return nil, err
	}

	config["TanggalPotong"] = tanggalPotong
	config["PersentasePotong"] = persentasePotong
	config["NominalTetap"] = nominalTetap
	config["TipePemotongan"] = tipePemotongan
	config["StatusAktif"] = statusAktif

	return config, nil
}

// SaveKonfigurasiSimpananWajib menyimpan atau update konfigurasi pemotongan simpanan wajib
func SaveKonfigurasiSimpananWajib(tanggalPotong int, persentasePotong, nominalTetap float64, tipePemotongan string, statusAktif bool) error {
	db := config.GetDB()

	// Check if config exists
	var count int
	var existingID int
	err := db.QueryRow("SELECT COUNT(*), COALESCE(MAX(id), 0) FROM konfigurasi_simpanan_wajib").Scan(&count, &existingID)
	if err != nil {
		log.Printf("❌ Error mengecek konfigurasi existing: %v", err)
		return err
	}

	log.Printf("📊 Jumlah record konfigurasi existing: %d", count)

	if count > 0 {
		// Update existing config dengan WHERE clause untuk memastikan hanya update record yang benar
		query := `UPDATE konfigurasi_simpanan_wajib 
		          SET tanggal_potong = $1, persentase_potong = $2, nominal_tetap = $3, 
		              tipe_pemotongan = $4, status_aktif = $5, updated_at = CURRENT_TIMESTAMP
		          WHERE id = $6`
		log.Printf("🔄 Updating konfigurasi dengan query: %s (ID: %d)", query, existingID)
		result, err := db.Exec(query, tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif, existingID)
		if err != nil {
			log.Printf("❌ Error UPDATE: %v", err)
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		log.Printf("✅ UPDATE berhasil, rows affected: %d", rowsAffected)

		// Hapus duplikat jika ada (safety measure)
		if count > 1 {
			log.Printf("⚠️ Ditemukan %d record, menghapus duplikat...", count)
			deleteQuery := `DELETE FROM konfigurasi_simpanan_wajib WHERE id != $1`
			deleteResult, deleteErr := db.Exec(deleteQuery, existingID)
			if deleteErr != nil {
				log.Printf("⚠️ Warning: Gagal menghapus duplikat: %v", deleteErr)
			} else {
				deletedRows, _ := deleteResult.RowsAffected()
				log.Printf("✅ Berhasil menghapus %d record duplikat", deletedRows)
			}
		}
	} else {
		// Insert new config
		query := `INSERT INTO konfigurasi_simpanan_wajib 
		          (tanggal_potong, persentase_potong, nominal_tetap, tipe_pemotongan, status_aktif) 
		          VALUES ($1, $2, $3, $4, $5)`
		log.Printf("➕ Inserting konfigurasi baru dengan query: %s", query)
		result, err := db.Exec(query, tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif)
		if err != nil {
			log.Printf("❌ Error INSERT: %v", err)
			return err
		}
		rowsAffected, _ := result.RowsAffected()
		log.Printf("✅ INSERT berhasil, rows affected: %d", rowsAffected)
	}

	return nil
}

// ProsesPemotonganSimpananWajib melakukan pemotongan otomatis untuk semua anggota
func ProsesPemotonganSimpananWajib() (successCount, failedCount int, errors []string) {
	db := config.GetDB()
	successCount = 0
	failedCount = 0
	errors = []string{}

	// Get konfigurasi
	configData, err := GetKonfigurasiSimpananWajib()
	if err != nil {
		errors = append(errors, "Gagal mengambil konfigurasi: "+err.Error())
		return
	}

	if !configData["StatusAktif"].(bool) {
		errors = append(errors, "Fitur pemotongan otomatis tidak aktif")
		return
	}

	// Get all active anggota dengan gaji > 0
	anggotas, err := GetAllAnggota()
	if err != nil {
		errors = append(errors, "Gagal mengambil data anggota: "+err.Error())
		return
	}

	fmt.Printf("=== Proses Pemotongan Simpanan Wajib ===\n")
	fmt.Printf("Total anggota: %d\n", len(anggotas))

	tipePemotongan := configData["TipePemotongan"].(string)
	persentasePotong := configData["PersentasePotong"].(float64)
	// nominalTetap := configData["NominalTetap"].(float64) // Tidak digunakan lagi

	// PENTING: Field PersentasePotong sebenarnya menyimpan Nominal Simpanan Wajib (bukan persentase)
	// Jadi kita gunakan PersentasePotong sebagai nominal tetap
	nominalSimpananWajib := persentasePotong

	fmt.Printf("Tipe pemotongan: %s, Nominal Simpanan Wajib: Rp %.0f\n", tipePemotongan, nominalSimpananWajib)

	// Get current month and year
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	fmt.Printf("Bulan: %d, Tahun: %d\n", bulan, tahun)

	// Hapus record SYSTEM jika ada (untuk memungkinkan proses ulang jika ada data baru)
	deleteSystemQuery := `DELETE FROM log_pemotongan_simpanan 
	                      WHERE bulan = $1 AND tahun = $2 AND id_anggota = 'SYSTEM'`
	db.Exec(deleteSystemQuery, bulan, tahun)

	var skippedNoGaji, skippedAlreadyProcessed, skippedHasSimpananWajib int

	for _, anggota := range anggotas {
		// Skip mahasiswa atau yang tidak punya gaji
		if anggota.GajiBulanan <= 0 {
			skippedNoGaji++
			fmt.Printf("  Skip %s (ID: %s): Gaji = %d\n", anggota.NamaAnggota, anggota.IDAnggota, anggota.GajiBulanan)
			continue
		}

		// Check if already processed this month (hanya check anggota real, bukan SYSTEM)
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM log_pemotongan_simpanan WHERE id_anggota = $1 AND bulan = $2 AND tahun = $3 AND status = 'berhasil')"
		db.QueryRow(checkQuery, anggota.IDAnggota, bulan, tahun).Scan(&exists)

		if exists {
			skippedAlreadyProcessed++
			fmt.Printf("  Skip %s (ID: %s): Sudah dipotong bulan ini\n", anggota.NamaAnggota, anggota.IDAnggota)
			continue
		}

		// PENTING: Check apakah anggota sudah memiliki simpanan wajib dari konfirmasi manual (detail)
		// Jika sudah ada, skip untuk tidak memotong gaji lagi
		var hasSimpananWajib bool
		checkSimpananWajibQuery := `SELECT EXISTS(
			SELECT 1 FROM detail 
			WHERE id_anggota = $1 AND id_simpanan = 2 AND COALESCE(status, 'confirmed') = 'confirmed'
		)`
		db.QueryRow(checkSimpananWajibQuery, anggota.IDAnggota).Scan(&hasSimpananWajib)

		if hasSimpananWajib {
			skippedHasSimpananWajib++
			fmt.Printf("  Skip %s (ID: %s): Sudah ada simpanan wajib dari konfirmasi manual\n", anggota.NamaAnggota, anggota.IDAnggota)
			continue
		}

		// Calculate potongan - gunakan nominalSimpananWajib (dari field PersentasePotong)
		jumlahPotong := nominalSimpananWajib

		fmt.Printf("  Processing %s (ID: %s): Gaji Rp %d, Potong Rp %.0f\n", anggota.NamaAnggota, anggota.IDAnggota, anggota.GajiBulanan, jumlahPotong)

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			failedCount++
			fmt.Printf("    ❌ Gagal memulai transaksi: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))
			continue
		}

		// Insert detail simpanan wajib
		detailQuery := `INSERT INTO detail (id_anggota, id_simpanan, id_pengelola, tgl_transaksi, jumlah_simpanan, status)
		                VALUES ($1, 2, 1, CURRENT_TIMESTAMP, $2, 'confirmed')`
		_, err = tx.Exec(detailQuery, anggota.IDAnggota, jumlahPotong)

		if err != nil {
			tx.Rollback()
			failedCount++
			fmt.Printf("    ❌ Gagal insert detail: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))

			// Log failed
			logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
			             VALUES ($1, $2, $3, $4, $5, 'gagal', $6)`
			db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, err.Error())
			continue
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			failedCount++
			fmt.Printf("    ❌ Gagal commit: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))

			// Log failed
			logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
			             VALUES ($1, $2, $3, $4, $5, 'gagal', $6)`
			db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, err.Error())
			continue
		}

		successCount++
		fmt.Printf("    ✓ Berhasil memotong Rp %.0f (Gaji tetap: Rp %d)\n", jumlahPotong, anggota.GajiBulanan)

		// Log success
		logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
		             VALUES ($1, $2, $3, $4, $5, 'berhasil', $6)`
		db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, fmt.Sprintf("Pemotongan simpanan wajib berhasil sebesar Rp %.0f", jumlahPotong))
	}

	fmt.Printf("\n=== Ringkasan ===\n")
	fmt.Printf("Berhasil: %d, Gagal: %d\n", successCount, failedCount)
	fmt.Printf("Dilewati (no gaji): %d, Dilewati (sudah dipotong): %d, Dilewati (sudah ada simpanan wajib): %d\n", skippedNoGaji, skippedAlreadyProcessed, skippedHasSimpananWajib)

	// Jika tidak ada yang diproses karena TIDAK ADA ANGGOTA DENGAN GAJI (bukan karena sudah dipotong),
	// catat sebagai SYSTEM agar tidak terus-menerus dicoba
	// PENTING: Jika ada anggota dengan gaji tapi sudah dipotong atau sudah ada simpanan wajib, jangan insert SYSTEM!
	if successCount == 0 && failedCount == 0 && skippedNoGaji > 0 && skippedAlreadyProcessed == 0 && skippedHasSimpananWajib == 0 {
		fmt.Printf("ℹ️ Tidak ada anggota dengan gaji yang perlu diproses. Menandai sebagai selesai.\n")

		// Insert log untuk menandai bulan ini sudah dicek
		logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
		             VALUES ('SYSTEM', $1, $2, 0, 0, 'berhasil', $3)`
		_, err := db.Exec(logQuery, bulan, tahun, fmt.Sprintf("Tidak ada anggota dengan gaji. Total anggota: %d, Skip (no gaji): %d", len(anggotas), skippedNoGaji))

		if err == nil {
			// Return successCount = 1 untuk menandai bahwa proses "berhasil" (walaupun tidak ada yang diproses)
			successCount = 1
		}
	}

	return successCount, failedCount, errors
}

// UpdateAnggotaStatus memperbarui status anggota berdasarkan ID
func UpdateAnggotaStatus(id string, status string) error {
	db := config.GetDB()

	// 🔧 Normalisasi status (WAJIB)
	status = strings.TrimSpace(strings.ToLower(status))

	query := `
		UPDATE anggota
		SET status = $1
		WHERE id_anggota = $2
	`

	_, err := db.Exec(query, status, id)
	return err
}

// // UpdateAnggotaStatus memperbarui status anggota berdasarkan ID
// func UpdateAnggotaStatus(id string, status string) error {
// 	db := config.GetDB()
// 	query := "UPDATE anggota SET status = $1 WHERE id_anggota = $2"
// 	_, err := db.Exec(query, status, id)
// 	return err
// }

// GetNomorRekening mengambil nomor rekening berdasarkan jenis (simpanan, angsuran, register)
func GetNomorRekening(jenis string) (string, error) {
	db := config.GetDB()
	var nomor string

	// 1. Coba ambil dari tabel nomor_rekening
	query := "SELECT nomor FROM nomor_rekening WHERE jenis = $1 LIMIT 1"
	err := db.QueryRow(query, jenis).Scan(&nomor)
	if err == nil && strings.TrimSpace(nomor) != "" {
		return nomor, nil
	}

	// 2. Fallback ke tabel pengaturan dengan nama spesifik per jenis
	if err == sql.ErrNoRows || strings.TrimSpace(nomor) == "" {
		err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = $1 LIMIT 1", "nomor_rekening_"+jenis).Scan(&nomor)
		if err == nil && strings.TrimSpace(nomor) != "" {
			return nomor, nil
		}
	}

	// 3. Fallback ke tabel pengaturan dengan nama umum (nomor_rekening)
	if err == sql.ErrNoRows || strings.TrimSpace(nomor) == "" {
		err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening' LIMIT 1").Scan(&nomor)
		if err == nil && strings.TrimSpace(nomor) != "" {
			return nomor, nil
		}
	}

	// 4. Jika semua tidak ditemukan, return default agar tidak kosong di UI
	if strings.TrimSpace(nomor) == "" {
		return "1234567890 (Bank ABC)", nil
	}
	return nomor, nil
}

// UpdateNomorRekening memperbarui atau menambahkan nomor rekening untuk jenis tertentu
func UpdateNomorRekening(jenis string, nomor string) error {
	db := config.GetDB()

	// Cek apakah sudah ada entri untuk jenis ini
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM nomor_rekening WHERE jenis = $1)", jenis).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update
		_, err = db.Exec("UPDATE nomor_rekening SET nomor = $1 WHERE jenis = $2", nomor, jenis)
	} else {
		// Insert
		_, err = db.Exec("INSERT INTO nomor_rekening (jenis, nomor) VALUES ($1, $2)", jenis, nomor)
	}
	return err
}

// GetTotalAnggota mengambil total anggota aktif
func GetTotalAnggota(db *sql.DB) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM anggota WHERE status = 'aktif'"
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// GetMenungguKonfirmasi mengambil jumlah anggota yang menunggu konfirmasi (status pending)
func GetMenungguKonfirmasi(db *sql.DB) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM anggota WHERE status = 'pending'"
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// BatchInsertAnggota melakukan insert batch data anggota dari file XLSX
func BatchInsertAnggota(db *sql.DB, anggotaList []models.Anggota) (successCount int, failedCount int, detailErrors []string, successfulIDs []string) {
	successCount = 0
	failedCount = 0
	detailErrors = []string{}
	successfulIDs = []string{}

	for idx, anggota := range anggotaList {
		rowNum := idx + 2 // +2 karena index 0 adalah header, dan Excel dimulai dari baris 1

		// Cek apakah username sudah ada
		var existingUsername string
		checkQuery := "SELECT COALESCE(username, '') FROM anggota WHERE username = $1 LIMIT 1"
		err := db.QueryRow(checkQuery, anggota.Username).Scan(&existingUsername)

		if err == nil {
			// Data ditemukan - ada duplikat
			if existingUsername == anggota.Username {
				detailErrors = append(detailErrors, fmt.Sprintf("Baris %d: Username '%s' sudah digunakan oleh anggota lain", rowNum, anggota.Username))
			}
			failedCount++
			continue
		}

		query := `
			INSERT INTO anggota (
				id_anggota, nama_anggota, username, password, tgl_lahir,
				no_telepon, alamat, jenis_kelamin, status_anggota,
				fakultas, tgl_gabung, unit_kerja, fakultas_code, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`

		_, err = db.Exec(query,
			anggota.IDAnggota,
			anggota.NamaAnggota,
			anggota.Username,
			anggota.Password,
			anggota.TglLahir,
			anggota.NoTelepon,
			anggota.Alamat,
			anggota.JenisKelamin,
			anggota.StatusAnggota,
			anggota.Fakultas,
			anggota.TglGabung,
			anggota.UnitKerja,
			anggota.FakultasCode,
			anggota.Status,
		)

		if err != nil {
			detailErrors = append(detailErrors, fmt.Sprintf("Baris %d: Gagal insert ke database - %s", rowNum, err.Error()))
			failedCount++
		} else {
			successCount++
			successfulIDs = append(successfulIDs, anggota.IDAnggota)
		}
	}

	return successCount, failedCount, detailErrors, successfulIDs
}

// ActivateAnggota mengaktifkan anggota dan mengisi id_anggota serta nomor_urut urut global
func ActivateAnggota(tempID string) error {
	db := config.GetDB()

	// Mulai transaksi
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Ambil data anggota berdasarkan tempID
	var unitKerja, fakultasCode, tahun string
	err = tx.QueryRow(`SELECT unit_kerja, fakultas_code, tahun FROM anggota WHERE id_anggota = $1 FOR UPDATE`, tempID).Scan(&unitKerja, &fakultasCode, &tahun)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Validasi dan pastikan kode unitKerja dan fakultasCode 2 digit
	if len(unitKerja) == 1 {
		unitKerja = "0" + unitKerja
	}
	if len(fakultasCode) == 1 {
		fakultasCode = "0" + fakultasCode
	}

	// Ambil nomor urut dari sequence database
	// Ambil nomor urut dari sequence global
	var nomorUrut int64
	err = tx.QueryRow(`SELECT nextval('anggota_nomor_urut_seq')`).Scan(&nomorUrut)
	if err != nil {
		tx.Rollback()
		return err
	}

	nomorUrutStr := fmt.Sprintf("%04d", nomorUrut)
	idAnggota := unitKerja + fakultasCode + tahun + nomorUrutStr
	// Update anggota: set id_anggota, nomor_urut, status aktif
	_, err = tx.Exec(`UPDATE anggota SET id_anggota = $1, nomor_urut = $2, status = 'aktif' WHERE id_anggota = $3`, idAnggota, nomorUrutStr, tempID)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}
