package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func getPengaturanValue(db *sql.DB, keys ...string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		var value string
		if err := db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&value); err == nil {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}

	return ""
}

func resolveWAGatewayURL(db *sql.DB, preferredKeys ...string) string {
	waURL := strings.TrimSpace(os.Getenv("WA_GATEWAY_URL"))
	if waURL == "" {
		keys := append([]string{}, preferredKeys...)
		keys = append(keys, "wa_gateway_url")
		waURL = getPengaturanValue(db, keys...)
	}
	if isSuspiciousWAGatewayURL(db, waURL) {
		log.Printf("[WA NOTIF] URL gateway '%s' terlihat mengarah ke aplikasi sendiri, fallback ke wa_gateway_url", waURL)
		waURL = getPengaturanValue(db, "wa_gateway_url")
	}
	if waURL == "" {
		waURL = "https://api.fonnte.com/send"
	}

	return waURL
}

func normalizeWAGatewayURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL gateway WA kosong")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("URL gateway WA tidak valid: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("URL gateway WA harus memakai format lengkap, misalnya https://api.fonnte.com/send")
	}
	if parsedURL.Path == "" || parsedURL.Path == "/" {
		parsedURL.Path = "/send"
	}

	return parsedURL.String(), nil
}

func formatWhatsAppPhone(rawPhone string) string {
	phone := strings.TrimSpace(rawPhone)
	phone = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(phone)
	if strings.HasPrefix(phone, "0") && len(phone) > 1 {
		return "62" + phone[1:]
	}
	if phone != "" && !strings.HasPrefix(phone, "62") {
		return "62" + phone
	}
	return phone
}

func buildWhatsAppChatURL(rawPhone, message string) string {
	phone := formatWhatsAppPhone(rawPhone)
	if phone == "" {
		return ""
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			return ""
		}
	}

	chatURL := "https://wa.me/" + phone
	if trimmedMessage := strings.TrimSpace(message); trimmedMessage != "" {
		params := url.Values{}
		params.Set("text", trimmedMessage)
		chatURL += "?" + params.Encode()
	}
	return chatURL
}

func describeWAGatewayError(waURL string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("gagal resolve host gateway WA %s. Server/aplikasi tidak bisa menemukan domain tujuan. Cek DNS/internet server atau ubah URL gateway jika salah", waURL)
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() {
			return fmt.Errorf("koneksi ke gateway WA %s timeout. Cek internet server atau firewall outbound", waURL)
		}
	}

	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return fmt.Errorf("koneksi ke gateway WA %s timeout. Cek internet server atau firewall outbound", waURL)
	}

	errText := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(errText, "no such host"):
		return fmt.Errorf("gagal resolve host gateway WA %s. Server/aplikasi tidak bisa menemukan domain tujuan. Cek DNS/internet server atau ubah URL gateway jika salah", waURL)
	case strings.Contains(errText, "connection refused"):
		return fmt.Errorf("gateway WA %s menolak koneksi. Cek URL gateway, port, atau firewall server", waURL)
	case strings.Contains(errText, "timeout"):
		return fmt.Errorf("koneksi ke gateway WA %s timeout. Cek internet server atau firewall outbound", waURL)
	default:
		return fmt.Errorf("gagal menghubungi gateway WA %s: %w", waURL, err)
	}
}

