package models

import "time"

type Detail struct {
	IDDetail       uint      `gorm:"primaryKey" json:"id_detail"`
	IDAnggota      uint      `json:"id_anggota"`
	IDPengelola    uint      `json:"id_pengelola"`
	TglTransaksi   time.Time `json:"tgl_transaksi"`
	JumlahSimpanan int       `json:"jumlah_simpanan"`
	TotalSimpanan  int       `json:"total_simpanan"`
}