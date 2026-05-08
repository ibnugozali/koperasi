package repository

import (
	"database/sql"
	"strings"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

func normalizeReferensiValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeReferensiPhone(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	if strings.HasPrefix(value, "+62") {
		value = "0" + strings.TrimPrefix(value, "+62")
	}
	if strings.HasPrefix(value, "62") {
		value = "0" + strings.TrimPrefix(value, "62")
	}
	return value
}

func UpsertReferensiPendaftaran(item models.ReferensiPendaftaran) error {
	db := config.GetDB()

	nik := strings.TrimSpace(item.NIKKTP)
	telepon := normalizeReferensiPhone(item.NoTelepon)
	nama := strings.TrimSpace(item.NamaLengkap)

	var existingID int
	var err error

	switch {
	case nik != "":
		err = db.QueryRow(`SELECT id FROM referensi_pendaftaran WHERE COALESCE(nik_ktp, '') = $1 LIMIT 1`, nik).Scan(&existingID)
	case telepon != "" && nama != "":
		err = db.QueryRow(`
			SELECT id
			FROM referensi_pendaftaran
			WHERE LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($1))
			  AND COALESCE(no_telepon, '') = $2
			LIMIT 1
		`, nama, telepon).Scan(&existingID)
	default:
		err = sql.ErrNoRows
	}

	if err == nil && existingID > 0 {
		_, err = db.Exec(`
			UPDATE referensi_pendaftaran
			SET nama_lengkap = $1,
			    nik_ktp = $2,
			    no_telepon = $3,
			    tgl_lahir = $4,
			    jenis_kelamin = $5,
			    status_anggota = $6,
			    fakultas = $7,
			    alamat = $8,
			    gaji_bulanan = $9,
			    status_keanggotaan = $10,
			    sumber_file = $11,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $12
		`,
			nama,
			nik,
			telepon,
			strings.TrimSpace(item.TglLahir),
			strings.TrimSpace(item.JenisKelamin),
			strings.TrimSpace(item.StatusAnggota),
			strings.TrimSpace(item.Fakultas),
			strings.TrimSpace(item.Alamat),
			item.GajiBulanan,
			strings.TrimSpace(item.StatusKeanggotaan),
			strings.TrimSpace(item.SumberFile),
			existingID,
		)
		return err
	}

	_, err = db.Exec(`
		INSERT INTO referensi_pendaftaran (
			nama_lengkap, nik_ktp, no_telepon, tgl_lahir, jenis_kelamin,
			status_anggota, fakultas, alamat, gaji_bulanan, status_keanggotaan, sumber_file
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		nama,
		nik,
		telepon,
		strings.TrimSpace(item.TglLahir),
		strings.TrimSpace(item.JenisKelamin),
		strings.TrimSpace(item.StatusAnggota),
		strings.TrimSpace(item.Fakultas),
		strings.TrimSpace(item.Alamat),
		item.GajiBulanan,
		strings.TrimSpace(item.StatusKeanggotaan),
		strings.TrimSpace(item.SumberFile),
	)
	return err
}

func FindReferensiPendaftaranForRegister(nama, identitas string, gaji int) (*models.ReferensiPendaftaran, error) {
	db := config.GetDB()

	nama = strings.TrimSpace(nama)
	identitas = strings.TrimSpace(identitas)

	row := db.QueryRow(`
		SELECT id, nama_lengkap, COALESCE(nik_ktp, ''), COALESCE(no_telepon, ''), COALESCE(tgl_lahir, ''),
		       COALESCE(jenis_kelamin, ''), COALESCE(status_anggota, ''), COALESCE(fakultas, ''),
		       COALESCE(alamat, ''), COALESCE(gaji_bulanan, 0), COALESCE(status_keanggotaan, ''),
		       COALESCE(sumber_file, ''), imported_at, updated_at
		FROM referensi_pendaftaran
		WHERE LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($1))
		  AND COALESCE(nik_ktp, '') = $2
		  AND COALESCE(gaji_bulanan, 0) = $3
		ORDER BY updated_at DESC, imported_at DESC
		LIMIT 1
	`, nama, identitas, gaji)

	var item models.ReferensiPendaftaran
	err := row.Scan(
		&item.ID,
		&item.NamaLengkap,
		&item.NIKKTP,
		&item.NoTelepon,
		&item.TglLahir,
		&item.JenisKelamin,
		&item.StatusAnggota,
		&item.Fakultas,
		&item.Alamat,
		&item.GajiBulanan,
		&item.StatusKeanggotaan,
		&item.SumberFile,
		&item.ImportedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func FindReferensiPendaftaranForAutofill(nama, identitas string) (*models.ReferensiPendaftaran, error) {
	db := config.GetDB()

	nama = strings.TrimSpace(nama)
	identitas = strings.TrimSpace(identitas)

	row := db.QueryRow(`
		SELECT id, nama_lengkap, COALESCE(nik_ktp, ''), COALESCE(no_telepon, ''), COALESCE(tgl_lahir, ''),
		       COALESCE(jenis_kelamin, ''), COALESCE(status_anggota, ''), COALESCE(fakultas, ''),
		       COALESCE(alamat, ''), COALESCE(gaji_bulanan, 0), COALESCE(status_keanggotaan, ''),
		       COALESCE(sumber_file, ''), imported_at, updated_at
		FROM referensi_pendaftaran
		WHERE LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($1))
		  AND COALESCE(nik_ktp, '') = $2
		ORDER BY updated_at DESC, imported_at DESC
		LIMIT 1
	`, nama, identitas)

	var item models.ReferensiPendaftaran
	err := row.Scan(
		&item.ID,
		&item.NamaLengkap,
		&item.NIKKTP,
		&item.NoTelepon,
		&item.TglLahir,
		&item.JenisKelamin,
		&item.StatusAnggota,
		&item.Fakultas,
		&item.Alamat,
		&item.GajiBulanan,
		&item.StatusKeanggotaan,
		&item.SumberFile,
		&item.ImportedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func CountReferensiPendaftaran() (int, error) {
	db := config.GetDB()
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM referensi_pendaftaran`).Scan(&count)
	return count, err
}

func CountReferensiPendaftaranByStatus(status string) (int, error) {
	db := config.GetDB()
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM referensi_pendaftaran
		WHERE LOWER(TRIM(COALESCE(status_keanggotaan, ''))) = LOWER(TRIM($1))
	`, status).Scan(&count)
	return count, err
}
