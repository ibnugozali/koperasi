package repository

import (
	"database/sql"
	"fmt"
	"time"

	"koperasi-simpan-pinjam/config" // Ganti dengan path config Anda
	"koperasi-simpan-pinjam/models" // Ganti dengan path models Anda
)

// Mengambil semua anggota dengan status pending
func GetPendingAnggota() ([]models.Anggota, error) {
	db := config.GetDB() // Fungsi untuk mendapatkan koneksi DB
	var anggotas []models.Anggota

	rows, err := db.Query("SELECT id_anggota, nama_anggota, username, nik_ktp, no_telepon, tgl_gabung, unit_kerja, fakultas_code, COALESCE(fakultas, '') FROM anggota WHERE status = 'pending' ORDER BY tgl_gabung ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota, &a.Username, &a.NikKTP, &a.NoTelepon, &a.TglGabung, &a.UnitKerja, &a.FakultasCode, &a.Fakultas); err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// Update status anggota dan tambahkan kode anggota
func UpdateAnggotaStatusWithCode(id string, newStatus string, memberCode string) error {
	db := config.GetDB()
	_, err := db.Exec("UPDATE anggota SET status = $1, kode_anggota = $2 WHERE id_anggota = $3", newStatus, memberCode, id)
	return err
}

// ... (fungsi lainnya)
// Membuat anggota baru (registrasi)
func CreateAnggota(anggota models.Anggota) error {
	db := config.GetDB()
	query := `
		INSERT INTO anggota (id_anggota, nama_anggota, username, password, tgl_lahir, nik_ktp, no_telepon, alamat, jenis_kelamin, status_anggota, fakultas, tgl_gabung, unit_kerja, fakultas_code, bukti_transfer, gaji_bulanan)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := db.Exec(query,
		anggota.IDAnggota,
		anggota.NamaAnggota,
		anggota.Username,
		anggota.Password,
		anggota.TglLahir,
		anggota.NikKTP,
		anggota.NoTelepon,
		anggota.Alamat,
		anggota.JenisKelamin,
		anggota.StatusAnggota,
		anggota.Fakultas,
		anggota.TglGabung,
		anggota.UnitKerja,
		anggota.FakultasCode,
		anggota.BuktiTransfer,
		anggota.GajiBulanan,
	)
	return err
}

// Di file: repository/anggota_repository.go

// ... (fungsi-fungsi lain yang sudah ada)

func GetAnggotaByID(id string) (models.Anggota, error) {
	db := config.GetDB()
	var a models.Anggota
	var encryptedPassword string
	query := `
		SELECT
			id_anggota, nama_anggota, username, password,
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			alamat, jenis_kelamin, status,
			unit_kerja, fakultas_code, COALESCE(tahun, ''), COALESCE(nomor_urut, ''), COALESCE(bukti_transfer, ''), COALESCE(status_anggota, ''), COALESCE(fakultas, ''), COALESCE(gaji_bulanan, 0)
		FROM anggota
		WHERE id_anggota = $1`

	err := db.QueryRow(query, id).Scan(
		&a.IDAnggota, &a.NamaAnggota, &a.Username, &encryptedPassword,
		&a.TglLahir, &a.NikKTP, &a.NoTelepon, &a.TglGabung,
		&a.Alamat, &a.JenisKelamin, &a.Status,
		&a.UnitKerja, &a.FakultasCode, &a.Tahun, &a.NomorUrut, &a.BuktiTransfer, &a.StatusAnggota, &a.Fakultas, &a.GajiBulanan,
	)
	if err != nil {
		return a, err
	}

	// Password sudah dalam bentuk plain text dari database
	a.Password = encryptedPassword

	return a, err
}

// GetAllAnggota mengambil semua anggota aktif
func GetAllAnggota() ([]models.Anggota, error) {
	db := config.GetDB()
	var anggotas []models.Anggota

	query := `
		SELECT
			id_anggota, nama_anggota, username,
			tgl_lahir, nik_ktp, no_telepon, tgl_gabung,
			alamat, jenis_kelamin, status, unit_kerja, fakultas, COALESCE(gaji_bulanan, 0)
		FROM anggota
		WHERE status = 'aktif'
		ORDER BY id_anggota DESC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a models.Anggota
		err := rows.Scan(
			&a.IDAnggota, &a.NamaAnggota, &a.Username,
			&a.TglLahir, &a.NikKTP, &a.NoTelepon, &a.TglGabung,
			&a.Alamat, &a.JenisKelamin, &a.Status, &a.UnitKerja, &a.Fakultas, &a.GajiBulanan,
		)
		if err != nil {
			return nil, err
		}
		anggotas = append(anggotas, a)
	}

	return anggotas, nil
}

// DeleteAnggota menghapus anggota berdasarkan ID (soft delete atau hard delete tergantung kebutuhan)
func DeleteAnggota(id string) error {
	db := config.GetDB()
	_, err := db.Exec("DELETE FROM anggota WHERE id_anggota = $1", id)
	return err
}

// UpdateAnggotaPassword memperbarui password anggota berdasarkan ID
func UpdateAnggotaPassword(id string, newPassword string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET password = $1 WHERE id_anggota = $2"
	_, err := db.Exec(query, newPassword, id)
	return err
}

// UpdateAnggotaUsernamePassword memperbarui username dan password anggota berdasarkan ID
func UpdateAnggotaUsernamePassword(id string, username, password string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET username = $1, password = $2 WHERE id_anggota = $3"
	_, err := db.Exec(query, username, password, id)
	return err
}

// GetSaldoAnggota mengambil total simpanan, total pinjaman, dan saldo bersih anggota
func GetSaldoAnggota(id string) (totalSimpanan, totalPinjaman, saldoBersih float64, err error) {
	db := config.GetDB()

	// Hitung total simpanan dari detail (hanya yang sudah dikonfirmasi/confirmed)
	querySimpanan := `
		SELECT COALESCE(SUM(d.jumlah_simpanan), 0)
		FROM detail d
		WHERE d.id_anggota = $1 AND d.status = 'confirmed'
	`
	err = db.QueryRow(querySimpanan, id).Scan(&totalSimpanan)
	if err != nil {
		return 0, 0, 0, err
	}

	// Hitung total pinjaman yang aktif (status = 'aktif')
	queryPinjaman := `
		SELECT COALESCE(SUM(p.jumlah_pinjaman), 0)
		FROM pinjaman p
		WHERE p.id_anggota = $1 AND p.status = 'aktif'
	`
	err = db.QueryRow(queryPinjaman, id).Scan(&totalPinjaman)
	if err != nil {
		return 0, 0, 0, err
	}

	// Saldo bersih = total simpanan - total pinjaman aktif
	saldoBersih = totalSimpanan - totalPinjaman

	return totalSimpanan, totalPinjaman, saldoBersih, nil
}

// GetDetailSimpananByJenis mengambil total simpanan per jenis
func GetDetailSimpananByJenis(id string) (map[string]float64, error) {
	db := config.GetDB()
	simpananByJenis := make(map[string]float64)

	query := `
		SELECT s.jenis_simpanan, COALESCE(SUM(d.jumlah_simpanan), 0) as total
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1 AND d.status = 'confirmed'
		GROUP BY s.jenis_simpanan
	`

	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var jenis string
		var total float64
		if err := rows.Scan(&jenis, &total); err != nil {
			return nil, err
		}
		simpananByJenis[jenis] = total
	}

	// Pastikan semua jenis simpanan ada dengan nilai 0 jika tidak ada data
	jenisList := []string{"pokok", "wajib", "sukarela", "hari_raya"}
	for _, jenis := range jenisList {
		if _, exists := simpananByJenis[jenis]; !exists {
			simpananByJenis[jenis] = 0
		}
	}

	return simpananByJenis, nil
}

