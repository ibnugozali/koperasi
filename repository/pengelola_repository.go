package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

// GetBendahara mengambil satu data bendahara yang memiliki nomor telepon.
// Prioritas diberikan pada bendahara berstatus aktif.
func GetBendahara() (models.Pengelola, error) {
	db := config.GetDB()
	var p models.Pengelola
	err := db.QueryRow(`
		SELECT id_pengelola, nama_pengelola, COALESCE(no_telepon, '')
		FROM pengelola
		WHERE LOWER(TRIM(level)) = 'bendahara'
		  AND TRIM(COALESCE(no_telepon, '')) <> ''
		ORDER BY
		  CASE WHEN LOWER(TRIM(COALESCE(status, ''))) = 'aktif' THEN 0 ELSE 1 END,
		  id_pengelola ASC
		LIMIT 1
	`).Scan(&p.IDPengelola, &p.NamaPengelola, &p.NoTelepon)
	return p, err
}

// GetAllPengelola mengambil semua data pengelola (user) dari tabel pengelola
func GetAllPengelola() ([]models.Pengelola, error) {
	db := config.GetDB()
	var pengelolas []models.Pengelola
	rows, err := db.Query("SELECT id_pengelola, nama_pengelola, username, password, COALESCE(plain_password, ''), jabatan_koperasi, COALESCE(no_telepon, ''), COALESCE(email, ''), tgl_gabung, level, status FROM pengelola ORDER BY id_pengelola")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p models.Pengelola
		err := rows.Scan(&p.IDPengelola, &p.NamaPengelola, &p.Username, &p.Password, &p.PlainPassword, &p.JabatanKoperasi, &p.NoTelepon, &p.Email, &p.TglGabung, &p.Level, &p.Status)
		if err != nil {
			return nil, err
		}
		pengelolas = append(pengelolas, p)
	}
	return pengelolas, nil
}

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
	query := "SELECT id_pengelola, nama_pengelola, username, password, COALESCE(plain_password, '') as plain_password, jabatan_koperasi, COALESCE(no_telepon, '') as no_telepon, COALESCE(email, '') as email, tgl_gabung, level, status FROM pengelola WHERE id_pengelola = $1"
	err := db.QueryRow(query, id).Scan(&p.IDPengelola, &p.NamaPengelola, &p.Username, &p.Password, &p.PlainPassword, &p.JabatanKoperasi, &p.NoTelepon, &p.Email, &p.TglGabung, &p.Level, &p.Status)
	return p, err
}

func UpdatePengelolaUsernamePassword(id int, username, password, plainPassword string) error {
	db := config.GetDB()
	query := "UPDATE pengelola SET username = $1, password = $2, plain_password = $3 WHERE id_pengelola = $4"
	_, err := db.Exec(query, username, password, plainPassword, id)
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
