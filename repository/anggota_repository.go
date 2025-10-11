package repository

import (
	"koperasi-simpan-pinjam/config" // Ganti dengan path config Anda
	"koperasi-simpan-pinjam/models" // Ganti dengan path models Anda

)

// Mengambil semua anggota dengan status pending
func GetPendingAnggota() ([]models.Anggota, error) {
	db := config.GetDB() // Fungsi untuk mendapatkan koneksi DB
	var anggotas []models.Anggota

	rows, err := db.Query("SELECT id_anggota, nama_anggota, nik_ktp, tgl_gabung FROM anggota WHERE status = 'pending' ORDER BY tgl_gabung ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota, &a.NikKTP, &a.TglGabung); err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// Update status anggota dan tambahkan kode anggota
func UpdateAnggotaStatus(id int, newStatus string, memberCode string) error {
	db := config.GetDB()
	_, err := db.Exec("UPDATE anggota SET status = $1, kode_anggota = $2 WHERE id_anggota = $3", newStatus, memberCode, id)
	return err
}

// ... (fungsi lainnya)
// Membuat anggota baru (registrasi)
func CreateAnggota(anggota models.Anggota) error {
	db := config.GetDB()
	query := `
		INSERT INTO anggota (nama_anggota, username, password, tgl_lahir, nik_ktp, no_telepon, provinsi, jenis_kelamin, status_anggota, fakultas, tgl_gabung)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := db.Exec(query,
		anggota.NamaAnggota,
		anggota.Username,
		anggota.Password,
		anggota.TglLahir,
		anggota.NikKTP,
		anggota.NoTelepon,
		anggota.Provinsi,
		anggota.JenisKelamin,
		anggota.StatusAnggota,
		anggota.Fakultas,
		anggota.TglGabung,
	)
	return err
}

// Di file: repository/anggota_repository.go

// ... (fungsi-fungsi lain yang sudah ada)

// GetAnggotaByID mengambil data lengkap anggota berdasarkan ID
func GetAnggotaByID(id int) (models.Anggota, error) {
	db := config.GetDB()
	var a models.Anggota
	query := `
		SELECT
			id_anggota, nama_anggota, username,
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			provinsi, jenis_kelamin, status, kode_anggota
		FROM anggota
		WHERE id_anggota = $1`

	// Perhatikan penggunaan &a.KodeAnggota karena tipenya sql.NullString
	err := db.QueryRow(query, id).Scan(
		&a.IDAnggota, &a.NamaAnggota, &a.Username,
		&a.TglLahir, &a.NikKTP, &a.NoTelepon, &a.TglGabung,
		&a.Provinsi, &a.JenisKelamin, &a.Status, &a.KodeAnggota,
	)
	return a, err
}

// GetAllAnggota mengambil semua anggota aktif
func GetAllAnggota() ([]models.Anggota, error) {
	db := config.GetDB()
	var anggotas []models.Anggota

	query := `
		SELECT
			id_anggota, nama_anggota, username,
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			provinsi, jenis_kelamin, status, kode_anggota
		FROM anggota
		WHERE status = 'aktif'
		ORDER BY tgl_gabung DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		err := rows.Scan(
			&a.IDAnggota, &a.NamaAnggota, &a.Username,
			&a.TglLahir, &a.NikKTP, &a.NoTelepon, &a.TglGabung,
			&a.Provinsi, &a.JenisKelamin, &a.Status, &a.KodeAnggota,
		)
		if err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// DeleteAnggota menghapus anggota berdasarkan ID (soft delete atau hard delete tergantung kebutuhan)
func DeleteAnggota(id int) error {
	db := config.GetDB()
	_, err := db.Exec("DELETE FROM anggota WHERE id_anggota = $1", id)
	return err
}
