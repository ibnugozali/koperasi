package repository

import (
	"database/sql"
	"time"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
)

// GetRiwayatTotalSimpananPerHari mengambil total simpanan per hari selama 30 hari terakhir
func GetRiwayatTotalSimpananPerHari(db *sql.DB) ([]map[string]interface{}, error) {
	query := `SELECT DATE(tgl_transaksi) as tanggal, SUM(jumlah_simpanan) as total FROM detail WHERE COALESCE(status, 'confirmed') = 'confirmed' AND tgl_transaksi >= CURRENT_DATE - INTERVAL '30 days' GROUP BY DATE(tgl_transaksi) ORDER BY tanggal ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var riwayat []map[string]interface{}
	for rows.Next() {
		var tanggal time.Time
		var total float64
		if err := rows.Scan(&tanggal, &total); err != nil {
			return nil, err
		}
		riwayat = append(riwayat, map[string]interface{}{
			"Tanggal": tanggal,
			"Jumlah":  total,
		})
	}
	return riwayat, nil
}

// GetRiwayatTotalPinjamanPerHari mengambil total pinjaman per hari selama 30 hari terakhir
func GetRiwayatTotalPinjamanPerHari(db *sql.DB) ([]map[string]interface{}, error) {
	query := `SELECT DATE(tgl_pinjaman) as tanggal, SUM(jumlah_pinjaman) as total FROM pinjaman WHERE status = 'aktif' AND tgl_pinjaman >= CURRENT_DATE - INTERVAL '30 days' GROUP BY DATE(tgl_pinjaman) ORDER BY tanggal ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var riwayat []map[string]interface{}
	for rows.Next() {
		var tanggal time.Time
		var total float64
		if err := rows.Scan(&tanggal, &total); err != nil {
			return nil, err
		}
		riwayat = append(riwayat, map[string]interface{}{
			"Tanggal": tanggal,
			"Jumlah":  total,
		})
	}
	return riwayat, nil
}

// GetPengambilanSimpananByID mengambil detail pengambilan simpanan berdasarkan ID
func GetPengambilanSimpananByID(id int) (models.PengambilanSimpanan, error) {
	db := config.GetDB()
	var ps models.PengambilanSimpanan
	query := `
	   SELECT ps.id_pengambilan, ps.id_anggota, a.nama_anggota, ps.id_simpanan, s.jenis_simpanan,
			  ps.jumlah, ps.alasan, ps.tgl_pengajuan, ps.tgl_proses, ps.status,
			  COALESCE(ps.catatan_bendahara, ''), ps.id_pengelola,
			  ps.no_rekening, ps.nama_bank, ps.nama_pemilik, ps.metode_pengambilan, ''
	   FROM pengambilan_simpanan ps
	   JOIN anggota a ON ps.id_anggota = a.id_anggota
	   JOIN simpanan s ON ps.id_simpanan = s.id_simpanan
	   WHERE ps.id_pengambilan = $1
	   LIMIT 1
   `
	err := db.QueryRow(query, id).Scan(
		&ps.IDPengambilan, &ps.IDAnggota, &ps.NamaAnggota, &ps.IDSimpanan, &ps.JenisSimpanan,
		&ps.Jumlah, &ps.Alasan, &ps.TglPengajuan, &ps.TglProses, &ps.Status,
		&ps.CatatanBendahara, &ps.IDPengelola,
		&ps.NomorRekening, &ps.NamaBank, &ps.NamaPemilikRekening, &ps.MetodePencairan, &ps.BuktiPengambilan,
	)
	return ps, err
}

// CreateSimpanan mencatat transaksi simpanan
func CreateSimpanan(detail models.Detail) error {
	db := config.GetDB()
	// Set waktu transaksi ke saat ini (server-side) untuk memastikan konsistensi
	detail.TglTransaksi = time.Now()
	// Set status default ke pending untuk menunggu konfirmasi bendahara
	if detail.Status == "" {
		detail.Status = "pending"
	}
	query := `
		INSERT INTO detail (id_anggota, id_simpanan, id_pengelola, tgl_transaksi, jumlah_simpanan, total_simpanan, status, bukti_pembayaran)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := db.Exec(query,
		detail.IDAnggota,
		detail.IDSimpanan,
		detail.IDPengelola,
		detail.TglTransaksi,
		detail.JumlahSimpanan,
		detail.TotalSimpanan,
		detail.Status,
		detail.BuktiPembayaran,
	)
	return err
}

// CreatePinjaman mencatat pinjaman baru
func CreatePinjaman(pinjaman models.Pinjaman) error {
	db := config.GetDB()
	// Set waktu pinjaman ke saat ini (server-side)
	pinjaman.TglPinjaman = time.Now()
	query := `
		INSERT INTO pinjaman (id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status, metode_pencairan, nomor_rekening, nama_bank, nama_pemilik_rekening, gaji_bulanan, tujuan_pinjaman)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := db.Exec(query,
		pinjaman.IDAnggota,
		pinjaman.IDPengelola,
		pinjaman.TglPinjaman,
		pinjaman.JumlahPinjaman,
		pinjaman.JangkaWaktu,
		pinjaman.Bunga,
		pinjaman.Status,
		pinjaman.MetodePencairan,
		pinjaman.NomorRekening,
		pinjaman.NamaBank,
		pinjaman.NamaPemilikRekening,
		pinjaman.GajiBulanan,
		pinjaman.TujuanPinjaman,
	)
	return err
}