// GetSimpananWajibAllAnggota mengambil total simpanan wajib untuk semua anggota
func GetSimpananWajibAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	simpananWajib := make(map[string]float64)

	query := `
		SELECT d.id_anggota, COALESCE(SUM(d.jumlah_simpanan), 0) as total_wajib
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE s.jenis_simpanan = 'wajib' AND d.status IN ('approved', 'confirmed')
		GROUP BY d.id_anggota
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var totalWajib float64
		if err := rows.Scan(&idAnggota, &totalWajib); err != nil {
			return nil, err
		}
		simpananWajib[idAnggota] = totalWajib
	}

	return simpananWajib, nil
}

// GetPotonganBulanIniAllAnggota mengambil total pemotongan bulan ini untuk semua anggota
func GetPotonganBulanIniAllAnggota() (map[string]float64, error) {
	db := config.GetDB()
	potonganBulanIni := make(map[string]float64)

	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	query := `
		SELECT id_anggota, COALESCE(SUM(jumlah_potong), 0) as total_potong
		FROM log_pemotongan_simpanan
		WHERE bulan = $1 AND tahun = $2 AND status = 'berhasil'
		GROUP BY id_anggota
	`

	rows, err := db.Query(query, bulan, tahun)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var idAnggota string
		var totalPotong float64
		if err := rows.Scan(&idAnggota, &totalPotong); err != nil {
			return nil, err
		}
		potonganBulanIni[idAnggota] = totalPotong
	}

	return potonganBulanIni, nil
}

// GetKonfigurasiSimpananWajib mengambil konfigurasi pemotongan simpanan wajib
func GetKonfigurasiSimpananWajib() (map[string]interface{}, error) {
	db := config.GetDB()
	config := make(map[string]interface{})

	query := `SELECT tanggal_potong, persentase_potong, nominal_tetap, tipe_pemotongan, status_aktif 
	          FROM konfigurasi_simpanan_wajib ORDER BY id DESC LIMIT 1`

	var tanggalPotong int
	var persentasePotong, nominalTetap float64
	var tipePemotongan string
	var statusAktif bool

	err := db.QueryRow(query).Scan(&tanggalPotong, &persentasePotong, &nominalTetap, &tipePemotongan, &statusAktif)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default values
			return map[string]interface{}{
				"TanggalPotong":    1,
				"PersentasePotong": 5.0,
				"NominalTetap":     0.0,
				"TipePemotongan":   "persentase",
				"StatusAktif":      false,
			}, nil
		}
		return nil, err
	}

	config["TanggalPotong"] = tanggalPotong
	config["PersentasePotong"] = persentasePotong
	config["NominalTetap"] = nominalTetap
	config["TipePemotongan"] = tipePemotongan
	config["StatusAktif"] = statusAktif

	return config, nil
}

// SaveKonfigurasiSimpananWajib menyimpan atau update konfigurasi pemotongan simpanan wajib
func SaveKonfigurasiSimpananWajib(tanggalPotong int, persentasePotong, nominalTetap float64, tipePemotongan string, statusAktif bool) error {
	db := config.GetDB()

	// Check if config exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM konfigurasi_simpanan_wajib").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update existing config
		query := `UPDATE konfigurasi_simpanan_wajib 
		          SET tanggal_potong = $1, persentase_potong = $2, nominal_tetap = $3, 
		              tipe_pemotongan = $4, status_aktif = $5, updated_at = CURRENT_TIMESTAMP`
		_, err = db.Exec(query, tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif)
	} else {
		// Insert new config
		query := `INSERT INTO konfigurasi_simpanan_wajib 
		          (tanggal_potong, persentase_potong, nominal_tetap, tipe_pemotongan, status_aktif) 
		          VALUES ($1, $2, $3, $4, $5)`
		_, err = db.Exec(query, tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif)
	}

	return err
}

// ProsesPemotonganSimpananWajib melakukan pemotongan otomatis untuk semua anggota
func ProsesPemotonganSimpananWajib() (successCount, failedCount int, errors []string) {
	db := config.GetDB()
	successCount = 0
	failedCount = 0
	errors = []string{}

	// Get konfigurasi
	configData, err := GetKonfigurasiSimpananWajib()
	if err != nil {
		errors = append(errors, "Gagal mengambil konfigurasi: "+err.Error())
		return
	}

	if !configData["StatusAktif"].(bool) {
		errors = append(errors, "Fitur pemotongan otomatis tidak aktif")
		return
	}

	// Get all active anggota dengan gaji > 0
	anggotas, err := GetAllAnggota()
	if err != nil {
		errors = append(errors, "Gagal mengambil data anggota: "+err.Error())
		return
	}

	fmt.Printf("=== Proses Pemotongan Simpanan Wajib ===\n")
	fmt.Printf("Total anggota: %d\n", len(anggotas))

	tipePemotongan := configData["TipePemotongan"].(string)
	persentasePotong := configData["PersentasePotong"].(float64)
	nominalTetap := configData["NominalTetap"].(float64)

	fmt.Printf("Tipe pemotongan: %s, Persentase: %.2f%%, Nominal: Rp %.0f\n", tipePemotongan, persentasePotong, nominalTetap)

	// Get current month and year
	now := time.Now()
	bulan := int(now.Month())
	tahun := now.Year()

	fmt.Printf("Bulan: %d, Tahun: %d\n", bulan, tahun)

	var skippedNoGaji, skippedAlreadyProcessed int

	for _, anggota := range anggotas {
		// Skip mahasiswa atau yang tidak punya gaji
		if anggota.GajiBulanan <= 0 {
			skippedNoGaji++
			fmt.Printf("  Skip %s (ID: %s): Gaji = %d\n", anggota.NamaAnggota, anggota.IDAnggota, anggota.GajiBulanan)
			continue
		}

		// Check if already processed this month
		var exists bool
		checkQuery := "SELECT EXISTS(SELECT 1 FROM log_pemotongan_simpanan WHERE id_anggota = $1 AND bulan = $2 AND tahun = $3)"
		db.QueryRow(checkQuery, anggota.IDAnggota, bulan, tahun).Scan(&exists)

		if exists {
			skippedAlreadyProcessed++
			fmt.Printf("  Skip %s (ID: %s): Sudah dipotong bulan ini\n", anggota.NamaAnggota, anggota.IDAnggota)
			continue
		}

		// Calculate potongan
		var jumlahPotong float64
		if tipePemotongan == "persentase" {
			jumlahPotong = float64(anggota.GajiBulanan) * (persentasePotong / 100)
		} else {
			jumlahPotong = nominalTetap
		}

		fmt.Printf("  Processing %s (ID: %s): Gaji Rp %d, Potong Rp %.0f\n", anggota.NamaAnggota, anggota.IDAnggota, anggota.GajiBulanan, jumlahPotong)

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			failedCount++
			fmt.Printf("    ❌ Gagal memulai transaksi: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))
			continue
		}

		// Insert detail simpanan wajib
		detailQuery := `INSERT INTO detail (id_anggota, id_simpanan, id_pengelola, tgl_transaksi, jumlah_simpanan, status)
		                VALUES ($1, 2, 1, CURRENT_TIMESTAMP, $2, 'confirmed')`
		_, err = tx.Exec(detailQuery, anggota.IDAnggota, jumlahPotong)

		if err != nil {
			tx.Rollback()
			failedCount++
			fmt.Printf("    ❌ Gagal insert detail: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))

			// Log failed
			logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
			             VALUES ($1, $2, $3, $4, $5, 'gagal', $6)`
			db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, err.Error())
			continue
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			failedCount++
			fmt.Printf("    ❌ Gagal commit: %v\n", err)
			errors = append(errors, fmt.Sprintf("Gagal memotong %s: %s", anggota.NamaAnggota, err.Error()))

			// Log failed
			logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
			             VALUES ($1, $2, $3, $4, $5, 'gagal', $6)`
			db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, err.Error())
			continue
		}

		successCount++
		fmt.Printf("    ✓ Berhasil memotong Rp %.0f (Gaji tetap: Rp %d)\n", jumlahPotong, anggota.GajiBulanan)

		// Log success
		logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
		             VALUES ($1, $2, $3, $4, $5, 'berhasil', $6)`
		db.Exec(logQuery, anggota.IDAnggota, bulan, tahun, float64(anggota.GajiBulanan), jumlahPotong, fmt.Sprintf("Pemotongan simpanan wajib berhasil sebesar Rp %.0f", jumlahPotong))
	}

	fmt.Printf("\n=== Ringkasan ===\n")
	fmt.Printf("Berhasil: %d, Gagal: %d\n", successCount, failedCount)
	fmt.Printf("Dilewati (no gaji): %d, Dilewati (sudah dipotong): %d\n", skippedNoGaji, skippedAlreadyProcessed)

	return successCount, failedCount, errors
}

