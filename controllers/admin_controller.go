package controllers

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// Menampilkan dashboard admin dengan data statistik
func AdminDashboard(c *gin.Context) {
	db := config.GetDB()

	// Ambil data dashboard
	totalAnggota, err := repository.GetTotalAnggota(db)
	if err != nil {
		totalAnggota = 0
	}

	menungguKonfirmasi, err := repository.GetMenungguKonfirmasi(db)
	if err != nil {
		menungguKonfirmasi = 0
	}

	totalSimpanan, err := repository.GetTotalSimpanan(db)
	if err != nil {
		totalSimpanan = 0
	}

	totalPinjaman, err := repository.GetTotalPinjaman(db)
	if err != nil {
		totalPinjaman = 0
	}

	// Ambil data aktivitas terbaru untuk grafik
	aktivitasData, err := repository.GetAktivitasTerbaru(db)
	if err != nil {
		aktivitasData = []map[string]interface{}{}
	}
	// Data untuk template
	data := map[string]interface{}{
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"AktivitasData":      aktivitasData,
		"LogoPath":           c.MustGet("LogoPath"),
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", data)
}

// ViewRegistration menampilkan detail registrasi anggota pending
func ShowEditHalamanForm(c *gin.Context) {
	slug := c.Param("slug")
	halaman, err := repository.GetHalamanBySlug(slug)
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Parse konten JSON untuk template
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
		konten = map[string]interface{}{}
	}

	// Ensure misi is always an array for visi-misi page
	if slug == "visi-misi" {
		if misi, exists := konten["misi"]; exists {
			// If misi exists, ensure it's an array
			switch v := misi.(type) {
			case []interface{}:
				// Already an array, do nothing
			case string:
				// If it's a string, convert to array with single element
				konten["misi"] = []interface{}{v}
			default:
				// If it's something else, set to empty array
				konten["misi"] = []interface{}{}
			}
		} else {
			// If misi doesn't exist, set to empty array
			konten["misi"] = []interface{}{}
		}
	}

	// Pilih template berdasarkan slug (ganti - dengan _ untuk nama file)
	templateSlug := strings.ReplaceAll(slug, "-", "_")
	var templateName string
	if templateSlug == "hubungi_kami" {
		templateName = "admin_halaman_edit_hubungi_kami.html"
	} else {
		templateName = "admin_halaman_edit_" + templateSlug + ".html"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Halaman":  halaman,
		"Konten":   konten,
		"LogoPath": c.MustGet("LogoPath"),
	})
}

