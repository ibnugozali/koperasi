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

func GetPengelolaByID(id int) (models.Pengelola, error) {
	db := config.GetDB()
	var p models.Pengelola
	query := "SELECT id_pengelola, nama_pengelola, username, password, jabatan_koperasi, no_telepon, email, tgl_gabung, level, status FROM pengelola WHERE id_pengelola = $1"
	err := db.QueryRow(query, id).Scan(&p.IDPengelola, &p.NamaPengelola, &p.Username, &p.Password, &p.JabatanKoperasi, &p.NoTelepon, &p.Email, &p.TglGabung, &p.Level, &p.Status)
	return p, err
}

func UpdatePengelolaUsernamePassword(id int, username, password string) error {
	db := config.GetDB()
	query := "UPDATE pengelola SET username = $1, password = $2 WHERE id_pengelola = $3"
	_, err := db.Exec(query, username, password, id)
	return err
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
