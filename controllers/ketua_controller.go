package controllers

import (
	"encoding/json"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

)

// Menampilkan dashboard ketua dengan daftar calon anggota
func KetuaDashboard(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	// Ambil konten dashboard anggota untuk form edit
	dashboardHalaman, err := repository.GetHalamanBySlug("dashboard_anggota")
	var dashboardKonten map[string]interface{}
	if err == nil {
		// Parse JSON
		json.Unmarshal([]byte(dashboardHalaman.Konten), &dashboardKonten)
	} else {
		dashboardKonten = map[string]interface{}{
			"teks":    "Selamat datang di dashboard anggota.",
			"gambar":  "/static/images/placeholder.png",
			"welcome": "Selamat Datang di Koperasi Wirya",
			"slogan":  "Dari Anggota, Oleh Anggota, dan Untuk Anggota",
		}
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"PendingMembers":  pendingMembers,
		"DashboardKonten": dashboardKonten,
		"ActivePage":      "dashboard",
	})
}

// Menampilkan halaman konfirmasi anggota
func KetuaKonfirmasi(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage":     "konfirmasi",
	})
}

// ListHalaman menampilkan daftar semua halaman statis untuk di-edit.
func KetuaListHalaman(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
		return
	}
	c.HTML(http.StatusOK, "ketua_halaman_list.html", gin.H{
		"AllHalaman": allHalaman,
		"ActivePage": "halaman",
	})
}

// ListAllAnggota menampilkan daftar semua anggota aktif
func KetuaListAllAnggota(c *gin.Context) {
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}
	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"Anggotas":   anggotas,
		"ActivePage": "anggota",
	})
}

// ViewAnggota menampilkan detail anggota berdasarkan ID
func KetuaViewAnggota(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID tidak valid"})
		return
	}

	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	c.HTML(http.StatusOK, "ketua_anggota_view.html", gin.H{
		"Anggota": anggota,
	})
}

// KetuaTransaksi menampilkan halaman transaksi ketua dengan data semua transaksi
func KetuaTransaksi(c *gin.Context) {
	simpanans, err := repository.GetAllSimpanan()
	if err != nil {
		simpanans = []models.Simpanan{} // Default kosong jika error
	}

	details, err := repository.GetAllDetails()
	if err != nil {
		details = []models.Detail{} // Default kosong jika error
	}

	pinjamans, err := repository.GetAllPinjamans()
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage": "transaksi",
		"Simpanans":  simpanans,
		"Details":    details,
		"Pinjamans":  pinjamans,
	})
}

// KetuaLaporan menampilkan halaman laporan keuangan ketua
func KetuaLaporan(c *gin.Context) {
	// Ambil bulan dan tahun dari query parameter, default bulan ini
	bulan := 1
	tahun := 2023
	if b := c.Query("bulan"); b != "" {
		if parsed, err := strconv.Atoi(b); err == nil {
			bulan = parsed
		}
	}
	if t := c.Query("tahun"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			tahun = parsed
		}
	}

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_layout.html", gin.H{
			"ActivePage": "laporan",
			"Error":      "Gagal mengambil laporan",
		})
		return
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage": "laporan",
		"Report":     report,
		"Bulan":      bulan,
		"Tahun":      tahun,
	})
}

// KetuaTentang menampilkan halaman tentang kami ketua
func KetuaTentang(c *gin.Context) {
	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage": "tentang",
	})
}

// KetuaPengaturan menampilkan halaman pengaturan ketua
func KetuaPengaturan(c *gin.Context) {
	// Ambil ID ketua dari session
	session := sessions.Default(c)
	ketuaID := session.Get("user_id")
	if ketuaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data ketua
	ketua, err := repository.GetPengelolaByID(ketuaID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data ketua: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage": "pengaturan",
		"Ketua":      ketua,
	})
}

// UpdateKetuaProfile memproses update username dan password ketua
func UpdateKetuaProfile(c *gin.Context) {
	// Ambil ID ketua dari session
	session := sessions.Default(c)
	ketuaID := session.Get("user_id")
	if ketuaID == nil {
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

	// Hash password jika ada
	passwordToUpdate := request.Password
	if passwordToUpdate != "" {
		// Import bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordToUpdate), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}
		passwordToUpdate = string(hashedPassword)
	} else {
		// Jika password kosong, ambil password lama
		ketua, err := repository.GetPengelolaByID(ketuaID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data ketua"})
			return
		}
		passwordToUpdate = ketua.Password
	}

	// Update username dan password
	err := repository.UpdatePengelolaUsernamePassword(ketuaID.(int), request.Username, passwordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}
