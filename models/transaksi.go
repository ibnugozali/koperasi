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
	IDDetail        int       `json:"id_detail" db:"id_detail"`
	IDAnggota       string    `json:"id_anggota" db:"id_anggota"`
	NamaAnggota     string    `json:"nama_anggota" db:"nama_anggota"`
	IDSimpanan      int       `json:"id_simpanan" db:"id_simpanan"`
	Simpanan        Simpanan  `json:"simpanan" db:"simpanan"`
	IDPengelola     int       `json:"id_pengelola" db:"id_pengelola"`
	TglTransaksi    time.Time `json:"tgl_transaksi" db:"tgl_transaksi"`
	JumlahSimpanan  float64   `json:"jumlah_simpanan" db:"jumlah_simpanan"`
	TotalSimpanan   float64   `json:"total_simpanan" db:"total_simpanan"`
	Status          string    `json:"status" db:"status"`
	StatusAngsuran  string    `json:"status_angsuran" db:"status_angsuran"`
	BuktiPembayaran string    `json:"bukti_pembayaran" db:"bukti_pembayaran"`
}

// Pinjaman mewakili pinjaman
type Pinjaman struct {
	IDPinjaman          int           `json:"id_pinjaman" db:"id_pinjaman"`
	IDAnggota           string        `json:"id_anggota" db:"id_anggota"`
	NamaAnggota         string        `json:"nama_anggota" db:"nama_anggota"`
	IDPengelola         sql.NullInt64 `json:"id_pengelola" db:"id_pengelola"`
	TglPinjaman         time.Time     `json:"tgl_pinjaman" db:"tgl_pinjaman"`
	JumlahPinjaman      float64       `form:"jumlah_pinjaman" json:"jumlah_pinjaman" db:"jumlah_pinjaman"`
	JangkaWaktu         int           `form:"jangka_waktu" json:"jangka_waktu" db:"jangka_waktu"`
	Bunga               float64       `form:"bunga" json:"bunga" db:"bunga"`
	Status              string        `json:"status" db:"status"`
	MetodePencairan     string        `form:"metode_pencairan" json:"metode_pencairan" db:"metode_pencairan"`
	NomorRekening       string        `form:"nomor_rekening" json:"nomor_rekening" db:"nomor_rekening"`
	NamaBank            string        `form:"nama_bank" json:"nama_bank" db:"nama_bank"`
	NamaPemilikRekening string        `form:"nama_pemilik" json:"nama_pemilik_rekening" db:"nama_pemilik_rekening"`
	GajiBulanan         float64       `form:"gaji_bulanan" json:"gaji_bulanan" db:"gaji_bulanan"`
	TujuanPinjaman      string        `form:"tujuan_pinjaman" json:"tujuan_pinjaman" db:"tujuan_pinjaman"`
}

// Angsuran mewakili pembayaran angsuran
type Angsuran struct {
	IDAngsuran     int           `json:"id_angsuran" db:"id_angsuran"`
	IDPinjaman     int           `json:"id_pinjaman" db:"id_pinjaman"`
	IDAnggota      string        `json:"id_anggota" db:"id_anggota"`
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
	IDAnggota   string    `json:"id_anggota" db:"id_anggota"`
	NoTelepon   string    `json:"no_telepon" db:"no_telepon"`
	GajiBulanan int       `json:"gaji_bulanan" db:"gaji_bulanan"`
}

// PengambilanSimpanan mewakili pengajuan pengambilan simpanan
type PengambilanSimpanan struct {
	IDPengambilan       int           `json:"id_pengambilan" db:"id_pengambilan"`
	IDAnggota           string        `json:"id_anggota" db:"id_anggota"`
	NamaAnggota         string        `json:"nama_anggota" db:"nama_anggota"`
	IDSimpanan          int           `json:"id_simpanan" db:"id_simpanan"`
	JenisSimpanan       string        `json:"jenis_simpanan" db:"jenis_simpanan"`
	Jumlah              float64       `json:"jumlah" db:"jumlah"`
	Alasan              string        `json:"alasan" db:"alasan"`
	TglPengajuan        time.Time     `json:"tgl_pengajuan" db:"tgl_pengajuan"`
	TglProses           sql.NullTime  `json:"tgl_proses" db:"tgl_proses"`
	Status              string        `json:"status" db:"status"`
	CatatanBendahara    string        `json:"catatan_bendahara" db:"catatan_bendahara"`
	IDPengelola         sql.NullInt64 `json:"id_pengelola" db:"id_pengelola"`
	NomorRekening       string        `json:"nomor_rekening" db:"nomor_rekening"`
	NamaBank            string        `json:"nama_bank" db:"nama_bank"`
	NamaPemilikRekening string        `json:"nama_pemilik_rekening" db:"nama_pemilik_rekening"`
	MetodePencairan     string        `json:"metode_pencairan" db:"metode_pencairan"`
	BuktiPengambilan    string        `json:"bukti_pengambilan" db:"bukti_pengambilan"`
}
