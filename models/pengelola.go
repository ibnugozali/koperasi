package models

import "time"

// Pengelola merepresentasikan struktur data untuk tabel pengelola di database.
type Pengelola struct {
	IDPengelola     int       `json:"id_pengelola"`
	NamaPengelola   string    `json:"nama_pengelola"`
	Username        string    `json:"username"`
	Password        string    `json:"password"`
	PlainPassword   string    `json:"plain_password"`
	JabatanKoperasi string    `json:"jabatan_koperasi"`
	NoTelepon       string    `json:"no_telepon"`
	Email           string    `json:"email"`
	TglGabung       time.Time `json:"tgl_gabung"`
	Level           string    `json:"level"`
	Status          string    `json:"status"`
}
