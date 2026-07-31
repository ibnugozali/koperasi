package controllers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

func DeleteAllLoginHistory(c *gin.Context) {
	err := repository.DeleteAllLoginHistory()
	if err != nil {
		c.JSON(500, gin.H{"error": "Gagal menghapus semua riwayat login"})
		return
	}
	c.JSON(200, gin.H{"message": "Semua riwayat login berhasil dihapus"})
}

// UpdateUser memproses update data user (pengelola)
func UpdateUser(c *gin.Context) {
	id := c.PostForm("id")
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validasi
	if id == "" || username == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID dan Username wajib diisi"})
		return
	}

	db := config.GetDB()
	var query string
	var args []interface{}
	if password != "" {
		// Hash password dengan bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengenkripsi password"})
			return
		}
		query = "UPDATE pengelola SET username = $1, password = $2 WHERE id_pengelola = $3"
		args = []interface{}{username, string(hashedPassword), id}
	} else {
		query = "UPDATE pengelola SET username = $1 WHERE id_pengelola = $2"
		args = []interface{}{username, id}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal memperbarui user"})
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

// UpdateAnggota memproses update data anggota
func UpdateAnggota(c *gin.Context) {
	id := c.PostForm("id")
	username := c.PostForm("username")
	password := c.PostForm("password")

	if id == "" || username == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID dan Username wajib diisi"})
		return
	}

	db := config.GetDB()
	var query string
	var args []interface{}
	if password != "" {
		query = "UPDATE anggota SET username = $1, password = $2 WHERE id_anggota = $3"
		args = []interface{}{username, password, id}
	} else {
		query = "UPDATE anggota SET username = $1 WHERE id_anggota = $2"
		args = []interface{}{username, id}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal memperbarui anggota"})
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

// Menampilkan dashboard admin dengan data statistik
func AdminDashboard(c *gin.Context) {
	db := config.GetDB()

	// Ambil data dashboard
	totalAnggota, err := repository.GetTotalAnggota(db)
	if err != nil {
		totalAnggota = 0
	}

	menungguKonfirmasi, err := repository.GetMenungguKonfirmasi(db)
	if err != nil {
		menungguKonfirmasi = 0
	}

	totalSimpanan, err := repository.GetTotalSimpanan(db)
	if err != nil {
		totalSimpanan = 0
	}

	totalPinjaman, err := repository.GetTotalPinjaman(db)
	if err != nil {
		totalPinjaman = 0
	}

	// Ambil riwayat total anggota, simpanan, pinjaman per hari
	riwayatAnggota, _ := repository.GetRiwayatTotalAnggotaPerHari(db)
	riwayatSimpanan, _ := repository.GetRiwayatTotalSimpananPerHari(db)
	riwayatPinjaman, _ := repository.GetRiwayatTotalPinjamanPerHari(db)

	aktivitasData := []map[string]interface{}{}
	for _, r := range riwayatAnggota {
		r["Jenis"] = "Anggota"
		aktivitasData = append(aktivitasData, r)
	}
	for _, r := range riwayatSimpanan {
		r["Jenis"] = "Simpanan"
		aktivitasData = append(aktivitasData, r)
	}
	for _, r := range riwayatPinjaman {
		r["Jenis"] = "Pinjaman"
		aktivitasData = append(aktivitasData, r)
	}
	// Jika semua riwayat kosong, fallback ke statistik utama
	if len(aktivitasData) == 0 {
		aktivitasData = []map[string]interface{}{
			{"Tanggal": time.Now(), "Jenis": "Anggota", "Jumlah": totalAnggota},
			{"Tanggal": time.Now(), "Jenis": "Simpanan", "Jumlah": totalSimpanan},
			{"Tanggal": time.Now(), "Jenis": "Pinjaman", "Jumlah": totalPinjaman},
		}
	}
	// Data untuk template
	data := map[string]interface{}{
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"AktivitasData":      aktivitasData,
		"LogoPath":           c.MustGet("LogoPath"),
		"CurrentLogo":        c.MustGet("LogoPath"),
		"ActivePage":         "dashboard",
		"Success":            c.Query("success"),
		"Error":              c.Query("error"),
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", data)
}

func AdminImportReferensiPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_import_referensi.html", gin.H{
		"LogoPath":    c.MustGet("LogoPath"),
		"CurrentLogo": c.MustGet("LogoPath"),
		"ActivePage":  "import_referensi",
		"Success":     c.Query("success"),
		"Warning":     c.Query("warning"),
		"Error":       c.Query("error"),
	})
}

func AdminDataAnggota(c *gin.Context) {
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	c.HTML(http.StatusOK, "admin_data_anggota.html", gin.H{
		"Anggotas":    anggotas,
		"Success":     c.Query("success"),
		"ActivePage":  "anggota",
		"LogoPath":    logoPath,
		"CurrentLogo": logoPath,
	})
}

func mapStatusToUnitKerja(statusAnggota string) string {
	switch strings.ToLower(strings.TrimSpace(statusAnggota)) {
	case "dosen":
		return "01"
	case "karyawan", "tenaga pendidikan":
		return "02"
	case "mahasiswa":
		return "03"
	default:
		return ""
	}
}

func mapFakultasToCode(fakultas string) string {
	switch strings.TrimSpace(fakultas) {
	case "Fakultas Agama Islam (FAI)":
		return "01"
	case "Fakultas Ekonomi (FE)":
		return "02"
	case "Fakultas Hukum (FH)":
		return "03"
	case "Fakultas Ilmu Sosial dan Ilmu Politik (FISIP)":
		return "04"
	case "Fakultas Keguruan dan Ilmu Pendidikan (FKIP)":
		return "05"
	case "Fakultas Kesehatan Masyarakat (FKM)":
		return "06"
	case "Fakultas Pertanian (FAPERTA)":
		return "07"
	case "Fakultas Teknik (FT)":
		return "08"
	case "Rektorat / Yayasan", "Rektorat / Yayasan / Staff", "Rektoriat":
		return "09"
	case "Paskasarjana", "Pascasarjana":
		return "10"
	default:
		return ""
	}
}

func normalizePhone(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	if strings.HasPrefix(value, "+62") {
		value = "0" + strings.TrimPrefix(value, "+62")
	}
	if strings.HasPrefix(value, "62") {
		value = "0" + strings.TrimPrefix(value, "62")
	}
	if value != "" && !strings.HasPrefix(value, "0") {
		value = "0" + value
	}
	return value
}

func renderAdminTambahAnggota(c *gin.Context, status int, data gin.H) {
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}
	data["ActivePage"] = "anggota"
	data["LogoPath"] = logoPath
	data["CurrentLogo"] = logoPath

	// Ambil nomor rekening dan nominal simpanan untuk tampilan form
	db := config.GetDB()
	var nomorRekening string
	if err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening); err != nil {
		nomorRekening = "-"
	}
	var nominalSimpanan string
	if err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpanan); err != nil {
		nominalSimpanan = "100000"
	}
	data["NomorRekening"] = nomorRekening
	data["NominalSimpanan"] = nominalSimpanan

	c.HTML(status, "admin_data_anggota_tambah.html", data)
}

// AdminTambahAnggotaForm menampilkan form tambah anggota (langsung aktif tanpa acc ketua).
func AdminTambahAnggotaForm(c *gin.Context) {
	renderAdminTambahAnggota(c, http.StatusOK, gin.H{
		"Success":     c.Query("success"),
		"ImportError": c.Query("error"),
	})
}