// CreateAngsuran mencatat pembayaran angsuran
func CreateAngsuran(angsuran models.Angsuran) error {
	db := config.GetDB()
	// Set waktu bayar ke saat ini (server-side)
	angsuran.TglBayar = time.Now()
	// Set status default ke 'pending' untuk menunggu konfirmasi bendahara
	if angsuran.Status == "" {
		angsuran.Status = "pending"
	}
	query := `
		INSERT INTO angsuran (id_pinjaman, id_pengelola, tgl_bayar, sisa_pinjaman, bukti_angsuran, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query,
		angsuran.IDPinjaman,
		angsuran.IDPengelola,
		angsuran.TglBayar,
		angsuran.SisaPinjaman,
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

// UpdateSimpananStatus memperbarui status simpanan
func UpdateSimpananStatus(id int, status string) error {
	db := config.GetDB()
	query := "UPDATE detail SET status = $1 WHERE id_detail = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// UpdateAngsuranStatus memperbarui status angsuran
func UpdateAngsuranStatus(id int, status string) error {
	db := config.GetDB()
	query := "UPDATE angsuran SET status = $1 WHERE id_angsuran = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// UpdatePengambilanSimpananStatus memperbarui status pengambilan simpanan
func UpdatePengambilanSimpananStatus(id int, status string) error {
	db := config.GetDB()
	query := "UPDATE pengambilan_simpanan SET status = $1, tgl_proses = CURRENT_TIMESTAMP WHERE id_pengambilan = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// GetPendingPinjaman mengambil pinjaman dengan status 'proses'
// Bunga yang ditampilkan adalah bunga terkini dari tabel pengaturan, bukan dari tabel pinjaman
func GetPendingPinjaman() ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman

	// Ambil bunga terkini dari pengaturan
	var currentBunga float64
	err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&currentBunga)
	if err != nil {
		// Jika belum ada, gunakan default 2.0
		currentBunga = 2.0
	}

	query := `
		SELECT p.id_pinjaman, p.id_anggota, a.nama_anggota, p.id_pengelola, p.tgl_pinjaman, p.jumlah_pinjaman, p.jangka_waktu, p.bunga, p.status
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.status = 'proses'
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
		// Override bunga dengan nilai terkini dari pengaturan
		p.Bunga = currentBunga
		pinjamans = append(pinjamans, p)
	}
	return pinjamans, nil
}

// GetPendingSimpanan mengambil detail simpanan dengan status 'pending'
func GetPendingSimpanan() ([]models.Detail, error) {
	db := config.GetDB()
	var details []models.Detail
	query := `
		SELECT d.id_detail, d.id_anggota, a.nama_anggota, d.id_simpanan, s.jenis_simpanan, d.id_pengelola, d.tgl_transaksi, d.jumlah_simpanan, d.total_simpanan, COALESCE(d.status, '')
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.status = 'pending'
		ORDER BY d.tgl_transaksi DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.Detail
		if err := rows.Scan(&d.IDDetail, &d.IDAnggota, &d.NamaAnggota, &d.IDSimpanan, &d.Simpanan.JenisSimpanan, &d.IDPengelola, &d.TglTransaksi, &d.JumlahSimpanan, &d.TotalSimpanan, &d.Status); err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, nil
}

// GetPendingAngsuran mengambil angsuran dengan status 'pending'
func GetPendingAngsuran() ([]models.Angsuran, error) {
	db := config.GetDB()
	var angsurans []models.Angsuran
	query := `
		SELECT a.id_angsuran, a.id_pinjaman, p.id_anggota, a.id_pengelola, a.tgl_bayar, a.sisa_pinjaman, 
		       COALESCE(a.status_angsuran, ''), COALESCE(a.status, 'pending'), ang.nama_anggota
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		JOIN anggota ang ON p.id_anggota = ang.id_anggota
		WHERE a.status = 'pending'
		ORDER BY a.tgl_bayar DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Angsuran
		if err := rows.Scan(&a.IDAngsuran, &a.IDPinjaman, &a.IDAnggota, &a.IDPengelola, &a.TglBayar, &a.SisaPinjaman, &a.StatusAngsuran, &a.Status, &a.NamaAnggota); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, a)
	}
	return angsurans, nil
}

