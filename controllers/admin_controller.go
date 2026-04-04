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
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

func DeleteAllLoginHistory(c *gin.Context) {
	err := repository.DeleteAllLoginHistory()
	if err != nil {
		c.JSON(500, gin.H{"error": "Gagal menghapus semua riwayat login"})
		return
	}
	c.JSON(200, gin.H{"message": "Semua riwayat login berhasil dihapus"})
}

// UpdateUser memproses update data user (pengelola)
func UpdateUser(c *gin.Context) {
	id := c.PostForm("id")
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validasi
	if id == "" || username == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID dan Username wajib diisi"})
		return
	}

	db := config.GetDB()
	var query string
	var args []interface{}
	if password != "" {
		// Hash password dengan bcrypt
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengenkripsi password"})
			return
		}
		query = "UPDATE pengelola SET username = $1, password = $2 WHERE id_pengelola = $3"
		args = []interface{}{username, string(hashedPassword), id}
	} else {
		query = "UPDATE pengelola SET username = $1 WHERE id_pengelola = $2"
		args = []interface{}{username, id}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal memperbarui user"})
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

// UpdateAnggota memproses update data anggota
func UpdateAnggota(c *gin.Context) {
	id := c.PostForm("id")
	username := c.PostForm("username")
	password := c.PostForm("password")

	if id == "" || username == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID dan Username wajib diisi"})
		return
	}

	db := config.GetDB()
	var query string
	var args []interface{}
	if password != "" {
		query = "UPDATE anggota SET username = $1, password = $2 WHERE id_anggota = $3"
		args = []interface{}{username, password, id}
	} else {
		query = "UPDATE anggota SET username = $1 WHERE id_anggota = $2"
		args = []interface{}{username, id}
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal memperbarui anggota"})
		return
	}
	c.Redirect(http.StatusFound, "/admin/pengaturan")
}

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

	// Ambil riwayat total anggota, simpanan, pinjaman per hari
	riwayatAnggota, _ := repository.GetRiwayatTotalAnggotaPerHari(db)
	riwayatSimpanan, _ := repository.GetRiwayatTotalSimpananPerHari(db)
	riwayatPinjaman, _ := repository.GetRiwayatTotalPinjamanPerHari(db)

	aktivitasData := []map[string]interface{}{}
	for _, r := range riwayatAnggota {
		r["Jenis"] = "Anggota"
		aktivitasData = append(aktivitasData, r)
	}
	for _, r := range riwayatSimpanan {
		r["Jenis"] = "Simpanan"
		aktivitasData = append(aktivitasData, r)
	}
	for _, r := range riwayatPinjaman {
		r["Jenis"] = "Pinjaman"
		aktivitasData = append(aktivitasData, r)
	}
	// Jika semua riwayat kosong, fallback ke statistik utama
	if len(aktivitasData) == 0 {
		aktivitasData = []map[string]interface{}{
			{"Tanggal": time.Now(), "Jenis": "Anggota", "Jumlah": totalAnggota},
			{"Tanggal": time.Now(), "Jenis": "Simpanan", "Jumlah": totalSimpanan},
			{"Tanggal": time.Now(), "Jenis": "Pinjaman", "Jumlah": totalPinjaman},
		}
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

func AdminDataAnggota(c *gin.Context) {
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	c.HTML(http.StatusOK, "admin_data_anggota.html", gin.H{
		"Anggotas":    anggotas,
		"ActivePage":  "anggota",
		"LogoPath":    logoPath,
		"CurrentLogo": logoPath,
	})
}

func AdminViewAnggota(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	simpananByJenis, err := repository.GetDetailSimpananByJenis(idStr)
	if err != nil {
		simpananByJenis = map[string]float64{
			"pokok":      0,
			"wajib":      0,
			"sukarela":   0,
			"hari_raya":  0,
			"umroh_haji": 0,
			"qurban":     0,
		}
	}

	totalSimpanan := simpananByJenis["pokok"] + simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

	_, totalPinjaman, _, err := repository.GetSaldoAnggota(idStr)
	if err != nil {
		totalPinjaman = 0
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	c.HTML(http.StatusOK, "admin_data_anggota_view.html", gin.H{
		"Anggota":           anggota,
		"ActivePage":        "anggota",
		"LogoPath":          logoPath,
		"CurrentLogo":       logoPath,
		"Title":             "Detail Anggota",
		"SimpananPokok":     simpananByJenis["pokok"],
		"SimpananWajib":     simpananByJenis["wajib"],
		"SimpananSukarela":  simpananByJenis["sukarela"],
		"SimpananHariRaya":  simpananByJenis["hari_raya"],
		"SimpananUmrohHaji": simpananByJenis["umroh_haji"],
		"SimpananQurban":    simpananByJenis["qurban"],
		"TotalSimpanan":     totalSimpanan,
		"TotalPinjaman":     totalPinjaman,
	})
}

func getLatestKopPath() string {
	files, err := os.ReadDir("static/uploads/kop")
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := strings.ToLower(file.Name())
		if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") && !strings.HasSuffix(name, ".png") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if modTime > latestTime {
			latestTime = modTime
			latestPath = "/static/uploads/kop/" + file.Name()
		}
	}

	return latestPath
}

