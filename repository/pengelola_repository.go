package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

func GetPengelolaByUsername(username string) (models.Pengelola, error) {
	db := config.GetDB()
	var p models.Pengelola
	query := "SELECT id_pengelola, username, password, level FROM pengelola WHERE username = $1 AND status = 'aktif'"
	err := db.QueryRow(query, username).Scan(&p.IDPengelola, &p.Username, &p.Password, &p.Level)
	return p, err
}
// ... (fungsi lainnya)

func GetAnggotaByUsername(username string) (models.Anggota, error) {
	db := config.GetDB()
	var a models.Anggota
	// Hanya anggota 'aktif' yang bisa login
	query := "SELECT id_anggota, username, password FROM anggota WHERE username = $1 AND status = 'aktif'"
	err := db.QueryRow(query, username).Scan(&a.IDAnggota, &a.Username, &a.Password)
	return a, err
}