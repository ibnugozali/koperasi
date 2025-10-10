package models

import (
	"database/sql"
	"time"
)

// Anggota merepresentasikan struktur data untuk tabel anggota di database.
type Anggota struct {
	IDAnggota    int            `json:"id_anggota"`
	NamaAnggota  string         `json:"nama_anggota" form:"NamaAnggota"`
	Username     string         `json:"username" form:"Username"`
	Password     string         `json:"password" form:"Password"`
	TglLahir     string         `json:"tgl_lahir" form:"TglLahir"`
	NikKTP       string         `json:"nik_ktp" form:"NikKTP"`
	NoTelepon    string         `json:"no_telepon" form:"NoTelepon"`
	TglGabung    time.Time      `json:"tgl_gabung"`
	Provinsi     string         `json:"provinsi" form:"Provinsi"`
	JenisKelamin string         `json:"jenis_kelamin" form:"JenisKelamin"`
	Status       string         `json:"status"`
	KodeAnggota  sql.NullString `json:"kode_anggota"` // Menggunakan sql.NullString karena bisa jadi NULL
}
