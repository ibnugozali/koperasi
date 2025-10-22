package controllers

import (
	"encoding/json" // Tambahkan ini
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"

)

func ShowHalaman(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		slug = "hubungi-kami"
	}
	halaman, err := repository.GetHalamanBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Cek session untuk info login di navbar
	session := sessions.Default(c)
	var anggota models.Anggota
	userID := session.Get("user_id")
	if userID != nil {
		anggota, _ = repository.GetAnggotaByID(userID.(int))
	}

	// Decode konten JSON menjadi sebuah map
	var kontenData map[string]interface{}
	json.Unmarshal([]byte(halaman.Konten), &kontenData)

	// Tentukan template yang akan digunakan berdasarkan slug
	var templateName string
	switch slug {
	case "pinjaman":
		templateName = "pinjaman.html"
	case "simpanan":
		templateName = "simpanan.html"
	case "angsuran":
		templateName = "angsuran.html"
	default:
		templateName = "halaman_statis.html"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Judul":   halaman.Judul,
		"Slug":    halaman.Slug,
		"Konten":  kontenData, // Kirim data yang sudah di-decode
		"Anggota": anggota,
	})
}

// ShowHubungiKami handles the /hubungi-kami route with a static page.
func ShowHubungiKami(c *gin.Context) {
	// Cek session untuk info login di navbar
	session := sessions.Default(c)
	var anggota models.Anggota
	userID := session.Get("user_id")
	if userID != nil {
		anggota, _ = repository.GetAnggotaByID(userID.(int))
	}

	c.HTML(http.StatusOK, "hubungi_kami.html", gin.H{
		"Judul":   "Hubungi Kami",
		"Anggota": anggota,
	})
}

// controllers/halaman_controller.go

// ... (kode controller lain)

// Controller untuk Riwayat (SUDAH DIPERBAIKI DAN DISESUAIKAN)
func ShowRiwayatPage(c *gin.Context) {
    session := sessions.Default(c)

    // ==========================================================
    // AMBIL DATA ANGGOTA DARI SESI (mengikuti pola dari ShowHalaman)
    // ==========================================================
    var anggota models.Anggota
    userID := session.Get("user_id")

    // Cek apakah pengguna sudah login atau belum
    if userID == nil {
        // Jika belum login, redirect ke halaman login
        c.Redirect(http.StatusFound, "/login")
        return
    }

    // Ambil data lengkap anggota dari repository
    // Pastikan untuk menangani error jika anggota tidak ditemukan
    anggota, err := repository.GetAnggotaByID(userID.(int))
    if err != nil {
        // Jika data anggota tidak ditemukan di DB (mungkin sesi aneh)
        // Sebaiknya redirect ke halaman login dan hapus sesi
        session.Clear()
        session.Save()
        c.Redirect(http.StatusFound, "/login?error=user_not_found")
        return
    }

    // Fetch all riwayat
    riwayatSimpanan, err := repository.GetRiwayatSimpananByAnggotaID(userID.(int))
    if err != nil {
        c.String(http.StatusInternalServerError, "Error fetching riwayat simpanan")
        return
    }
    riwayatPinjaman, err := repository.GetRiwayatPinjamanByAnggotaID(userID.(int))
    if err != nil {
        c.String(http.StatusInternalServerError, "Error fetching riwayat pinjaman")
        return
    }
    riwayatAngsuran, err := repository.GetRiwayatAngsuranByAnggotaID(userID.(int))
    if err != nil {
        c.String(http.StatusInternalServerError, "Error fetching riwayat angsuran")
        return
    }

    judulHalaman := "Riwayat Transaksi"

    // =========================================================================
    // KIRIM OBJEK ANGGOTA KE TEMPLATE (sama seperti di ShowHalaman)
    // =========================================================================
    c.HTML(http.StatusOK, "riwayat.html", gin.H{
        "title":            judulHalaman,
        "Judul":            judulHalaman,
        "RiwayatSimpanan": riwayatSimpanan,
        "RiwayatPinjaman": riwayatPinjaman,
        "RiwayatAngsuran": riwayatAngsuran,
        "Anggota":          anggota, // <-- INI KUNCI UTAMANYA
    })
}
