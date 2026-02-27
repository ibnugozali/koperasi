package controllers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

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
			"success":     "Pendaftaran berhasil! Silakan tunggu konfirmasi dari admin sebelum login.",
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

	c.HTML(http.StatusOK, "register.html", gin.H{
		"NomorRekening":   nomorRekening,
		"NominalSimpanan": nominalSimpanan,
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

	// Validasi: Username dan No. Telepon tidak boleh sama
	if newAnggota.Username == newAnggota.NoTelepon {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":           "Nama Pengguna dan No. Telepon tidak boleh sama.",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
		return
	}

	// Validasi: Username dan No. Telepon tidak boleh sama dengan anggota lain
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM anggota WHERE username = $1 OR no_telepon = $2", newAnggota.Username, newAnggota.NoTelepon).Scan(&count)
	if err == nil && count > 0 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":           "Nama Pengguna atau No. Telepon sudah terdaftar. Silakan gunakan data lain.",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
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

	// Handle file upload
	file, err := c.FormFile("BuktiTransfer")
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":           "Bukti transfer wajib diupload",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
		return
	}

	// Save the uploaded file
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	dst := "./static/uploads/" + filename
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error":           "Gagal menyimpan file",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
		return
	}
	newAnggota.BuktiTransfer = filename

	// Validate required fields
	if newAnggota.NamaAnggota == "" || newAnggota.Username == "" || newAnggota.Password == "" ||
		newAnggota.TglLahir == "" || newAnggota.NoTelepon == "" ||
		newAnggota.Alamat == "" || newAnggota.JenisKelamin == "" || newAnggota.StatusAnggota == "" ||
		newAnggota.Fakultas == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":           "Semua field wajib diisi dengan benar",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
		return
	}

	// Validasi gaji hanya untuk non-mahasiswa
	if newAnggota.StatusAnggota != "mahasiswa" && newAnggota.GajiBulanan <= 0 {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":           "Gaji bulanan wajib diisi untuk dosen dan karyawan",
			"NomorRekening":   nomorRekening,
			"NominalSimpanan": nominalSimpanan,
		})
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
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Gagal menyimpan data: " + err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/login?status=success_register")
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
