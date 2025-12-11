package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/repository"

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

	c.HTML(http.StatusOK, "ketua_dashboard.html", gin.H{
		"PendingMembers":  pendingMembers,
		"DashboardKonten": dashboardKonten,
		"ActivePage":      "dashboard",
	})
}

// Menampilkan halaman data anggota dengan status 'pending'
func KetuaDataAnggota(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	c.HTML(http.StatusOK, "ketua_data_anggota.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage":     "anggota",
	})
}

// Menampilkan halaman riwayat login
func KetuaRiwayat(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
		return
	}
	c.HTML(http.StatusOK, "ketua_riwayat.html", gin.H{
		"AllHalaman": allHalaman,
		"ActivePage": "halaman",
	})
}

// Menampilkan halaman laporan anggota
func KetuaLaporan(c *gin.Context) {
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}
	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"Anggotas":   anggotas,
		"ActivePage": "anggota",
	})
}

// Menampilkan halaman pengaturan ketua
func KetuaPengaturan(c *gin.Context) {		
	c.HTML(http.StatusOK, "ketua_pengaturan.html", gin.H{
		"ActivePage": "pengaturan",
	})
}