// GetLaporanKeuangan menghasilkan laporan keuangan bulanan
func GetLaporanKeuangan(bulan, tahun int) (map[string]interface{}, error) {
	db := config.GetDB()
	report := make(map[string]interface{})

	// Total simpanan bulan ini (atau tahun ini jika bulan = 0)
	querySimpanan := `
		SELECT COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE ($1 = 0 OR EXTRACT(MONTH FROM d.tgl_transaksi) = $1) AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
	`
	var totalSimpanan float64
	err := db.QueryRow(querySimpanan, bulan, tahun).Scan(&totalSimpanan)
	if err != nil {
		return nil, err
	}
	report["total_simpanan"] = totalSimpanan

	// Total pinjaman bulan ini (atau tahun ini jika bulan = 0)
	queryPinjaman := `
		SELECT COALESCE(SUM(p.jumlah_pinjaman), 0)
		FROM pinjaman p
		WHERE ($1 = 0 OR EXTRACT(MONTH FROM p.tgl_pinjaman) = $1) AND EXTRACT(YEAR FROM p.tgl_pinjaman) = $2
	`
	var totalPinjaman float64
	err = db.QueryRow(queryPinjaman, bulan, tahun).Scan(&totalPinjaman)
	if err != nil {
		return nil, err
	}
	report["total_pinjaman"] = totalPinjaman

	// Total angsuran bulan ini (atau tahun ini jika bulan = 0)
	queryAngsuran := `
		SELECT COALESCE(SUM(a.sisa_pinjaman), 0)
		FROM angsuran a
		WHERE ($1 = 0 OR EXTRACT(MONTH FROM a.tgl_bayar) = $1) AND EXTRACT(YEAR FROM a.tgl_bayar) = $2
	`
	var totalAngsuran float64
	err = db.QueryRow(queryAngsuran, bulan, tahun).Scan(&totalAngsuran)
	if err != nil {
		return nil, err
	}
	report["total_angsuran"] = totalAngsuran

	// Total pengambilan simpanan bulan ini (atau tahun ini jika bulan = 0)
	queryPengambilan := `
		SELECT COALESCE(SUM(ps.jumlah), 0)
		FROM pengambilan_simpanan ps
		WHERE ($1 = 0 OR EXTRACT(MONTH FROM ps.tgl_proses) = $1) AND EXTRACT(YEAR FROM ps.tgl_proses) = $2
		  AND ps.status = 'approved'
	`
	var totalPengambilan float64
	err = db.QueryRow(queryPengambilan, bulan, tahun).Scan(&totalPengambilan)
	if err != nil {
		return nil, err
	}
	report["total_pengambilan"] = totalPengambilan

	// Hitung arus kas: Pemasukan - Pengeluaran
	// Pemasukan: total_simpanan
	// Pengeluaran: total_pinjaman + total_pengambilan
	arusKas := totalSimpanan - (totalPinjaman + totalPengambilan)
	report["arus_kas"] = arusKas

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
		SELECT a.id_angsuran, a.id_pinjaman, p.id_anggota, a.id_pengelola, a.tgl_bayar, a.sisa_pinjaman, a.status_angsuran, a.status, ang.nama_anggota
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
		if err := rows.Scan(&ang.IDAngsuran, &ang.IDPinjaman, &ang.IDAnggota, &ang.IDPengelola, &ang.TglBayar, &ang.SisaPinjaman, &ang.StatusAngsuran, &ang.Status, &ang.NamaAnggota); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, ang)
	}
	return angsurans, nil
}

// GetUnitKerjaName mengkonversi kode unit kerja menjadi nama yang readable
func GetUnitKerjaName(unitKerja string) string {
	switch unitKerja {
	case "01":
		return "Dosen"
	case "02":
		return "Karyawan/Staff"
	case "03":
		return "Mahasiswa"
	default:
		return unitKerja // Return as-is jika tidak dikenali
	}
}

