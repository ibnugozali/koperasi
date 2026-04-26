package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

// GetBuktiTransferGajiByBulanTahun mengambil bukti transfer gaji berdasarkan bulan dan tahun
func GetBuktiTransferGajiByBulanTahun(bulan, tahun int) (*models.BuktiTransferGaji, error) {
	db := config.GetDB()
	var bukti models.BuktiTransferGaji
	query := `
		SELECT id, bulan, tahun, nama_file, path_file, diupload_oleh, tgl_upload, status, catatan
		FROM bukti_transfer_gaji
		WHERE bulan = $1 AND tahun = $2
	`
	err := db.QueryRow(query, bulan, tahun).Scan(
		&bukti.ID, &bukti.Bulan, &bukti.Tahun, &bukti.NamaFile, &bukti.PathFile,
		&bukti.DiuploadOleh, &bukti.TglUpload, &bukti.Status, &bukti.Catatan,
	)
	if err != nil {
		return nil, err
	}
	return &bukti, nil
}

// CheckBuktiTransferGajiExists memeriksa apakah bukti transfer gaji sudah ada untuk bulan dan tahun tertentu
func CheckBuktiTransferGajiExists(bulan, tahun int) (bool, error) {
	db := config.GetDB()
	var count int
	query := `SELECT COUNT(*) FROM bukti_transfer_gaji WHERE bulan = $1 AND tahun = $2`
	err := db.QueryRow(query, bulan, tahun).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SaveBuktiTransferGaji menyimpan bukti transfer gaji baru
func SaveBuktiTransferGaji(bukti *models.BuktiTransferGaji) error {
	db := config.GetDB()
	query := `
		INSERT INTO bukti_transfer_gaji (bulan, tahun, nama_file, path_file, diupload_oleh, status, catatan)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (bulan, tahun) DO UPDATE SET
			nama_file = EXCLUDED.nama_file,
			path_file = EXCLUDED.path_file,
			diupload_oleh = EXCLUDED.diupload_oleh,
			tgl_upload = CURRENT_TIMESTAMP,
			status = EXCLUDED.status,
			catatan = EXCLUDED.catatan
		RETURNING id, tgl_upload
	`
	return db.QueryRow(query, bukti.Bulan, bukti.Tahun, bukti.NamaFile, bukti.PathFile,
		bukti.DiuploadOleh, bukti.Status, bukti.Catatan).Scan(&bukti.ID, &bukti.TglUpload)
}

// GetAllBuktiTransferGaji mengambil semua bukti transfer gaji
func GetAllBuktiTransferGaji() ([]models.BuktiTransferGaji, error) {
	db := config.GetDB()
	var buktiList []models.BuktiTransferGaji
	query := `
		SELECT id, bulan, tahun, nama_file, path_file, diupload_oleh, tgl_upload, status, catatan
		FROM bukti_transfer_gaji
		ORDER BY tahun DESC, bulan DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bukti models.BuktiTransferGaji
		if err := rows.Scan(
			&bukti.ID, &bukti.Bulan, &bukti.Tahun, &bukti.NamaFile, &bukti.PathFile,
			&bukti.DiuploadOleh, &bukti.TglUpload, &bukti.Status, &bukti.Catatan,
		); err != nil {
			return nil, err
		}
		buktiList = append(buktiList, bukti)
	}
	return buktiList, nil
}
