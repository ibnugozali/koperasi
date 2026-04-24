package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// sendBendaharaWhatsAppNotification mengirim notifikasi ke WA Bendahara (mirip ketua)
func sendBendaharaWhatsAppNotification(rawBendaharaPhone, namaAnggota, jenisTransaksi, nominal, appBaseURL string) error {
	db := config.GetDB()
	token := strings.TrimSpace(os.Getenv("WA_GATEWAY_TOKEN"))
	if token == "" {
		var tokenDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_token'").Scan(&tokenDB); err == nil {
			token = strings.TrimSpace(tokenDB)
		}
	}
	if token == "" {
		return fmt.Errorf("WA_GATEWAY_TOKEN belum diset (env/db)")
	}
	configuredPhone := ""
	if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_bendahara_phone'").Scan(&configuredPhone); err != nil && err != sql.ErrNoRows {
		log.Printf("[WA NOTIF] gagal baca wa_bendahara_phone: %v", err)
	}

	bendaharaPhone := strings.TrimSpace(configuredPhone)
	if bendaharaPhone == "" {
		bendaharaPhone = strings.TrimSpace(rawBendaharaPhone)
	}
	if bendaharaPhone == "" {
		return fmt.Errorf("nomor bendahara kosong")
	}
	bendaharaPhone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(bendaharaPhone)
	if strings.HasPrefix(bendaharaPhone, "0") {
		bendaharaPhone = "62" + bendaharaPhone[1:]
	} else if !strings.HasPrefix(bendaharaPhone, "62") {
		bendaharaPhone = "62" + bendaharaPhone
	}

	waURL := strings.TrimSpace(os.Getenv("WA_GATEWAY_URL"))
	if waURL == "" {
		var urlDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_url'").Scan(&urlDB); err == nil {
			waURL = strings.TrimSpace(urlDB)
		}
	}
	if waURL == "" {
		waURL = "https://api.fonnte.com/send"
	}

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	message := "Notifikasi transaksi anggota baru:\n" +
		"- Nama: " + namaAnggota + "\n" +
		"- Jenis Transaksi: " + jenisTransaksi + "\n" +
		"- Nominal: " + nominal + "\n"

	linkKonfirmasiBendahara := strings.TrimRight(appBaseURL, "/") + "/bendahara/konfirmasi-transaksi"
	message += "Silakan cek menu konfirmasi transaksi:\n" + linkKonfirmasiBendahara

	form := url.Values{"target": {bendaharaPhone}, "message": {message}}
	jsonBody, _ := json.Marshal(map[string]string{
		"target":  bendaharaPhone,
		"message": message,
	})

	type waAttempt struct {
		name        string
		contentType string
		body        string
		auth        string
	}

	attempts := []waAttempt{
		{name: "form/raw-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: token},
		{name: "form/bearer-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: "Bearer " + token},
		{name: "json/raw-token", contentType: "application/json", body: string(jsonBody), auth: token},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error

	for _, at := range attempts {
		req, err := http.NewRequest(http.MethodPost, waURL, strings.NewReader(at.body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", at.contentType)
		req.Header.Set("Authorization", at.auth)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[WA NOTIF] attempt=%s error=%v", at.name, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(bodyBytes))
		log.Printf("[WA NOTIF] attempt=%s status=%d response=%s", at.name, resp.StatusCode, bodyStr)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("gateway status %d", resp.StatusCode)
			continue
		}

		var parsed map[string]interface{}
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			if okVal, exists := parsed["status"]; exists {
				if okBool, ok := okVal.(bool); ok && !okBool {
					lastErr = fmt.Errorf("gateway reject: %s", bodyStr)
					continue
				}
			}
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("semua percobaan kirim WA gagal")
}

// ShowLoginPage menampilkan halaman login utama.
func ShowLoginPage(c *gin.Context) {
	status := c.Query("status")
	logoPath, _ := c.Get("LogoPath")

	// Ambil data halaman hubungi_kami dari database
	var kontak map[string]interface{}
	halaman, err := repository.GetHalamanBySlug("hubungi_kami")
	if err == nil {
		_ = json.Unmarshal([]byte(halaman.Konten), &kontak)
	} else {
		kontak = map[string]interface{}{}
	}
	// // Cari logo.png jika ada, jika tidak cari logo_ terbaru, jika tidak ada fallback ke placeholder.png
	// dirFiles, err := os.ReadDir("static/images")
	// var latestLogo string
	// var latestTime int64
	// foundLogoPNG := false
	// if err == nil {
	// 	for _, file := range dirFiles {
	// 		name := file.Name()
	// 		if name == "logo.png" {
	// 			latestLogo = "/static/images/logo.png"
	// 			foundLogoPNG = true
	// 			break
	// 		}
	// 	}
	// 	if !foundLogoPNG {
	// 		for _, file := range dirFiles {
	// 			name := file.Name()
	// 			if len(name) > 5 && name[:5] == "logo_" && (strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg")) {
	// 				info, err := file.Info()
	// 				if err == nil {
	// 					modTime := info.ModTime().Unix()
	// 					if modTime > latestTime {
	// 						latestTime = modTime
	// 						latestLogo = "/static/images/" + name
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	// if latestLogo == "" {
	// 	latestLogo = "/static/images/placeholder.png"
	// }
	// Cari logo terbaru di static/images
	dirFiles, errLogo := os.ReadDir("static/images")
	var latestLogo string
	var latestTime int64
	if errLogo == nil {
		for _, file := range dirFiles {
			name := file.Name()
			if (len(name) > 5 && name[:5] == "logo_" && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".jpg")) || name == "logo.png" {
				info, err := file.Info()
				if err == nil {
					modTime := info.ModTime().Unix()
					if modTime > latestTime {
						latestTime = modTime
						latestLogo = "/static/images/" + name
					}
				}
			}
		}
	}
	if latestLogo == "" {
		latestLogo = "/static/images/placeholder.png"
	}

	if status == "success_register" {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"success":     "Pendaftaran berhasil! Silakan tunggu konfirmasi dari ketua sebelum login.",
			"LogoPath":    logoPath,
			"CurrentLogo": latestLogo,
			"Konten":      kontak,
		})
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{
		"LogoPath":    logoPath,
		"CurrentLogo": latestLogo,
		"Konten":      kontak,
	})
}

// ShowRegisterPage menampilkan halaman registrasi.
func ShowRegisterPage(c *gin.Context) {
	db := config.GetDB()

	// Ambil nomor rekening dari database
	var nomorRekening string
	err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening)
	if err != nil {
		nomorRekening = "1234567890 (Bank ABC)" // Default jika belum diset
	}

	// Ambil nominal simpanan dari database
	var nominalSimpanan string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpanan)
	if err != nil {
		nominalSimpanan = "100000" // Default jika belum diset
	}

	// Ambil nomor ketua dari konten halaman hubungi_kami (fallback ke telepon admin)
	ketuaTelepon := ""
	halaman, errHalaman := repository.GetHalamanBySlug("hubungi_kami")
	if errHalaman == nil {
		var kontak map[string]interface{}
		if json.Unmarshal([]byte(halaman.Konten), &kontak) == nil {
			if v, ok := kontak["telepon_ketua"].(string); ok {
				ketuaTelepon = strings.TrimSpace(v)
			}
			if ketuaTelepon == "" {
				if v, ok := kontak["telepon"].(string); ok {
					ketuaTelepon = strings.TrimSpace(v)
				}
			}
		}
	}

	c.HTML(http.StatusOK, "register.html", gin.H{
		"NomorRekening":   nomorRekening,
		"NominalSimpanan": nominalSimpanan,
		"KetuaTelepon":    ketuaTelepon,
	})
}

