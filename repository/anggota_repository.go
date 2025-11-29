package repository

import (
	"database/sql"

	"koperasi-simpan-pinjam/config" // Ganti dengan path config Anda
	"koperasi-simpan-pinjam/models" // Ganti dengan path models Anda
)

// Mengambil semua anggota dengan status pending
func GetPendingAnggota() ([]models.Anggota, error) {
	db := config.GetDB() // Fungsi untuk mendapatkan koneksi DB
	var anggotas []models.Anggota

	rows, err := db.Query("SELECT id_anggota, nama_anggota, username, nik_ktp, no_telepon, tgl_gabung, unit_kerja, fakultas_code, COALESCE(fakultas, '') FROM anggota WHERE status = 'pending' ORDER BY tgl_gabung ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota, &a.Username, &a.NikKTP, &a.NoTelepon, &a.TglGabung, &a.UnitKerja, &a.FakultasCode, &a.Fakultas); err != nil {
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
	query := `
		INSERT INTO anggota (id_anggota, nama_anggota, username, password, tgl_lahir, nik_ktp, no_telepon, alamat, jenis_kelamin, status_anggota, fakultas, tgl_gabung, unit_kerja, fakultas_code, bukti_transfer)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := db.Exec(query,
		anggota.IDAnggota,
		anggota.NamaAnggota,
		anggota.Username,
		anggota.Password,
		anggota.TglLahir,
		anggota.NikKTP,
		anggota.NoTelepon,
		anggota.Alamat,
		anggota.JenisKelamin,
		anggota.StatusAnggota,
		anggota.Fakultas,
		anggota.TglGabung,
		anggota.UnitKerja,
		anggota.FakultasCode,
		anggota.BuktiTransfer,
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
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			alamat, jenis_kelamin, status,
			unit_kerja, fakultas_code, COALESCE(tahun, ''), COALESCE(nomor_urut, ''), COALESCE(bukti_transfer, ''), COALESCE(status_anggota, ''), COALESCE(fakultas, '')
		FROM anggota
		WHERE id_anggota = $1`

	err := db.QueryRow(query, id).Scan(
		&a.IDAnggota, &a.NamaAnggota, &a.Username, &encryptedPassword,
		&a.TglLahir, &a.NikKTP, &a.NoTelepon, &a.TglGabung,
		&a.Alamat, &a.JenisKelamin, &a.Status,
		&a.UnitKerja, &a.FakultasCode, &a.Tahun, &a.NomorUrut, &a.BuktiTransfer, &a.StatusAnggota, &a.Fakultas,
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
			id_anggota, nama_anggota, username,
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			alamat, jenis_kelamin, status, unit_kerja, fakultas
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
			&a.Alamat, &a.JenisKelamin, &a.Status, &a.UnitKerja, &a.Fakultas,
		)
		if err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// DeleteAnggota menghapus anggota berdasarkan ID (soft delete atau hard delete tergantung kebutuhan)
func DeleteAnggota(id string) error {
	db := config.GetDB()
	_, err := db.Exec("DELETE FROM anggota WHERE id_anggota = $1", id)
	return err
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
func GetSaldoAnggota(id string) (totalSimpanan, totalPinjaman, saldoBersih float64, err error) {
	db := config.GetDB()

	// Hitung total simpanan dari detail
	querySimpanan := `
		SELECT COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE d.id_anggota = $1
	`
	err = db.QueryRow(querySimpanan, id).Scan(&totalSimpanan)
	if err != nil {
		return 0, 0, 0, err
	}

	// Hitung total pinjaman yang belum lunas (status != 'lunas')
	queryPinjaman := `
		SELECT COALESCE(SUM(p.jumlah_pinjaman), 0)
		FROM pinjaman p
		WHERE p.id_anggota = $1 AND p.status != 'lunas'
	`
	err = db.QueryRow(queryPinjaman, id).Scan(&totalPinjaman)
	if err != nil {
		return 0, 0, 0, err
	}

	// Saldo bersih = total simpanan - total pinjaman belum lunas
	saldoBersih = totalSimpanan - totalPinjaman

	return totalSimpanan, totalPinjaman, saldoBersih, nil
}

// UpdateAnggotaStatus memperbarui status anggota berdasarkan ID
func UpdateAnggotaStatus(id string, status string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET status = $1 WHERE id_anggota = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// GetNomorRekening mengambil nomor rekening berdasarkan jenis (simpanan, angsuran, register)
func GetNomorRekening(jenis string) (string, error) {
	db := config.GetDB()
	var nomor string
	query := "SELECT nomor FROM nomor_rekening WHERE jenis = $1 LIMIT 1"
	err := db.QueryRow(query, jenis).Scan(&nomor)
	if err == sql.ErrNoRows {
		// Jika tidak ada, return string kosong dan no error
		return "", nil
	}
	return nomor, err
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
