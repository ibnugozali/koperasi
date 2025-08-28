package models

import "time"

type Anggota struct {
	IDAnggota    uint      `gorm:"primaryKey" json:"id_anggota"`
	NamaAnggota  string    `json:"nama_anggota"`
	Password     string    `json:"password"` // Hash in service
	TglLahir     time.Time `json:"tgl_lahir"`
	TlbKtr       int       `json:"tlb_ktr"` // Sesuaikan nama jika typo
	NikKrp       int       `json:"nik_krp"`
	NoTelepon    int       `json:"no_telepon"`
	Provesi      string    `json:"provesi"`
	JenisKelamin string    `json:"jenis_kelamin"`
	Status       string    `json:"status"`
	Role         string    `gorm:"default:anggota" json:"role"` // Tambah untuk auth
}