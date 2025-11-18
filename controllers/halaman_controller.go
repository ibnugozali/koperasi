package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

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
		anggota, _ = repository.GetAnggotaByID(userID.(string))
	}

	// Decode konten JSON menjadi sebuah map
	var kontenData map[string]interface{}
	json.Unmarshal([]byte(halaman.Konten), &kontenData)

	// Tentukan template yang akan digunakan berdasarkan slug
	var templateName string
	switch slug {
	case "pinjaman":
		// Redirect ke halaman ajukan pinjaman anggota
		c.Redirect(http.StatusFound, "/anggota/ajukan-pinjaman")
		return
	case "simpanan":
		// Redirect ke halaman simpanan anggota
		c.Redirect(http.StatusFound, "/anggota/simpanan")
		return
	case "angsuran":
		// Redirect ke halaman angsuran anggota
		c.Redirect(http.StatusFound, "/anggota/angsuran")
		return
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

// ShowTentang handles the /tentang/:slug route for about pages.
func ShowTentang(c *gin.Context) {
	slug := c.Param("slug")

	// Cek session untuk info login di navbar
	session := sessions.Default(c)
	var anggota models.Anggota
	userID := session.Get("user_id")
	if userID != nil {
		anggota, _ = repository.GetAnggotaByID(userID.(string))
	}

	switch slug {
	case "sejarah":
		// Ambil data dari database
		halaman, err := repository.GetHalamanBySlug(slug)
		if err != nil {
			c.String(http.StatusNotFound, "Halaman tidak ditemukan")
			return
		}

		// Parse konten JSON
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
			konten = map[string]interface{}{}
		}

		c.HTML(http.StatusOK, "anggota_sejarah.html", gin.H{
			"Judul":   halaman.Judul,
			"Konten":  konten,
			"Anggota": anggota,
		})
	case "visi-misi":
		// Ambil data dari database
		halaman, err := repository.GetHalamanBySlug(slug)
		if err != nil {
			c.String(http.StatusNotFound, "Halaman tidak ditemukan")
			return
		}

		// Parse konten JSON
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
			konten = map[string]interface{}{}
		}

		c.HTML(http.StatusOK, "anggota_visi_misi.html", gin.H{
			"Judul":   halaman.Judul,
			"Konten":  konten,
			"Anggota": anggota,
		})
	case "struktur":
		// Ambil data dari database
		halaman, err := repository.GetHalamanBySlug(slug)
		if err != nil {
			c.String(http.StatusNotFound, "Halaman tidak ditemukan")
			return
		}

		// Parse konten JSON
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
			konten = map[string]interface{}{}
		}

		c.HTML(http.StatusOK, "anggota_struktur.html", gin.H{
			"Judul":   halaman.Judul,
			"Konten":  konten,
			"Anggota": anggota,
		})
	default:
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
	}
}

// ShowHubungiKami handles the /hubungi-kami route with a static page.
func ShowHubungiKami(c *gin.Context) {
	// Cek session untuk info login di navbar
	session := sessions.Default(c)
	var anggota models.Anggota
	userID := session.Get("user_id")
	if userID != nil {
		anggota, _ = repository.GetAnggotaByID(userID.(string))
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
	anggota, err := repository.GetAnggotaByID(userID.(string))
	if err != nil {
		// Jika data anggota tidak ditemukan di DB (mungkin sesi aneh)
		// Sebaiknya redirect ke halaman login dan hapus sesi
		session.Clear()
		session.Save()
		c.Redirect(http.StatusFound, "/login?error=user_not_found")
		return
	}

	// Get search query parameter
	search := c.DefaultQuery("search", "")

	// Fetch all riwayat without search filter first
	riwayatSimpanan, err := repository.GetRiwayatSimpananByAnggotaID(userID.(string), "")
	if err != nil {
		// Jika gagal ambil riwayat simpanan, tetap tampilkan halaman dengan simpanan kosong
		riwayatSimpanan = []models.Detail{}
	}
	riwayatPinjaman, err := repository.GetRiwayatPinjamanByAnggotaID(userID.(string), "")
	if err != nil {
		// Jika gagal ambil riwayat pinjaman, tetap tampilkan halaman dengan pinjaman kosong
		riwayatPinjaman = []models.Pinjaman{}
	}
	riwayatAngsuran, err := repository.GetRiwayatAngsuranByAnggotaID(userID.(string), "")
	if err != nil {
		// Jika gagal ambil riwayat angsuran, tetap tampilkan halaman dengan angsuran kosong
		riwayatAngsuran = []models.Angsuran{}
	}

	// Define a unified transaction struct
	type UnifiedTransaction struct {
		Date        time.Time
		Type        string
		Description string
		Amount      string
		Status      string
	}

	var allTransactions []UnifiedTransaction

	// Add simpanan transactions
	for _, s := range riwayatSimpanan {
		desc := "Simpanan: " + s.Simpanan.JenisSimpanan
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", s.JumlahSimpanan)), " ", "")
		if strings.Contains(strings.ToLower(desc+" "+amount), strings.ToLower(search)) {
			allTransactions = append(allTransactions, UnifiedTransaction{
				Date:        s.TglTransaksi,
				Type:        "Simpanan",
				Description: desc,
				Amount:      amount,
				Status:      "",
			})
		}
	}

	// Add pinjaman transactions
	for _, p := range riwayatPinjaman {
		desc := "Pinjaman"
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", p.JumlahPinjaman)), " ", "")
		status := p.Status
		if strings.Contains(strings.ToLower(desc+" "+amount+" "+status), strings.ToLower(search)) {
			allTransactions = append(allTransactions, UnifiedTransaction{
				Date:        p.TglPinjaman,
				Type:        "Pinjaman",
				Description: desc,
				Amount:      amount,
				Status:      status,
			})
		}
	}

	// Add angsuran transactions
	for _, a := range riwayatAngsuran {
		desc := "Angsuran"
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", a.SisaPinjaman)), " ", "")
		status := a.StatusAngsuran + " - " + a.Status
		if strings.Contains(strings.ToLower(desc+" "+amount+" "+status), strings.ToLower(search)) {
			allTransactions = append(allTransactions, UnifiedTransaction{
				Date:        a.TglBayar,
				Type:        "Angsuran",
				Description: desc,
				Amount:      amount,
				Status:      status,
			})
		}
	}

	// Sort by date descending
	sort.Slice(allTransactions, func(i, j int) bool {
		return allTransactions[i].Date.After(allTransactions[j].Date)
	})

	judulHalaman := "Riwayat Transaksi"

	// =========================================================================
	// KIRIM OBJEK ANGGOTA KE TEMPLATE (sama seperti di ShowHalaman)
	// =========================================================================
	c.HTML(http.StatusOK, "anggota_riwayat.html", gin.H{
		"Title": judulHalaman,

		"Judul":   judulHalaman,
		"Riwayat": allTransactions,
		"Anggota": anggota,
		"Search":  search,
	})
}
