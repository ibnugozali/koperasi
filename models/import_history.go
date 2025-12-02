package models

import "time"

type ImportHistory struct {
	IDImport      string    `db:"id_import"`
	IDPengelola   int       `db:"id_pengelola"`
	Username      string    `db:"username"`
	FileName      string    `db:"file_name"`
	TotalData     int       `db:"total_data"`
	SuccessCount  int       `db:"success_count"`
	FailedCount   int       `db:"failed_count"`
	ImportedData  string    `db:"imported_data"` // JSON string
	ParseErrors   string    `db:"parse_errors"`  // JSON string
	TanggalImport time.Time `db:"tanggal_import"`
}