// AdminTambahAnggotaPost menyimpan anggota baru langsung aktif.
func AdminTambahAnggotaPost(c *gin.Context) {
	namaAnggota := strings.TrimSpace(c.PostForm("NamaAnggota"))
	username := strings.TrimSpace(c.PostForm("Username"))
	password := strings.TrimSpace(c.PostForm("Password"))
	noTelepon := normalizePhone(strings.TrimSpace(c.PostForm("NoTelepon")))
	tglLahir := strings.TrimSpace(c.PostForm("TglLahir"))
	jenisKelamin := strings.TrimSpace(c.PostForm("JenisKelamin"))
	statusAnggota := strings.TrimSpace(c.PostForm("StatusAnggota"))
	fakultas := strings.TrimSpace(c.PostForm("Fakultas"))
	alamat := strings.TrimSpace(c.PostForm("Alamat"))
	gajiBulananStr := strings.TrimSpace(c.PostForm("GajiBulanan"))
	metodePembayaran := strings.TrimSpace(c.PostForm("MetodePembayaran"))
	noIdentitasPegawai := strings.TrimSpace(c.PostForm("NoIdentitasPegawai"))

	formData := gin.H{
		"FormNamaAnggota":        namaAnggota,
		"FormUsername":           username,
		"FormNoTelepon":          noTelepon,
		"FormTglLahir":           tglLahir,
		"FormJenisKelamin":       jenisKelamin,
		"FormStatusAnggota":      statusAnggota,
		"FormFakultas":           fakultas,
		"FormAlamat":             alamat,
		"FormGajiBulanan":        gajiBulananStr,
		"FormMetodePembayaran":   metodePembayaran,
		"FormNoIdentitasPegawai": noIdentitasPegawai,
	}

	if namaAnggota == "" || username == "" || password == "" || noTelepon == "" || tglLahir == "" ||
		jenisKelamin == "" || statusAnggota == "" || fakultas == "" || alamat == "" {
		formData["Error"] = "Semua field wajib diisi."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	if username == noTelepon {
		formData["Error"] = "Nama Pengguna dan No. Telepon tidak boleh sama."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	gajiBulanan := 0
	if gajiBulananStr != "" {
		parsedGaji, err := strconv.Atoi(gajiBulananStr)
		if err != nil || parsedGaji < 0 {
			formData["Error"] = "Format gaji bulanan tidak valid."
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}
		gajiBulanan = parsedGaji
	}

	if metodePembayaran == "" {
		metodePembayaran = "transfer_bank"
	}
	if metodePembayaran == "transfer" {
		metodePembayaran = "transfer_bank"
	}

	if strings.ToLower(statusAnggota) != "mahasiswa" && gajiBulanan <= 0 {
		formData["Error"] = "Gaji bulanan wajib diisi untuk dosen dan tenaga pendidikan."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	// Validasi referensi untuk non-mahasiswa (sama seperti alur register)
	db := config.GetDB()
	var err error
	if strings.ToLower(statusAnggota) != "mahasiswa" {
		if noIdentitasPegawai == "" {
			formData["Error"] = "Nomer identitas wajib diisi sesuai data master import referensi."
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}
		username = noIdentitasPegawai
		formData["FormUsername"] = username

		referensiByIdentitas, err := repository.FindReferensiPendaftaranByIdentitas(noIdentitasPegawai)
		if err == nil {
			if strings.ToLower(strings.TrimSpace(referensiByIdentitas.StatusKeanggotaan)) == "anggota" {
				hasCurrentAnggota, err := repository.HasCurrentAnggotaForReferensi(
					referensiByIdentitas.NamaLengkap,
					referensiByIdentitas.NomorIdentitas,
					referensiByIdentitas.GajiBulanan,
				)
				if err != nil {
					formData["Error"] = "Gagal memvalidasi data referensi pendaftaran."
					renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
					return
				}
				if hasCurrentAnggota {
					formData["Error"] = "Data ini di master sudah tercatat sebagai anggota. Pendaftaran baru tidak dapat diproses."
					renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
					return
				}
			}
		} else if err != sql.ErrNoRows {
			formData["Error"] = "Gagal memvalidasi data referensi pendaftaran."
			renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
			return
		}

		_, err = repository.FindReferensiPendaftaranForRegister(namaAnggota, noIdentitasPegawai, gajiBulanan)
		if err != nil {
			if err == sql.ErrNoRows {
				formData["Error"] = "Data referensi tidak cocok. Pastikan Nama Lengkap, Nomer Identitas, dan Gaji Bersih sama dengan data di import referensi admin."
				renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
				return
			}
			formData["Error"] = "Gagal memvalidasi data referensi pendaftaran."
			renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
			return
		}
	}

	unitKerja := mapStatusToUnitKerja(statusAnggota)
	fakultasCode := mapFakultasToCode(fakultas)
	if unitKerja == "" || fakultasCode == "" {
		formData["Error"] = "Jabatan atau Unit Kerja tidak valid."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	// Validasi metode pembayaran seperti register
	var buktiTransfer string
	var nominalSimpanan string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpanan)
	if err != nil {
		nominalSimpanan = "100000"
	}
	nominalSimpananPokok := parseNominalPengaturan(nominalSimpanan, 100000)

	switch metodePembayaran {
	case "transfer_bank":
		file, err := c.FormFile("BuktiTransfer")
		if err != nil {
			formData["Error"] = "Bukti transfer wajib diupload jika memilih metode transfer."
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}

		filename := time.Now().Format("20060102150405") + "_" + file.Filename
		dst := "./static/uploads/" + filename
		if err := c.SaveUploadedFile(file, dst); err != nil {
			formData["Error"] = "Gagal menyimpan file bukti transfer."
			renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
			return
		}
		buktiTransfer = filename
	case "potong_gaji":
		if strings.ToLower(statusAnggota) == "mahasiswa" {
			formData["Error"] = "Metode potong gaji hanya untuk anggota dengan gaji. Untuk mahasiswa gunakan transfer."
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}
		if gajiBulanan < nominalSimpananPokok {
			formData["Error"] = fmt.Sprintf("Gaji bersih harus minimal Rp %d untuk menggunakan metode potong gaji karena nominal simpanan pokok saat ini Rp %d.", nominalSimpananPokok, nominalSimpananPokok)
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}
		buktiTransfer = "POTONG_GAJI"
	case "tunai":
		buktiTransfer = "TUNAI"
	default:
		formData["Error"] = "Metode pembayaran tidak valid."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	if strings.ToLower(statusAnggota) != "mahasiswa" {
		var duplicateCount int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM anggota
			WHERE LOWER(TRIM(COALESCE(nama_anggota, ''))) = LOWER(TRIM($1))
			  AND COALESCE(username, '') = $2
			  AND COALESCE(gaji_bulanan, 0) = $3
			  AND LOWER(TRIM(COALESCE(status, ''))) <> 'keluar'
		`, namaAnggota, noIdentitasPegawai, gajiBulanan).Scan(&duplicateCount)
		if err == nil && duplicateCount > 0 {
			formData["Error"] = "Data dengan Nama Lengkap, Nomer Identitas, dan Gaji Bersih tersebut sudah terdaftar."
			renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
			return
		}
	}

	// Validasi unik username / telepon agar tidak bentrok akun.
	var count int
	usernameDipakaiAnggotaAktif, err := repository.HasCurrentAnggotaUsername(username)
	if err != nil {
		formData["Error"] = "Gagal memvalidasi data anggota."
		renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
		return
	}
	if usernameDipakaiAnggotaAktif {
		formData["Error"] = "Nama Pengguna sudah terdaftar."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}
	if err := repository.ReleaseUsernameFromAnggotaKeluar(username); err != nil {
		formData["Error"] = "Gagal memvalidasi data anggota."
		renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
		return
	}

	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM anggota
		WHERE COALESCE(no_telepon, '') = $1
		  AND LOWER(TRIM(COALESCE(status, ''))) <> 'keluar'
	`, noTelepon).Scan(&count)
	if err != nil {
		formData["Error"] = "Gagal memvalidasi data anggota."
		renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
		return
	}
	if count > 0 {
		formData["Error"] = "No. Telepon sudah terdaftar."
		renderAdminTambahAnggota(c, http.StatusBadRequest, formData)
		return
	}

	// Generate nomor urut global seperti proses konfirmasi ketua/bendahara.
	var lastNumber int
	err = db.QueryRow("SELECT COALESCE(MAX(CAST(nomor_urut AS INTEGER)), 0) FROM anggota WHERE id_anggota NOT LIKE 'TEMP%'").Scan(&lastNumber)
	if err != nil {
		formData["Error"] = "Gagal membuat nomor anggota."
		renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
		return
	}
	nomorUrut := fmt.Sprintf("%04d", lastNumber+1)
	tahun := time.Now().Format("06")
	idAnggota := fmt.Sprintf("%s%s%s%s", unitKerja, fakultasCode, tahun, nomorUrut)

	insertQuery := `
		INSERT INTO anggota (
			id_anggota, nama_anggota, username, password, tgl_lahir,
			no_telepon, tgl_gabung, alamat, jenis_kelamin,
			status_anggota, fakultas, status, unit_kerja, fakultas_code,
			bukti_transfer, gaji_bulanan, tahun, nomor_urut
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, CURRENT_DATE, $7, $8,
			$9, $10, 'aktif', $11, $12,
			$13, $14, $15, $16
		)
	`
	_, err = db.Exec(
		insertQuery,
		idAnggota, namaAnggota, username, password, tglLahir,
		noTelepon, alamat, jenisKelamin,
		statusAnggota, fakultas, unitKerja, fakultasCode,
		buktiTransfer, gajiBulanan, tahun, nomorUrut,
	)
	if err != nil {
		log.Printf("[ERROR] AdminTambahAnggota simpan anggota baru gagal: %v", err)
		formData["Error"] = "Gagal menyimpan anggota baru"
		renderAdminTambahAnggota(c, http.StatusInternalServerError, formData)
		return
	}

	// Jika metode pembayaran potong_gaji, catat simpanan pokok sebagai 'confirmed' seperti proses konfirmasi
	if strings.EqualFold(strings.TrimSpace(buktiTransfer), "POTONG_GAJI") {
		var nominalSimpananPokok float64
		err = db.QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
		if err != nil {
			nominalSimpananPokok = 100000
		}

		if err := repository.EnsureSimpananPokokPotongGaji(db, idAnggota, nominalSimpananPokok); err != nil {
			log.Printf("[ERROR] gagal mencatat simpanan pokok potong gaji untuk anggota %s: %v", idAnggota, err)
		}
	}

	c.Redirect(http.StatusFound, "/admin/anggota?success=Anggota baru berhasil ditambahkan dan langsung aktif")
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer(" ", "", "_", "", ".", "", "-", "", "/", "")
	return replacer.Replace(s)
}

func findHeaderIndex(headerMap map[string]int, keys ...string) int {
	for _, k := range keys {
		if idx, ok := headerMap[normalizeHeader(k)]; ok {
			return idx
		}
	}
	return -1
}

func getCell(row []string, idx int) string {
	if idx >= 0 && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func findReferensiIdentitasHeaderIndex(headerMap map[string]int) int {
	for _, key := range referensiIdentitasHeaderKeys() {
		if idx, ok := headerMap[normalizeHeader(key)]; ok {
			return idx
		}
	}
	return -1
}

func referensiIdentitasHeaderKeys() []string {
	return []string{
		"nomer identitas",
		"nomor identitas",
		"no identitas",
		"no. identitas",
		"nomer induk",
		"nomor induk",
		"nip",
		"nidn",
		"nidk",
		"id pegawai",
		"kode pegawai",
	}
}

func findReferensiIdentitasColumn(rows [][]string, headerRowIdx int, headerMap map[string]int) int {
	for _, key := range referensiIdentitasHeaderKeys() {
		idx, ok := headerMap[normalizeHeader(key)]
		if !ok {
			continue
		}
		if !isLikelyRowNumberColumn(rows, headerRowIdx, idx) {
			return idx
		}
	}
	return -1
}

func isLikelyRowNumberColumn(rows [][]string, headerRowIdx int, idx int) bool {
	if idx < 0 {
		return false
	}

	expected := 1
	checked := 0
	matches := 0
	for _, row := range rows[headerRowIdx+1:] {
		value := getCell(row, idx)
		if value == "" {
			continue
		}
		checked++
		if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed == expected {
			matches++
		}
		expected++
		if checked >= 8 {
			break
		}
	}

	return checked >= 3 && matches == checked
}

func buildReferensiHeaderSampleMessage(rows [][]string, headerRowIdx int) string {
	if headerRowIdx < 0 || headerRowIdx >= len(rows) {
		return ""
	}

	var sampleRow []string
	for _, row := range rows[headerRowIdx+1:] {
		hasValue := false
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				hasValue = true
				break
			}
		}
		if hasValue {
			sampleRow = row
			break
		}
	}

	parts := []string{}
	for i, header := range rows[headerRowIdx] {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}

		if sampleRow != nil {
			if sample := getCell(sampleRow, i); sample != "" {
				parts = append(parts, fmt.Sprintf("%s=%s", header, sample))
			} else {
				parts = append(parts, header)
			}
		} else {
			parts = append(parts, header)
		}

		if len(parts) >= 6 {
			break
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return " Header terdeteksi: " + strings.Join(parts, "; ")
}

func containsHeaderToken(cell string, tokens ...string) bool {
	n := normalizeHeader(cell)
	for _, t := range tokens {
		if strings.Contains(n, normalizeHeader(t)) {
			return true
		}
	}
	return false
}

// AdminImportAnggotaExcel import anggota dari file Excel dan langsung aktif (tanpa acc ketua).
func AdminImportAnggotaExcel(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=File+Excel+tidak+ditemukan")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Format+file+harus+.xlsx+atau+.xls")
		return
	}

	if file.Size > 10*1024*1024 {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Ukuran+file+maksimal+10MB")
		return
	}

	tempPath := "./static/uploads/" + uuid.New().String() + ext
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Gagal+menyimpan+file+upload")
		return
	}
	defer os.Remove(tempPath)

	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=File+Excel+tidak+bisa+dibaca")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=File+Excel+tidak+memiliki+sheet")
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Data+Excel+kosong+atau+tidak+valid")
		return
	}

	// Cari baris header secara dinamis (tidak selalu di baris pertama),
	// agar file laporan/export juga bisa diimpor.
	headerRowIdx := -1
	headerMap := map[string]int{}
	for r, row := range rows {
		tmp := map[string]int{}
		for i, h := range row {
			norm := normalizeHeader(h)
			if norm != "" {
				tmp[norm] = i
			}
		}
		idxNamaTmp := findHeaderIndex(tmp, "nama anggota", "nama", "nama_anggota")
		if idxNamaTmp >= 0 {
			headerRowIdx = r
			headerMap = tmp
			break
		}
		// Fallback khusus format laporan: baris yang berisi No|Kode|Nama Anggota|Unit
		if len(row) >= 4 &&
			containsHeaderToken(getCell(row, 0), "no") &&
			containsHeaderToken(getCell(row, 1), "kode", "id") &&
			containsHeaderToken(getCell(row, 2), "nama anggota", "nama") {
			headerRowIdx = r
			headerMap = tmp
			break
		}
	}

	idxNama := findHeaderIndex(headerMap, "nama anggota", "nama", "nama_anggota")
	if idxNama < 0 {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Kolom+Nama+Anggota+tidak+ditemukan+di+file+Excel")
		return
	}

	idxUsername := findHeaderIndex(headerMap, "username", "nama pengguna")
	idxPassword := findHeaderIndex(headerMap, "password", "kata sandi")
	idxTelepon := findHeaderIndex(headerMap, "no telepon", "no hp", "telepon", "notelepon")
	idxStatus := findHeaderIndex(headerMap, "jabatan", "status anggota", "status_anggota")
	idxFakultas := findHeaderIndex(headerMap, "unit kerja", "fakultas")
	idxTglLahir := findHeaderIndex(headerMap, "tanggal lahir", "tgl lahir", "tgllahir")
	idxJenisKelamin := findHeaderIndex(headerMap, "jenis kelamin", "jeniskelamin")
	idxAlamat := findHeaderIndex(headerMap, "alamat")
	idxGaji := findHeaderIndex(headerMap, "gaji bulanan", "gaji", "gajibulanan")

	db := config.GetDB()
	var lastNumber int
	if err := db.QueryRow("SELECT COALESCE(MAX(CAST(nomor_urut AS INTEGER)), 0) FROM anggota WHERE id_anggota NOT LIKE 'TEMP%'").Scan(&lastNumber); err != nil {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error=Gagal+membaca+nomor+urut+anggota")
		return
	}

	successCount := 0
	failedCount := 0
	var parseErrors []string
	tahun := time.Now().Format("2006")

	// Fallback kolom untuk format laporan jika header tertentu tidak ada.
	if idxStatus < 0 {
		idxStatus = findHeaderIndex(headerMap, "unit")
	}
	if idxFakultas < 0 {
		idxFakultas = findHeaderIndex(headerMap, "unit")
	}

	for i, row := range rows[headerRowIdx+1:] {
		rowNum := headerRowIdx + i + 2

		nama := getCell(row, idxNama)
		if nama == "" || containsHeaderToken(nama, "nama", "anggota") {
			// Lewati baris kosong / subheader tanpa dihitung gagal.
			continue
		}

		username := getCell(row, idxUsername)
		if username == "" {
			username = strings.ToLower(strings.ReplaceAll(nama, " ", ""))
			username = strings.ReplaceAll(username, ".", "")
			if username == "" {
				username = fmt.Sprintf("anggota%d", time.Now().UnixNano())
			}
		}

		password := getCell(row, idxPassword)
		if password == "" {
			password = "12345678"
		}

		noTelepon := getCell(row, idxTelepon)
		if noTelepon == "" {
			noTelepon = fmt.Sprintf("8%d", time.Now().UnixNano()%1000000000000)
		}

		statusAnggota := strings.ToLower(getCell(row, idxStatus))
		if statusAnggota == "" {
			// Jika kolom status tidak ada, coba infer dari kolom unit.
			unitText := strings.ToLower(getCell(row, idxFakultas))
			if strings.Contains(unitText, "dosen") {
				statusAnggota = "dosen"
			} else if strings.Contains(unitText, "mahasiswa") {
				statusAnggota = "mahasiswa"
			} else {
				statusAnggota = "karyawan"
			}
		}
		if statusAnggota == "tenaga pendidikan" {
			statusAnggota = "karyawan"
		}

		fakultas := getCell(row, idxFakultas)
		if fakultas == "" {
			fakultas = "Rektoriat"
		}

		tglLahir := getCell(row, idxTglLahir)
		if tglLahir == "" {
			tglLahir = "2000-01-01"
		}

		jenisKelamin := getCell(row, idxJenisKelamin)
		if jenisKelamin == "" {
			jenisKelamin = "Laki-laki"
		}

		alamat := getCell(row, idxAlamat)
		if alamat == "" {
			alamat = "-"
		}

		gajiBulanan := 0
		gajiStr := getCell(row, idxGaji)
		if gajiStr != "" {
			gajiStr = strings.ReplaceAll(gajiStr, ".", "")
			gajiStr = strings.ReplaceAll(gajiStr, ",", "")
			if g, err := strconv.Atoi(gajiStr); err == nil && g >= 0 {
				gajiBulanan = g
			}
		}

		unitKerja := mapStatusToUnitKerja(statusAnggota)
		fakultasCode := mapFakultasToCode(fakultas)
		if unitKerja == "" {
			unitKerja = "02"
		}
		if fakultasCode == "" {
			fakultasCode = "09"
			fakultas = "Rektoriat"
		}

		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM anggota WHERE username = $1 OR no_telepon = $2", username, noTelepon).Scan(&exists); err != nil {
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal validasi data", rowNum))
			continue
		}
		if exists > 0 {
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: username/no telepon sudah terdaftar", rowNum))
			continue
		}

		lastNumber++
		nomorUrut := fmt.Sprintf("%04d", lastNumber)
		idAnggota := fmt.Sprintf("%s%s%s%s", unitKerja, fakultasCode, tahun, nomorUrut)

		insertQuery := `
			INSERT INTO anggota (
				id_anggota, nama_anggota, username, password, tgl_lahir,
				no_telepon, tgl_gabung, alamat, jenis_kelamin,
				status_anggota, fakultas, status, unit_kerja, fakultas_code,
				bukti_transfer, gaji_bulanan, tahun, nomor_urut
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, CURRENT_DATE, $7, $8,
				$9, $10, 'aktif', $11, $12,
				'', $13, $14, $15
			)
		`
		_, err = db.Exec(
			insertQuery,
			idAnggota, nama, username, password, tglLahir,
			noTelepon, alamat, jenisKelamin,
			statusAnggota, fakultas, unitKerja, fakultasCode,
			gajiBulanan, tahun, nomorUrut,
		)
		if err != nil {
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal simpan (%v)", rowNum, err))
			continue
		}

		successCount++
	}

	if successCount == 0 {
		c.Redirect(http.StatusFound, "/admin/anggota/tambah?error="+url.QueryEscape(fmt.Sprintf("Tidak ada data yang berhasil diimport. Gagal: %d", failedCount)))
		return
	}

	msg := fmt.Sprintf("Import anggota selesai. Berhasil: %d, Gagal: %d", successCount, failedCount)
	c.Redirect(http.StatusFound, "/admin/anggota/tambah?success="+url.QueryEscape(msg))
}