// Register memproses data registrasi anggota baru.
func Register(c *gin.Context) {
	// Ambil nomor rekening dari database
	var nomorRekening string
	db := config.GetDB()
	err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening)
	if err != nil {
		nomorRekening = "1234567890 (Bank ABC)" // Default jika belum diset
	}

	// Ambil nominal simpanan dari database
	var nominalSimpanan string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpanan)
	if err != nil {
		nominalSimpanan = "100000" // Default jika belum diset
	}

	// Ambil nomor ketua dari konten halaman hubungi_kami (fallback ke telepon admin)
	ketuaTelepon := ""
	halaman, errHalaman := repository.GetHalamanBySlug("hubungi_kami")
	if errHalaman == nil {
		var kontak map[string]interface{}
		if json.Unmarshal([]byte(halaman.Konten), &kontak) == nil {
			if v, ok := kontak["telepon_ketua"].(string); ok {
				ketuaTelepon = strings.TrimSpace(v)
			}
			if ketuaTelepon == "" {
				if v, ok := kontak["telepon"].(string); ok {
					ketuaTelepon = strings.TrimSpace(v)
				}
			}
		}
	}

	renderRegisterError := func(status int, message string) {
		c.HTML(status, "register.html", gin.H{
			"error":           message,
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
			"KetuaTelepon":    ketuaTelepon,
		})
	}

	var newAnggota models.Anggota

	// Bind form data manually for multipart/form-data
	newAnggota.NamaAnggota = c.PostForm("NamaAnggota")
	newAnggota.Username = c.PostForm("Username")
	newAnggota.Password = c.PostForm("Password")
	newAnggota.TglLahir = c.PostForm("TglLahir")
	newAnggota.NikKTP = c.PostForm("NikKTP")
	// Jika NikKTP kosong, gunakan username sebagai default
	if newAnggota.NikKTP == "" {
		newAnggota.NikKTP = newAnggota.Username
	}
	newAnggota.NoTelepon = c.PostForm("NoTelepon")
	newAnggota.Alamat = c.PostForm("Alamat")
	newAnggota.JenisKelamin = c.PostForm("JenisKelamin")
	newAnggota.StatusAnggota = c.PostForm("StatusAnggota")
	newAnggota.Fakultas = c.PostForm("Fakultas")
	metodePembayaran := c.PostForm("MetodePembayaran")
	if metodePembayaran == "" {
		metodePembayaran = "transfer"
	}

	// Validasi: Username dan No. Telepon tidak boleh sama
	if newAnggota.Username == newAnggota.NoTelepon {
		renderRegisterError(http.StatusBadRequest, "Nama Pengguna dan No. Telepon tidak boleh sama.")
		return
	}

	// Validasi: Username dan No. Telepon tidak boleh sama dengan anggota lain
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM anggota WHERE username = $1 OR no_telepon = $2", newAnggota.Username, newAnggota.NoTelepon).Scan(&count)
	if err == nil && count > 0 {
		renderRegisterError(http.StatusBadRequest, "Nama Pengguna atau No. Telepon sudah terdaftar. Silakan gunakan data lain.")
		return
	}

	// Parse GajiBulanan
	gajiBulananStr := c.PostForm("GajiBulanan")
	if gajiBulananStr == "" {
		newAnggota.GajiBulanan = 0
	} else {
		gajiBulanan, err := strconv.Atoi(gajiBulananStr)
		if err == nil {
			newAnggota.GajiBulanan = gajiBulanan
		} else {
			newAnggota.GajiBulanan = 0
		}
	}

	// Metode pembayaran simpanan pokok: transfer atau potong gaji
	switch metodePembayaran {
	case "transfer":
		file, err := c.FormFile("BuktiTransfer")
		if err != nil {
			renderRegisterError(http.StatusBadRequest, "Bukti transfer wajib diupload jika memilih metode transfer.")
			return
		}

		filename := time.Now().Format("20060102150405") + "_" + file.Filename
		dst := "./static/uploads/" + filename
		if err := c.SaveUploadedFile(file, dst); err != nil {
			renderRegisterError(http.StatusInternalServerError, "Gagal menyimpan file")
			return
		}
		newAnggota.BuktiTransfer = filename
	case "potong_gaji":
		if newAnggota.StatusAnggota == "mahasiswa" {
			renderRegisterError(http.StatusBadRequest, "Metode potong gaji hanya untuk anggota dengan gaji. Untuk mahasiswa gunakan transfer.")
			return
		}
		newAnggota.BuktiTransfer = "POTONG_GAJI"
	default:
		renderRegisterError(http.StatusBadRequest, "Metode pembayaran tidak valid.")
		return
	}

	// Validate required fields
	if newAnggota.NamaAnggota == "" || newAnggota.Username == "" || newAnggota.Password == "" ||
		newAnggota.TglLahir == "" || newAnggota.NoTelepon == "" ||
		newAnggota.Alamat == "" || newAnggota.JenisKelamin == "" || newAnggota.StatusAnggota == "" ||
		newAnggota.Fakultas == "" {
		renderRegisterError(http.StatusBadRequest, "Semua field wajib diisi dengan benar")
		return
	}

	// Validasi gaji hanya untuk non-mahasiswa
	if newAnggota.StatusAnggota != "mahasiswa" && newAnggota.GajiBulanan <= 0 {
		renderRegisterError(http.StatusBadRequest, "Gaji bulanan wajib diisi untuk dosen dan tenaga pendidikan")
		return
	}

	// Map status_anggota to unit_kerja
	switch newAnggota.StatusAnggota {
	case "dosen":
		newAnggota.UnitKerja = "01"
	case "karyawan":
		newAnggota.UnitKerja = "02"
	case "mahasiswa":
		newAnggota.UnitKerja = "03"
	}

	// Map fakultas to fakultas_code
	switch newAnggota.Fakultas {
	case "Fakultas Agama Islam (FAI)":
		newAnggota.FakultasCode = "01"
	case "Fakultas Ekonomi (FE)":
		newAnggota.FakultasCode = "02"
	case "Fakultas Hukum (FH)":
		newAnggota.FakultasCode = "03"
	case "Fakultas Ilmu Sosial dan Ilmu Politik (FISIP)":
		newAnggota.FakultasCode = "04"
	case "Fakultas Keguruan dan Ilmu Pendidikan (FKIP)":
		newAnggota.FakultasCode = "05"
	case "Fakultas Kesehatan Masyarakat (FKM)":
		newAnggota.FakultasCode = "06"
	case "Fakultas Pertanian (FAPERTA)":
		newAnggota.FakultasCode = "07"
	case "Fakultas Teknik (FT)":
		newAnggota.FakultasCode = "08"
	case "Paskasarjana", "Pascasarjana":
		newAnggota.FakultasCode = "10"
	case "Rektorat / Yayasan", "Rektorat / Yayasan / Staff":
		newAnggota.FakultasCode = "09"
	}

	// Password disimpan dalam bentuk plain text sesuai permintaan
	newAnggota.TglGabung, _ = time.Parse("2006-01-02", time.Now().Format("2006-01-02"))

	// Generate temporary id_anggota for registration (will be updated during confirmation)
	// Use a temporary ID that will be replaced when admin confirms
	newAnggota.IDAnggota = "TEMP" + time.Now().Format("060102150405") // Temporary ID with timestamp

	err = repository.CreateAnggota(newAnggota)
	if err != nil {
		// Tambahkan log error ke console/server log
		println("[REGISTER ERROR]", err.Error())
		renderRegisterError(http.StatusInternalServerError, "Gagal menyimpan data: "+err.Error())
		return
	}

	// Notifikasi otomatis ke WhatsApp Ketua (jika gateway dikonfigurasi)
	appBaseURL := resolveAppBaseURL(c, db)
	if err := sendKetuaWhatsAppNotification(ketuaTelepon, newAnggota, metodePembayaran, appBaseURL); err != nil {
		log.Printf("[WA NOTIF] gagal kirim notifikasi ketua: %v", err)
	}

	c.Redirect(http.StatusFound, "/login?status=success_register")
}

