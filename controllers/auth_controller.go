package controllers

import (
	"net/http"
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
	if status == "success_register" {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"success": "Pendaftaran berhasil! Silakan tunggu konfirmasi dari admin sebelum login.",
		})
		return
	}
	c.HTML(http.StatusOK, "login.html", nil)
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
	var newAnggota models.Anggota

	// Bind form data manually for multipart/form-data
	newAnggota.NamaAnggota = c.PostForm("NamaAnggota")
	newAnggota.Username = c.PostForm("Username")
	newAnggota.Password = c.PostForm("Password")
	newAnggota.TglLahir = c.PostForm("TglLahir")
	newAnggota.NikKTP = c.PostForm("NikKTP")
	newAnggota.NoTelepon = c.PostForm("NoTelepon")
	newAnggota.Alamat = c.PostForm("Alamat")
	newAnggota.JenisKelamin = c.PostForm("JenisKelamin")
	newAnggota.StatusAnggota = c.PostForm("StatusAnggota")
	newAnggota.Fakultas = c.PostForm("Fakultas")

	// Handle file upload
	file, err := c.FormFile("BuktiTransfer")
	if err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "Bukti transfer wajib diupload"})
		return
	}

	// Save the uploaded file
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	dst := "./static/uploads/" + filename
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Gagal menyimpan file"})
		return
	}
	newAnggota.BuktiTransfer = filename

	// Validate required fields
	if newAnggota.NamaAnggota == "" || newAnggota.Username == "" || newAnggota.Password == "" ||
		newAnggota.TglLahir == "" || newAnggota.NikKTP == "" || newAnggota.NoTelepon == "" ||
		newAnggota.Alamat == "" || newAnggota.JenisKelamin == "" || newAnggota.StatusAnggota == "" ||
		newAnggota.Fakultas == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "Semua field wajib diisi"})
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
	case "Rektorat / Yayasan / Staff":
		newAnggota.FakultasCode = "09"
	}

	// Password disimpan dalam bentuk plain text sesuai permintaan
	newAnggota.TglGabung, _ = time.Parse("2006-01-02", time.Now().Format("2006-01-02"))

	// Generate temporary id_anggota for registration (will be updated during confirmation)
	// Use a temporary ID that will be replaced when admin confirms
	newAnggota.IDAnggota = "TEMP" + time.Now().Format("060102150405") // Temporary ID with timestamp

	err = repository.CreateAnggota(newAnggota)
	if err != nil {
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

	// Cek di tabel pengelola
	pengelola, err := repository.GetPengelolaByUsername(username)
	if err == nil {
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

			switch pengelola.Level {
			case "admin":
				c.Redirect(http.StatusFound, "/admin/dashboard")
			case "bendahara":
				c.Redirect(http.StatusFound, "/bendahara/dashboard")
			case "ketua":
				c.Redirect(http.StatusFound, "/ketua/dashboard")
			default:
				c.Redirect(http.StatusFound, "/")
			}
			return
		}
	}

	// Cek di tabel anggota
	anggota, err := repository.GetAnggotaByUsername(username)
	if err == nil {
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

			c.Redirect(http.StatusFound, "/anggota/dashboard")
			return
		}
	}

	// Log failed login attempt
	loginHistory := models.LoginHistory{
		Username:  username,
		Role:      "unknown",
		LoginTime: time.Now(),
		IPAddress: ipAddress,
		Status:    "failed",
	}
	repository.CreateLoginHistory(loginHistory)

	c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Username atau password salah."})
}

// Logout menghapus session pengguna.
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
