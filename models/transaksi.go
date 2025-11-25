package models

import (
	"database/sql"
	"time"
)

// Simpanan mewakili jenis simpanan
type Simpanan struct {
	IDSimpanan    int    `json:"id_simpanan" db:"id_simpanan"`
	JenisSimpanan string `json:"jenis_simpanan" db:"jenis_simpanan"`
}

// Detail mewakili detail transaksi simpanan
type Detail struct {
	IDDetail       int       `json:"id_detail" db:"id_detail"`
	IDAnggota      string    `json:"id_anggota" db:"id_anggota"`
	NamaAnggota    string    `json:"nama_anggota" db:"nama_anggota"`
	IDSimpanan     int       `json:"id_simpanan" db:"id_simpanan"`
	Simpanan       Simpanan  `json:"simpanan" db:"simpanan"`
	IDPengelola    int       `json:"id_pengelola" db:"id_pengelola"`
	TglTransaksi   time.Time `json:"tgl_transaksi" db:"tgl_transaksi"`
	JumlahSimpanan float64   `json:"jumlah_simpanan" db:"jumlah_simpanan"`
	TotalSimpanan  float64   `json:"total_simpanan" db:"total_simpanan"`
	Status         string    `json:"status" db:"status"`
	StatusAngsuran string    `json:"status_angsuran" db:"status_angsuran"`
}

// Pinjaman mewakili pinjaman
type Pinjaman struct {
	IDPinjaman     int           `json:"id_pinjaman" db:"id_pinjaman"`
	IDAnggota      string        `json:"id_anggota" db:"id_anggota"`
	NamaAnggota    string        `json:"nama_anggota" db:"nama_anggota"`
	IDPengelola    sql.NullInt64 `json:"id_pengelola" db:"id_pengelola"`
	TglPinjaman    time.Time     `json:"tgl_pinjaman" db:"tgl_pinjaman"`
	JumlahPinjaman float64       `form:"jumlah_pinjaman" json:"jumlah_pinjaman" db:"jumlah_pinjaman"`
	JangkaWaktu    int           `form:"jangka_waktu" json:"jangka_waktu" db:"jangka_waktu"`
	Bunga          float64       `form:"bunga" json:"bunga" db:"bunga"`
	Status         string        `json:"status" db:"status"`
}

// Angsuran mewakili pembayaran angsuran
type Angsuran struct {
	IDAngsuran     int           `json:"id_angsuran" db:"id_angsuran"`
	IDPinjaman     int           `json:"id_pinjaman" db:"id_pinjaman"`
	IDPengelola    sql.NullInt64 `json:"id_pengelola" db:"id_pengelola"`
	TglBayar       time.Time     `json:"tgl_bayar" db:"tgl_bayar"`
	SisaPinjaman   float64       `json:"sisa_pinjaman" db:"sisa_pinjaman"`
	BuktiAngsuran  string        `json:"bukti_angsuran" db:"bukti_angsuran"`
	StatusAngsuran string        `json:"status_angsuran" db:"status_angsuran"`
	Status         string        `json:"status" db:"status"`
	NamaAnggota    string        `json:"nama_anggota" db:"nama_anggota"`
}

// Riwayat mewakili riwayat transaksi gabungan
type Riwayat struct {
	ID          int       `json:"id" db:"id"`
	Tanggal     time.Time `json:"tanggal" db:"tanggal"`
	Jenis       string    `json:"jenis" db:"jenis"`
	Jumlah      float64   `json:"jumlah" db:"jumlah"`
	Status      string    `json:"status" db:"status"`
	NamaAnggota string    `json:"nama_anggota" db:"nama_anggota"`
}