// AdminImportReferensiPendaftaran mengimpor data master referensi untuk validasi register.
func AdminImportReferensiPendaftaran(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=File referensi tidak ditemukan")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=Format file harus .xlsx atau .xls")
		return
	}

	tempPath := "./static/uploads/" + uuid.New().String() + ext
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=Gagal menyimpan file upload")
		return
	}
	defer os.Remove(tempPath)

	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=File Excel tidak bisa dibaca")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=File Excel tidak memiliki sheet")
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=Data referensi kosong atau tidak valid")
		return
	}

	db := config.GetDB()

	headerRowIdx := -1
	headerMap := map[string]int{}
	for r, row := range rows {
		tmp := map[string]int{}
		for i, header := range row {
			norm := normalizeHeader(header)
			if norm != "" {
				tmp[norm] = i
			}
		}

		idxNamaTmp := findHeaderIndex(tmp, "nama lengkap", "nama anggota", "nama")
		idxIdentitasTmp := findReferensiIdentitasHeaderIndex(tmp)
		if idxNamaTmp >= 0 && idxIdentitasTmp >= 0 {
			headerRowIdx = r
			headerMap = tmp
			break
		}
	}

	if headerRowIdx < 0 {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=Baris header template tidak ditemukan. Pastikan file memiliki kolom Nama Lengkap dan Nomer Identitas. Jika ada kolom Identitas berisi Dosen/Tendik, kolom itu akan dibaca sebagai Jabatan, bukan Nomer Identitas.")
		return
	}

	idxNama := findHeaderIndex(headerMap, "nama lengkap", "nama anggota", "nama")
	idxIdentitas := findReferensiIdentitasColumn(rows, headerRowIdx, headerMap)
	idxJabatan := findHeaderIndex(headerMap, "jabatan", "status anggota", "status_anggota", "status calon anggota", "kategori pegawai")
	if idxJabatan < 0 {
		idxJabatan = findHeaderIndex(headerMap, "identitas", "jenis pegawai", "status pegawai", "status")
		if idxJabatan == idxIdentitas {
			idxJabatan = -1
		}
	}
	idxGaji := findHeaderIndex(headerMap, "gajih bersih", "gaji bersih", "gaji bulanan", "gaji", "gajibulanan")
	idxStatusKeanggotaan := findHeaderIndex(headerMap, "status keanggotaan", "status data", "keanggotaan")

	if idxNama < 0 {
		c.Redirect(http.StatusFound, "/admin/import-referensi?error=Kolom Nama Lengkap wajib ada di file referensi")
		return
	}

	if idxIdentitas < 0 {
		message := "Kolom Nomer Identitas wajib ada di file referensi dan tidak boleh berisi nomor urut 1, 2, 3. Gunakan header Nomer Identitas/NIP/NIDN pada kolom nomor induk asli; kolom Identitas yang berisi Dosen/Tendik bukan nomor identitas."
		message += buildReferensiHeaderSampleMessage(rows, headerRowIdx)
		c.Redirect(http.StatusFound, "/admin/import-referensi?error="+url.QueryEscape(message))
		return
	}

	successCount := 0
	failedCount := 0
	insertedCount := 0
	updatedCount := 0
	warningCount := 0
	warnings := []string{}
	seenIdentitas := map[string]int{}
	addWarning := func(format string, args ...interface{}) {
		warning := fmt.Sprintf(format, args...)
		warningCount++
		log.Printf("[IMPORT-REFERENSI] %s", warning)
		if len(warnings) < 8 {
			warnings = append(warnings, warning)
		}
	}
	buildWarningMessage := func() string {
		if warningCount == 0 {
			return ""
		}

		msg := fmt.Sprintf("%d peringatan import: %s", warningCount, strings.Join(warnings, "; "))
		if warningCount > len(warnings) {
			msg += fmt.Sprintf("; dan %d peringatan lainnya. Cek log terminal untuk detail lengkap.", warningCount-len(warnings))
		}
		return msg
	}
	buildRedirectURL := func(key, message string) string {
		redirectURL := "/admin/import-referensi?" + key + "=" + url.QueryEscape(message)
		if warningMessage := buildWarningMessage(); warningMessage != "" {
			redirectURL += "&warning=" + url.QueryEscape(warningMessage)
		}
		return redirectURL
	}

	for i, row := range rows[headerRowIdx+1:] {
		rowNum := headerRowIdx + i + 2
		nama := getCell(row, idxNama)
		identitas := getCell(row, idxIdentitas)

		if nama == "" {
			failedCount++
			log.Printf("[IMPORT-REFERENSI] baris %d dilewati: nama kosong", rowNum)
			continue
		}
		if identitas == "" {
			failedCount++
			log.Printf("[IMPORT-REFERENSI] baris %d dilewati: identitas kosong", rowNum)
			continue
		}

		if firstRow, exists := seenIdentitas[identitas]; exists {
			addWarning("Baris %d: Nomer Identitas %s duplikat dengan baris %d di file upload; data baris ini tetap diproses dan akan memperbarui data dengan identitas tersebut.", rowNum, identitas, firstRow)
		} else {
			seenIdentitas[identitas] = rowNum
		}

		existingReferensi := false
		var existingID int
		checkErr := db.QueryRow(`
			SELECT id
			FROM referensi_pendaftaran
			WHERE COALESCE(nomor_identitas, '') = $1
			LIMIT 1
		`, identitas).Scan(&existingID)
		if checkErr == nil {
			existingReferensi = true
		} else if checkErr != sql.ErrNoRows {
			log.Printf("[IMPORT-REFERENSI] baris %d gagal cek duplikat identitas %s: %v", rowNum, identitas, checkErr)
		}

		statusKeanggotaan := strings.ToLower(strings.TrimSpace(getCell(row, idxStatusKeanggotaan)))
		switch statusKeanggotaan {
		case "anggota", "member", "sudah_anggota", "sudah anggota":
			statusKeanggotaan = "anggota"
		case "belum_anggota", "belum anggota", "non_anggota", "non anggota", "":
			statusKeanggotaan = "belum_anggota"
		default:
			statusKeanggotaan = "belum_anggota"
		}

		// Jika file template baru tidak menyertakan status keanggotaan, infer dari anggota yang sudah ada.
		if idxStatusKeanggotaan < 0 {
			var anggotaCount int
			checkErr := db.QueryRow(`
				SELECT COUNT(*)
				FROM anggota
				WHERE LOWER(TRIM(COALESCE(status, ''))) <> 'keluar'
				  AND (
					COALESCE(username, '') = $1
					OR COALESCE(no_telepon, '') = $1
				  )
			`, identitas).Scan(&anggotaCount)
			if checkErr == nil && anggotaCount > 0 {
				statusKeanggotaan = "anggota"
			} else {
				statusKeanggotaan = "belum_anggota"
			}
		}

		gajiBulanan := 0
		gajiStr := getCell(row, idxGaji)
		if gajiStr != "" {
			gajiStr = strings.ReplaceAll(gajiStr, ".", "")
			gajiStr = strings.ReplaceAll(gajiStr, ",", "")
			if parsedGaji, parseErr := strconv.Atoi(gajiStr); parseErr == nil && parsedGaji >= 0 {
				gajiBulanan = parsedGaji
			}
		}

		item := models.ReferensiPendaftaran{
			NamaLengkap:       nama,
			NomorIdentitas:    identitas,
			Jabatan:           strings.TrimSpace(getCell(row, idxJabatan)),
			GajiBulanan:       gajiBulanan,
			StatusKeanggotaan: statusKeanggotaan,
			SumberFile:        file.Filename,
		}

		if err := repository.UpsertReferensiPendaftaran(item); err != nil {
			failedCount++
			log.Printf("[IMPORT-REFERENSI] baris %d gagal disimpan: %v", rowNum, err)
			continue
		}

		successCount++
		if existingReferensi {
			updatedCount++
		} else {
			insertedCount++
		}
	}

	if successCount == 0 {
		c.Redirect(http.StatusFound, buildRedirectURL("error", "Tidak ada data referensi yang berhasil diimport"))
		return
	}

	msg := fmt.Sprintf("Import referensi pendaftaran berhasil: %d data diproses (%d data baru, %d data diperbarui), %d data gagal", successCount, insertedCount, updatedCount, failedCount)
	c.Redirect(http.StatusFound, buildRedirectURL("success", msg))
}

