package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// Menampilkan dashboard bendahara dengan data statistik
func BendaharaDashboard(c *gin.Context) {
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

	// Encode aktivitasData ke JSON untuk template
	aktivitasJSON, err := json.Marshal(aktivitasData)
	if err != nil {
		aktivitasJSON = []byte("[]")
	}

	// Data untuk template
	data := map[string]interface{}{
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"AktivitasData":      string(aktivitasJSON),
		"ActivePage":         "dashboard",
	}

	c.HTML(http.StatusOK, "bendahara_dashboard_content.html", data)
}

// Menampilkan halaman konfirmasi anggota
// BendaharaEditRekeningRegister menampilkan halaman edit nomor rekening koperasi
// BendaharaKonfirmasi menampilkan halaman konfirmasi anggota
func BendaharaKonfirmasi(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal mengambil data anggota")
		return
	}

	// Get LogoPath from context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	c.HTML(http.StatusOK, "bendahara_anggota_konfirmasi.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage":     "konfirmasi_anggota",
		"LogoPath":       logoPath,
		"Title":          "Konfirmasi Anggota",
	})
}

func BendaharaEditRekeningRegister(c *gin.Context) {
	db := config.GetDB()

	// Buat tabel pengaturan jika belum ada
	db.Exec(`
		CREATE TABLE IF NOT EXISTS pengaturan (
			id SERIAL PRIMARY KEY,
			nama_pengaturan VARCHAR(50) UNIQUE NOT NULL,
			nilai TEXT NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)

	// Ambil nomor rekening dari database
	var nomorRekening string
	err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening)
	if err != nil {
		// Jika belum ada, insert nilai default
		db.Exec("INSERT INTO pengaturan (nama_pengaturan, nilai) VALUES ('nomor_rekening', '1234567890 (Bank ABC)')")
		nomorRekening = "1234567890 (Bank ABC)"
	}

	// Ambil nominal simpanan dari database
	var nominalSimpanan string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpanan)
	if err != nil {
		// Jika belum ada, insert nilai default
		db.Exec("INSERT INTO pengaturan (nama_pengaturan, nilai) VALUES ('nominal_simpanan', '100000')")
		nominalSimpanan = "100000"
	}

	keteranganBuktiTransfer := "Transfer dari rekening pribadi ke rekening koperasi sebesar Rp. " + nominalSimpanan + " untuk simpanan pokok wajib."

	c.HTML(http.StatusOK, "bendahara_edit_rekening_register.html", gin.H{
		"NomorRekening":           nomorRekening,
		"NominalSimpanan":         nominalSimpanan,
		"KeteranganBuktiTransfer": keteranganBuktiTransfer,
		"ActivePage":              "edit-rekening-register",
	})
}

// BendaharaUpdateRekeningRegister memproses update nomor rekening koperasi
func BendaharaUpdateRekeningRegister(c *gin.Context) {
	db := config.GetDB()
	fieldType := c.PostForm("field_type")
	nomorRekening := c.PostForm("nomor_rekening")
	nominalSimpanan := c.PostForm("nominal_simpanan")

	// Validasi berdasarkan field_type
	switch fieldType {
	case "rekening":
		if nomorRekening == "" {
			c.HTML(http.StatusBadRequest, "bendahara_edit_rekening_register.html", gin.H{
				"Error":           "Nomor rekening harus diisi",
				"NomorRekening":   nomorRekening,
				"NominalSimpanan": "100000",
				"ActivePage":      "edit-rekening-register",
			})
			return
		}
		// Simpan nomor rekening ke database
		_, err := db.Exec(`
			INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at) 
			VALUES ('nomor_rekening', $1, NOW())
			ON CONFLICT (nama_pengaturan) 
			DO UPDATE SET nilai = $1, updated_at = NOW()
		`, nomorRekening)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "bendahara_edit_rekening_register.html", gin.H{
				"Error":           "Gagal menyimpan nomor rekening",
				"NomorRekening":   nomorRekening,
				"NominalSimpanan": "100000",
				"ActivePage":      "edit-rekening-register",
			})
			return
		}
	case "simpanan":
		if nominalSimpanan == "" {
			c.HTML(http.StatusBadRequest, "bendahara_edit_rekening_register.html", gin.H{
				"Error":           "Nominal simpanan harus diisi",
				"NomorRekening":   "1234567890 (Bank ABC)",
				"NominalSimpanan": nominalSimpanan,
				"ActivePage":      "edit-rekening-register",
			})
			return
		}
		// Simpan nominal simpanan ke database
		_, err := db.Exec(`
			INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at) 
			VALUES ('nominal_simpanan', $1, NOW())
			ON CONFLICT (nama_pengaturan) 
			DO UPDATE SET nilai = $1, updated_at = NOW()
		`, nominalSimpanan)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "bendahara_edit_rekening_register.html", gin.H{
				"Error":           "Gagal menyimpan nominal simpanan",
				"NomorRekening":   "1234567890 (Bank ABC)",
				"NominalSimpanan": nominalSimpanan,
				"ActivePage":      "edit-rekening-register",
			})
			return
		}
	default:
		c.HTML(http.StatusBadRequest, "bendahara_edit_rekening_register.html", gin.H{
			"Error":           "Tipe field tidak valid",
			"NomorRekening":   "1234567890 (Bank ABC)",
			"NominalSimpanan": "100000",
			"ActivePage":      "edit-rekening-register",
		})
		return
	}

	// Jika berhasil simpan, redirect ke halaman dashboard bendahara
	c.Redirect(http.StatusFound, "/bendahara/dashboard")
}

// Mengkonfirmasi keanggotaan
func BendaharaConfirmMembership(c *gin.Context) {
	// Ambil id anggota dari URL (ini masih TEMP id)
	tempID := c.Param("id")

	// Ambil data anggota untuk mendapatkan informasi unit_kerja, fakultas, dan tahun
	anggota, err := repository.GetAnggotaByID(tempID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data anggota"})
		return
	}

	// Generate ID anggota yang benar berdasarkan unit_kerja, fakultas_code, tahun konfirmasi, dan nomor urut
	// Format: {unit_kerja}{fakultas_code}{tahun}{nomor_urut}
	// Contoh: 010120250001
	// - 01: Unit Kerja (01=Dosen, 02=Karyawan/Staff, 03=Mahasiswa)
	// - 01: Fakultas Code (01=FAI, 02=FE, 03=FH, 04=FISIP, 05=FKIP, 06=FKM, 07=FAPERTA, 08=FT, 09=Rektorat/Yayasan/Staff)
	// - 2025: Tahun konfirmasi
	// - 0001: Nomor urut

	db := config.GetDB()

	// Ambil tahun konfirmasi saat ini
	tahunKonfirmasi := time.Now().Format("2006")

	// Ambil nomor urut terakhir untuk kombinasi unit_kerja, fakultas_code, dan tahun konfirmasi ini
	var lastNumber int
	query := `SELECT COALESCE(MAX(CAST(nomor_urut AS INTEGER)), 0) 
	          FROM anggota 
	          WHERE unit_kerja = $1 AND fakultas_code = $2 AND tahun = $3 AND id_anggota NOT LIKE 'TEMP%'`

	err = db.QueryRow(query, anggota.UnitKerja, anggota.FakultasCode, tahunKonfirmasi).Scan(&lastNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate nomor urut"})
		return
	}

	// Nomor urut berikutnya (4 digit)
	newNumber := lastNumber + 1
	nomorUrut := fmt.Sprintf("%04d", newNumber)

	// Generate ID anggota baru: {unit_kerja}{fakultas_code}{tahun}{nomor_urut}
	newIDAnggota := fmt.Sprintf("%s%s%s%s", anggota.UnitKerja, anggota.FakultasCode, tahunKonfirmasi, nomorUrut)

	// Update id_anggota, status, tahun, dan nomor_urut
	updateQuery := `UPDATE anggota 
	                SET id_anggota = $1, status = $2, tahun = $3, nomor_urut = $4 
	                WHERE id_anggota = $5`

	_, err = db.Exec(updateQuery, newIDAnggota, "aktif", tahunKonfirmasi, nomorUrut, tempID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengkonfirmasi anggota"})
		return
	}

	// Arahkan kembali ke halaman konfirmasi bendahara
	c.Redirect(http.StatusFound, "/bendahara/konfirmasi")
}

// BendaharaRejectMembership menolak pendaftaran anggota
func BendaharaRejectMembership(c *gin.Context) {
	// Ambil id anggota dari URL (ini masih TEMP id)
	tempID := c.Param("id")

	// Hapus anggota dari database
	db := config.GetDB()
	deleteQuery := `DELETE FROM anggota WHERE id_anggota = $1`

	_, err := db.Exec(deleteQuery, tempID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menolak pendaftaran anggota"})
		return
	}

	// Arahkan kembali ke halaman konfirmasi bendahara
	c.Redirect(http.StatusFound, "/bendahara/konfirmasi")
}

// ShowEditHalamanForm menampilkan form untuk mengedit halaman.
func BendaharaShowEditHalamanForm(c *gin.Context) {
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

	// Get LogoPath from context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	// Gunakan template khusus untuk simpanan
	templateName := "bendahara_halaman_edit.html"
	activePage := "halaman"
	if slug == "simpanan" {
		templateName = "bendahara_halaman_edit_simpanan.html"
		activePage = "edit_simpanan"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Halaman":    halaman,
		"Konten":     konten,
		"LogoPath":   logoPath,
		"ActivePage": activePage,
	})
}

// UpdateHalaman memproses update konten halaman.
func BendaharaUpdateHalaman(c *gin.Context) {
	slug := c.Param("slug")

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
		c.Redirect(http.StatusFound, "/bendahara/dashboard")
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
	c.Redirect(http.StatusFound, "/bendahara/halaman")
}

func BendaharaUploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada file yang diterima"})
		return
	}

	// Buat nama file yang unik untuk menghindari konflik
	extension := filepath.Ext(file.Filename)
	newFileName := uuid.New().String() + extension

	// Simpan file ke folder static/uploads
	err = c.SaveUploadedFile(file, "static/uploads/"+newFileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file"})
		return
	}

	// Kembalikan path file yang bisa diakses publik
	filePath := "/static/uploads/" + newFileName
	c.JSON(http.StatusOK, gin.H{"filePath": filePath})
}

// ListAllAnggota menampilkan daftar semua anggota aktif
func BendaharaListAllAnggota(c *gin.Context) {
	logoPath, _ := c.Get("LogoPath")

	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}
	c.HTML(http.StatusOK, "bendahara_data_anggota.html", gin.H{
		"Anggotas":   anggotas,
		"ActivePage": "anggota",
		"LogoPath":   logoPath,
		"Title":      "Data Anggota",
	})
}

// ViewAnggota menampilkan detail anggota berdasarkan ID
func BendaharaViewAnggota(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Get LogoPath from context
	logoPath, _ := c.Get("LogoPath")

	// Use bendahara template
	c.HTML(http.StatusOK, "bendahara_data_anggota_view.html", gin.H{
		"Anggota":    anggota,
		"ActivePage": "anggota",
		"LogoPath":   logoPath,
		"Title":      "Detail Anggota",
	})
}

// EditAnggota menampilkan form edit anggota
func BendaharaEditAnggota(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Get LogoPath from context
	logoPath, _ := c.Get("LogoPath")

	c.HTML(http.StatusOK, "bendahara_data_anggota_edit.html", gin.H{
		"Anggota":    anggota,
		"ActivePage": "anggota",
		"LogoPath":   logoPath,
		"Title":      "Edit Data Anggota",
	})
}

// UpdateAnggota memproses update data anggota
func BendaharaUpdateAnggota(c *gin.Context) {
	idStr := c.Param("id")

	var anggota models.Anggota
	if err := c.ShouldBind(&anggota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Update query (assuming we update all fields except password for simplicity)
	db := config.GetDB()
	query := `
		UPDATE anggota SET
			nama_anggota = $1, username = $2, tgl_lahir = $3, nik_ktp = $4,
			no_telepon = $5, alamat = $6, jenis_kelamin = $7, status_anggota = $8, fakultas = $9
		WHERE id_anggota = $10`
	_, err := db.Exec(query,
		anggota.NamaAnggota, anggota.Username, anggota.TglLahir, anggota.NikKTP,
		anggota.NoTelepon, anggota.Alamat, anggota.JenisKelamin, anggota.StatusAnggota, anggota.Fakultas, idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui anggota"})
		return
	}

	c.Redirect(http.StatusFound, "/bendahara/anggota/"+idStr)
}

// DeleteAnggota menghapus anggota
func BendaharaDeleteAnggota(c *gin.Context) {
	idStr := c.Param("id")

	err := repository.DeleteAnggota(idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus anggota"})
		return
	}

	c.Redirect(http.StatusFound, "/bendahara/anggota")
}

// BendaharaTransaksi menampilkan halaman transaksi bendahara
func BendaharaTransaksi(c *gin.Context) {
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

	c.HTML(http.StatusOK, "bendahara_layout.html", gin.H{
		"ActivePage": "transaksi",
		"Simpanans":  simpanans,
		"Details":    details,
		"Pinjamans":  pinjamans,
	})
}

// BendaharaCatatSimpanan memproses pencatatan simpanan
func BendaharaCatatSimpanan(c *gin.Context) {
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var detail models.Detail
	if err := c.ShouldBind(&detail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	detail.IDPengelola = bendaharaID.(int)
	detail.TglTransaksi = time.Now()
	detail.Status = "confirmed" // Langsung confirmed karena dicatat bendahara

	// Tentukan id_simpanan berdasarkan jenis_simpanan
	jenisSimpanan := c.PostForm("jenis_simpanan")
	switch jenisSimpanan {
	case "wajib":
		detail.IDSimpanan = 2
	case "sukarela":
		detail.IDSimpanan = 3
	case "hari_raya":
		detail.IDSimpanan = 4
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenis simpanan tidak valid"})
		return
	}

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

// BendaharaCatatPinjaman memproses pencatatan pinjaman
func BendaharaCatatPinjaman(c *gin.Context) {
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	var pinjaman models.Pinjaman
	if err := c.ShouldBind(&pinjaman); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	pinjaman.IDPengelola.Int64 = int64(bendaharaID.(int))
	pinjaman.TglPinjaman = time.Now()
	pinjaman.Status = "aktif"

	err := repository.CreatePinjaman(pinjaman)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat pinjaman"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pinjaman berhasil dicatat"})
}

// BendaharaRiwayat menampilkan halaman riwayat transaksi bendahara
func BendaharaRiwayat(c *gin.Context) {
	c.HTML(http.StatusOK, "bendahara_riwayat_content.html", gin.H{
		"ActivePage": "riwayat",
	})
}

func BendaharaLaporan(c *gin.Context) {
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
		c.HTML(http.StatusInternalServerError, "bendahara_laporan.html", gin.H{
			"ActivePage": "laporan",
			"Error":      "Gagal mengambil laporan",
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_laporan.html", gin.H{
		"ActivePage": "laporan",
		"Report":     report,
		"Bulan":      bulan,
		"Tahun":      tahun,
	})
}

// BendaharaTentang menampilkan halaman tentang kami bendahara
func BendaharaTentang(c *gin.Context) {
	c.HTML(http.StatusOK, "bendahara_layout.html", gin.H{
		"ActivePage": "tentang",
	})
}

// BendaharaPengaturan menampilkan halaman pengaturan bendahara
func BendaharaPengaturan(c *gin.Context) {
	// Ambil ID bendahara dari session
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data bendahara
	bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_layout.html", gin.H{
		"ActivePage": "pengaturan",
		"Bendahara":  bendahara,
	})
}

// UpdateBendaharaProfile memproses update username dan password bendahara
func UpdateBendaharaProfile(c *gin.Context) {
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
		bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bendahara"})
			return
		}
		passwordToUpdate = bendahara.Password
	}

	// Update username dan password
	err := repository.UpdatePengelolaUsernamePassword(bendaharaID.(int), request.Username, passwordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// BendaharaKonfirmasiTransaksi menampilkan halaman konfirmasi transaksi
func BendaharaKonfirmasiTransaksi(c *gin.Context) {
	// Ambil pending simpanan
	pendingSimpanan, err := repository.GetPendingSimpanan()
	if err != nil {
		pendingSimpanan = []models.Detail{}
	}

	// Ambil pending pinjaman
	pendingPinjaman, err := repository.GetPendingPinjaman()
	if err != nil {
		pendingPinjaman = []models.Pinjaman{}
	}

	// Ambil pending angsuran
	pendingAngsuran, err := repository.GetPendingAngsuran()
	if err != nil {
		pendingAngsuran = []models.Angsuran{}
	}

	// Ambil pending pengambilan simpanan
	pendingPengambilan, err := repository.GetPendingPengambilanSimpanan()
	if err != nil {
		pendingPengambilan = []models.PengambilanSimpanan{}
	}

	// Tambahkan nomor urut (No) mulai dari 1 untuk setiap daftar
	type numberedDetail struct {
		No int
		models.Detail
	}
	type numberedPinjaman struct {
		No int
		models.Pinjaman
	}
	type numberedAngsuran struct {
		No int
		models.Angsuran
	}
	type numberedPengambilan struct {
		No int
		models.PengambilanSimpanan
	}

	var numberedSimpanan []numberedDetail
	for i, d := range pendingSimpanan {
		numberedSimpanan = append(numberedSimpanan, numberedDetail{No: i + 1, Detail: d})
	}

	var numberedPinjamans []numberedPinjaman
	for i, p := range pendingPinjaman {
		numberedPinjamans = append(numberedPinjamans, numberedPinjaman{No: i + 1, Pinjaman: p})
	}

	var numberedAngsurans []numberedAngsuran
	for i, a := range pendingAngsuran {
		numberedAngsurans = append(numberedAngsurans, numberedAngsuran{No: i + 1, Angsuran: a})
	}

	var numberedPengambilans []numberedPengambilan
	for i, ps := range pendingPengambilan {
		numberedPengambilans = append(numberedPengambilans, numberedPengambilan{No: i + 1, PengambilanSimpanan: ps})
	}

	// Get LogoPath from context
	logoPath, _ := c.Get("LogoPath")

	c.HTML(http.StatusOK, "bendahara_konfirmasi_transaksi.html", gin.H{
		"PendingSimpanan":    numberedSimpanan,
		"PendingPinjaman":    numberedPinjamans,
		"PendingAngsuran":    numberedAngsurans,
		"PendingPengambilan": numberedPengambilans,
		"ActivePage":         "konfirmasi-transaksi",
		"LogoPath":           logoPath,
		"Title":              "Konfirmasi Transaksi",
	})
}

// BendaharaLihatDetailSimpanan menampilkan detail simpanan pending untuk anggota
func BendaharaLihatDetailSimpanan(c *gin.Context) {
	id := c.Param("id")

	// Ambil data anggota
	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Ambil semua simpanan pending dari anggota ini
	db := config.GetDB()
	query := `
		SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.tgl_transaksi, 
		       d.jumlah_simpanan, d.total_simpanan, s.jenis_simpanan,
		       COALESCE(d.status, 'pending') as status,
		       COALESCE(d.bukti_pembayaran, '') as bukti_pembayaran
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1 AND d.status = 'pending'
		ORDER BY d.tgl_transaksi DESC
	`

	rows, err := db.Query(query, id)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data simpanan"})
		return
	}
	defer rows.Close()

	var detailSimpanan []models.Detail
	var totalWajib, totalSukarela, totalHariRaya, grandTotal float64
	var buktiPembayaran string

	for rows.Next() {
		var d models.Detail
		var s models.Simpanan
		var bukti string
		err := rows.Scan(&d.IDDetail, &d.IDAnggota, &d.IDSimpanan, &d.TglTransaksi,
			&d.JumlahSimpanan, &d.TotalSimpanan, &s.JenisSimpanan, &d.Status, &bukti)
		if err != nil {
			continue
		}
		d.Simpanan = s
		d.BuktiPembayaran = bukti // Set bukti pembayaran untuk setiap detail
		detailSimpanan = append(detailSimpanan, d)

		// Ambil bukti pembayaran pertama yang ada (untuk backward compatibility)
		if buktiPembayaran == "" && bukti != "" {
			buktiPembayaran = bukti
		}

		// Hitung total per jenis
		// Note: jenis_simpanan from database: 'pokok', 'wajib', 'sukarela', 'hari_raya'
		switch s.JenisSimpanan {
		case "pokok":
			totalWajib += d.JumlahSimpanan // Simpanan pokok masuk ke kategori wajib
		case "wajib":
			totalWajib += d.JumlahSimpanan
		case "sukarela":
			totalSukarela += d.JumlahSimpanan
		case "hari_raya":
			totalHariRaya += d.JumlahSimpanan
		}
		grandTotal += d.JumlahSimpanan
	}

	// Ambil nomor rekening koperasi
	nomorRekening, _ := repository.GetNomorRekening("simpanan")

	c.HTML(http.StatusOK, "bendahara_detail_simpanan.html", gin.H{
		"Anggota":         anggota,
		"DetailSimpanan":  detailSimpanan,
		"TotalWajib":      totalWajib,
		"TotalSukarela":   totalSukarela,
		"TotalHariRaya":   totalHariRaya,
		"GrandTotal":      grandTotal,
		"NomorRekening":   nomorRekening,
		"BuktiPembayaran": buktiPembayaran,
		"Judul":           "Detail Simpanan Pending",
	})
}

// BendaharaLihatPersyaratanPinjaman menampilkan halaman persyaratan ajukan pinjaman untuk anggota (read-only)
func BendaharaLihatPersyaratanPinjaman(c *gin.Context) {
	id := c.Param("id")

	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Hitung total simpanan untuk menampilkan limit
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(id)
	if err != nil {
		totalSimpanan = 0
	}

	// Hitung limit pinjaman berdasarkan jenis anggota
	var limitPinjaman float64
	var jenisAnggota string

	switch anggota.UnitKerja {
	case "03": // Mahasiswa
		jenisAnggota = "Mahasiswa"
		limitPinjaman = 5 * totalSimpanan // 5x total simpanan
	case "01", "02": // Dosen/Staff
		jenisAnggota = "Dosen/Staff"
		limitPinjaman = 0 // Akan dihitung berdasarkan gaji di frontend
	default:
		jenisAnggota = "Tidak Diketahui"
		limitPinjaman = 0
	}

	// Ambil data pinjaman pending dari anggota ini (jika ada)
	db := config.GetDB()
	var pinjaman models.Pinjaman
	var hasPinjaman bool
	queryPinjaman := `
		SELECT id_pinjaman, id_anggota, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status, 
		       COALESCE(metode_pencairan, '') as metode_pencairan, COALESCE(nomor_rekening, '') as nomor_rekening,
		       COALESCE(gaji_bulanan, 0) as gaji_bulanan, COALESCE(tujuan_pinjaman, '') as tujuan_pinjaman
		FROM pinjaman 
		WHERE id_anggota = $1 AND status = 'proses'
		ORDER BY tgl_pinjaman DESC 
		LIMIT 1
	`
	err = db.QueryRow(queryPinjaman, id).Scan(
		&pinjaman.IDPinjaman,
		&pinjaman.IDAnggota,
		&pinjaman.TglPinjaman,
		&pinjaman.JumlahPinjaman,
		&pinjaman.JangkaWaktu,
		&pinjaman.Bunga,
		&pinjaman.Status,
		&pinjaman.MetodePencairan,
		&pinjaman.NomorRekening,
		&pinjaman.GajiBulanan,
		&pinjaman.TujuanPinjaman,
	)
	if err == nil {
		hasPinjaman = true
	}

	// Ambil bunga terkini dari database
	var bungaTerkini float64
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&bungaTerkini)
	if err != nil {
		// Jika belum ada pengaturan, gunakan default 2.0
		bungaTerkini = 2.0
	}

	// Render template persyaratan pinjaman bendahara
	c.HTML(http.StatusOK, "bendahara_persyaratan_pinjaman.html", gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"LimitPinjaman": limitPinjaman,
		"JenisAnggota":  jenisAnggota,
		"Judul":         "Lihat Persyaratan Pengajuan Pinjaman",
		"Pinjaman":      pinjaman,
		"HasPinjaman":   hasPinjaman,
		"Bunga":         bungaTerkini,
	})
}

// BendaharaKonfirmasiTransaksiPost menangani konfirmasi/reject transaksi
func BendaharaKonfirmasiTransaksiPost(c *gin.Context) {
	transactionType := c.Param("type")
	idStr := c.Param("id")
	action := c.PostForm("action")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	switch transactionType {
	case "simpanan":
		// Update status simpanan
		if action == "confirm" {
			err = repository.UpdateSimpananStatus(id, "confirmed")
		} else {
			err = repository.UpdateSimpananStatus(id, "rejected")
		}
	case "pinjaman":
		if action == "confirm" {
			err = repository.UpdatePinjamanStatus(id, "aktif")
		} else {
			err = repository.UpdatePinjamanStatus(id, "gagal")
		}
	case "angsuran":
		if action == "confirm" {
			err = repository.UpdateAngsuranStatus(id, "confirmed")
		} else {
			err = repository.UpdateAngsuranStatus(id, "rejected")
		}
	case "pengambilan":
		if action == "confirm" {
			err = repository.UpdatePengambilanSimpananStatus(id, "approved")
		} else {
			err = repository.UpdatePengambilanSimpananStatus(id, "rejected")
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe transaksi tidak valid"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaksi berhasil diproses"})
}

// GetCurrentBunga mengambil nilai bunga terkini dari database
func GetCurrentBunga() float64 {
	db := config.GetDB()
	var bunga float64

	// Buat tabel pengaturan jika belum ada
	db.Exec(`
		CREATE TABLE IF NOT EXISTS pengaturan (
			id SERIAL PRIMARY KEY,
			nama_pengaturan VARCHAR(50) UNIQUE NOT NULL,
			nilai VARCHAR(100) NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)

	// Cek apakah bunga sudah ada di database
	err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&bunga)
	if err != nil {
		// Jika belum ada, insert nilai default 2.0
		db.Exec("INSERT INTO pengaturan (nama_pengaturan, nilai) VALUES ('bunga_pinjaman', '2.0')")
		bunga = 2.0
	}

	return bunga
}

