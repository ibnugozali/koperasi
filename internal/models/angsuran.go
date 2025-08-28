package models

import "time"

type Angsuran struct {
	IDAngsuran    uint   `gorm:"primaryKey" json:"id_angsuran"`
	IDPinjaman    uint   `json:"id_pinjaman"`
	IDPengelola   uint   `json:"id_pengelola"`
	TglBayar      time.Time `json:"tgl_bayar"`
	SisaPinjaman  string `json:"sisa_pinjaman"` // Varchar di diagram
	StatusAngsuran string `json:"status_angsuran"`
	BuktiAngsuran string `json:"bukti_angsuran"`
	Status        string `json:"status"`
}