func AdminViewAnggota(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	simpananByJenis, err := repository.GetDetailSimpananByJenis(idStr)
	if err != nil {
		simpananByJenis = map[string]float64{
			"pokok":      0,
			"wajib":      0,
			"sukarela":   0,
			"hari_raya":  0,
			"umroh_haji": 0,
			"qurban":     0,
		}
	}

	totalSimpanan := totalSimpananTanpaPokokWajib(simpananByJenis)
	profilSimpananRows := buildProfilSimpananRows(simpananByJenis)

	_, totalPinjaman, _, err := repository.GetSaldoAnggota(idStr)
	if err != nil {
		totalPinjaman = 0
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	c.HTML(http.StatusOK, "admin_data_anggota_view.html", gin.H{
		"Anggota":            anggota,
		"ActivePage":         "anggota",
		"LogoPath":           logoPath,
		"CurrentLogo":        logoPath,
		"Title":              "Detail Anggota",
		"ProfilSimpananRows": profilSimpananRows,
		"SimpananPokok":      simpananByJenis["pokok"],
		"SimpananWajib":      simpananByJenis["wajib"],
		"SimpananSukarela":   simpananByJenis["sukarela"],
		"SimpananHariRaya":   simpananByJenis["hari_raya"],
		"SimpananUmrohHaji":  simpananByJenis["umroh_haji"],
		"SimpananQurban":     simpananByJenis["qurban"],
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
	})
}

func getLatestKopPath() string {
	files, err := os.ReadDir("static/uploads/kop")
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := strings.ToLower(file.Name())
		if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") && !strings.HasSuffix(name, ".png") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if modTime > latestTime {
			latestTime = modTime
			latestPath = "/static/uploads/kop/" + file.Name()
		}
	}

	return latestPath
}

