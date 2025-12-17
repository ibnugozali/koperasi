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

	"koperasi-simpan-pinjam/models"
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

	c.HTML(http.StatusOK, "ketua_dashboard.html", gin.H{
		"PendingMembers":  pendingMembers,
		"DashboardKonten": dashboardKonten,
		"ActivePage":      "dashboard",
		"CurrentLogo":     latestLogo,
	})
}

// Menampilkan halaman data anggota dengan status 'pending'
func KetuaDataAnggota(c *gin.Context) {
	// Ambil semua anggota (bukan hanya pending)
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
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

	c.HTML(http.StatusOK, "ketua_data_anggota.html", gin.H{
		"Anggotas":    anggotas,
		"ActivePage":  "anggota",
		"CurrentLogo": latestLogo,
	})
}

// Menampilkan halaman riwayat login
func KetuaRiwayat(c *gin.Context) {
	// Ambil data riwayat login dari database
	loginHistory, err := repository.GetLoginHistory()
	if err != nil {
		loginHistory = []models.LoginHistory{} // Default kosong jika error
	}

	// Cari logo terbaru di static/images (opsional, jika ingin tampilkan logo)
	// ... (bisa ditambahkan jika ingin)

	c.HTML(http.StatusOK, "ketua_riwayat_login.html", gin.H{
		"ActivePage":   "login_history",
		"LoginHistory": loginHistory,
	})
}

// Menampilkan halaman laporan anggota
func KetuaLaporan(c *gin.Context) {
	// Ambil bulan dan tahun dari query parameter, default bulan dan tahun saat ini
	currentTime := time.Now()
	bulan := int(currentTime.Month())
	tahun := currentTime.Year()
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

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_laporan.html", gin.H{
			"ActivePage":  "laporan",
			"Error":       "Gagal mengambil laporan",
			"CurrentLogo": latestLogo,
			"Bulan":       bulan,
			"Tahun":       tahun,
		})
		return
	}

	riwayats, err := repository.GetAllRiwayat()
	var uniqueRiwayats []models.Riwayat
	if err == nil {
		uniqueMap := make(map[string]models.Riwayat)
		for _, r := range riwayats {
			key := r.IDAnggota + "-" + r.NoTelepon
			if key == "-" || r.IDAnggota == "" || r.NoTelepon == "" {
				continue // skip jika data tidak valid
			}
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = r
			}
		}
		uniqueRiwayats = make([]models.Riwayat, 0, len(uniqueMap))
		for _, v := range uniqueMap {
			uniqueRiwayats = append(uniqueRiwayats, v)
		}
	} else {
		uniqueRiwayats = []models.Riwayat{} // slice kosong jika error
	}

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":  "laporan",
		"Report":      report,
		"Bulan":       bulan,
		"Tahun":       tahun,
		"CurrentLogo": latestLogo,
		"Riwayats":    uniqueRiwayats,
		"Error": func() string {
			if err != nil {
				return "Gagal mengambil data riwayat"
			} else {
				return ""
			}
		}(),
	})
}

// BendaharaPengaturan menampilkan halaman pengaturan bendahara
func KetuaPengaturan(c *gin.Context) {
	// Ambil ID bendahara dari session
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
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

	// Ambil data bendahara
	bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara: " + err.Error(),
			"LogoPath":   latestLogo,
		})
		return
	}

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage":  "pengaturan",
		"Bendahara":   bendahara,
		"LogoPath":    latestLogo,
		"CurrentLogo": latestLogo,
	})
}

// UpdateKetuaProfile memproses update username dan password ketua
func UpdateKetuaProfile(c *gin.Context) {

	// Ambil ID bendahara dari session
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
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
	plainPasswordToUpdate := ""
	if passwordToUpdate != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordToUpdate), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}
		plainPasswordToUpdate = request.Password
		passwordToUpdate = string(hashedPassword)
	} else {
		// Jika password kosong, ambil password lama
		bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bendahara"})
			return
		}
		passwordToUpdate = bendahara.Password
		plainPasswordToUpdate = bendahara.PlainPassword
	}

	// Update username, password, dan plain_password
	err := repository.UpdatePengelolaUsernamePassword(bendaharaID.(int), request.Username, passwordToUpdate, plainPasswordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}
