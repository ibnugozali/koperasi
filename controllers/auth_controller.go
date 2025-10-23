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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newAnggota.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Gagal memproses pendaftaran"})
		return
	}
	newAnggota.Password = string(hashedPassword)
	newAnggota.TglGabung, _ = time.Parse("2006-01-02", time.Now().Format("2006-01-02"))

	err = repository.CreateAnggota(newAnggota)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{"error": "Username atau NIK mungkin sudah terdaftar"})
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
