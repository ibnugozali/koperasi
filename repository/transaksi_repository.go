package repository

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"time"
)

// CreateSimpanan mencatat transaksi simpanan
func CreateSimpanan(detail models.Detail) error {
	db := config.GetDB()
	query := `
		INSERT INTO detail (id_anggota, id_simpanan, id_pengelola, tgl_transaksi, jumlah_simpanan, total_simpanan)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query,
		detail.IDAnggota,
		detail.IDSimpanan,
		detail.IDPengelola,
		detail.TglTransaksi,
		detail.JumlahSimpanan,
		detail.TotalSimpanan,
	)
	return err
}

// CreatePinjaman mencatat pinjaman baru
func CreatePinjaman(pinjaman models.Pinjaman) error {
	db := config.GetDB()
	query := `
		INSERT INTO pinjaman (id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(query,
		pinjaman.IDAnggota,
		pinjaman.IDPengelola,
		pinjaman.TglPinjaman,
		pinjaman.JumlahPinjaman,
		pinjaman.JangkaWaktu,
		pinjaman.Bunga,
		pinjaman.Status,
	)
	return err
}

// CreateAngsuran mencatat pembayaran angsuran
func CreateAngsuran(angsuran models.Angsuran) error {
	db := config.GetDB()
	query := `
		INSERT INTO angsuran (id_pinjaman, id_pengelola, tgl_bayar, sisa_pinjaman, status_angsuran, bukti_angsuran, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(query,
		angsuran.IDPinjaman,
		angsuran.IDPengelola,
		angsuran.TglBayar,
		angsuran.SisaPinjaman,
		angsuran.StatusAngsuran,
		angsuran.BuktiAngsuran,
		angsuran.Status,
	)
	return err
}

// GetAllSimpanan mengambil semua jenis simpanan
func GetAllSimpanan() ([]models.Simpanan, error) {
	db := config.GetDB()
	var simpanans []models.Simpanan
	query := "SELECT id_simpanan, jenis_simpanan FROM simpanan"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s models.Simpanan
		if err := rows.Scan(&s.IDSimpanan, &s.JenisSimpanan); err != nil {
			return nil, err
		}
		simpanans = append(simpanans, s)
	}
	return simpanans, nil
}

// GetPinjamanByID mengambil pinjaman berdasarkan ID
func GetPinjamanByID(id int) (models.Pinjaman, error) {
	db := config.GetDB()
	var p models.Pinjaman
	query := "SELECT id_pinjaman, id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status FROM pinjaman WHERE id_pinjaman = $1"
	err := db.QueryRow(query, id).Scan(&p.IDPinjaman, &p.IDAnggota, &p.IDPengelola, &p.TglPinjaman, &p.JumlahPinjaman, &p.JangkaWaktu, &p.Bunga, &p.Status)
	return p, err
}

// UpdatePinjamanStatus memperbarui status pinjaman
func UpdatePinjamanStatus(id int, status string) error {
	db := config.GetDB()
	query := "UPDATE pinjaman SET status = $1 WHERE id_pinjaman = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// GetPendingPinjaman mengambil pinjaman dengan status 'proses'
func GetPendingPinjaman() ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman
	query := `
		SELECT p.id_pinjaman, p.id_anggota, a.nama_anggota, p.tgl_pinjaman, p.jumlah_pinjaman, p.jangka_waktu, p.bunga
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.status = 'proses'
		ORDER BY p.tgl_pinjaman ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Pinjaman
		if err := rows.Scan(&p.IDPinjaman, &p.IDAnggota, &p.IDPengelola, &p.TglPinjaman, &p.JumlahPinjaman, &p.JangkaWaktu, &p.Bunga); err != nil {
			return nil, err
		}
		pinjamans = append(pinjamans, p)
	}
	return pinjamans, nil
}

// GetLaporanKeuangan menghasilkan laporan keuangan bulanan
func GetLaporanKeuangan(bulan, tahun int) (map[string]interface{}, error) {
	db := config.GetDB()
	report := make(map[string]interface{})

	// Total simpanan bulan ini
	querySimpanan := `
		SELECT COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE EXTRACT(MONTH FROM d.tgl_transaksi) = $1 AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
	`
	var totalSimpanan float64
	err := db.QueryRow(querySimpanan, bulan, tahun).Scan(&totalSimpanan)
	if err != nil {
		return nil, err
	}
	report["total_simpanan"] = totalSimpanan

	// Total pinjaman bulan ini
	queryPinjaman := `
		SELECT COALESCE(SUM(p.jumlah_pinjaman), 0)
		FROM pinjaman p
		WHERE EXTRACT(MONTH FROM p.tgl_pinjaman) = $1 AND EXTRACT(YEAR FROM p.tgl_pinjaman) = $2
	`
	var totalPinjaman float64
	err = db.QueryRow(queryPinjaman, bulan, tahun).Scan(&totalPinjaman)
	if err != nil {
		return nil, err
	}
	report["total_pinjaman"] = totalPinjaman

	// Total angsuran bulan ini
	queryAngsuran := `
		SELECT COALESCE(SUM(a.sisa_pinjaman), 0)
		FROM angsuran a
		WHERE EXTRACT(MONTH FROM a.tgl_bayar) = $1 AND EXTRACT(YEAR FROM a.tgl_bayar) = $2
	`
	var totalAngsuran float64
	err = db.QueryRow(queryAngsuran, bulan, tahun).Scan(&totalAngsuran)
	if err != nil {
		return nil, err
	}
	report["total_angsuran"] = totalAngsuran

	// Arus kas = total simpanan - total pinjaman + total angsuran
	report["arus_kas"] = totalSimpanan - totalPinjaman + totalAngsuran

	return report, nil
}

