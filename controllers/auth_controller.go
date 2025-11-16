package controllers

import (
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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
	c.HTML(http.StatusOK, "register.html", nil)
}

// Register memproses data registrasi anggota baru.
func Register(c *gin.Context) {
	var newAnggota models.Anggota
	if err := c.ShouldBind(&newAnggota); err != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{"error": "Data tidak valid"})
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newAnggota.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Gagal memproses pendaftaran"})
		return
	}
	newAnggota.Password = string(hashedPassword)
	newAnggota.TglGabung, _ = time.Parse("2006-01-02", time.Now().Format("2006-01-02"))

	// Generate temporary id_anggota for registration (will be updated during confirmation)
	// Use a temporary ID that will be replaced when admin confirms
	newAnggota.IDAnggota = "TEMP" + newAnggota.Username // Temporary ID

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
		err = bcrypt.CompareHashAndPassword([]byte(anggota.Password), []byte(password))
		if err == nil { // Password cocok
			session := sessions.Default(c)
			session.Set("user_id", anggota.IDAnggota)
			session.Set("username", anggota.Username)
			session.Set("role", "anggota")
			session.Save()
			c.Redirect(http.StatusFound, "/anggota/dashboard")
			return
		}
	}

	c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Username atau password salah."})
}

// Logout menghapus session pengguna.
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}
