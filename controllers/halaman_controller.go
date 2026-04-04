package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
		c.HTML(http.StatusOK, "anggota_sejarah.html", gin.H{
			"Judul":       halaman.Judul,
			"Konten":      konten,
			"Anggota":     anggota,
			"CurrentLogo": latestLogo,
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

		// // Cari logo.png jika ada, jika tidak cari logo_ terbaru, jika tidak ada fallback ke placeholder.png
		// dirFiles, errLogo := os.ReadDir("static/images")
		// var latestLogo string
		// var latestTime int64
		// foundLogoPNG := false
		// if errLogo == nil {
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
		c.HTML(http.StatusOK, "anggota_visi_misi.html", gin.H{
			"Judul":       halaman.Judul,
			"Konten":      konten,
			"Anggota":     anggota,
			"CurrentLogo": latestLogo,
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

		// // Cari logo.png jika ada, jika tidak cari logo_ terbaru, jika tidak ada fallback ke placeholder.png
		// dirFiles, errLogo := os.ReadDir("static/images")
		// var latestLogo string
		// var latestTime int64
		// foundLogoPNG := false
		// if errLogo == nil {
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
		c.HTML(http.StatusOK, "anggota_struktur.html", gin.H{
			"Judul":       halaman.Judul,
			"Konten":      konten,
			"Anggota":     anggota,
			"CurrentLogo": latestLogo,
		})
	default:
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
	}
}

// ShowHubungiKami handles the /hubungi-kami route with a static page.
func ShowHubungiKami(c *gin.Context) {
	// Ambil data halaman hubungi_kami dari database
	halaman, err := repository.GetHalamanBySlug("hubungi_kami")
	var konten map[string]interface{}
	if err == nil {
		_ = json.Unmarshal([]byte(halaman.Konten), &konten)
	} else {
		konten = map[string]interface{}{}
	}

	// Cek session untuk info login di navbar
	session := sessions.Default(c)
	var anggota models.Anggota
	userID := session.Get("user_id")
	if userID != nil {
		anggota, _ = repository.GetAnggotaByID(userID.(string))
	}

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

	c.HTML(http.StatusOK, "hubungi_kami.html", gin.H{
		"Judul":       "Hubungi Kami",
		"Anggota":     anggota,
		"CurrentLogo": latestLogo,
		"Konten":      konten,
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
	riwayatPengambilan, err := repository.GetRiwayatPengambilanSimpananByAnggotaID(userID.(string), "")
	if err != nil {
		// Jika gagal ambil riwayat pengambilan simpanan, tetap tampilkan halaman dengan pengambilan kosong
		riwayatPengambilan = []models.PengambilanSimpanan{}
	}

	// Define a unified transaction struct
	type UnifiedTransaction struct {
		ID          int
		Date        time.Time
		Time        string
		Type        string
		Description string
		Amount      string
		Status      string
	}

	var allTransactions []UnifiedTransaction
	// Add simpanan transactions
	for _, s := range riwayatSimpanan {
		desc := "Simpanan " + s.Simpanan.JenisSimpanan
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", s.JumlahSimpanan)), " ", "")
		timeStr := s.TglTransaksi.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		var status string
		switch s.Status {
		case "pending":
			status = "Dalam Proses"
		case "confirmed":
			status = "Diterima"
		case "rejected":
			status = "Ditolak"
		default:
			status = "Diterima" // Default untuk data lama tanpa status
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          s.IDDetail,
			Date:        s.TglTransaksi,
			Time:        timeStr,
			Type:        desc,
			Description: desc,
			Amount:      amount,
			Status:      status,
		})
	}

	// Add pinjaman transactions
	for _, p := range riwayatPinjaman {
		desc := "Pinjaman"
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", p.JumlahPinjaman)), " ", "")
		var status string
		switch p.Status {
		case "proses":
			status = "Dalam Proses"
		case "aktif":
			status = "Diterima"
		case "gagal":
			status = "Ditolak"
		case "lunas":
			status = "Lunas"
		default:
			status = p.Status
		}
		timeStr := p.TglPinjaman.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          p.IDPinjaman,
			Date:        p.TglPinjaman,
			Time:        timeStr,
			Type:        "Pinjaman",
			Description: desc,
			Amount:      amount,
			Status:      status,
		})
	}

	// Add angsuran transactions
	for _, a := range riwayatAngsuran {
		desc := "Angsuran"
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", a.SisaPinjaman)), " ", "")
		var status string
		switch a.Status {
		case "pending":
			status = "Dalam Proses"
		case "confirmed":
			status = "Diterima"
		case "rejected":
			status = "Ditolak"
		case "valid":
			status = "Diterima" // Backward compatibility
		case "invalid":
			status = "Ditolak" // Backward compatibility
		default:
			status = "Diterima" // Default untuk data lama
		}
		timeStr := a.TglBayar.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          a.IDAngsuran,
			Date:        a.TglBayar,
			Time:        timeStr,
			Type:        "Angsuran",
			Description: desc,
			Amount:      amount,
			Status:      status,
		})
	}

	// Add pengambilan simpanan transactions
	for _, ps := range riwayatPengambilan {
		desc := "Penarikan " + ps.JenisSimpanan
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", ps.Jumlah)), " ", "")
		var status string
		switch ps.Status {
		case "pending":
			status = "Dalam Proses"
		case "approved":
			status = "Diterima"
		case "rejected":
			status = "Ditolak"
		default:
			status = ps.Status
		}
		timeStr := ps.TglPengajuan.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          ps.IDPengambilan,
			Date:        ps.TglPengajuan,
			Time:        timeStr,
			Type:        "Penarikan Simpanan",
			Description: desc,
			Amount:      amount,
			Status:      status,
		})
	}

	// Sort by date descending (newest first)
	sort.Slice(allTransactions, func(i, j int) bool {
		// Use UnixNano for strict comparison (descending - newest first)
		ti := allTransactions[i].Date.UnixNano()
		tj := allTransactions[j].Date.UnixNano()
		if ti == tj {
			// If timestamps exactly equal, tie-break by ID (descending - higher ID first, assuming auto-increment)
			return allTransactions[i].ID > allTransactions[j].ID
		}
		// Newest date first (descending order)
		return ti > tj
	})

	judulHalaman := "Riwayat Transaksi"

	// Cari logo terbaru di static/images
	dirFiles, errLogo := os.ReadDir("static/images")
	var latestLogo string
	var latestTime int64
	if errLogo == nil {
		for _, file := range dirFiles {
			name := file.Name()
			if len(name) > 5 && name[:5] == "logo_" && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".jpg") {
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

	// =========================================================================
	// KIRIM OBJEK ANGGOTA KE TEMPLATE (sama seperti di ShowHalaman)
	// =========================================================================
	c.HTML(http.StatusOK, "anggota_riwayat.html", gin.H{
		"Title":       judulHalaman,
		"Judul":       judulHalaman,
		"Riwayat":     allTransactions,
		"Anggota":     anggota,
		"Search":      search,
		"CurrentLogo": latestLogo,
	})
}