// GetAngsuranTerlambat mengambil angsuran yang terlambat (belum lunas dan tgl_bayar > tgl_pinjaman + jangka_waktu)
func GetAngsuranTerlambat() ([]map[string]interface{}, error) {
	db := config.GetDB()
	var terlambats []map[string]interface{}
	query := `
		SELECT a.nama_anggota, p.id_pinjaman, p.tgl_pinjaman, p.jangka_waktu, p.jumlah_pinjaman
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.status = 'aktif' AND p.tgl_pinjaman + INTERVAL '1 month' * p.jangka_waktu < CURRENT_DATE
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var nama string
		var idPinjaman int
		var tglPinjaman time.Time
		var jangkaWaktu int
		var jumlahPinjaman float64
		if err := rows.Scan(&nama, &idPinjaman, &tglPinjaman, &jangkaWaktu, &jumlahPinjaman); err != nil {
			return nil, err
		}
		terlambat := map[string]interface{}{
			"nama_anggota":    nama,
			"id_pinjaman":     idPinjaman,
			"tgl_pinjaman":    tglPinjaman,
			"jangka_waktu":    jangkaWaktu,
			"jumlah_pinjaman": jumlahPinjaman,
		}
		terlambats = append(terlambats, terlambat)
	}
	return terlambats, nil
}

// GetAllDetails mengambil semua detail transaksi simpanan dengan join anggota
func GetAllDetails() ([]models.Detail, error) {
	db := config.GetDB()
	var details []models.Detail
	query := `
		SELECT d.id_detail, d.id_anggota, a.nama_anggota, d.id_simpanan, s.jenis_simpanan, d.id_pengelola, d.tgl_transaksi, d.jumlah_simpanan, d.total_simpanan
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		ORDER BY d.tgl_transaksi DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Detail
		if err := rows.Scan(&d.IDDetail, &d.IDAnggota, &d.NamaAnggota, &d.IDSimpanan, &d.Simpanan.JenisSimpanan, &d.IDPengelola, &d.TglTransaksi, &d.JumlahSimpanan, &d.TotalSimpanan); err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, nil
}

// GetAllPinjamans mengambil semua pinjaman dengan join anggota
func GetAllPinjamans() ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman
	query := `
		SELECT p.id_pinjaman, p.id_anggota, a.nama_anggota, p.id_pengelola, p.tgl_pinjaman, p.jumlah_pinjaman, p.jangka_waktu, p.bunga, p.status
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		ORDER BY p.tgl_pinjaman DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Pinjaman
		if err := rows.Scan(&p.IDPinjaman, &p.IDAnggota, &p.NamaAnggota, &p.IDPengelola, &p.TglPinjaman, &p.JumlahPinjaman, &p.JangkaWaktu, &p.Bunga, &p.Status); err != nil {
			return nil, err
		}
		pinjamans = append(pinjamans, p)
	}
	return pinjamans, nil
}

// GetAllAngsurans mengambil semua angsuran dengan join anggota
func GetAllAngsurans() ([]models.Angsuran, error) {
	db := config.GetDB()
	var angsurans []models.Angsuran
	query := `
		SELECT a.id_angsuran, a.id_pinjaman, a.id_pengelola, a.tgl_bayar, a.sisa_pinjaman, a.status_angsuran, a.status, ang.nama_anggota
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		JOIN anggota ang ON p.id_anggota = ang.id_anggota
		ORDER BY a.tgl_bayar DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ang models.Angsuran
		if err := rows.Scan(&ang.IDAngsuran, &ang.IDPinjaman, &ang.IDPengelola, &ang.TglBayar, &ang.SisaPinjaman, &ang.StatusAngsuran, &ang.Status, &ang.NamaAnggota); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, ang)
	}
	return angsurans, nil
}

// GetAllRiwayat mengambil semua riwayat transaksi gabungan
func GetAllRiwayat() ([]models.Riwayat, error) {
	db := config.GetDB()
	var riwayats []models.Riwayat

	// Simpanan
	querySimpanan := `
		SELECT d.id_detail, d.tgl_transaksi, 'Simpanan' as jenis, d.jumlah_simpanan, 'Selesai' as status, a.nama_anggota
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
	`
	rows, err := db.Query(querySimpanan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Riwayat
		if err := rows.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota); err != nil {
			return nil, err
		}
		riwayats = append(riwayats, r)
	}

	// Pinjaman
	queryPinjaman := `
		SELECT p.id_pinjaman, p.tgl_pinjaman, 'Pinjaman' as jenis, p.jumlah_pinjaman, p.status, a.nama_anggota
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
	`
	rows2, err := db.Query(queryPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var r models.Riwayat
		if err := rows2.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota); err != nil {
			return nil, err
		}
		riwayats = append(riwayats, r)
	}

	// Angsuran
	queryAngsuran := `
		SELECT a.id_angsuran, a.tgl_bayar, 'Angsuran' as jenis, a.sisa_pinjaman, a.status, ang.nama_anggota
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		JOIN anggota ang ON p.id_anggota = ang.id_anggota
	`
	rows3, err := db.Query(queryAngsuran)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	for rows3.Next() {
		var r models.Riwayat
		if err := rows3.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota); err != nil {
			return nil, err
		}
		riwayats = append(riwayats, r)
	}

	return riwayats, nil
}