// sendKetuaWhatsAppNotification mengirim notifikasi pendaftaran baru ke WA Ketua.
// Gunakan env:
// - WA_GATEWAY_TOKEN (wajib)
// - WA_GATEWAY_URL (opsional, default: https://api.fonnte.com/send)
func resolveAppBaseURL(c *gin.Context, db *sql.DB) string {
	baseURL := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	if baseURL == "" {
		var baseFromDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'app_base_url'").Scan(&baseFromDB); err == nil {
			baseURL = strings.TrimSpace(baseFromDB)
		}
	}
	if baseURL == "" && c != nil && c.Request != nil {
		scheme := "http"
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		if host := strings.TrimSpace(c.Request.Host); host != "" {
			baseURL = scheme + "://" + host
		}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL
}

func sendKetuaWhatsAppNotification(rawKetuaPhone string, anggota models.Anggota, metodePembayaran string, appBaseURL string) error {
	db := config.GetDB()
	token := strings.TrimSpace(os.Getenv("WA_GATEWAY_TOKEN"))
	if token == "" {
		var tokenDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_token'").Scan(&tokenDB); err == nil {
			token = strings.TrimSpace(tokenDB)
		}
	}
	if token == "" {
		return fmt.Errorf("WA_GATEWAY_TOKEN belum diset (env/db)")
	}

	ketuaPhone := strings.TrimSpace(rawKetuaPhone)
	if ketuaPhone == "" {
		return fmt.Errorf("nomor ketua kosong")
	}
	ketuaPhone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(ketuaPhone)
	if strings.HasPrefix(ketuaPhone, "0") {
		ketuaPhone = "62" + ketuaPhone[1:]
	} else if !strings.HasPrefix(ketuaPhone, "62") {
		ketuaPhone = "62" + ketuaPhone
	}

	waURL := strings.TrimSpace(os.Getenv("WA_GATEWAY_URL"))
	if waURL == "" {
		var urlDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_url'").Scan(&urlDB); err == nil {
			waURL = strings.TrimSpace(urlDB)
		}
	}
	if waURL == "" {
		waURL = "https://api.fonnte.com/send"
	}

	metode := "Transfer"
	if metodePembayaran == "potong_gaji" {
		metode = "Potong Gaji"
	}

	message := "Notifikasi pendaftaran calon anggota baru:\n" +
		"- Nama: " + anggota.NamaAnggota + "\n" +
		"- Username: " + anggota.Username + "\n" +
		"- No Telepon: +62" + anggota.NoTelepon + "\n" +
		"- Jabatan: " + anggota.StatusAnggota + "\n" +
		"- Unit Kerja: " + anggota.Fakultas + "\n" +
		"- Metode Simpanan Pokok: " + metode + "\n"

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}
	linkKonfirmasiKetua := strings.TrimRight(appBaseURL, "/") + "/ketua/konfirmasi"

	message += "Silakan cek menu konfirmasi anggota:\n" + linkKonfirmasiKetua

	form := url.Values{"target": {ketuaPhone}, "message": {message}}
	jsonBody, _ := json.Marshal(map[string]string{
		"target":  ketuaPhone,
		"message": message,
	})

	type waAttempt struct {
		name        string
		contentType string
		body        string
		auth        string
	}

	attempts := []waAttempt{
		{name: "form/raw-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: token},
		{name: "form/bearer-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: "Bearer " + token},
		{name: "json/raw-token", contentType: "application/json", body: string(jsonBody), auth: token},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error

	for _, at := range attempts {
		req, err := http.NewRequest(http.MethodPost, waURL, strings.NewReader(at.body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", at.contentType)
		req.Header.Set("Authorization", at.auth)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[WA NOTIF] attempt=%s error=%v", at.name, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(bodyBytes))
		log.Printf("[WA NOTIF] attempt=%s status=%d response=%s", at.name, resp.StatusCode, bodyStr)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("gateway status %d", resp.StatusCode)
			continue
		}

		var parsed map[string]interface{}
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			if okVal, exists := parsed["status"]; exists {
				if okBool, ok := okVal.(bool); ok && !okBool {
					lastErr = fmt.Errorf("gateway reject: %s", bodyStr)
					continue
				}
			}
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("semua percobaan kirim WA gagal")
}

func getKetuaWhatsAppPhone() string {
	db := config.GetDB()

	// 1) Prioritas dari pengaturan WA ketua di tabel pengaturan (akomodasi beberapa key lama/baru)
	for _, key := range []string{"wa_ketua_phone", "nomor_wa_ketua", "telepon_ketua"} {
		var configuredPhone string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&configuredPhone); err == nil {
			if p := strings.TrimSpace(configuredPhone); p != "" {
				return p
			}
		}
	}

	// 2) Ambil dari halaman hubungi_kami (telepon_ketua -> telepon)
	halaman, err := repository.GetHalamanBySlug("hubungi_kami")
	if err == nil && strings.TrimSpace(halaman.Konten) != "" {
		var kontak map[string]interface{}
		if json.Unmarshal([]byte(halaman.Konten), &kontak) == nil {
			if v, ok := kontak["telepon_ketua"].(string); ok {
				if p := strings.TrimSpace(v); p != "" {
					return p
				}
			}
			if v, ok := kontak["telepon"].(string); ok {
				if p := strings.TrimSpace(v); p != "" {
					return p
				}
			}
		}
	}

	// 3) Fallback ke user pengelola level ketua aktif
	var ketuaPhone string
	if err := db.QueryRow(`
		SELECT COALESCE(no_telepon, '')
		FROM pengelola
		WHERE LOWER(TRIM(level)) = 'ketua'
		  AND LOWER(TRIM(COALESCE(status, ''))) = 'aktif'
		ORDER BY id_pengelola ASC
		LIMIT 1
	`).Scan(&ketuaPhone); err == nil {
		return strings.TrimSpace(ketuaPhone)
	}

	return ""
}

// sendKetuaWhatsAppTransactionNotification mengirim notifikasi transaksi ke WA Ketua.
func sendKetuaWhatsAppTransactionNotification(rawKetuaPhone, namaAnggota, jenisTransaksi, nominal, appBaseURL string) error {
	db := config.GetDB()
	token := strings.TrimSpace(os.Getenv("WA_GATEWAY_TOKEN"))
	if token == "" {
		var tokenDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_token'").Scan(&tokenDB); err == nil {
			token = strings.TrimSpace(tokenDB)
		}
	}
	if token == "" {
		return fmt.Errorf("WA_GATEWAY_TOKEN belum diset (env/db)")
	}

	ketuaPhone := strings.TrimSpace(rawKetuaPhone)
	if ketuaPhone == "" {
		ketuaPhone = getKetuaWhatsAppPhone()
	}
	if ketuaPhone == "" {
		return fmt.Errorf("nomor ketua kosong")
	}
	ketuaPhone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(ketuaPhone)
	if strings.HasPrefix(ketuaPhone, "0") {
		ketuaPhone = "62" + ketuaPhone[1:]
	} else if !strings.HasPrefix(ketuaPhone, "62") {
		ketuaPhone = "62" + ketuaPhone
	}

	waURL := strings.TrimSpace(os.Getenv("WA_GATEWAY_URL"))
	if waURL == "" {
		var urlDB string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = 'wa_gateway_url'").Scan(&urlDB); err == nil {
			waURL = strings.TrimSpace(urlDB)
		}
	}
	if waURL == "" {
		waURL = "https://api.fonnte.com/send"
	}

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	linkKonfirmasiKetua := strings.TrimRight(appBaseURL, "/") + "/ketua/konfirmasi-transaksi"
	message := "Notifikasi transaksi dari bendahara:\n" +
		"- Nama Anggota: " + namaAnggota + "\n" +
		"- Jenis Transaksi: " + jenisTransaksi + "\n" +
		"- Nominal: " + nominal + "\n" +
		"Silakan cek menu konfirmasi transaksi Ketua:\n" + linkKonfirmasiKetua

	form := url.Values{"target": {ketuaPhone}, "message": {message}}
	jsonBody, _ := json.Marshal(map[string]string{
		"target":  ketuaPhone,
		"message": message,
	})

	type waAttempt struct {
		name        string
		contentType string
		body        string
		auth        string
	}

	attempts := []waAttempt{
		{name: "form/raw-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: token},
		{name: "form/bearer-token", contentType: "application/x-www-form-urlencoded", body: form.Encode(), auth: "Bearer " + token},
		{name: "json/raw-token", contentType: "application/json", body: string(jsonBody), auth: token},
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error

	for _, at := range attempts {
		req, err := http.NewRequest(http.MethodPost, waURL, strings.NewReader(at.body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", at.contentType)
		req.Header.Set("Authorization", at.auth)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[WA NOTIF KETUA] attempt=%s error=%v", at.name, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(bodyBytes))
		log.Printf("[WA NOTIF KETUA] attempt=%s status=%d response=%s", at.name, resp.StatusCode, bodyStr)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("gateway status %d", resp.StatusCode)
			continue
		}

		var parsed map[string]interface{}
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			if okVal, exists := parsed["status"]; exists {
				if okBool, ok := okVal.(bool); ok && !okBool {
					lastErr = fmt.Errorf("gateway reject: %s", bodyStr)
					continue
				}
			}
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("semua percobaan kirim WA ke ketua gagal")
}

// Login memproses otentikasi pengguna.
func Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	ipAddress := c.ClientIP()
	failedRole := "unknown"

	// Cek di tabel pengelola
	pengelola, err := repository.GetPengelolaByUsername(username)
	if err == nil {
		failedRole = pengelola.Level
		err = bcrypt.CompareHashAndPassword([]byte(pengelola.Password), []byte(password))
		if err == nil { // Password cocok
			session := sessions.Default(c)
			session.Set("user_id", pengelola.IDPengelola)
			session.Set("username", pengelola.Username)
			session.Set("role", pengelola.Level)
			session.Save()

			// Log successful login
			loginHistory := models.LoginHistory{
				Username:  username,
				Role:      pengelola.Level,
				LoginTime: time.Now(),
				IPAddress: ipAddress,
				Status:    "success",
			}
			repository.CreateLoginHistory(loginHistory)

			var redirectURL string
			switch pengelola.Level {
			case "admin":
				redirectURL = "/admin/dashboard"
			case "bendahara":
				redirectURL = "/bendahara/dashboard"
			case "ketua":
				redirectURL = "/ketua/dashboard"
			default:
				redirectURL = "/"
			}

			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"message":  "Login berhasil",
				"redirect": redirectURL,
			})
			return
		}
	}

	// Cek di tabel anggota
	anggota, err := repository.GetAnggotaByUsername(username)
	if err == nil {
		failedRole = "anggota"
		// Password disimpan dalam plain text, jadi bandingkan langsung
		if anggota.Password == password { // Password cocok
			session := sessions.Default(c)
			session.Set("user_id", anggota.IDAnggota)
			session.Set("username", anggota.Username)
			session.Set("role", "anggota")
			session.Save()

			// Log successful login
			loginHistory := models.LoginHistory{
				Username:  username,
				Role:      "anggota",
				LoginTime: time.Now(),
				IPAddress: ipAddress,
				Status:    "success",
			}
			repository.CreateLoginHistory(loginHistory)

			c.JSON(http.StatusOK, gin.H{
				"success":  true,
				"message":  "Login berhasil",
				"redirect": "/anggota/dashboard",
			})
			return
		}
	}

	// Log failed login attempt
	loginHistory := models.LoginHistory{
		Username:  username,
		Role:      failedRole,
		LoginTime: time.Now(),
		IPAddress: ipAddress,
		Status:    "failed",
	}
	repository.CreateLoginHistory(loginHistory)

	// Check if request expects JSON (AJAX request)
	if c.GetHeader("Accept") == "application/json" || c.ContentType() == "application/x-www-form-urlencoded" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Username atau password salah.",
		})
		return
	}

	// Cari logo terbaru di static/images untuk ditampilkan saat error login
	dirFiles, errLogo := os.ReadDir("static/images")
	var latestLogo string
	var latestTime int64
	if errLogo == nil {
		for _, file := range dirFiles {
			name := file.Name()
			if (len(name) > 5 && name[:5] == "logo_" && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".jpg")) || name == "logo.png" {
				info, err := file.Info()
				if err == nil {
					modTime := info.ModTime().Unix()
					if modTime > latestTime {
						latestTime = modTime
						latestLogo = "/static/images/" + name
					}
				}
			}
		}
	}
	if latestLogo == "" {
		latestLogo = "/static/images/placeholder.png"
	}

	// Ambil data halaman hubungi_kami dari database untuk modal kontak admin
	var kontak map[string]interface{}
	halaman, err := repository.GetHalamanBySlug("hubungi_kami")
	if err == nil {
		_ = json.Unmarshal([]byte(halaman.Konten), &kontak)
	} else {
		kontak = map[string]interface{}{}
	}
	c.HTML(http.StatusUnauthorized, "login.html", gin.H{
		"error":       "Username atau password salah.",
		"CurrentLogo": latestLogo,
		"Konten":      kontak,
	})
}

// Logout menghapus session pengguna.
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Logout berhasil",
		"redirect": "/login",
	})
}