func getLatestSignaturePath(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return ""
	}

	signDir := "static/uploads/signatures"
	files, err := os.ReadDir(signDir)
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	prefix := "ttd_" + role + "_"
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := strings.ToLower(file.Name())
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		ext := filepath.Ext(name)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if modTime > latestTime {
			latestTime = modTime
			latestPath = "/static/uploads/signatures/" + file.Name()
		}
	}

	return latestPath
}

// ViewRegistration menampilkan detail registrasi anggota pending
func ShowEditHalamanForm(c *gin.Context) {
	slug := c.Param("slug")
	halaman, err := repository.GetHalamanBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Parse konten JSON untuk template
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
		konten = map[string]interface{}{}
	}

	// Ensure misi is always an array for visi-misi page
	if slug == "visi-misi" {
		if misi, exists := konten["misi"]; exists {
			// If misi exists, ensure it's an array
			switch v := misi.(type) {
			case []interface{}:
				// Already an array, do nothing
			case string:
				// If it's a string, convert to array with single element
				konten["misi"] = []interface{}{v}
			default:
				// If it's something else, set to empty array
				konten["misi"] = []interface{}{}
			}
		} else {
			// If misi doesn't exist, set to empty array
			konten["misi"] = []interface{}{}
		}
	}

	// Pilih template berdasarkan slug (ganti - dengan _ untuk nama file)
	templateSlug := strings.ReplaceAll(slug, "-", "_")
	var templateName string
	activePage := templateSlug
	switch templateSlug {
	case "hubungi_kami":
		templateName = "admin_halaman_edit_hubungi_kami.html"
	case "dashboard_anggota":
		templateName = "admin_halaman_edit_dashboard.html"
		activePage = "dashboard_content"
	case "simpanan":
		templateName = "admin_halaman_edit_simpanan.html"
		activePage = "edit_simpanan"
	default:
		templateName = "admin_halaman_edit_" + templateSlug + ".html"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Halaman":                    halaman,
		"Konten":                     konten,
		"LogoPath":                   c.MustGet("LogoPath"),
		"CurrentKop":                 getLatestKopPath(),
		"CurrentSignatureKetua":      getLatestSignaturePath("ketua"),
		"CurrentSignatureBendahara":  getLatestSignaturePath("bendahara"),
		"CurrentSignatureSekretaris": getLatestSignaturePath("sekretaris"),
		"success":                    c.Query("success"),
		"error":                      c.Query("error"),
		"ActivePage":                 activePage,
	})
}

func AdminUploadKop(c *gin.Context) {
	file, err := c.FormFile("kop_file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal menerima file kop")
		return
	}

	kopDir := "static/uploads/kop/"
	if err := os.MkdirAll(kopDir, os.ModePerm); err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal membuat folder kop")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Hanya file JPG, JPEG, PNG yang diperbolehkan")
		return
	}

	filename := "kop_" + strconv.FormatInt(time.Now().Unix(), 10) + ext
	savePath := filepath.Join(kopDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal menyimpan file kop")
		return
	}

	c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?success=Kop surat berhasil diupload")
}

func AdminUploadSignature(c *gin.Context) {
	role := strings.ToLower(strings.TrimSpace(c.PostForm("sign_role")))
	if role != "ketua" && role != "bendahara" && role != "sekretaris" {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Peran tanda tangan tidak valid")
		return
	}

	file, err := c.FormFile("sign_file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menerima file tanda tangan")
		return
	}

	signDir := "static/uploads/signatures/"
	if err := os.MkdirAll(signDir, os.ModePerm); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal membuat folder tanda tangan")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Hanya file JPG, JPEG, PNG yang diperbolehkan untuk tanda tangan")
		return
	}

	filename := "ttd_" + role + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ext
	savePath := filepath.Join(signDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan file tanda tangan")
		return
	}

	c.Redirect(http.StatusFound, "/admin/tanda-tangan?success=Template tanda tangan berhasil diupload")
}

func AdminEditTandaTangan(c *gin.Context) {
	logoPath := c.MustGet("LogoPath")
	db := config.GetDB()
	getNamaTtd := func(key, def string) string {
		var val string
		err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&val)
		if err != nil || strings.TrimSpace(val) == "" {
			return def
		}
		return val
	}

	c.HTML(http.StatusOK, "admin_edit_tanda_tangan.html", gin.H{
		"LogoPath":                   logoPath,
		"CurrentLogo":                logoPath,
		"CurrentSignatureKetua":      getLatestSignaturePath("ketua"),
		"CurrentSignatureBendahara":  getLatestSignaturePath("bendahara"),
		"CurrentSignatureSekretaris": getLatestSignaturePath("sekretaris"),
		"NamaTtdKetua":               getNamaTtd("ttd_nama_ketua", "Ketua KOPMA"),
		"NamaTtdBendahara":           getNamaTtd("ttd_nama_bendahara", "Bendahara"),
		"NamaTtdSekretaris":          getNamaTtd("ttd_nama_sekretaris", "Sekretaris"),
		"success":                    c.Query("success"),
		"error":                      c.Query("error"),
		"ActivePage":                 "edit_tanda_tangan",
	})
}

