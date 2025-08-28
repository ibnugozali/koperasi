package models

import "time"

type Pinjaman struct {
	IDPinjaman     uint      `gorm:"primaryKey" json:"id_pinjaman"`
	IDAnggota      uint      `json:"id_anggota"`
	IDPengelola    uint      `json:"id_pengelola"`
	JumlahPinjaman int       `json:"jumlah_pinjaman"`
	TglPinjaman    time.Time `json:"tgl_pinjaman"`
	JangkaWaktu    time.Time `json:"jangka_waktu"` // Jatuh tempo
	Bunga          int       `json:"bunga"`
	Status         string    `json:"status"`
}