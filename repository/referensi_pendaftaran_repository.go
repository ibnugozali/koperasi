package repository

import (
	"database/sql"
	"strings"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

const syncReferensiStatusFromAnggotaSQL = `
	UPDATE referensi_pendaftaran r
	SET status_keanggotaan = 'anggota',
	    updated_at = CURRENT_TIMESTAMP
	WHERE LOWER(TRIM(COALESCE(r.status_keanggotaan, ''))) <> 'anggota'
	  AND EXISTS (
		SELECT 1
		FROM anggota a
		WHERE (
			COALESCE(r.nomor_identitas, '') <> ''
			AND COALESCE(a.username, '') = COALESCE(r.nomor_identitas, '')
		)
		OR (
			LOWER(TRIM(COALESCE(a.nama_anggota, ''))) = LOWER(TRIM(COALESCE(r.nama_lengkap, '')))
			AND COALESCE(a.gaji_bulanan, 0) = COALESCE(r.gaji_bulanan, 0)
		)
	  )
`

func SyncReferensiPendaftaranStatusFromAnggota() error {
	db := config.GetDB()
	_, err := db.Exec(syncReferensiStatusFromAnggotaSQL)
	return err
}

func UpsertReferensiPendaftaran(item models.ReferensiPendaftaran) error {
	db := config.GetDB()

	nomorIdentitas := strings.TrimSpace(item.NomorIdentitas)
	nama := strings.TrimSpace(item.NamaLengkap)

	var existingID int
	var err error

	switch {
	case nomorIdentitas != "":
		err = db.QueryRow(`SELECT id FROM referensi_pendaftaran WHERE COALESCE(nomor_identitas, '') = $1 LIMIT 1`, nomorIdentitas).Scan(&existingID)
	case nama != "":
		err = db.QueryRow(`
			SELECT id
			FROM referensi_pendaftaran
			WHERE LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($1))
			LIMIT 1
		`, nama).Scan(&existingID)
	default:
		err = sql.ErrNoRows
	}

	if err == nil && existingID > 0 {
		_, err = db.Exec(`
			UPDATE referensi_pendaftaran
			SET nama_lengkap = $1,
			    nomor_identitas = $2,
			    jabatan = $3,
			    gaji_bulanan = $4,
			    status_keanggotaan = $5,
			    sumber_file = $6,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $7
		`,
			nama,
			nomorIdentitas,
			strings.TrimSpace(item.Jabatan),
			item.GajiBulanan,
			strings.TrimSpace(item.StatusKeanggotaan),
			strings.TrimSpace(item.SumberFile),
			existingID,
		)
		if err != nil {
			return err
		}
		return SyncReferensiPendaftaranStatusFromAnggota()
	}

	_, err = db.Exec(`
		INSERT INTO referensi_pendaftaran (
			nama_lengkap, nomor_identitas, jabatan, gaji_bulanan, status_keanggotaan, sumber_file
		) VALUES ($1, $2, $3, $4, $5, $6)
	`,
		nama,
		nomorIdentitas,
		strings.TrimSpace(item.Jabatan),
		item.GajiBulanan,
		strings.TrimSpace(item.StatusKeanggotaan),
		strings.TrimSpace(item.SumberFile),
	)
	if err != nil {
		return err
	}
	return SyncReferensiPendaftaranStatusFromAnggota()
}

func FindReferensiPendaftaranForRegister(nama, identitas string, gaji int) (*models.ReferensiPendaftaran, error) {
	db := config.GetDB()

	nama = strings.TrimSpace(nama)
	identitas = strings.TrimSpace(identitas)

	row := db.QueryRow(`
		SELECT id, nama_lengkap, COALESCE(nomor_identitas, ''), COALESCE(jabatan, ''), COALESCE(gaji_bulanan, 0),
		       COALESCE(status_keanggotaan, ''), COALESCE(sumber_file, ''), imported_at, updated_at
		FROM referensi_pendaftaran
		WHERE LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($1))
		  AND COALESCE(nomor_identitas, '') = $2
		  AND COALESCE(gaji_bulanan, 0) = $3
		ORDER BY updated_at DESC, imported_at DESC
		LIMIT 1
	`, nama, identitas, gaji)

	var item models.ReferensiPendaftaran
	err := row.Scan(
		&item.ID,
		&item.NamaLengkap,
		&item.NomorIdentitas,
		&item.Jabatan,
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

	query := `
		SELECT id, nama_lengkap, COALESCE(nomor_identitas, ''), COALESCE(jabatan, ''), COALESCE(gaji_bulanan, 0),
		       COALESCE(status_keanggotaan, ''), COALESCE(sumber_file, ''), imported_at, updated_at
		FROM referensi_pendaftaran
		WHERE COALESCE(nomor_identitas, '') = $1
	`

	args := []interface{}{identitas}
	if nama != "" {
		query += ` AND LOWER(TRIM(COALESCE(nama_lengkap, ''))) = LOWER(TRIM($2))`
		args = append(args, nama)
	}

	query += `
		ORDER BY updated_at DESC, imported_at DESC
		LIMIT 1
	`

	row := db.QueryRow(query, args...)

	var item models.ReferensiPendaftaran
	err := row.Scan(
		&item.ID,
		&item.NamaLengkap,
		&item.NomorIdentitas,
		&item.Jabatan,
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

func FindReferensiPendaftaranByIdentitas(identitas string) (*models.ReferensiPendaftaran, error) {
	db := config.GetDB()

	identitas = strings.TrimSpace(identitas)

	row := db.QueryRow(`
		SELECT id, nama_lengkap, COALESCE(nomor_identitas, ''), COALESCE(jabatan, ''), COALESCE(gaji_bulanan, 0),
		       COALESCE(status_keanggotaan, ''), COALESCE(sumber_file, ''), imported_at, updated_at
		FROM referensi_pendaftaran
		WHERE COALESCE(nomor_identitas, '') = $1
		ORDER BY
			CASE WHEN COALESCE(jabatan, '') <> '' THEN 0 ELSE 1 END,
			updated_at DESC,
			imported_at DESC
		LIMIT 1
	`, identitas)

	var item models.ReferensiPendaftaran
	err := row.Scan(
		&item.ID,
		&item.NamaLengkap,
		&item.NomorIdentitas,
		&item.Jabatan,
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
