package models

import "time"

type Simpanan struct {
	IDSimpanan     uint      `gorm:"primaryKey" json:"id_simpanan"`
	IDAnggota      uint      `json:"id_anggota"`
	JenisSimpanan  string    `json:"jenis_simpanan"`
	JumlahSimpanan int       `json:"jumlah_simpanan"`
	TglSimpanan    time.Time `json:"tgl_simpanan"`
}