// UpdateAnggotaStatus memperbarui status anggota berdasarkan ID
func UpdateAnggotaStatus(id string, status string) error {
	db := config.GetDB()
	query := "UPDATE anggota SET status = $1 WHERE id_anggota = $2"
	_, err := db.Exec(query, status, id)
	return err
}

// GetNomorRekening mengambil nomor rekening berdasarkan jenis (simpanan, angsuran, register)
func GetNomorRekening(jenis string) (string, error) {
	db := config.GetDB()
	var nomor string
	query := "SELECT nomor FROM nomor_rekening WHERE jenis = $1 LIMIT 1"
	err := db.QueryRow(query, jenis).Scan(&nomor)
	if err == sql.ErrNoRows {
		// Jika tidak ada, return string kosong dan no error
		return "", nil
	}
	return nomor, err
}

// UpdateNomorRekening memperbarui atau menambahkan nomor rekening untuk jenis tertentu
func UpdateNomorRekening(jenis string, nomor string) error {
	db := config.GetDB()

	// Cek apakah sudah ada entri untuk jenis ini
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM nomor_rekening WHERE jenis = $1)", jenis).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update
		_, err = db.Exec("UPDATE nomor_rekening SET nomor = $1 WHERE jenis = $2", nomor, jenis)
	} else {
		// Insert
		_, err = db.Exec("INSERT INTO nomor_rekening (jenis, nomor) VALUES ($1, $2)", jenis, nomor)
	}
	return err
}