func getLatestSignaturePath(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return ""
	}

	signDir := "static/uploads/signatures"
	files, err := os.ReadDir(signDir)
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	prefix := "ttd_" + role + "_"
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := strings.ToLower(file.Name())
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		ext := filepath.Ext(name)
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		modTime := info.ModTime().Unix()
		if modTime > latestTime {
			latestTime = modTime
			latestPath = "/static/uploads/signatures/" + file.Name()
		}
	}

	return latestPath
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
	activePage := templateSlug
	switch templateSlug {
	case "hubungi_kami":
		templateName = "admin_halaman_edit_hubungi_kami.html"
	case "dashboard_anggota":
		templateName = "admin_halaman_edit_dashboard.html"
		activePage = "dashboard_content"
	case "simpanan":
		templateName = "admin_halaman_edit_simpanan.html"
		activePage = "edit_simpanan"
	default:
		templateName = "admin_halaman_edit_" + templateSlug + ".html"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Halaman":    halaman,
		"Konten":     konten,
		"LogoPath":   c.MustGet("LogoPath"),
		"CurrentKop": getLatestKopPath(),
		"CurrentSignatureKetua":     getLatestSignaturePath("ketua"),
		"CurrentSignatureBendahara": getLatestSignaturePath("bendahara"),
		"CurrentSignatureSekretaris": getLatestSignaturePath("sekretaris"),
		"success":    c.Query("success"),
		"error":      c.Query("error"),
		"ActivePage": activePage,
	})
}

func AdminUploadKop(c *gin.Context) {
	file, err := c.FormFile("kop_file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal menerima file kop")
		return
	}

	kopDir := "static/uploads/kop/"
	if err := os.MkdirAll(kopDir, os.ModePerm); err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal membuat folder kop")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Hanya file JPG, JPEG, PNG yang diperbolehkan")
		return
	}

	filename := "kop_" + strconv.FormatInt(time.Now().Unix(), 10) + ext
	savePath := filepath.Join(kopDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?error=Gagal menyimpan file kop")
		return
	}

	c.Redirect(http.StatusFound, "/admin/halaman/edit/dashboard_anggota?success=Kop surat berhasil diupload")
}

func AdminUploadSignature(c *gin.Context) {
	role := strings.ToLower(strings.TrimSpace(c.PostForm("sign_role")))
	if role != "ketua" && role != "bendahara" && role != "sekretaris" {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Peran tanda tangan tidak valid")
		return
	}

	file, err := c.FormFile("sign_file")
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menerima file tanda tangan")
		return
	}

	signDir := "static/uploads/signatures/"
	if err := os.MkdirAll(signDir, os.ModePerm); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal membuat folder tanda tangan")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Hanya file JPG, JPEG, PNG yang diperbolehkan untuk tanda tangan")
		return
	}

	filename := "ttd_" + role + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ext
	savePath := filepath.Join(signDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan file tanda tangan")
		return
	}

	c.Redirect(http.StatusFound, "/admin/tanda-tangan?success=Template tanda tangan berhasil diupload")
}

func AdminEditTandaTangan(c *gin.Context) {
	logoPath := c.MustGet("LogoPath")
	db := config.GetDB()
	getNamaTtd := func(key, def string) string {
		var val string
		err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&val)
		if err != nil || strings.TrimSpace(val) == "" {
			return def
		}
		return val
	}

	c.HTML(http.StatusOK, "admin_edit_tanda_tangan.html", gin.H{
		"LogoPath":                 logoPath,
		"CurrentLogo":              logoPath,
		"CurrentSignatureKetua":    getLatestSignaturePath("ketua"),
		"CurrentSignatureBendahara": getLatestSignaturePath("bendahara"),
		"CurrentSignatureSekretaris": getLatestSignaturePath("sekretaris"),
		"NamaTtdKetua":             getNamaTtd("ttd_nama_ketua", "Ketua KOPMA"),
		"NamaTtdBendahara":         getNamaTtd("ttd_nama_bendahara", "Bendahara"),
		"NamaTtdSekretaris":        getNamaTtd("ttd_nama_sekretaris", "Sekretaris"),
		"success":                  c.Query("success"),
		"error":                    c.Query("error"),
		"ActivePage":               "edit_tanda_tangan",
	})
}