// GetAllRiwayat mengambil semua riwayat transaksi gabungan
func GetAllRiwayat() ([]models.Riwayat, error) {
	db := config.GetDB()
	var riwayats []models.Riwayat

	// Ambil data potongan bulan ini untuk semua anggota
	potonganBulanIni, err := GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64) // Default kosong jika error
	}

	// Simpanan
	querySimpanan := `
		SELECT d.id_detail, d.tgl_transaksi, s.jenis_simpanan as jenis, d.jumlah_simpanan, 'Selesai' as status, a.nama_anggota, a.id_anggota, a.no_telepon, a.unit_kerja, a.gaji_bulanan
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
	`
	rows, err := db.Query(querySimpanan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Riwayat
		if err := rows.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota, &r.IDAnggota, &r.NoTelepon, &r.UnitKerja, &r.GajiBulanan); err != nil {
			return nil, err
		}
		// Konversi kode unit kerja ke nama readable
		r.UnitKerja = GetUnitKerjaName(r.UnitKerja)
		// Hitung sisa gaji: Gaji Bulanan - Potongan Bulan Ini
		potongan := int(potonganBulanIni[r.IDAnggota])
		r.SisaGaji = r.GajiBulanan - potongan
		riwayats = append(riwayats, r)
	}

	// Pinjaman
	queryPinjaman := `
		SELECT p.id_pinjaman, p.tgl_pinjaman, 'Pinjaman' as jenis, p.jumlah_pinjaman, p.status, a.nama_anggota, a.id_anggota, a.no_telepon, a.unit_kerja, a.gaji_bulanan
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
		if err := rows2.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota, &r.IDAnggota, &r.NoTelepon, &r.UnitKerja, &r.GajiBulanan); err != nil {
			return nil, err
		}
		// Konversi kode unit kerja ke nama readable
		r.UnitKerja = GetUnitKerjaName(r.UnitKerja)
		// Hitung sisa gaji: Gaji Bulanan - Potongan Bulan Ini
		potongan := int(potonganBulanIni[r.IDAnggota])
		r.SisaGaji = r.GajiBulanan - potongan
		riwayats = append(riwayats, r)
	}

	// Angsuran
	queryAngsuran := `
		SELECT a.id_angsuran, a.tgl_bayar, 'Angsuran' as jenis, a.sisa_pinjaman, a.status, ang.nama_anggota, ang.id_anggota, ang.no_telepon, ang.unit_kerja, ang.gaji_bulanan
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
		if err := rows3.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota, &r.IDAnggota, &r.NoTelepon, &r.UnitKerja, &r.GajiBulanan); err != nil {
			return nil, err
		}
		// Hitung sisa gaji: Gaji Bulanan - Potongan Bulan Ini
		potongan := int(potonganBulanIni[r.IDAnggota])
		r.SisaGaji = r.GajiBulanan - potongan
		riwayats = append(riwayats, r)
	}

	// Anggota aktif (untuk menampilkan semua anggota di laporan)
	queryAnggota := `
		SELECT '0' as id, a.tgl_gabung as tanggal, 'Pendaftaran' as jenis, 0.0 as jumlah, 'Aktif' as status,
		       a.nama_anggota, a.id_anggota, a.no_telepon, a.unit_kerja, a.gaji_bulanan
		FROM anggota a
		WHERE a.status = 'aktif'
	`
	rows4, err := db.Query(queryAnggota)
	if err != nil {
		return nil, err
	}
	defer rows4.Close()

	for rows4.Next() {
		var r models.Riwayat
		if err := rows4.Scan(&r.ID, &r.Tanggal, &r.Jenis, &r.Jumlah, &r.Status, &r.NamaAnggota, &r.IDAnggota, &r.NoTelepon, &r.UnitKerja, &r.GajiBulanan); err != nil {
			return nil, err
		}
		// Konversi kode unit kerja ke nama readable
		r.UnitKerja = GetUnitKerjaName(r.UnitKerja)
		// Hitung sisa gaji: Gaji Bulanan - Potongan Bulan Ini
		potongan := int(potonganBulanIni[r.IDAnggota])
		r.SisaGaji = r.GajiBulanan - potongan
		riwayats = append(riwayats, r)
	}

	return riwayats, nil
}

// GetTotalSimpanan mengambil total simpanan semua anggota yang sudah dikonfirmasi
func GetTotalSimpanan(db *sql.DB) (float64, error) {
	var total float64
	query := "SELECT COALESCE(SUM(jumlah_simpanan), 0) FROM detail WHERE COALESCE(status, 'confirmed') = 'confirmed'"
	err := db.QueryRow(query).Scan(&total)
	return total, err
}

