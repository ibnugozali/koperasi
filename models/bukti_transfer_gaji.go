package models

import "time"

// BuktiTransferGaji mewakili bukti transfer gaji dari bendahara universitas yang diupload oleh ketua
type BuktiTransferGaji struct {
	ID           int       `json:"id" db:"id"`
	Bulan        int       `json:"bulan" db:"bulan"`
	Tahun        int       `json:"tahun" db:"tahun"`
	NamaFile     string    `json:"nama_file" db:"nama_file"`
	PathFile     string    `json:"path_file" db:"path_file"`
	DiuploadOleh int       `json:"diupload_oleh" db:"diupload_oleh"`
	TglUpload    time.Time `json:"tgl_upload" db:"tgl_upload"`
	Status       string    `json:"status" db:"status"`
	Catatan      string    `json:"catatan" db:"catatan"`
}
