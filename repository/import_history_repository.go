package repository

import (
	"database/sql"

	"koperasi-simpan-pinjam/models"
)

// SaveImportHistory menyimpan riwayat import ke database
func SaveImportHistory(db *sql.DB, history models.ImportHistory) error {
	query := `
		INSERT INTO import_history 
		(id_import, id_pengelola, username, file_name, total_data, success_count, failed_count, imported_data, parse_errors, tanggal_import) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := db.Exec(query,
		history.IDImport,
		history.IDPengelola,
		history.Username,
		history.FileName,
		history.TotalData,
		history.SuccessCount,
		history.FailedCount,
		history.ImportedData,
		history.ParseErrors,
		history.TanggalImport,
	)
	return err
}

// GetLatestImportHistory mengambil riwayat import terbaru untuk user tertentu
func GetLatestImportHistory(db *sql.DB, idPengelola int) (*models.ImportHistory, error) {
	query := `
		SELECT id_import, id_pengelola, username, file_name, total_data, success_count, failed_count, 
		       imported_data, parse_errors, tanggal_import 
		FROM import_history 
		WHERE id_pengelola = $1 
		ORDER BY tanggal_import DESC 
		LIMIT 1
	`

	var history models.ImportHistory
	err := db.QueryRow(query, idPengelola).Scan(
		&history.IDImport,
		&history.IDPengelola,
		&history.Username,
		&history.FileName,
		&history.TotalData,
		&history.SuccessCount,
		&history.FailedCount,
		&history.ImportedData,
		&history.ParseErrors,
		&history.TanggalImport,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &history, nil
}

// GetAllImportHistory mengambil semua riwayat import untuk user tertentu
func GetAllImportHistory(db *sql.DB, idPengelola int, limit int) ([]models.ImportHistory, error) {
	query := `
		SELECT id_import, id_pengelola, username, file_name, total_data, success_count, failed_count, 
		       imported_data, parse_errors, tanggal_import 
		FROM import_history 
		WHERE id_pengelola = $1 
		ORDER BY tanggal_import DESC 
		LIMIT $2
	`

	rows, err := db.Query(query, idPengelola, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []models.ImportHistory
	for rows.Next() {
		var history models.ImportHistory
		err := rows.Scan(
			&history.IDImport,
			&history.IDPengelola,
			&history.Username,
			&history.FileName,
			&history.TotalData,
			&history.SuccessCount,
			&history.FailedCount,
			&history.ImportedData,
			&history.ParseErrors,
			&history.TanggalImport,
		)
		if err != nil {
			return nil, err
		}
		histories = append(histories, history)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return histories, nil
}

// DeleteImportHistory menghapus riwayat import berdasarkan ID
func DeleteImportHistory(db *sql.DB, idImport string) error {
	query := `DELETE FROM import_history WHERE id_import = $1`
	_, err := db.Exec(query, idImport)
	return err
}

// DeleteAllImportHistoryByPengelola menghapus semua riwayat import untuk pengelola tertentu
func DeleteAllImportHistoryByPengelola(db *sql.DB, idPengelola int) error {
	query := `DELETE FROM import_history WHERE id_pengelola = $1`
	_, err := db.Exec(query, idPengelola)
	return err
}

// UpdateImportHistory memperbarui data import history
func UpdateImportHistory(db *sql.DB, history *models.ImportHistory) error {
	query := `
		UPDATE import_history 
		SET imported_data = $1, 
		    success_count = $2, 
		    failed_count = $3,
		    total_data = $4
		WHERE id_import = $5
	`
	_, err := db.Exec(query,
		history.ImportedData,
		history.SuccessCount,
		history.FailedCount,
		history.TotalData,
		history.IDImport,
	)
	return err
}