func AdminUpdateSignatureNames(c *gin.Context) {
	namaKetua := strings.TrimSpace(c.PostForm("nama_ttd_ketua"))
	namaBendahara := strings.TrimSpace(c.PostForm("nama_ttd_bendahara"))
	namaSekretaris := strings.TrimSpace(c.PostForm("nama_ttd_sekretaris"))

	if namaKetua == "" {
		namaKetua = "Ketua KOPMA"
	}
	if namaBendahara == "" {
		namaBendahara = "Bendahara"
	}
	if namaSekretaris == "" {
		namaSekretaris = "Sekretaris"
	}

	db := config.GetDB()
	upsert := func(key, value string) error {
		_, err := db.Exec(`
			INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (nama_pengaturan)
			DO UPDATE SET nilai = EXCLUDED.nilai, updated_at = NOW()
		`, key, value)
		return err
	}

	if err := upsert("ttd_nama_ketua", namaKetua); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Ketua")
		return
	}
	if err := upsert("ttd_nama_bendahara", namaBendahara); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Bendahara")
		return
	}
	if err := upsert("ttd_nama_sekretaris", namaSekretaris); err != nil {
		c.Redirect(http.StatusFound, "/admin/tanda-tangan?error=Gagal menyimpan nama Sekretaris")
		return
	}

	c.Redirect(http.StatusFound, "/admin/tanda-tangan?success=Nama penandatangan berhasil disimpan")
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
		"ActivePage":  "riwayat",
		"Riwayats":    riwayats,
		"LogoPath":    c.MustGet("LogoPath"),
		"CurrentLogo": c.MustGet("LogoPath"),
	})
}

// AdminLaporan menampilkan halaman laporan keuangan admin
func AdminLaporan(c *gin.Context) {
	// Ambil tipe laporan dari query parameter (default: bulanan)
	tipeLaporan := c.Query("tipe_laporan")
	if tipeLaporan == "" {
		tipeLaporan = "bulanan"
	}

	currentTime := time.Now()
	bulan := int(currentTime.Month())
	tahun := currentTime.Year()

	if tipeLaporan == "tahunan" {
		bulan = 0
	} else {
		if b := c.Query("bulan"); b != "" {
			if parsed, convErr := strconv.Atoi(b); convErr == nil {
				bulan = parsed
			}
		}
	}

	if t := c.Query("tahun"); t != "" {
		if parsed, convErr := strconv.Atoi(t); convErr == nil {
			tahun = parsed
		}
	}

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_laporan.html", gin.H{
			"ActivePage":      "laporan",
			"Error":           "Gagal mengambil laporan",
			"CurrentLogo":     logoPath,
			"LogoPath":        logoPath,
			"Bulan":           bulan,
			"Tahun":           tahun,
			"TipeLaporan":     tipeLaporan,
			"LaporanBasePath": "/admin/laporan",
			"UseAdminLayout":  true,
			"ReadOnlyMode":    true,
		})
		return
	}

	// Ambil data neraca dari repository untuk ditampilkan read-only
	userIDInt := resolveNeracaOwnerID(c)

	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)
	neraca, _ := neracaRepo.GetNeraca(userIDInt)
	var data2024, data2023 map[string]interface{}
	if neraca != nil {
		_ = json.Unmarshal([]byte(neraca.Data2024), &data2024)
		_ = json.Unmarshal([]byte(neraca.Data2023), &data2023)
	}

	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		anggotas = []models.Anggota{}
	}

	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64)
	}

	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potongan := potonganBulanIni[anggota.IDAnggota]
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan
	}

	laporanDetail, err := repository.GetLaporanBulananPerAnggota(bulan, tahun)
	if err != nil {
		laporanDetail = []map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":       "laporan",
		"Report":           report,
		"Bulan":            bulan,
		"Tahun":            tahun,
		"TipeLaporan":      tipeLaporan,
		"CurrentLogo":      logoPath,
		"LogoPath":         logoPath,
		"Anggotas":         anggotas,
		"LaporanDetail":    laporanDetail,
		"SisaGaji":         sisaGaji,
		"GetUnitKerjaName": repository.GetUnitKerjaName,
		"NeracaData2024":   data2024,
		"NeracaData2023":   data2023,
		"LaporanBasePath":  "/admin/laporan",
		"UseAdminLayout":   true,
		"ReadOnlyMode":     true,
	})
}

func AdminDownloadLaporan(c *gin.Context) {
	KetuaDownloadLaporan(c)
}

func AdminGetNeraca(c *gin.Context) {
	KetuaGetNeraca(c)
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
		return
	}

	userList, err := repository.GetAllPengelola()
	if err != nil {
		userList = []models.Pengelola{}
	}

	anggotaList, err := repository.GetAnggotaByStatus("aktif")
	if err != nil {
		anggotaList = []models.Anggota{}
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
		"AllHalaman":  filteredHalaman,
		"UserList":    userList,
		"AnggotaList": anggotaList,
		"ActivePage":  "pengaturan",
		"LogoPath":    logoPath,
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

	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}
	c.HTML(http.StatusOK, "admin_keamanan_login.html", gin.H{
		"ActivePage":   "keamanan_login",
		"LoginHistory": loginHistory,
		"LogoPath":     logoPath,
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