// GetTotalPinjaman mengambil total pinjaman semua anggota yang aktif
func GetTotalPinjaman(db *sql.DB) (float64, error) {
	var total float64
	query := "SELECT COALESCE(SUM(jumlah_pinjaman), 0) FROM pinjaman WHERE status = 'aktif'"
	err := db.QueryRow(query).Scan(&total)
	return total, err
}

// GetTotalAngsuran mengambil total angsuran yang sudah dibayarkan
func GetTotalAngsuran(db *sql.DB) (float64, error) {
	var total float64
	query := "SELECT COALESCE(SUM(sisa_pinjaman), 0) FROM angsuran WHERE COALESCE(status, 'confirmed') = 'confirmed'"
	err := db.QueryRow(query).Scan(&total)
	return total, err
}

// GetTotalPengambilan mengambil total pengambilan simpanan yang sudah disetujui
func GetTotalPengambilan(db *sql.DB) (float64, error) {
	var total float64
	query := "SELECT COALESCE(SUM(jumlah), 0) FROM pengambilan_simpanan WHERE status = 'approved'"
	err := db.QueryRow(query).Scan(&total)
	return total, err
}

// GetAktivitasTerbaru mengambil aktivitas terbaru untuk grafik (simpanan, pinjaman, angsuran dalam 30 hari terakhir)
func GetAktivitasTerbaru(db *sql.DB) ([]map[string]interface{}, error) {
	var aktivitas []map[string]interface{}

	// Simpanan terbaru
	querySimpanan := `
		SELECT d.tgl_transaksi, 'Simpanan' as jenis, d.jumlah_simpanan
		FROM detail d
		WHERE d.tgl_transaksi >= CURRENT_DATE - INTERVAL '30 days'
		ORDER BY d.tgl_transaksi DESC
		LIMIT 10
	`
	rows, err := db.Query(querySimpanan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tanggal time.Time
		var jenis string
		var jumlah float64
		if err := rows.Scan(&tanggal, &jenis, &jumlah); err != nil {
			return nil, err
		}
		aktivitas = append(aktivitas, map[string]interface{}{
			"Tanggal": tanggal,
			"Jenis":   jenis,
			"Jumlah":  jumlah,
		})
	}

	// Pinjaman terbaru
	queryPinjaman := `
		SELECT p.tgl_pinjaman, 'Pinjaman' as jenis, p.jumlah_pinjaman
		FROM pinjaman p
		WHERE p.tgl_pinjaman >= CURRENT_DATE - INTERVAL '30 days'
		ORDER BY p.tgl_pinjaman DESC
		LIMIT 10
	`
	rows2, err := db.Query(queryPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var tanggal time.Time
		var jenis string
		var jumlah float64
		if err := rows2.Scan(&tanggal, &jenis, &jumlah); err != nil {
			return nil, err
		}
		aktivitas = append(aktivitas, map[string]interface{}{
			"Tanggal": tanggal,
			"Jenis":   jenis,
			"Jumlah":  jumlah,
		})
	}

	// Angsuran terbaru
	queryAngsuran := `
		SELECT a.tgl_bayar, 'Angsuran' as jenis, a.sisa_pinjaman
		FROM angsuran a
		WHERE a.tgl_bayar >= CURRENT_DATE - INTERVAL '30 days'
		ORDER BY a.tgl_bayar DESC
		LIMIT 10
	`
	rows3, err := db.Query(queryAngsuran)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	for rows3.Next() {
		var tanggal time.Time
		var jenis string
		var jumlah float64
		if err := rows3.Scan(&tanggal, &jenis, &jumlah); err != nil {
			return nil, err
		}
		aktivitas = append(aktivitas, map[string]interface{}{
			"Tanggal": tanggal,
			"Jenis":   jenis,
			"Jumlah":  jumlah,
		})
	}

	return aktivitas, nil
}

func GetPinjamanAktifByAnggotaID(idAnggota string) ([]models.Pinjaman, error) {
	db := config.GetDB()
	var pinjamans []models.Pinjaman
	query := "SELECT id_pinjaman, id_anggota, id_pengelola, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status FROM pinjaman WHERE id_anggota = $1 AND (status = 'aktif' OR status = 'proses')"
	rows, err := db.Query(query, idAnggota)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Pinjaman
		if err := rows.Scan(&p.IDPinjaman, &p.IDAnggota, &p.IDPengelola, &p.TglPinjaman, &p.JumlahPinjaman, &p.JangkaWaktu, &p.Bunga, &p.Status); err != nil {
			return nil, err
		}
		pinjamans = append(pinjamans, p)
	}
	return pinjamans, nil
}

