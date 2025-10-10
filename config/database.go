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
}

// GetDB mengembalikan instance koneksi database yang sudah ada
func GetDB() *sql.DB {
	return db
}