func sendWhatsAppMessage(waURL, token, phone, message, logPrefix string) error {
	normalizedURL, err := normalizeWAGatewayURL(waURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token gateway WA kosong")
	}
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("nomor tujuan WA kosong")
	}

	form := url.Values{"target": {phone}, "message": {message}}
	jsonBody, _ := json.Marshal(map[string]string{
		"target":  phone,
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
		req, reqErr := http.NewRequest(http.MethodPost, normalizedURL, strings.NewReader(at.body))
		if reqErr != nil {
			lastErr = reqErr
			continue
		}
		req.Header.Set("Content-Type", at.contentType)
		req.Header.Set("Authorization", at.auth)

		resp, doErr := client.Do(req)
		if doErr != nil {
			lastErr = describeWAGatewayError(normalizedURL, doErr)
			log.Printf("%s attempt=%s error=%v", logPrefix, at.name, lastErr)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := strings.TrimSpace(string(bodyBytes))
		log.Printf("%s attempt=%s status=%d response=%s", logPrefix, at.name, resp.StatusCode, bodyStr)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("gateway WA merespons status %d", resp.StatusCode)
			continue
		}

		var parsed map[string]interface{}
		if json.Unmarshal(bodyBytes, &parsed) == nil {
			if okVal, exists := parsed["status"]; exists {
				if okBool, ok := okVal.(bool); ok && !okBool {
					lastErr = fmt.Errorf("gateway WA menolak request: %s", bodyStr)
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

func isSuspiciousWAGatewayURL(db *sql.DB, rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(parsedURL.Host))
	path := strings.TrimSpace(parsedURL.Path)
	if host == "" {
		return false
	}

	isLocalAppHost := strings.Contains(host, "localhost:8081") || strings.Contains(host, "127.0.0.1:8081")
	if isLocalAppHost && (path == "" || path == "/") {
		return true
	}

	appBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/")
	if appBaseURL == "" && db != nil {
		appBaseURL = strings.TrimRight(getPengaturanValue(db, "app_base_url"), "/")
	}
	if appBaseURL == "" {
		return false
	}

	parsedAppURL, err := url.Parse(appBaseURL)
	if err != nil {
		return false
	}

	appHost := strings.ToLower(strings.TrimSpace(parsedAppURL.Host))
	appPath := strings.TrimSpace(parsedAppURL.Path)
	return host == appHost && path == appPath
}

// getBendaharaWhatsAppPhone mengambil nomor WhatsApp bendahara dari pengaturan/profil aktif
func getBendaharaWhatsAppPhone() string {
	db := config.GetDB()

	// 1) Prioritas dari pengaturan WA bendahara di tabel pengaturan
	if configuredPhone := getPengaturanValue(db, "wa_bendahara_phone", "nomor_wa_bendahara", "telepon_bendahara"); configuredPhone != "" {
		return configuredPhone
	}

	// 2) Fallback ke user pengelola level bendahara aktif
	var bendaharaPhone string
	if err := db.QueryRow(`
		SELECT COALESCE(no_telepon, '')
		FROM pengelola
		WHERE LOWER(TRIM(level)) = 'bendahara'
		  AND LOWER(TRIM(COALESCE(status, ''))) = 'aktif'
		ORDER BY id_pengelola ASC
		LIMIT 1
	`).Scan(&bendaharaPhone); err == nil {
		return strings.TrimSpace(bendaharaPhone)
	}

	return ""
}

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

	bendaharaPhone := strings.TrimSpace(rawBendaharaPhone)
	if bendaharaPhone == "" {
		bendaharaPhone = getBendaharaWhatsAppPhone()
	}
	if bendaharaPhone == "" {
		return fmt.Errorf("nomor bendahara kosong")
	}
	bendaharaPhone = formatWhatsAppPhone(bendaharaPhone)

	waURL := resolveWAGatewayURL(db)

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	message := "Notifikasi transaksi anggota baru:\n" +
		"- Nama: " + namaAnggota + "\n" +
		"- Jenis Transaksi: " + jenisTransaksi + "\n" +
		"- Nominal: " + nominal + "\n"

	linkKonfirmasiBendahara := strings.TrimRight(appBaseURL, "/") + "/bendahara/konfirmasi-transaksi"
	message += "Silakan cek menu konfirmasi transaksi:\n" + linkKonfirmasiBendahara

	return sendWhatsAppMessage(waURL, token, bendaharaPhone, message, "[WA NOTIF]")
}

// sendAnggotaWhatsAppPesanNotification mengirim notifikasi WA ke anggota saat bendahara mengirim pesan.
func sendAnggotaWhatsAppPesanNotification(rawAnggotaPhone, namaAnggota, judulPesan, isiPesan, appBaseURL string) error {
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

	anggotaPhone := strings.TrimSpace(rawAnggotaPhone)
	if anggotaPhone == "" {
		return fmt.Errorf("nomor anggota kosong")
	}
	anggotaPhone = formatWhatsAppPhone(anggotaPhone)

	waURL := resolveWAGatewayURL(db)

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	message := "Halo " + namaAnggota + ", Anda menerima pesan baru dari Bendahara Koperasi.\n" +
		"- Judul: " + strings.TrimSpace(judulPesan) + "\n" +
		"- Isi: " + strings.TrimSpace(isiPesan) + "\n"

	linkPesan := strings.TrimRight(appBaseURL, "/") + "/anggota/pesan"
	message += "Silakan cek detail pesan di:\n" + linkPesan

	return sendWhatsAppMessage(waURL, token, anggotaPhone, message, "[WA PESAN ANGGOTA]")
}

// sendAnggotaWhatsAppTransactionApprovalNotification mengirim notifikasi transaksi ke anggota setelah ACC ketua.
func sendAnggotaWhatsAppTransactionApprovalNotification(rawAnggotaPhone, namaAnggota, jenisTransaksi, nominal, appBaseURL string) error {
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

	anggotaPhone := strings.TrimSpace(rawAnggotaPhone)
	if anggotaPhone == "" {
		return fmt.Errorf("nomor anggota kosong")
	}
	anggotaPhone = formatWhatsAppPhone(anggotaPhone)

	waURL := resolveWAGatewayURL(db)

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	message := "Halo " + namaAnggota + ", transaksi Anda telah disetujui oleh Ketua Koperasi.\n" +
		"- Jenis Transaksi: " + jenisTransaksi + "\n" +
		"- Nominal: " + nominal + "\n"

	linkRiwayat := strings.TrimRight(appBaseURL, "/") + "/anggota/riwayat"
	message += "Silakan cek riwayat transaksi di:\n" + linkRiwayat

	return sendWhatsAppMessage(waURL, token, anggotaPhone, message, "[WA NOTIF ANGGOTA]")
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
			"success":     "Pendaftaran berhasil! Silakan tunggu konfirmasi dari pengurus sebelum login.",
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

func normalizeRegisterCompare(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeReferensiJabatan(value string) string {
	value = normalizeRegisterCompare(value)
	switch {
	case strings.Contains(value, "dosen"):
		return "dosen"
	case strings.Contains(value, "tenaga pendidikan"),
		strings.Contains(value, "tendik"),
		strings.Contains(value, "karyawan"),
		strings.Contains(value, "pegawai"),
		strings.Contains(value, "staff"),
		strings.Contains(value, "staf"):
		return "karyawan"
	default:
		return ""
	}
}

func validateReferensiStatusAnggota(selectedStatus, jabatan string) string {
	selectedStatus = normalizeReferensiJabatan(selectedStatus)
	rawJabatan := strings.TrimSpace(jabatan)
	jabatan = normalizeReferensiJabatan(jabatan)

	if selectedStatus == "" {
		return ""
	}

	if rawJabatan == "" {
		return "Data referensi untuk Nomor Identitas ini belum memiliki jabatan. Silakan lengkapi jabatan pada import referensi admin sebelum mendaftar."
	}

	if jabatan == "" {
		return "Jabatan pada data referensi untuk Nomor Identitas ini belum dikenali sistem. Silakan perbarui data referensi admin sebelum mendaftar."
	}

	if selectedStatus == jabatan {
		return ""
	}

	if jabatan == "dosen" {
		return "Nomor Identitas ini terdaftar sebagai dosen di data referensi. Ubah Status Calon Anggota ke Dosen."
	}

	if jabatan == "karyawan" {
		return "Nomor Identitas ini terdaftar sebagai tenaga pendidikan di data referensi. Ubah Status Calon Anggota ke Tenaga Pendidikan."
	}

	return ""
}

func normalizeRegisterPhone(value string) string {
	value = strings.TrimSpace(value)

	var digits strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}

	phone := digits.String()
	phone = strings.TrimPrefix(phone, "62")
	phone = strings.TrimLeft(phone, "0")
	return phone
}

func isRegisterPhoneValid(value string) bool {
	if len(value) < 10 || len(value) > 13 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func parseNominalPengaturan(value string, fallback int) int {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func RegisterReferensiLookup(c *gin.Context) {
	nama := strings.TrimSpace(c.Query("nama"))
	identitas := strings.TrimSpace(c.Query("identitas"))

	if identitas == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"found": false,
			"error": "Nomer Identitas wajib diisi",
		})
		return
	}

	referensi, err := repository.FindReferensiPendaftaranForAutofill(nama, identitas)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"found": false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"found": false,
			"error": "Gagal mengambil data referensi",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"found":              true,
		"jabatan":            referensi.Jabatan,
		"status_referensi":   normalizeReferensiJabatan(referensi.Jabatan),
		"nama_lengkap":       referensi.NamaLengkap,
		"gaji_bulanan":       referensi.GajiBulanan,
		"status_keanggotaan": referensi.StatusKeanggotaan,
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
	nominalSimpananPokok := parseNominalPengaturan(nominalSimpanan, 100000)

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
	newAnggota.NamaAnggota = strings.TrimSpace(c.PostForm("NamaAnggota"))
	newAnggota.Username = strings.TrimSpace(c.PostForm("Username"))
	newAnggota.Password = strings.TrimSpace(c.PostForm("Password"))
	newAnggota.TglLahir = strings.TrimSpace(c.PostForm("TglLahir"))
	newAnggota.NoTelepon = normalizeRegisterPhone(c.PostForm("NoTelepon"))
	newAnggota.Alamat = strings.TrimSpace(c.PostForm("Alamat"))
	newAnggota.JenisKelamin = strings.TrimSpace(c.PostForm("JenisKelamin"))
	newAnggota.StatusAnggota = strings.TrimSpace(c.PostForm("StatusAnggota"))
	newAnggota.Fakultas = strings.TrimSpace(c.PostForm("Fakultas"))
	noIdentitasPegawai := strings.TrimSpace(c.PostForm("NoIdentitasPegawai"))
	metodePembayaran := c.PostForm("MetodePembayaran")
	if metodePembayaran == "" {
		metodePembayaran = "transfer_bank"
	}
	if metodePembayaran == "transfer" {
		metodePembayaran = "transfer_bank"
	}

	if !isRegisterPhoneValid(newAnggota.NoTelepon) {
		renderRegisterError(http.StatusBadRequest, "No. Telepon harus berisi 10 hingga 13 digit angka.")
		return
	}

	// Validasi: Username dan No. Telepon tidak boleh sama
	if newAnggota.Username == newAnggota.NoTelepon {
		renderRegisterError(http.StatusBadRequest, "Nama Pengguna dan No. Telepon tidak boleh sama.")
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

	if newAnggota.StatusAnggota != "mahasiswa" {
		if noIdentitasPegawai == "" {
			renderRegisterError(http.StatusBadRequest, "Nomer identitas wajib diisi sesuai data master import referensi.")
			return
		}

		referensiByIdentitas, err := repository.FindReferensiPendaftaranByIdentitas(noIdentitasPegawai)
		if err == nil {
			if normalizeRegisterCompare(referensiByIdentitas.StatusKeanggotaan) == "anggota" {
				renderRegisterError(http.StatusBadRequest, "Data Anda di master sudah tercatat sebagai anggota. Pendaftaran baru tidak dapat diproses.")
				return
			}

			if statusWarning := validateReferensiStatusAnggota(newAnggota.StatusAnggota, referensiByIdentitas.Jabatan); statusWarning != "" {
				renderRegisterError(http.StatusBadRequest, statusWarning)
				return
			}
		} else if err != sql.ErrNoRows {
			renderRegisterError(http.StatusInternalServerError, "Gagal memvalidasi data referensi pendaftaran.")
			return
		}

		_, err = repository.FindReferensiPendaftaranForRegister(
			newAnggota.NamaAnggota,
			noIdentitasPegawai,
			newAnggota.GajiBulanan,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				renderRegisterError(http.StatusBadRequest, "Data referensi tidak cocok. Pastikan Nama Lengkap, Nomer Identitas, dan Gajih Bersih sama dengan data di import referensi admin.")
				return
			}
			renderRegisterError(http.StatusInternalServerError, "Gagal memvalidasi data referensi pendaftaran.")
			return
		}

		var count int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM anggota
			WHERE LOWER(TRIM(COALESCE(nama_anggota, ''))) = LOWER(TRIM($1))
			  AND COALESCE(username, '') = $2
			  AND COALESCE(gaji_bulanan, 0) = $3
		`, newAnggota.NamaAnggota, noIdentitasPegawai, newAnggota.GajiBulanan).Scan(&count)
		if err == nil && count > 0 {
			renderRegisterError(http.StatusBadRequest, "Data dengan Nama Lengkap, Nomer Identitas, dan Gajih Bersih tersebut sudah terdaftar.")
			return
		}
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM anggota WHERE username = $1", newAnggota.Username).Scan(&count)
	if err == nil && count > 0 {
		renderRegisterError(http.StatusBadRequest, "Nama Pengguna sudah terdaftar. Silakan gunakan nama pengguna lain.")
		return
	}

	// Metode pembayaran simpanan pokok: transfer bank, potong gaji, atau tunai
	switch metodePembayaran {
	case "transfer_bank":
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
		if newAnggota.GajiBulanan < nominalSimpananPokok {
			renderRegisterError(http.StatusBadRequest, fmt.Sprintf("Gaji bersih harus minimal Rp %d untuk menggunakan metode potong gaji karena nominal simpanan pokok saat ini Rp %d.", nominalSimpananPokok, nominalSimpananPokok))
			return
		}
		newAnggota.BuktiTransfer = "POTONG_GAJI"
	case "tunai":
		newAnggota.BuktiTransfer = "TUNAI"
	default:
		renderRegisterError(http.StatusBadRequest, "Metode pembayaran tidak valid.")
		return
	}
	if metodePembayaran == "transfer_bank" || metodePembayaran == "tunai" {
		newAnggota.Status = repository.StatusAnggotaPendingBendahara
	} else {
		newAnggota.Status = repository.StatusAnggotaPending
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
		log.Printf("[ERROR] RegisterAnggota simpan data gagal: %v", err)
		renderRegisterError(http.StatusInternalServerError, "Gagal menyimpan data")
		return
	}
	if err := repository.SyncReferensiPendaftaranStatusFromAnggota(); err != nil {
		log.Printf("[WARN] gagal sinkron status referensi pendaftaran setelah register: %v", err)
	}

	// Notifikasi otomatis ke WhatsApp pengurus sesuai tahap konfirmasi awal.
	appBaseURL := resolveAppBaseURL(c, db)
	if newAnggota.Status == repository.StatusAnggotaPendingBendahara {
		nominalLabel := fmt.Sprintf("Rp %d", nominalSimpananPokok)
		if err := sendBendaharaWhatsAppNotification("", newAnggota.NamaAnggota, "Simpanan Pokok Registrasi", nominalLabel, appBaseURL); err != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi bendahara: %v", err)
		}
	} else {
		// Kirim dengan nomor kosong agar helper selalu memakai prioritas konfigurasi resmi:
		// wa_ketua_phone -> halaman hubungi_kami -> profil pengelola ketua aktif.
		if err := sendKetuaWhatsAppNotification("", newAnggota, metodePembayaran, appBaseURL); err != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi ketua: %v", err)
		}
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
		ketuaPhone = getKetuaWhatsAppPhone()
	}
	if ketuaPhone == "" {
		return fmt.Errorf("nomor ketua kosong")
	}
	ketuaPhone = formatWhatsAppPhone(ketuaPhone)

	waURL := resolveWAGatewayURL(db, "wa_url_ketua")

	metode := "Transfer Bank"
	switch metodePembayaran {
	case "potong_gaji":
		metode = "Potong Gaji"
	case "tunai":
		metode = "Tunai"
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

	return sendWhatsAppMessage(waURL, token, ketuaPhone, message, "[WA NOTIF]")
}

func getKetuaWhatsAppPhone() string {
	db := config.GetDB()

	// 1) Prioritas dari pengaturan WA ketua di tabel pengaturan (akomodasi beberapa key lama/baru)
	if configuredPhone := getPengaturanValue(db, "wa_ketua_phone", "nomor_wa_ketua", "telepon_ketua"); configuredPhone != "" {
		return configuredPhone
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
	ketuaPhone = formatWhatsAppPhone(ketuaPhone)

	waURL := resolveWAGatewayURL(db, "wa_url_ketua")

	if strings.TrimSpace(appBaseURL) == "" {
		appBaseURL = "http://localhost:8081"
	}

	linkKonfirmasiKetua := strings.TrimRight(appBaseURL, "/") + "/ketua/konfirmasi-transaksi"
	message := "Notifikasi transaksi dari bendahara:\n" +
		"- Nama Anggota: " + namaAnggota + "\n" +
		"- Jenis Transaksi: " + jenisTransaksi + "\n" +
		"- Nominal: " + nominal + "\n" +
		"Silakan cek menu konfirmasi transaksi Ketua:\n" + linkKonfirmasiKetua

	return sendWhatsAppMessage(waURL, token, ketuaPhone, message, "[WA NOTIF KETUA]")
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
