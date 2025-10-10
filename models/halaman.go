package models

import "time"

// Halaman merepresentasikan data dari tabel halaman
type Halaman struct {
	ID        int       `json:"id"`
	Slug      string    `json:"slug" form:"slug"`
	Judul     string    `json:"judul" form:"judul"`
	Konten    string    `json:"konten" form:"konten"`
	UpdatedAt time.Time `json:"updated_at"`
}