package models

import (
	"time"
)

// Anggota merepresentasikan struktur data untuk tabel anggota di database.
type Anggota struct {
	IDAnggota     string     `json:"id_anggota"`
	NamaAnggota   string     `json:"nama_anggota" form:"NamaAnggota"`
	Username      string     `json:"username" form:"Username"`
	Password      string     `json:"password" form:"Password"`
	TglLahir      string     `json:"tgl_lahir" form:"TglLahir"`
	NikKTP        string     `json:"nik_ktp" form:"NikKTP"`
	NoTelepon     string     `json:"no_telepon" form:"NoTelepon"`
	TglGabung     time.Time  `json:"tgl_gabung"`
	TglKeluar     *time.Time `json:"tgl_keluar"`
	Alamat        string     `json:"alamat" form:"Alamat"`
	Provinsi      string     `json:"provinsi" form:"Provinsi"`
	JenisKelamin  string     `json:"jenis_kelamin" form:"JenisKelamin"`
	StatusAnggota string     `json:"status_anggota" form:"StatusAnggota"`
	Fakultas      string     `json:"fakultas" form:"Fakultas"`
	Status        string     `json:"status"`
	UnitKerja     string     `json:"unit_kerja"`
	FakultasCode  string     `json:"fakultas_code"`
	Tahun         string     `json:"tahun"`
	NomorUrut     string     `json:"nomor_urut"`
	BuktiTransfer string     `json:"bukti_transfer"`
	GajiBulanan   int        `json:"gaji_bulanan" form:"GajiBulanan"`
}
