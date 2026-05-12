package models

import "time"

// ReferensiPendaftaran menyimpan data master hasil import admin
// untuk validasi calon anggota saat proses register.
type ReferensiPendaftaran struct {
	ID                int
	NamaLengkap       string
	NomorIdentitas    string
	Jabatan           string
	GajiBulanan       int
	StatusKeanggotaan string
	SumberFile        string
	ImportedAt        time.Time
	UpdatedAt         time.Time
}