// UpdateHalaman memproses update konten halaman.
func UpdateHalaman(c *gin.Context) {
	slug := c.Param("slug")

	// Check if request is JSON (AJAX) or form data
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Handle JSON request (AJAX)
		var request struct {
			Judul  string `json:"judul"`
			Konten string `json:"konten"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Data tidak valid",
			})
			return
		}

		halaman := models.Halaman{
			Slug:   slug,
			Judul:  request.Judul,
			Konten: request.Konten,
		}

		err := repository.UpdateHalaman(halaman)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal memperbarui halaman",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Halaman berhasil diperbarui",
		})
		return
	}

	// Handle form data (fallback for non-AJAX requests)
	if slug == "dashboard_anggota" {
		// Handle special case for dashboard_anggota with separate fields
		teks := c.PostForm("teks")
		gambar := c.PostForm("gambar")
		if teks == "" || gambar == "" {
			c.String(http.StatusBadRequest, "Data tidak valid")
			return
		}
		kontenMap := map[string]string{
			"teks":   teks,
			"gambar": gambar,
		}
		kontenBytes, err := json.Marshal(kontenMap)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat konten")
			return
		}
		// Get existing halaman to keep judul
		existing, err := repository.GetHalamanBySlug(slug)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
			return
		}
		halaman := models.Halaman{
			Slug:   slug,
			Judul:  existing.Judul,
			Konten: string(kontenBytes),
		}
		err = repository.UpdateHalaman(halaman)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
			return
		}
		c.Redirect(http.StatusFound, "/admin/dashboard")
		return
	}

	// Handle halaman edit with JSON konten (like sejarah, visi-misi, struktur)
	kontenStr := c.PostForm("konten")
	if kontenStr != "" {
		// Parse JSON konten
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(kontenStr), &konten); err != nil {
			c.String(http.StatusBadRequest, "Konten tidak valid")
			return
		}

		// Get judul from form
		judul := c.PostForm("judul")
		if judul == "" {
			// Get existing judul if not provided
			existing, err := repository.GetHalamanBySlug(slug)
			if err != nil {
				c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
				return
			}
			judul = existing.Judul
		}

		// Convert konten back to JSON string
		kontenBytes, err := json.Marshal(konten)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat konten")
			return
		}

		halaman := models.Halaman{
			Slug:   slug,
			Judul:  judul,
			Konten: string(kontenBytes),
		}

		err = repository.UpdateHalaman(halaman)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
			return
		}

		// Redirect to admin pengaturan instead of dashboard for consistency
		c.Redirect(http.StatusFound, "/admin/pengaturan")
		return
	}

	var halaman models.Halaman
	if err := c.ShouldBind(&halaman); err != nil {
		c.String(http.StatusBadRequest, "Data tidak valid")
		return
	}
	halaman.Slug = slug

	err := repository.UpdateHalaman(halaman)
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal memperbarui halaman")
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

func UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada file yang diterima"})
		return
	}

	// Buat nama file yang unik untuk menghindari konflik
	extension := filepath.Ext(file.Filename)
	newFileName := uuid.New().String() + extension

	// Simpan file ke folder static/uploads
	err = c.SaveUploadedFile(file, "static/images/"+newFileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}

	// Kembalikan path file yang bisa diakses publik
	filePath := "/static/images/" + newFileName
	c.JSON(http.StatusOK, gin.H{"filePath": filePath})
}

// UploadStruktur handles file upload specifically for struktur page images
func UploadStruktur(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Tidak ada file yang diterima",
		})
		return
	}

	// Validasi tipe file
	allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}
	fileType := file.Header.Get("Content-Type")
	isAllowed := false
	for _, allowedType := range allowedTypes {
		if fileType == allowedType {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Format file tidak didukung. Gunakan JPG, PNG, atau GIF.",
		})
		return
	}

	// Validasi ukuran file (2MB)
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Ukuran file terlalu besar. Maksimal 2MB.",
		})
		return
	}

	// Buat nama file yang unik untuk struktur
	extension := filepath.Ext(file.Filename)
	if extension == "" {
		extension = ".png" // Default to PNG if no extension
	}
	newFileName := "struktur_" + uuid.New().String() + extension

	// Simpan file ke folder static/images
	err = c.SaveUploadedFile(file, "static/images/"+newFileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan file",
		})
		return
	}

	// Kembalikan path file yang bisa diakses publik
	filePath := "/static/images/" + newFileName
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Gambar berhasil diupload",
		"filePath": filePath,
	})
}

// AdminTransaksi menampilkan halaman transaksi admin dengan form input
func AdminTransaksi(c *gin.Context) {
	details, err := repository.GetAllDetails()
	if err != nil {
		details = []models.Detail{} // Default kosong jika error
	}

	pinjamans, err := repository.GetPendingPinjaman()
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	c.HTML(http.StatusOK, "admin_transaksi.html", gin.H{
		"ActivePage": "transaksi",
		"Details":    details,
		"Pinjamans":  pinjamans,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// CatatSimpanan memproses pencatatan simpanan
func CatatSimpanan(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var detail models.Detail
	if err := c.ShouldBind(&detail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	detail.TglTransaksi = time.Now()

	// Hitung total simpanan (kumulatif)
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(detail.IDAnggota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total simpanan"})
		return
	}
	detail.TotalSimpanan = totalSimpanan + detail.JumlahSimpanan

	err = repository.CreateSimpanan(detail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat simpanan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Simpanan berhasil dicatat"})
}

// CatatPinjaman memproses pencatatan pinjaman
func CatatPinjaman(c *gin.Context) {
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var pinjaman models.Pinjaman
	if err := c.ShouldBind(&pinjaman); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	pinjaman.IDPengelola.Int64 = int64(adminID.(int))
	pinjaman.TglPinjaman = time.Now()
	pinjaman.Status = "aktif"

	err := repository.CreatePinjaman(pinjaman)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat pinjaman"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pinjaman berhasil dicatat"})
}

// AdminRiwayat menampilkan halaman riwayat transaksi admin
func AdminRiwayat(c *gin.Context) {
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data riwayat"})
		return
	}

	c.HTML(http.StatusOK, "admin_riwayat.html", gin.H{
		"ActivePage": "riwayat",
		"Riwayats":   riwayats,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminLaporan menampilkan halaman laporan keuangan admin
func AdminLaporan(c *gin.Context) {
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data laporan"})
		return
	}

	c.HTML(http.StatusOK, "admin_laporan.html", gin.H{
		"ActivePage": "laporan",
		"Riwayats":   riwayats,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminTentang menampilkan halaman tentang kami admin
func AdminTentang(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_layout.html", gin.H{
		"ActivePage": "tentang",
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminPengaturan menampilkan halaman pengaturan admin (sekarang menampilkan daftar halaman statis)
func AdminPengaturan(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data halaman"})
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data halaman"})
		return

	}

	// Map judul ke nama keamanan
	securityTitles := map[string]string{
		"pinjaman": "Pengaturan Keamanan Data Pinjaman",
		"simpanan": "Pengaturan Keamanan Data Simpanan",
		"angsuran": "Pengaturan Keamanan Pembayaran",
	}

	// Filter out dashboard_anggota and struktur from the list
	var filteredHalaman []models.Halaman
	for _, halaman := range allHalaman {
		if halaman.Slug != "dashboard_anggota" && halaman.Slug != "struktur" {
			filteredHalaman = append(filteredHalaman, halaman)
		}
	}

	for i, halaman := range filteredHalaman {
		if title, ok := securityTitles[halaman.Slug]; ok {
			filteredHalaman[i].Judul = title
		}
	}

	// Ambil LogoPath dari context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}
	c.HTML(http.StatusOK, "admin_pengaturan.html", gin.H{
		"AllHalaman": filteredHalaman,
		"ActivePage": "pengaturan",
		"LogoPath":   logoPath,
	})
}

// UpdateAdminProfile memproses update username dan password admin
func UpdateAdminProfile(c *gin.Context) {
	// Ambil ID admin dari session
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
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

	// Password disimpan dalam bentuk plain text sesuai permintaan
	passwordToUpdate := request.Password
	plainPasswordToUpdate := ""
	if passwordToUpdate == "" {
		// Jika password kosong, ambil password lama
		admin, err := repository.GetPengelolaByID(adminID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data admin"})
			return
		}
		passwordToUpdate = admin.Password
		plainPasswordToUpdate = admin.PlainPassword
	} else {
		plainPasswordToUpdate = request.Password
	}

	// Update username, password, dan plain_password
	err := repository.UpdatePengelolaUsernamePassword(adminID.(int), request.Username, passwordToUpdate, plainPasswordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// AdminKeamananLogin menampilkan halaman keamanan login
func AdminKeamananLogin(c *gin.Context) {
	// Ambil data riwayat login dari database
	loginHistory, err := repository.GetLoginHistory()
	if err != nil {
		loginHistory = []models.LoginHistory{} // Default kosong jika error
	}

	c.HTML(http.StatusOK, "admin_keamanan_login.html", gin.H{
		"ActivePage":   "keamanan_login",
		"LoginHistory": loginHistory,
	})
}

// DeleteLoginHistory menghapus riwayat login berdasarkan ID
func DeleteLoginHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	err = repository.DeleteLoginHistory(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus riwayat login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Riwayat login berhasil dihapus"})
}

// AdminKeamananSimpanan menampilkan halaman keamanan data simpanan
func AdminKeamananSimpanan(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_simpanan.html", gin.H{
		"ActivePage": "pengaturan",
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminKeamananPinjaman menampilkan halaman keamanan data pinjaman
func AdminKeamananPinjaman(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_pinjaman.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananPembayaran menampilkan halaman keamanan pembayaran
func AdminKeamananPembayaran(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_pembayaran.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananDashboard menampilkan halaman keamanan dashboard
func AdminKeamananDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_dashboard.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananRiwayat menampilkan halaman keamanan riwayat
func AdminKeamananRiwayat(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_riwayat.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminKeamananOrganisasi menampilkan halaman keamanan organisasi
func AdminKeamananOrganisasi(c *gin.Context) {
	c.HTML(http.StatusOK, "admin_keamanan_organisasi.html", gin.H{
		"ActivePage": "pengaturan",
	})
}

// AdminPesan menampilkan halaman pesan admin
func AdminPesan(c *gin.Context) {
	// Ambil session admin
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data admin
	admin, err := repository.GetPengelolaByID(adminID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "admin_pesan.html", gin.H{
			"ActivePage": "pesan",
			"Error":      "Gagal mengambil data admin: " + err.Error(),
		})
		return
	}

	// Ambil daftar anggota untuk dropdown
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		anggotas = []models.Anggota{}
	}

	c.HTML(http.StatusOK, "admin_pesan.html", gin.H{
		"ActivePage": "pesan",
		"Admin":      admin,
		"Anggotas":   anggotas,
		"LogoPath":   c.MustGet("LogoPath"),
	})
}

// AdminLogo menampilkan halaman edit logo admin
func AdminLogo(c *gin.Context) {
	// Cari logo terbaru yang sudah diupload
	files, err := os.ReadDir("static/images")
	if err != nil {
		c.HTML(http.StatusOK, "admin_logo.html", gin.H{
			"ActivePage": "edit_logo",
		})
		return
	}

	// Cari file logo terbaru (berdasarkan waktu modifikasi file)
	var latestLogo string
	var latestTime int64
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "logo_") && (strings.HasSuffix(file.Name(), ".png") || strings.HasSuffix(file.Name(), ".jpg")) {
			info, err := file.Info()
			if err == nil {
				modTime := info.ModTime().Unix()
				if modTime > latestTime {
					latestTime = modTime
					latestLogo = "/static/images/" + file.Name()
				}
			}
		}
	}
	// Jika tidak ada logo yang ditemukan, gunakan placeholder
	if latestLogo == "" {
		latestLogo = "/static/images/placeholder.png"
	}
	c.HTML(http.StatusOK, "admin_logo.html", gin.H{
		"ActivePage":  "edit_logo",
		"CurrentLogo": latestLogo,
		"LogoPath":    latestLogo,
	})
}

// UploadLogo memproses upload logo baru
func UploadLogo(c *gin.Context) {
	// Cek apakah request menggunakan JSON atau FormData
	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "application/json") {
		var request struct {
			LogoData string `json:"logoData" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format data tidak valid",
			})
			return
		}

		logoData := request.LogoData
		if logoData == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Tidak ada data logo yang diterima",
			})
			return
		}

		// Decode base64 data
		// Format: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA..."
		parts := strings.Split(logoData, ",")
		if len(parts) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format data logo tidak valid",
			})
			return
		}

		// Decode base64
		imageData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Gagal decode data logo",
			})
			return
		}

		// Decode image (coba berbagai format)
		img, _, err := image.Decode(strings.NewReader(string(imageData)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Gagal decode gambar",
			})
			return
		}

		// Buat nama file unik
		newFileName := "logo_" + uuid.New().String() + ".png"

		// Simpan sebagai PNG transparan
		filePath := "static/images/" + newFileName
		file, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal membuat file",
			})
			return
		}
		defer file.Close()

		// Encode sebagai PNG
		err = png.Encode(file, img)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan gambar",
			})
			return
		}

		// Path file yang bisa diakses publik
		logoPath := "/static/images/" + newFileName

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "Logo berhasil diupload",
			"logoPath": logoPath,
		})
	} else {
		// Handle FormData request (fallback untuk file upload langsung)
		file, err := c.FormFile("logoFile")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Tidak ada file yang dipilih",
			})
			return
		}

		// Validasi tipe file
		allowedTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif"}
		fileType := file.Header.Get("Content-Type")
		isAllowed := false
		for _, allowedType := range allowedTypes {
			if fileType == allowedType {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Format file tidak didukung. Gunakan JPG, PNG, atau GIF.",
			})
			return
		}

		// Validasi ukuran file (2MB)
		if file.Size > 2*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Ukuran file terlalu besar. Maksimal 2MB.",
			})
			return
		}

		// Buat nama file unik
		extension := filepath.Ext(file.Filename)
		newFileName := "logo_" + uuid.New().String() + extension

		// Simpan file ke folder static/images
		err = c.SaveUploadedFile(file, "static/images/"+newFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan file",
			})
			return
		}

		// Path file yang bisa diakses publik
		logoPath := "/static/images/" + newFileName

		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"message":  "Logo berhasil diupload",
			"logoPath": logoPath,
		})
	}
}

// Handle JSON request (data canvas dari JavaScript)
