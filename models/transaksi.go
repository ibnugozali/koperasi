package models

import (
	"database/sql"
	"time"
)

// Simpanan merepresentasikan jenis simpanan (misal: pokok, wajib, sukarela)
type Simpanan struct {
	IDSimpanan    int    `json:"id_simpanan"`
	JenisSimpanan string `json:"jenis_simpanan"`
}

// Detail merepresentasikan detail transaksi simpanan anggota
type Detail struct {
	IDDetail       int       `json:"id_detail"`
	IDAnggota      string    `json:"id_anggota"`
	NamaAnggota    string    `json:"nama_anggota"`
	IDSimpanan     int       `json:"id_simpanan"`
	IDPengelola    int       `json:"id_pengelola"`
	TglTransaksi   time.Time `json:"tgl_transaksi"`
	JumlahSimpanan float64   `json:"jumlah_simpanan"`
	TotalSimpanan  float64   `json:"total_simpanan"`
	Simpanan       Simpanan  `json:"simpanan"`
}

// Pinjaman merepresentasikan data pinjaman anggota
type Pinjaman struct {
	IDPinjaman     int           `json:"id_pinjaman"`
	IDAnggota      string        `json:"id_anggota"`
	NamaAnggota    string        `json:"nama_anggota"`
	IDPengelola    sql.NullInt64 `json:"id_pengelola"` // Bisa NULL
	TglPinjaman    time.Time     `json:"tgl_pinjaman"`
	JumlahPinjaman float64       `json:"jumlah_pinjaman" form:"jumlah_pinjaman"`
	JangkaWaktu    int           `json:"jangka_waktu" form:"jangka_waktu"` // Dalam bulan
	Bunga          float64       `json:"bunga" form:"bunga"`
	Status         string        `json:"status"` // (proses, aktif, lunas, gagal)
}

// Angsuran merepresentasikan data pembayaran angsuran pinjaman
type Angsuran struct {
	IDAngsuran     int           `json:"id_angsuran"`
	IDPinjaman     int           `json:"id_pinjaman"`
	IDPengelola    sql.NullInt64 `json:"id_pengelola"` // Bisa NULL
	TglBayar       time.Time     `json:"tgl_bayar"`
	SisaPinjaman   float64       `json:"sisa_pinjaman"`
	StatusAngsuran string        `json:"status_angsuran"` // (belum_lunas, lunas, terlambat)
	BuktiAngsuran  []byte        `json:"bukti_angsuran"`  // Untuk data biner
	Status         string        `json:"status"`          // (valid, invalid)
	NamaAnggota    string        `json:"nama_anggota"`
}

// Riwayat merepresentasikan riwayat transaksi gabungan
type Riwayat struct {
	ID          int       `json:"id"`
	Tanggal     time.Time `json:"tanggal"`
	Jenis       string    `json:"jenis"`
	Jumlah      float64   `json:"jumlah"`
	Status      string    `json:"status"`
	NamaAnggota string    `json:"nama_anggota"`
}
