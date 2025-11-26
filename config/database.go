package config

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // Driver PostgreSQL

)

var db *sql.DB

// InitDB menginisialisasi koneksi ke database PostgreSQL
func InitDB() {
	var err error
	connStr := "postgres://postgres:SuTa@localhost:5432/koperasi?sslmode=disable"

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

// GetDB mengembalikan instance koneksi database yang sudah ada
func GetDB() *sql.DB {
	return db
}