func BendaharaEditBunga(c *gin.Context) {
	bunga := GetCurrentBunga()
	c.HTML(http.StatusOK, "bendahara_edit_bunga.html", gin.H{
		"CurrentBunga": fmt.Sprintf("%.2f", bunga),
		"ActivePage":   "edit-bunga",
	})
}

func BendaharaUpdateBunga(c *gin.Context) {
	bungaStr := c.PostForm("bunga")
	if bungaStr == "" {
		c.HTML(http.StatusBadRequest, "bendahara_edit_bunga.html", gin.H{
			"Error":        "Nilai bunga harus diisi",
			"CurrentBunga": "",
			"ActivePage":   "edit-bunga",
		})
		return
	}

	bungaVal, err := strconv.ParseFloat(bungaStr, 64)
	if err != nil || bungaVal < 0 {
		c.HTML(http.StatusBadRequest, "bendahara_edit_bunga.html", gin.H{
			"Error":        "Nilai bunga tidak valid",
			"CurrentBunga": bungaStr,
			"ActivePage":   "edit-bunga",
		})
		return
	}

	// Simpan bunga ke database
	db := config.GetDB()
	_, err = db.Exec(`
		INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at) 
		VALUES ('bunga_pinjaman', $1, NOW())
		ON CONFLICT (nama_pengaturan) 
		DO UPDATE SET nilai = $1, updated_at = NOW()
	`, bungaStr)

	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_edit_bunga.html", gin.H{
			"Error":        "Gagal menyimpan bunga ke database",
			"CurrentBunga": bungaStr,
			"ActivePage":   "edit-bunga",
		})
		return
	}

	// Redirect back to edit page after update
	c.Redirect(http.StatusFound, "/bendahara/edit-bunga")
}
