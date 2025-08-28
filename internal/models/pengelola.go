package models

import "time"

type Pengelola struct {
	IDPengelola   uint      `gorm:"primaryKey" json:"id_pengelola"`
	NamaPengelola string    `json:"nama_pengelola"`
	Password      string    `json:"password"`
	Jabatan       string    `json:"jabatan"` // administrator, bendahara, pembina
	NoTelepon     int       `json:"no_telepon"`
	Email         string    `json:"email"`
	TglGabung     time.Time `json:"tgl_gabung"`
	Level         string    `json:"level"`
	Status        string    `json:"status"`
	Role          string    `gorm:"default:pengelola" json:"role"` // Base role, jabatan untuk sub-role
}