// GetTotalAnggota mengambil total anggota aktif
func GetTotalAnggota(db *sql.DB) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM anggota WHERE status = 'aktif'"
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// GetMenungguKonfirmasi mengambil jumlah anggota yang menunggu konfirmasi (status pending)
func GetMenungguKonfirmasi(db *sql.DB) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM anggota WHERE status = 'pending'"
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

// BatchInsertAnggota melakukan insert batch data anggota dari file XLSX
func BatchInsertAnggota(db *sql.DB, anggotaList []models.Anggota) (successCount int, failedCount int, detailErrors []string, successfulIDs []string) {
	successCount = 0
	failedCount = 0
	detailErrors = []string{}
	successfulIDs = []string{}

	for idx, anggota := range anggotaList {
		rowNum := idx + 2 // +2 karena index 0 adalah header, dan Excel dimulai dari baris 1

		// Cek apakah username sudah ada
		var existingUsername string
		var existingNIK string
		checkQuery := "SELECT COALESCE(username, ''), COALESCE(nik_ktp, '') FROM anggota WHERE username = $1 OR nik_ktp = $2 LIMIT 1"
		err := db.QueryRow(checkQuery, anggota.Username, anggota.NikKTP).Scan(&existingUsername, &existingNIK)

		if err == nil {
			// Data ditemukan - ada duplikat
			if existingUsername == anggota.Username {
				detailErrors = append(detailErrors, fmt.Sprintf("Baris %d: Username '%s' sudah digunakan oleh anggota lain", rowNum, anggota.Username))
			}
			if existingNIK == anggota.NikKTP && anggota.NikKTP != "" {
				detailErrors = append(detailErrors, fmt.Sprintf("Baris %d: NIK '%s' sudah terdaftar", rowNum, anggota.NikKTP))
			}
			failedCount++
			continue
		}

		query := `
			INSERT INTO anggota (
				id_anggota, nama_anggota, username, password, tgl_lahir, 
				nik_ktp, no_telepon, alamat, jenis_kelamin, status_anggota, 
				fakultas, tgl_gabung, unit_kerja, fakultas_code, status
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`

		_, err = db.Exec(query,
			anggota.IDAnggota,
			anggota.NamaAnggota,
			anggota.Username,
			anggota.Password,
			anggota.TglLahir,
			anggota.NikKTP,
			anggota.NoTelepon,
			anggota.Alamat,
			anggota.JenisKelamin,
			anggota.StatusAnggota,
			anggota.Fakultas,
			anggota.TglGabung,
			anggota.UnitKerja,
			anggota.FakultasCode,
			anggota.Status,
		)

		if err != nil {
			detailErrors = append(detailErrors, fmt.Sprintf("Baris %d: Gagal insert ke database - %s", rowNum, err.Error()))
			failedCount++
		} else {
			successCount++
			successfulIDs = append(successfulIDs, anggota.IDAnggota)
		}
	}

	return successCount, failedCount, detailErrors, successfulIDs
}