func AdminUpdateSignatureNames(c *gin.Context) {
	namaKetua := strings.TrimSpace(c.PostForm("nama_ttd_ketua"))
	namaBendahara := strings.TrimSpace(c.PostForm("nama_ttd_bendahara"))
	namaSekretaris := strings.TrimSpace(c.PostForm("nama_ttd_sekretaris"))

	if namaKetua == "" {
		namaKetua = "Ketua KOPMA"
	}
	if namaBendahara == "" {
		namaBendahara = "Bendahara"
	}
	if namaSekretaris == "" {
		namaSekretaris = "Sekretaris"
	}

	db := config.GetDB()
	upsert := func(key, value string) error {
		_, err := db.Exec(`
			INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (nama_pengaturan)
			DO UPDATE SET nilai = EXCLUDED.nilai, updated_at = NOW()
		`, key, value)
		return err
	}

	if err := upsert("ttd_nama_ketua", namaKetua); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Ketua")
		return
	}
	if err := upsert("ttd_nama_bendahara", namaBendahara); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Bendahara")
		return
	}
	if err := upsert("ttd_nama_sekretaris", namaSekretaris); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Sekretaris")
		return
	}

	c.Redirect(http.StatusFound, "/admin/tanda-tangan?success=Nama penandatangan berhasil disimpan")
}

