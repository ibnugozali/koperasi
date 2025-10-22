package models

import (
	"time"
)

// Pesan merepresentasikan struktur data untuk tabel pesan di database.
type Pesan struct {
	IDPesan   int       `json:"id_pesan"`
	IDAnggota int       `json:"id_anggota"`
	Judul     string    `json:"judul" form:"Judul"`
	Isi       string    `json:"isi" form:"Isi"`
	TglKirim  time.Time `json:"tgl_kirim"`
	Status    string    `json:"status"`
}