// GetAngsuranByPinjamanID mengambil angsuran berdasarkan ID pinjaman
func GetAngsuranByPinjamanID(idPinjaman int) ([]models.Angsuran, error) {
	db := config.GetDB()
	var angsurans []models.Angsuran
	query := "SELECT id_angsuran, id_pinjaman, id_pengelola, tgl_bayar, sisa_pinjaman, COALESCE(bukti_angsuran, ''), status FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar DESC"
	rows, err := db.Query(query, idPinjaman)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a models.Angsuran
		if err := rows.Scan(&a.IDAngsuran, &a.IDPinjaman, &a.IDPengelola, &a.TglBayar, &a.SisaPinjaman, &a.BuktiAngsuran, &a.Status); err != nil {
			return nil, err
		}
		angsurans = append(angsurans, a)
	}
	return angsurans, nil
}

// GetPendingPengambilanSimpanan mengambil pengambilan simpanan dengan status 'pending'
func GetPendingPengambilanSimpanan() ([]models.PengambilanSimpanan, error) {
	db := config.GetDB()
	var pengambilans []models.PengambilanSimpanan
	query := `
		SELECT ps.id_pengambilan, ps.id_anggota, a.nama_anggota, ps.id_simpanan, s.jenis_simpanan, 
		       ps.jumlah, ps.alasan, ps.tgl_pengajuan, ps.tgl_proses, ps.status, 
		       COALESCE(ps.catatan_bendahara, ''), ps.id_pengelola
		FROM pengambilan_simpanan ps
		JOIN anggota a ON ps.id_anggota = a.id_anggota
		JOIN simpanan s ON ps.id_simpanan = s.id_simpanan
		WHERE ps.status = 'pending'
		ORDER BY ps.tgl_pengajuan DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ps models.PengambilanSimpanan
		if err := rows.Scan(&ps.IDPengambilan, &ps.IDAnggota, &ps.NamaAnggota, &ps.IDSimpanan, &ps.JenisSimpanan,
			&ps.Jumlah, &ps.Alasan, &ps.TglPengajuan, &ps.TglProses, &ps.Status,
			&ps.CatatanBendahara, &ps.IDPengelola); err != nil {
			return nil, err
		}
		pengambilans = append(pengambilans, ps)
	}
	return pengambilans, nil
}

// GetRiwayatPengambilanSimpananByAnggotaID mengambil riwayat pengambilan simpanan berdasarkan ID anggota
func GetRiwayatPengambilanSimpananByAnggotaID(idAnggota string, search string) ([]models.PengambilanSimpanan, error) {
	db := config.GetDB()
	var pengambilans []models.PengambilanSimpanan

	query := `
		SELECT ps.id_pengambilan, ps.id_anggota, a.nama_anggota, ps.id_simpanan, s.jenis_simpanan, 
		       ps.jumlah, ps.alasan, ps.tgl_pengajuan, ps.tgl_proses, ps.status, 
		       COALESCE(ps.catatan_bendahara, ''), ps.id_pengelola
		FROM pengambilan_simpanan ps
		JOIN anggota a ON ps.id_anggota = a.id_anggota
		JOIN simpanan s ON ps.id_simpanan = s.id_simpanan
		WHERE ps.id_anggota = $1
		ORDER BY ps.tgl_pengajuan DESC, ps.id_pengambilan DESC
	`

	rows, err := db.Query(query, idAnggota)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ps models.PengambilanSimpanan
		if err := rows.Scan(&ps.IDPengambilan, &ps.IDAnggota, &ps.NamaAnggota, &ps.IDSimpanan, &ps.JenisSimpanan,
			&ps.Jumlah, &ps.Alasan, &ps.TglPengajuan, &ps.TglProses, &ps.Status,
			&ps.CatatanBendahara, &ps.IDPengelola); err != nil {
			return nil, err
		}
		pengambilans = append(pengambilans, ps)
	}
	return pengambilans, nil
}

// GetLaporanBulananPerAnggota mengambil laporan detail bulanan per anggota
func GetLaporanBulananPerAnggota(bulan, tahun int) ([]map[string]interface{}, error) {
	db := config.GetDB()
	var reports []map[string]interface{}

	// Query untuk mendapatkan data per anggota aktif
	query := `
		SELECT 
			a.id_anggota,
			a.nama_anggota,
			a.unit_kerja,
			-- Simpanan Pokok (dari tabel detail dengan jenis Pokok)
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'pokok'
			), 0) as simpanan_pokok,
			-- Simpanan Wajib bulan ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'wajib'
				AND ($1 = 0 OR EXTRACT(MONTH FROM d.tgl_transaksi) = $1) AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
			), 0) as simpanan_wajib_bulanan,
			-- Total Simpanan Wajib sampai saat ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'wajib'
			), 0) as total_simpanan_wajib,
			-- Simpanan Hari Raya bulan ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'hari_raya'
				AND ($1 = 0 OR EXTRACT(MONTH FROM d.tgl_transaksi) = $1) AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
			), 0) as simpanan_hariraya_bulanan,
			-- Total Simpanan Hari Raya sampai saat ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'hari_raya'
			), 0) as total_simpanan_hariraya,
			-- Simpanan Sukarela bulan ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'sukarela'
				AND ($1 = 0 OR EXTRACT(MONTH FROM d.tgl_transaksi) = $1) AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
			), 0) as simpanan_sukarela_bulanan,
			-- Total Simpanan Sukarela sampai saat ini
			COALESCE((SELECT SUM(d.jumlah_simpanan) 
				FROM detail d 
				JOIN simpanan s ON d.id_simpanan = s.id_simpanan
				WHERE d.id_anggota = a.id_anggota 
				AND s.jenis_simpanan = 'sukarela'
			), 0) as total_simpanan_sukarela,
			-- Pinjaman yang diambil bulan ini
			COALESCE((SELECT SUM(p.jumlah_pinjaman) 
				FROM pinjaman p
				WHERE p.id_anggota = a.id_anggota 
				AND ($1 = 0 OR EXTRACT(MONTH FROM p.tgl_pinjaman) = $1) AND EXTRACT(YEAR FROM p.tgl_pinjaman) = $2
			), 0) as pinjaman_bulanan,
			-- Total Pinjaman Aktif (belum lunas)
			COALESCE((SELECT SUM(p.jumlah_pinjaman) 
				FROM pinjaman p
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
			), 0) as total_pinjaman_aktif,
			-- Sisa Pinjaman (jumlah pinjaman - total angsuran yang sudah dibayar)
			COALESCE((SELECT p.jumlah_pinjaman - COALESCE((SELECT SUM(ang.sisa_pinjaman) FROM angsuran ang WHERE ang.id_pinjaman = p.id_pinjaman), 0)
				FROM pinjaman p
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
				ORDER BY p.tgl_pinjaman DESC
				LIMIT 1
			), 0) as sisa_pinjaman,
			-- Angsuran bulan ini (total pembayaran angsuran pada bulan ini)
			COALESCE((SELECT COUNT(*) * (
				SELECT (p.jumlah_pinjaman / p.jangka_waktu) + ((p.jumlah_pinjaman * p.bunga / 100) / p.jangka_waktu)
				FROM pinjaman p
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
				ORDER BY p.tgl_pinjaman DESC
				LIMIT 1
			)
				FROM angsuran ang
				JOIN pinjaman p ON ang.id_pinjaman = p.id_pinjaman
				WHERE p.id_anggota = a.id_anggota 
				AND ($1 = 0 OR EXTRACT(MONTH FROM ang.tgl_bayar) = $1) AND EXTRACT(YEAR FROM ang.tgl_bayar) = $2
			), 0) as angsuran_bulanan,
			-- Total Angsuran yang sudah dibayar
			COALESCE((SELECT COUNT(*) 
				FROM angsuran ang
				JOIN pinjaman p ON ang.id_pinjaman = p.id_pinjaman
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
			), 0) as total_angsuran_dibayar,
			-- Sisa Angsuran yang belum dibayar
			COALESCE((
				SELECT p.jangka_waktu - COUNT(ang.id_angsuran)
				FROM pinjaman p
				LEFT JOIN angsuran ang ON p.id_pinjaman = ang.id_pinjaman
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
				AND p.id_pinjaman = (
					SELECT id_pinjaman FROM pinjaman 
					WHERE id_anggota = a.id_anggota AND status = 'aktif' 
					ORDER BY tgl_pinjaman DESC LIMIT 1
				)
				GROUP BY p.jangka_waktu
			), 0) as sisa_angsuran,
			-- Jangka Waktu/Tenor (dari pinjaman aktif terbaru)
			COALESCE((SELECT p.jangka_waktu 
				FROM pinjaman p
				WHERE p.id_anggota = a.id_anggota 
				AND p.status = 'aktif'
				ORDER BY p.tgl_pinjaman DESC
				LIMIT 1
			), 0) as jangka_waktu,
			-- Pokok Pinjaman per bulan
			CASE 
				WHEN (SELECT p.jangka_waktu FROM pinjaman p WHERE p.id_anggota = a.id_anggota AND p.status = 'aktif' ORDER BY p.tgl_pinjaman DESC LIMIT 1) > 0
				THEN COALESCE((SELECT p.jumlah_pinjaman / p.jangka_waktu FROM pinjaman p WHERE p.id_anggota = a.id_anggota AND p.status = 'aktif' ORDER BY p.tgl_pinjaman DESC LIMIT 1), 0)
				ELSE 0
			END as pokok_per_bulan,
			-- Jasa/Bunga per bulan
			CASE 
				WHEN (SELECT p.jangka_waktu FROM pinjaman p WHERE p.id_anggota = a.id_anggota AND p.status = 'aktif' ORDER BY p.tgl_pinjaman DESC LIMIT 1) > 0
				THEN COALESCE((SELECT (p.jumlah_pinjaman * p.bunga / 100) / p.jangka_waktu FROM pinjaman p WHERE p.id_anggota = a.id_anggota AND p.status = 'aktif' ORDER BY p.tgl_pinjaman DESC LIMIT 1), 0)
				ELSE 0
			END as jasa_per_bulan
		FROM anggota a
		WHERE a.status_anggota IN ('dosen', 'karyawan')
		ORDER BY a.id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var report map[string]interface{} = make(map[string]interface{})
		var idAnggota, unitKerja string
		var namaAnggota string
		var simpananPokok, simpananWajibBulanan, totalSimpananWajib float64
		var simpananHariRayaBulanan, totalSimpananHariRaya float64
		var simpananSukarelaBulanan, totalSimpananSukarela float64
		var pinjamanBulanan, totalPinjamanAktif, sisaPinjaman float64
		var angsuranBulanan float64
		var totalAngsuranDibayar, sisaAngsuran, jangkaWaktu int
		var pokokPerBulan, jasaPerBulan float64

		if err := rows.Scan(
			&idAnggota, &namaAnggota, &unitKerja,
			&simpananPokok,
			&simpananWajibBulanan, &totalSimpananWajib,
			&simpananHariRayaBulanan, &totalSimpananHariRaya,
			&simpananSukarelaBulanan, &totalSimpananSukarela,
			&pinjamanBulanan, &totalPinjamanAktif, &sisaPinjaman,
			&angsuranBulanan, &totalAngsuranDibayar, &sisaAngsuran,
			&jangkaWaktu, &pokokPerBulan, &jasaPerBulan,
		); err != nil {
			return nil, err
		}

		// Hitung jumlah angsuran per bulan (pokok + jasa)
		jumlahAngsuranPerBulan := pokokPerBulan + jasaPerBulan

		// Hitung total pembayaran bulan ini (simpanan + angsuran)
		totalPembayaran := simpananWajibBulanan + simpananHariRayaBulanan + simpananSukarelaBulanan + angsuranBulanan

		report["id_anggota"] = idAnggota
		report["nama_anggota"] = namaAnggota
		report["unit_kerja"] = unitKerja
		report["simpanan_pokok"] = simpananPokok
		report["simpanan_wajib_bulanan"] = simpananWajibBulanan
		report["total_simpanan_wajib"] = totalSimpananWajib
		report["simpanan_hariraya_bulanan"] = simpananHariRayaBulanan
		report["total_simpanan_hariraya"] = totalSimpananHariRaya
		report["simpanan_sukarela_bulanan"] = simpananSukarelaBulanan
		report["total_simpanan_sukarela"] = totalSimpananSukarela
		report["pinjaman_bulanan"] = pinjamanBulanan
		report["total_pinjaman_aktif"] = totalPinjamanAktif
		report["sisa_pinjaman"] = sisaPinjaman
		report["angsuran_bulanan"] = angsuranBulanan
		report["total_angsuran_dibayar"] = totalAngsuranDibayar
		report["sisa_angsuran"] = sisaAngsuran
		report["jangka_waktu"] = jangkaWaktu
		report["pokok_per_bulan"] = pokokPerBulan
		report["jasa_per_bulan"] = jasaPerBulan
		report["jumlah_angsuran_per_bulan"] = jumlahAngsuranPerBulan
		report["total_pembayaran"] = totalPembayaran

		reports = append(reports, report)
	}

	return reports, nil
}