// UpdateHalaman memproses update konten halaman.
func UpdateHalaman(c *gin.Context) {
	slug := c.Param("slug")

	// Check if request is JSON (AJAX) or form data
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Handle JSON request (AJAX)
		var request struct {
			Judul  string `json:"judul"`
			Konten string `json:"konten"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Data tidak valid",
			})
			return
		}

		halaman := models.Halaman{
			Slug:   slug,
			Judul:  request.Judul,
			Konten: request.Konten,
		}

		err := repository.UpdateHalaman(halaman)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal memperbarui halaman",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Halaman berhasil diperbarui",
		})
		return
	}

	// Handle form data (fallback for non-AJAX requests)
	if slug == "dashboard_anggota" {
		// Handle special case for dashboard_anggota with separate fields
		teks := c.PostForm("teks")
		gambar := c.PostForm("gambar")
		if teks == "" || gambar == "" {
			c.String(http.StatusBadRequest, "Data tidak valid")
			return
		}
		kontenMap := map[string]string{
			"teks":   teks,
			"gambar": gambar,
		}
		kontenBytes, err := json.Marshal(kontenMap)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat konten")
			return
		}
		// Get existing halaman to keep judul
		existing, err := repository.GetHalamanBySlug(slug)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
			return
		}
		halaman := models.Halaman{
			Slug:   slug,
			Judul:  existing.Judul,
			Konten: string(kontenBytes),
		}
		err = repository.UpdateHalaman(halaman)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
			return
		}
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	// Handle halaman edit with JSON konten (like sejarah, visi-misi, struktur)
	kontenStr := c.PostForm("konten")
	if kontenStr != "" {
		// Parse JSON konten
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(kontenStr), &konten); err != nil {
			c.String(http.StatusBadRequest, "Konten tidak valid")
			return
		}

		// Get judul from form
		judul := c.PostForm("judul")
		if judul == "" {
			// Get existing judul if not provided
			existing, err := repository.GetHalamanBySlug(slug)
			if err != nil {
				c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
				return
			}
			judul = existing.Judul
		}

		// Convert konten back to JSON string
		kontenBytes, err := json.Marshal(konten)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat konten")
			return
		}

		halaman := models.Halaman{
			Slug:   slug,
			Judul:  judul,
			Konten: string(kontenBytes),
		}

		err = repository.UpdateHalaman(halaman)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
			return
		}

		c.Redirect(http.StatusFound, "/admin/halaman/edit/"+slug)
		return
	}

	var halaman models.Halaman
	if err := c.ShouldBind(&halaman); err != nil {
		c.String(http.StatusBadRequest, "Data tidak valid")
		return
	}
	halaman.Slug = slug

	err := repository.UpdateHalaman(halaman)
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada file yang diterima"})
		return
	}

	// Buat nama file yang unik untuk menghindari konflik
	extension := filepath.Ext(file.Filename)
	newFileName := uuid.New().String() + extension

	// Simpan file ke folder static/uploads
	err = c.SaveUploadedFile(file, "static/images/"+newFileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}

	// Kembalikan path file yang bisa diakses publik
	filePath := "/static/images/" + newFileName
	c.JSON(http.StatusOK, gin.H{"filePath": filePath})
}

// UploadStruktur handles file upload specifically for struktur page images
func UploadStruktur(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Tidak ada file yang diterima",
		})
		return
	}

	// Validasi tipe file
	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}
	fileType := file.Header.Get("Content-Type")
	isAllowed := false
	for _, allowedType := range allowedTypes {
		if fileType == allowedType {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format file tidak didukung. Gunakan JPG, PNG, atau GIF.",
		})
		return
	}

	// Validasi ukuran file (2MB)
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Ukuran file terlalu besar. Maksimal 2MB.",
		})
		return
	}

	// Buat nama file yang unik untuk struktur
	extension := filepath.Ext(file.Filename)
	if extension == "" {
		extension = ".png" // Default to PNG if no extension
	}
	newFileName := "struktur_" + uuid.New().String() + extension

	// Simpan file ke folder static/images
	err = c.SaveUploadedFile(file, "static/images/"+newFileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan file",
		})
		return
	}

	// Kembalikan path file yang bisa diakses publik
	filePath := "/static/images/" + newFileName
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Gambar berhasil diupload",
		"filePath": filePath,
	})
}

// AdminTransaksi menampilkan halaman transaksi admin dengan form input
func AdminTransaksi(c *gin.Context) {
	details, err := repository.GetAllDetails()
	if err != nil {
		details = []models.Detail{} // Default kosong jika error
	}

	pinjamans, err := repository.GetPendingPinjaman()
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	c.HTML(http.StatusOK, "admin_transaksi.html", gin.H{
		"ActivePage": "transaksi",
		"Details":    details,
		"Pinjamans":  pinjamans,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// CatatSimpanan memproses pencatatan simpanan
func CatatSimpanan(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var detail models.Detail
	if err := c.ShouldBind(&detail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	detail.TglTransaksi = time.Now()

	// Hitung total simpanan (kumulatif)
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(detail.IDAnggota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total simpanan"})
		return
	}
	detail.TotalSimpanan = totalSimpanan + detail.JumlahSimpanan

	err = repository.CreateSimpanan(detail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat simpanan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Simpanan berhasil dicatat"})
}

// CatatPinjaman memproses pencatatan pinjaman
func CatatPinjaman(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var pinjaman models.Pinjaman
	if err := c.ShouldBind(&pinjaman); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	pinjaman.IDPengelola.Int64 = int64(adminID.(int))
	pinjaman.TglPinjaman = time.Now()
	pinjaman.Status = "aktif"

	err := repository.CreatePinjaman(pinjaman)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat pinjaman"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pinjaman berhasil dicatat"})
}

// AdminRiwayat menampilkan halaman riwayat transaksi admin
func AdminRiwayat(c *gin.Context) {
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data riwayat"})
		return
	}

	c.HTML(http.StatusOK, "admin_riwayat.html", gin.H{
		"ActivePage":  "riwayat",
		"Riwayats":    riwayats,
		"LogoPath":    c.MustGet("LogoPath"),
		"CurrentLogo": c.MustGet("LogoPath"),
	})
}

// AdminLaporan menampilkan halaman laporan keuangan admin
func AdminLaporan(c *gin.Context) {
	// Ambil tipe laporan dari query parameter (default: bulanan)
	tipeLaporan := c.Query("tipe_laporan")
	if tipeLaporan == "" {
		tipeLaporan = "bulanan"
	}

	currentTime := time.Now()
	bulan := int(currentTime.Month())
	tahun := currentTime.Year()

	if tipeLaporan == "tahunan" {
		bulan = 0
	} else {
		if b := c.Query("bulan"); b != "" {
			if parsed, convErr := strconv.Atoi(b); convErr == nil {
				bulan = parsed
			}
		}
	}

	if t := c.Query("tahun"); t != "" {
		if parsed, convErr := strconv.Atoi(t); convErr == nil {
			tahun = parsed
		}
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_laporan.html", gin.H{
			"ActivePage":      "laporan",
			"Error":           "Gagal mengambil laporan",
			"CurrentLogo":     logoPath,
			"LogoPath":        logoPath,
			"Bulan":           bulan,
			"Tahun":           tahun,
			"TipeLaporan":     tipeLaporan,
			"LaporanBasePath": "/admin/laporan",
			"UseAdminLayout":  true,
			"ReadOnlyMode":    true,
		})
		return
	}

	// Ambil data neraca dari repository untuk ditampilkan read-only
	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)
	neraca, _ := getNeracaForRequest(c, neracaRepo)
	var data2024, data2023 map[string]interface{}
	if neraca != nil {
		_ = json.Unmarshal([]byte(neraca.Data2024), &data2024)
		_ = json.Unmarshal([]byte(neraca.Data2023), &data2023)
	}

	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		anggotas = []models.Anggota{}
	}

	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64)
	}

	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potongan := potonganBulanIni[anggota.IDAnggota]
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan
	}

	laporanDetail, err := repository.GetLaporanBulananPerAnggota(bulan, tahun)
	if err != nil {
		laporanDetail = []map[string]interface{}{}
	}
	labelByKey, customSimpananColumns := getLaporanSimpananColumns()
	hydrateCustomSimpananValuesToLaporanDetail(laporanDetail, customSimpananColumns, bulan, tahun)

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":            "laporan",
		"Report":                report,
		"Bulan":                 bulan,
		"Tahun":                 tahun,
		"TipeLaporan":           tipeLaporan,
		"CurrentLogo":           logoPath,
		"LogoPath":              logoPath,
		"Anggotas":              anggotas,
		"LaporanDetail":         laporanDetail,
		"SisaGaji":              sisaGaji,
		"SimpananLabelPokok":    labelByKey["simpanan_pokok"],
		"SimpananLabelWajib":    labelByKey["simpanan_wajib"],
		"SimpananLabelHariRaya": labelByKey["simpanan_hari_raya"],
		"SimpananLabelSukarela": labelByKey["simpanan_sukarela"],
		"CustomSimpananColumns": customSimpananColumns,
		"GetUnitKerjaName":      repository.GetUnitKerjaName,
		"NeracaData2024":        data2024,
		"NeracaData2023":        data2023,
		"LaporanBasePath":       "/admin/laporan",
		"UseAdminLayout":        true,
		"ReadOnlyMode":          true,
	})
}

func AdminDownloadLaporan(c *gin.Context) {
	KetuaDownloadLaporan(c)
}

func AdminGetNeraca(c *gin.Context) {
	KetuaGetNeraca(c)
}

// AdminTentang menampilkan halaman tentang kami admin
func AdminTentang(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_layout.html", gin.H{
		"ActivePage": "tentang",
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminPengaturan menampilkan halaman pengaturan admin (sekarang menampilkan daftar halaman statis)
func AdminPengaturan(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data halaman"})
		return
	}

	userList, err := repository.GetAllPengelola()
	if err != nil {
		userList = []models.Pengelola{}
	}

	anggotaList, err := repository.GetAnggotaByStatus("aktif")
	if err != nil {
		anggotaList = []models.Anggota{}
	}

	// Map judul ke nama keamanan
	securityTitles := map[string]string{
		"pinjaman": "Pengaturan Keamanan Data Pinjaman",
		"simpanan": "Pengaturan Keamanan Data Simpanan",
		"angsuran": "Pengaturan Keamanan Pembayaran",
	}

	// Filter out dashboard_anggota and struktur from the list
	var filteredHalaman []models.Halaman
	for _, halaman := range allHalaman {
		if halaman.Slug != "dashboard_anggota" && halaman.Slug != "struktur" {
			filteredHalaman = append(filteredHalaman, halaman)
		}
	}

	for i, halaman := range filteredHalaman {
		if title, ok := securityTitles[halaman.Slug]; ok {
			filteredHalaman[i].Judul = title
		}
	}

	// Ambil LogoPath dari context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	db := config.GetDB()
	getSetting := func(key string) string {
		var val string
		err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&val)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(val)
	}

	c.HTML(http.StatusOK, "admin_pengaturan.html", gin.H{
		"AllHalaman":  filteredHalaman,
		"UserList":    userList,
		"AnggotaList": anggotaList,
		"ActivePage":  "pengaturan",
		"LogoPath":    logoPath,
		"WAToken":     getSetting("wa_gateway_token"),
		"WAURL":       getSetting("wa_gateway_url"),
		"WAURLKetua":  getSetting("wa_url_ketua"),
		"AppBaseURL":  getSetting("app_base_url"),
		"WABendahara": getSetting("wa_bendahara_phone"),
		"WAKetua":     getSetting("wa_ketua_phone"),
		"WASuccess":   c.Query("wa_success"),
		"WAError":     c.Query("wa_error"),
		"CurrentLogo": logoPath,
	})
}

// AdminSaveWAGatewayConfig menyimpan konfigurasi token/url gateway WhatsApp notifikasi.
func AdminSaveWAGatewayConfig(c *gin.Context) {
	token := strings.TrimSpace(c.PostForm("wa_gateway_token"))
	waURLInput := strings.TrimSpace(c.PostForm("wa_gateway_url"))
	urlKetua := strings.TrimSpace(c.PostForm("wa_url_ketua"))
	appBaseURL := strings.TrimSpace(c.PostForm("app_base_url"))
	bendaharaPhone := strings.TrimSpace(c.PostForm("wa_bendahara_phone"))
	ketuaPhone := strings.TrimSpace(c.PostForm("wa_ketua_phone"))

	if token == "" {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Token WA wajib diisi")
		return
	}
	if waURLInput == "" {
		waURLInput = "https://api.fonnte.com/send"
	}
	normalizedURL, err := normalizeWAGatewayURL(waURLInput)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error="+url.QueryEscape(err.Error()))
		return
	}
	waURLInput = normalizedURL
	if isSuspiciousWAGatewayURL(nil, waURLInput) {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=URL gateway WA tidak boleh mengarah ke aplikasi sendiri")
		return
	}
	if urlKetua != "" {
		normalizedKetuaURL, err := normalizeWAGatewayURL(urlKetua)
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error="+url.QueryEscape("URL gateway WA Ketua tidak valid: "+err.Error()))
			return
		}
		if isSuspiciousWAGatewayURL(nil, normalizedKetuaURL) {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=URL gateway WA Ketua tidak boleh mengarah ke aplikasi sendiri")
			return
		}
		urlKetua = normalizedKetuaURL
	}

	db := config.GetDB()
	upsert := func(key, value string) error {
		_, err := db.Exec(`
			INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (nama_pengaturan)
			DO UPDATE SET nilai = EXCLUDED.nilai, updated_at = NOW()
		`, key, value)
		return err
	}

	if err := upsert("wa_gateway_token", token); err != nil {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan token WA")
		return
	}
	if err := upsert("wa_gateway_url", waURLInput); err != nil {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan URL gateway WA")
		return
	}
	if urlKetua != "" {
		if err := upsert("wa_url_ketua", urlKetua); err != nil {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan URL gateway WA Ketua")
			return
		}
	}
	if appBaseURL != "" {
		if err := upsert("app_base_url", appBaseURL); err != nil {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan App Base URL")
			return
		}
	}
	if bendaharaPhone != "" {
		if err := upsert("wa_bendahara_phone", bendaharaPhone); err != nil {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan nomor WA Bendahara")
			return
		}
	}
	if ketuaPhone != "" {
		if err := upsert("wa_ketua_phone", ketuaPhone); err != nil {
			c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Gagal menyimpan nomor WA Ketua")
			return
		}
	}

	c.Redirect(http.StatusFound, "/admin/pengaturan?wa_success=Konfigurasi WA berhasil disimpan")
}

func AdminTestWAGatewayConfig(c *gin.Context) {
	db := config.GetDB()
	token := strings.TrimSpace(getPengaturanValue(db, "wa_gateway_token"))
	if token == "" {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Token gateway WA belum disimpan")
		return
	}

	waURL := resolveWAGatewayURL(db, "wa_url_ketua")
	if _, err := normalizeWAGatewayURL(waURL); err != nil {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error="+url.QueryEscape(err.Error()))
		return
	}

	ketuaPhone := formatWhatsAppPhone(getKetuaWhatsAppPhone())
	if ketuaPhone == "" {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error=Nomor WA Ketua belum diisi")
		return
	}

	message := "Tes koneksi gateway WhatsApp dari aplikasi koperasi pada " + time.Now().Format("02-01-2006 15:04:05")
	if err := sendWhatsAppMessage(waURL, token, ketuaPhone, message, "[WA TEST]"); err != nil {
		c.Redirect(http.StatusFound, "/admin/pengaturan?wa_error="+url.QueryEscape(err.Error()))
		return
	}

	c.Redirect(http.StatusFound, "/admin/pengaturan?wa_success=Tes notifikasi WA berhasil dikirim ke nomor Ketua")
}

// UpdateAdminProfile memproses update username dan password admin
func UpdateAdminProfile(c *gin.Context) {
	// Ambil ID admin dari session
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var request struct {
		Username        string `form:"username" binding:"required"`
		Password        string `form:"password"`
		ConfirmPassword string `form:"confirm_password"`
	}

	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Validasi password jika diisi
	if request.Password != "" {
		if request.Password != request.ConfirmPassword {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password tidak cocok"})
			return
		}
		if len(request.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimal 6 karakter"})
			return
		}
	}

	// Password disimpan dalam bentuk plain text sesuai permintaan
	passwordToUpdate := request.Password
	plainPasswordToUpdate := ""
	if passwordToUpdate == "" {
		// Jika password kosong, ambil password lama
		admin, err := repository.GetPengelolaByID(adminID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data admin"})
			return
		}
		passwordToUpdate = admin.Password
		plainPasswordToUpdate = admin.PlainPassword
	} else {
		plainPasswordToUpdate = request.Password
	}

	// Update username, password, dan plain_password
	err := repository.UpdatePengelolaUsernamePassword(adminID.(int), request.Username, passwordToUpdate, plainPasswordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// AdminKeamananLogin menampilkan halaman keamanan login
func AdminKeamananLogin(c *gin.Context) {
	// Ambil data riwayat login dari database
	loginHistory, err := repository.GetLoginHistory()
	if err != nil {
		loginHistory = []models.LoginHistory{} // Default kosong jika error
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}
	c.HTML(http.StatusOK, "admin_keamanan_login.html", gin.H{
		"ActivePage":   "keamanan_login",
		"LoginHistory": loginHistory,
		"LogoPath":     logoPath,
	})
}

// DeleteLoginHistory menghapus riwayat login berdasarkan ID
func DeleteLoginHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	err = repository.DeleteLoginHistory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus riwayat login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Riwayat login berhasil dihapus"})
}

// AdminKeamananSimpanan menampilkan halaman keamanan data simpanan
func AdminKeamananSimpanan(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_simpanan.html", gin.H{
		"ActivePage": "pengaturan",
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminKeamananPinjaman menampilkan halaman keamanan data pinjaman
func AdminKeamananPinjaman(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_pinjaman.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananPembayaran menampilkan halaman keamanan pembayaran
func AdminKeamananPembayaran(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_pembayaran.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananDashboard menampilkan halaman keamanan dashboard
func AdminKeamananDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_dashboard.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananRiwayat menampilkan halaman keamanan riwayat
func AdminKeamananRiwayat(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_riwayat.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananOrganisasi menampilkan halaman keamanan organisasi
func AdminKeamananOrganisasi(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_organisasi.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminPesan menampilkan halaman pesan admin
func AdminPesan(c *gin.Context) {
	// Ambil session admin
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data admin
	admin, err := repository.GetPengelolaByID(adminID.(int))
	if err != nil {
		log.Printf("[ERROR] AdminPesan ambil data admin gagal (id=%v): %v", adminID, err)
		c.HTML(http.StatusInternalServerError, "admin_pesan.html", gin.H{
			"ActivePage": "pesan",
			"Error":      "Gagal mengambil data admin",
		})
		return
	}

	// Ambil daftar anggota untuk dropdown
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		anggotas = []models.Anggota{}
	}

	c.HTML(http.StatusOK, "admin_pesan.html", gin.H{
		"ActivePage": "pesan",
		"Admin":      admin,
		"Anggotas":   anggotas,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminBackground menampilkan halaman edit background dashboard anggota.
func AdminBackground(c *gin.Context) {
	logoPath := "/static/images/placeholder.png"
	if v, ok := c.Get("LogoPath"); ok {
		if s, okCast := v.(string); okCast && s != "" {
			logoPath = s
		}
	}

	currentBackground := "/static/images/placeholder.png"
	files, err := os.ReadDir("static/images")
	if err == nil {
		var latestTime int64
		for _, file := range files {
			name := file.Name()
			if strings.HasPrefix(name, "background_") && (strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
				info, errInfo := file.Info()
				if errInfo == nil && info.ModTime().Unix() > latestTime {
					latestTime = info.ModTime().Unix()
					currentBackground = "/static/images/" + name
				}
			}
		}
	}

	c.HTML(http.StatusOK, "admin_background.html", gin.H{
		"ActivePage":        "edit_background",
		"CurrentBackground": currentBackground,
		"CurrentLogo":       logoPath,
		"LogoPath":          logoPath,
	})
}

// AdminLogo menampilkan halaman edit logo admin
func AdminLogo(c *gin.Context) {
	// Cari logo terbaru yang sudah diupload
	files, err := os.ReadDir("static/images")
	if err != nil {
		c.HTML(http.StatusOK, "admin_logo.html", gin.H{
			"ActivePage": "edit_logo",
		})
		return
	}

	// Cari file logo terbaru (berdasarkan waktu modifikasi file)
	var latestLogo string
	var latestTime int64
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "logo_") && (strings.HasSuffix(file.Name(), ".png") || strings.HasSuffix(file.Name(), ".jpg")) {
			info, err := file.Info()
			if err == nil {
				modTime := info.ModTime().Unix()
				if modTime > latestTime {
					latestTime = modTime
					latestLogo = "/static/images/" + file.Name()
				}
			}
		}
	}
	// Jika tidak ada logo yang ditemukan, gunakan placeholder
	if latestLogo == "" {
		latestLogo = "/static/images/placeholder.png"
	}
	c.HTML(http.StatusOK, "admin_logo.html", gin.H{
		"ActivePage":  "edit_logo",
		"CurrentLogo": latestLogo,
		"LogoPath":    latestLogo,
	})
}

// UploadBackground memproses upload background dashboard anggota.
func UploadBackground(c *gin.Context) {
	file, err := c.FormFile("backgroundFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Tidak ada file yang dipilih",
		})
		return
	}

	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}
	fileType := file.Header.Get("Content-Type")
	isAllowed := false
	for _, allowedType := range allowedTypes {
		if fileType == allowedType {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format file tidak didukung. Gunakan JPG, PNG, atau GIF.",
		})
		return
	}

	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Ukuran file terlalu besar. Maksimal 5MB.",
		})
		return
	}

	extension := strings.ToLower(filepath.Ext(file.Filename))
	if extension == ".jpeg" {
		extension = ".jpg"
	}
	newFileName := "background_" + uuid.New().String() + extension
	savePath := "static/images/" + newFileName

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan file background",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "Background berhasil diupload",
		"backgroundPath": "/static/images/" + newFileName,
	})
}

// UploadLogo memproses upload logo baru
func UploadLogo(c *gin.Context) {
	// Cek apakah request menggunakan JSON atau FormData
	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var request struct {
			LogoData string `json:"logoData" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format data tidak valid",
			})
			return
		}

		logoData := request.LogoData
		if logoData == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Tidak ada data logo yang diterima",
			})
			return
		}

		// Decode base64 data
		// Format: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
		parts := strings.Split(logoData, ",")
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format data logo tidak valid",
			})
			return
		}

		// Decode base64
		imageData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Gagal decode data logo",
			})
			return
		}

		// Decode image (coba berbagai format)
		img, _, err := image.Decode(strings.NewReader(string(imageData)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Gagal decode gambar",
			})
			return
		}

		// Buat nama file unik
		newFileName := "logo_" + uuid.New().String() + ".png"

		// Simpan sebagai PNG transparan
		filePath := "static/images/" + newFileName
		file, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal membuat file",
			})
			return
		}
		defer file.Close()

		// Encode sebagai PNG
		err = png.Encode(file, img)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan gambar",
			})
			return
		}

		// Path file yang bisa diakses publik
		logoPath := "/static/images/" + newFileName

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "Logo berhasil diupload",
			"logoPath": logoPath,
		})
	} else {
		// Handle FormData request (fallback untuk file upload langsung)
		file, err := c.FormFile("logoFile")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Tidak ada file yang dipilih",
			})
			return
		}

		// Validasi tipe file
		allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}
		fileType := file.Header.Get("Content-Type")
		isAllowed := false
		for _, allowedType := range allowedTypes {
			if fileType == allowedType {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format file tidak didukung. Gunakan JPG, PNG, atau GIF.",
			})
			return
		}

		// Validasi ukuran file (2MB)
		if file.Size > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Ukuran file terlalu besar. Maksimal 2MB.",
			})
			return
		}

		// Buat nama file unik
		extension := filepath.Ext(file.Filename)
		newFileName := "logo_" + uuid.New().String() + extension

		// Simpan file ke folder static/images
		err = c.SaveUploadedFile(file, "static/images/"+newFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan file",
			})
			return
		}

		// Path file yang bisa diakses publik
		logoPath := "/static/images/" + newFileName

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "Logo berhasil diupload",
			"logoPath": logoPath,
		})
	}
}

// Handle JSON request (data canvas dari JavaScript)
