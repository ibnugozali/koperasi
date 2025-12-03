package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
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

	totalAngsuran, err := repository.GetTotalAngsuran(db)
	if err != nil {
		totalAngsuran = 0
	}

	totalPengambilan, err := repository.GetTotalPengambilan(db)
	if err != nil {
		totalPengambilan = 0
	}

	// Data untuk template
	data := map[string]interface{}{
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"TotalAngsuran":      totalAngsuran,
		"TotalPengambilan":   totalPengambilan,
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

	if slug == "simpanan" {
		// Handle special case for simpanan with JSON konten
		konten := c.PostForm("konten")
		if konten == "" {
			c.String(http.StatusBadRequest, "Data konten tidak valid")
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
			Konten: konten,
		}

		err = repository.UpdateHalaman(halaman)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal memperbarui halaman: "+err.Error())
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

	// Ambil data simpanan wajib untuk semua anggota
	simpananWajib, err := repository.GetSimpananWajibAllAnggota()
	if err != nil {
		simpananWajib = make(map[string]float64) // Default ke map kosong jika error
	}

	// Ambil data pemotongan bulan ini untuk semua anggota
	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64) // Default ke map kosong jika error
	}

	// Hitung sisa gaji untuk setiap anggota
	sisaGaji := make(map[string]int)
	for _, anggota := range anggotas {
		potongan := int(potonganBulanIni[anggota.IDAnggota])
		sisaGaji[anggota.IDAnggota] = anggota.GajiBulanan - potongan
	}

	c.HTML(http.StatusOK, "bendahara_data_anggota.html", gin.H{
		"Anggotas":         anggotas,
		"SimpananWajib":    simpananWajib,
		"PotonganBulanIni": potonganBulanIni,
		"SisaGaji":         sisaGaji,
		"ActivePage":       "anggota",
		"LogoPath":         logoPath,
		"Title":            "Data Anggota",
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
		log.Printf("Error binding data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	log.Printf("Data anggota yang diterima: NamaAnggota=%s, TglLahir=%s, GajiBulanan=%d",
		anggota.NamaAnggota, anggota.TglLahir, anggota.GajiBulanan)

	// Update query (assuming we update all fields except password for simplicity)
	db := config.GetDB()
	query := `
		UPDATE anggota SET
			nama_anggota = $1, username = $2, tgl_lahir = $3, nik_ktp = $4,
			no_telepon = $5, alamat = $6, jenis_kelamin = $7, status_anggota = $8, fakultas = $9, gaji_bulanan = $10
		WHERE id_anggota = $11`
	_, err := db.Exec(query,
		anggota.NamaAnggota, anggota.Username, anggota.TglLahir, anggota.NikKTP,
		anggota.NoTelepon, anggota.Alamat, anggota.JenisKelamin, anggota.StatusAnggota, anggota.Fakultas, anggota.GajiBulanan, idStr)
	if err != nil {
		log.Printf("Error updating anggota %s: %v", idStr, err)
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
	// Ambil semua data riwayat transaksi dari database
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_riwayat_content.html", gin.H{
			"ActivePage": "riwayat",
			"Error":      "Gagal mengambil data riwayat: " + err.Error(),
		})
		return
	}

	// Ambil daftar anggota untuk filter
	db := config.GetDB()
	var anggotas []models.Anggota
	rows, err := db.Query("SELECT id_anggota, nama_anggota FROM anggota ORDER BY nama_anggota")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a models.Anggota
			if err := rows.Scan(&a.IDAnggota, &a.NamaAnggota); err == nil {
				anggotas = append(anggotas, a)
			}
		}
	}

	c.HTML(http.StatusOK, "bendahara_riwayat_content.html", gin.H{
		"ActivePage": "riwayat",
		"Riwayats":   riwayats,
		"Anggotas":   anggotas,
	})
}

func BendaharaLaporan(c *gin.Context) {
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

	// Get LogoPath from context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_laporan.html", gin.H{
			"ActivePage": "laporan",
			"Error":      "Gagal mengambil laporan",
			"LogoPath":   logoPath,
			"Bulan":      bulan,
			"Tahun":      tahun,
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_laporan.html", gin.H{
		"ActivePage": "laporan",
		"Report":     report,
		"Bulan":      bulan,
		"Tahun":      tahun,
		"LogoPath":   logoPath,
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

	// Get LogoPath from context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	// Ambil data bendahara
	bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara: " + err.Error(),
			"LogoPath":   logoPath,
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_layout.html", gin.H{
		"ActivePage": "pengaturan",
		"Bendahara":  bendahara,
		"LogoPath":   logoPath,
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

// BendaharaLoginHistory menampilkan halaman riwayat login
func BendaharaLoginHistory(c *gin.Context) {
	// Get LogoPath from context (set by middleware)
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	// Ambil data riwayat login dari database
	loginHistory, err := repository.GetLoginHistory()
	if err != nil {
		loginHistory = []models.LoginHistory{} // Default kosong jika error
	}

	c.HTML(http.StatusOK, "bendahara_login_history.html", gin.H{
		"ActivePage":   "login_history",
		"LoginHistory": loginHistory,
		"LogoPath":     logoPath,
	})
}

// BendaharaDeleteLoginHistory menghapus riwayat login berdasarkan ID
func BendaharaDeleteLoginHistory(c *gin.Context) {
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

// BendaharaDeleteAllLoginHistory menghapus semua riwayat login
func BendaharaDeleteAllLoginHistory(c *gin.Context) {
	err := repository.DeleteAllLoginHistory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus semua riwayat login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Semua riwayat login berhasil dihapus"})
}

// BendaharaImportAnggotaPage menampilkan halaman import data anggota
func BendaharaImportAnggotaPage(c *gin.Context) {
	// Ambil logo path dari context (sudah di-set oleh middleware)
	logoPath, exists := c.Get("logoPath")
	if !exists {
		logoPath = "/static/images/logo.png"
	}

	// Ambil session untuk mendapatkan ID pengelola
	session := sessions.Default(c)
	idPengelola := session.Get("user_id") // Gunakan "user_id" sesuai dengan key yang diset saat login

	// Ambil riwayat import terbaru
	db := config.GetDB()
	var latestImport *models.ImportHistory
	var allImportedData []map[string]interface{} // Gabungan SEMUA data dari semua import
	var parseErrors []string
	var totalSuccessCount int
	var totalFailedCount int

	fmt.Println("=== BendaharaImportAnggotaPage called ===")
	fmt.Printf("Session user_id: %v (type: %T)\n", idPengelola, idPengelola)

	if idPengelola != nil {
		// Convert ke int
		pengelolaID := 0
		if id, ok := idPengelola.(int); ok {
			pengelolaID = id
		} else if idStr, ok := idPengelola.(string); ok {
			pengelolaID, _ = strconv.Atoi(idStr)
		}

		fmt.Printf("=== Loading ALL import history for pengelola ID: %d ===\n", pengelolaID)

		// Ambil SEMUA riwayat import (untuk akumulasi data)
		allImports, err := repository.GetAllImportHistory(db, pengelolaID, 100) // Ambil max 100 import terakhir
		if err != nil {
			fmt.Printf("❌ Error loading import history: %v\n", err)
		} else if len(allImports) > 0 {
			fmt.Printf("✓ Found %d import records in database\n", len(allImports))

			// Set latest import untuk info header
			latestImport = &allImports[0]
			fmt.Printf("✓ Latest import: %s (Date: %v)\n", latestImport.FileName, latestImport.TanggalImport)

			// Gabungkan SEMUA data dari semua import
			for idx, imp := range allImports {
				fmt.Printf("  [%d] Processing import: %s (Success: %d, Failed: %d)\n",
					idx+1, imp.FileName, imp.SuccessCount, imp.FailedCount)

				totalSuccessCount += imp.SuccessCount
				totalFailedCount += imp.FailedCount

				// Parse dan gabungkan imported data
				if imp.ImportedData != "" {
					var importData []map[string]interface{}
					if err := json.Unmarshal([]byte(imp.ImportedData), &importData); err != nil {
						fmt.Printf("    ❌ Error parsing ImportedData: %v\n", err)
					} else {
						fmt.Printf("    ✓ Adding %d records from this import\n", len(importData))
						allImportedData = append(allImportedData, importData...)
					}
				} else {
					fmt.Printf("    ⚠️ No ImportedData for this record\n")
				}

				// Gabungkan parse errors dari latest import saja
				if idx == 0 && imp.ParseErrors != "" {
					if err := json.Unmarshal([]byte(imp.ParseErrors), &parseErrors); err != nil {
						fmt.Printf("    ❌ Error parsing ParseErrors: %v\n", err)
					}
				}
			}

			fmt.Printf("✓ Total accumulated data: %d records (Success: %d, Failed: %d)\n",
				len(allImportedData), totalSuccessCount, totalFailedCount)
		} else {
			fmt.Printf("ℹ️ No import history found for pengelola ID: %d (database is empty)\n", pengelolaID)
		}
	} else {
		fmt.Println("⚠️ No pengelola ID found in session - user not logged in?")
	}

	// Ambil data anggota real-time dari database untuk menampilkan gaji terbaru
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		fmt.Printf("❌ Error loading anggota data: %v\n", err)
		anggotas = []models.Anggota{}
	}

	// Ambil data potongan bulan ini untuk menghitung sisa gaji
	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		fmt.Printf("❌ Error loading potongan data: %v\n", err)
		potonganBulanIni = make(map[string]float64)
	}

	// Convert anggota ke format map untuk template dengan sisa gaji yang sudah dikurangi potongan
	var realTimeData []map[string]interface{}
	for _, anggota := range anggotas {
		// Hitung sisa gaji = gaji_bulanan - potongan_bulan_ini
		potongan := potonganBulanIni[anggota.IDAnggota]
		sisaGaji := float64(anggota.GajiBulanan) - potongan
		if sisaGaji < 0 {
			sisaGaji = 0
		}

		realTimeData = append(realTimeData, map[string]interface{}{
			"id_anggota":     anggota.IDAnggota,
			"nama_anggota":   anggota.NamaAnggota,
			"username":       anggota.Username,
			"nik_ktp":        anggota.NikKTP,
			"no_telepon":     anggota.NoTelepon,
			"alamat":         anggota.Alamat,
			"provinsi":       anggota.Provinsi,
			"jenis_kelamin":  anggota.JenisKelamin,
			"tgl_lahir":      anggota.TglLahir,
			"fakultas":       anggota.Fakultas,
			"unit_kerja":     anggota.UnitKerja,
			"gaji_bulanan":   int(sisaGaji), // Kirim sisa gaji (gaji - potongan)
			"status":         anggota.Status,
			"status_anggota": anggota.StatusAnggota,
		})
	}

	fmt.Printf("=== Rendering template with %d total records (real-time from database) ===\n", len(realTimeData))

	c.HTML(http.StatusOK, "bendahara_import_anggota.html", gin.H{
		"ActivePage":        "import_anggota",
		"LogoPath":          logoPath,
		"LatestImport":      latestImport,
		"ImportedData":      realTimeData, // Kirim data real-time dari database
		"ParseErrors":       parseErrors,
		"TotalSuccessCount": totalSuccessCount, // Total success dari semua import
		"TotalFailedCount":  totalFailedCount,  // Total failed dari semua import
	})
}

// BendaharaImportAnggota memproses upload file XLSX dan import data anggota
func BendaharaImportAnggota(c *gin.Context) {
	// Ambil file dari form
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println("Error getting file:", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File tidak ditemukan. Pastikan Anda telah memilih file.",
		})
		return
	}

	fmt.Printf("File received: %s, Size: %d bytes\n", file.Filename, file.Size)

	// Validasi ekstensi file
	ext := filepath.Ext(file.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		fmt.Printf("Invalid extension: %s\n", ext)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("File harus berformat .xlsx atau .xls (Anda mengupload: %s)", ext),
		})
		return
	}

	// Validasi ukuran file (max 10MB)
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Ukuran file terlalu besar: %.2f MB (maksimal 10MB)", float64(file.Size)/(1024*1024)),
		})
		return
	}

	// Simpan file sementara
	tempPath := "./static/uploads/" + uuid.New().String() + ext
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		fmt.Println("Error saving file:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan file: " + err.Error(),
		})
		return
	}

	fmt.Printf("File saved to: %s\n", tempPath)

	// Hapus file temporary setelah selesai
	defer func() {
		// Hapus file temporary untuk menghemat space
		os.Remove(tempPath)
	}()

	// Buka file Excel
	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file Excel: " + err.Error()})
		return
	}
	defer f.Close()

	// Ambil sheet pertama
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak memiliki sheet"})
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca data dari Excel"})
		return
	}

	if len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak memiliki data (minimal harus ada header dan 1 baris data)"})
		return
	}

	fmt.Printf("Total rows: %d, Header columns: %v\n", len(rows), rows[0])

	// Parse data dari Excel (skip header row)
	var anggotaList []models.Anggota
	var errors []string

	// Ambil header untuk deteksi otomatis
	headers := rows[0]
	fmt.Printf("Detected headers: %v\n", headers)
	fmt.Printf("File diterima dengan %d kolom\n", len(headers))

	// Helper function untuk mendapatkan value atau default
	getValue := func(row []string, index int) string {
		if index < len(row) {
			return row[index]
		}
		return ""
	}

	// Helper function untuk mapping unit kerja ke kode 2 digit
	mapUnitKerja := func(unitKerja string) string {
		if len(unitKerja) <= 2 {
			return unitKerja
		}
		unitKerja = strings.ToLower(strings.TrimSpace(unitKerja))
		switch {
		case strings.Contains(unitKerja, "dosen"):
			return "01"
		case strings.Contains(unitKerja, "karyawan") || strings.Contains(unitKerja, "staff"):
			return "02"
		case strings.Contains(unitKerja, "mahasiswa"):
			return "03"
		default:
			return "" // Kosongkan jika tidak dikenali
		}
	}

	// Helper function untuk mapping fakultas ke kode 2 digit
	mapFakultasCode := func(fakultas string) string {
		if len(fakultas) <= 2 {
			return fakultas
		}
		fakultas = strings.ToUpper(strings.TrimSpace(fakultas))
		switch {
		case strings.Contains(fakultas, "FAI") || strings.Contains(fakultas, "AGAMA"):
			return "01"
		case strings.Contains(fakultas, "FE") || strings.Contains(fakultas, "EKONOMI"):
			return "02"
		case strings.Contains(fakultas, "FH") || strings.Contains(fakultas, "HUKUM"):
			return "03"
		case strings.Contains(fakultas, "FISIP") || strings.Contains(fakultas, "SOSIAL") || strings.Contains(fakultas, "POLITIK"):
			return "04"
		case strings.Contains(fakultas, "FKIP") || strings.Contains(fakultas, "KEGURUAN"):
			return "05"
		case strings.Contains(fakultas, "FKM") || strings.Contains(fakultas, "KESEHATAN MASYARAKAT"):
			return "06"
		case strings.Contains(fakultas, "FAPERTA") || strings.Contains(fakultas, "PERTANIAN"):
			return "07"
		case strings.Contains(fakultas, "FT") || strings.Contains(fakultas, "TEKNIK"):
			return "08"
		case strings.Contains(fakultas, "REKTORAT") || strings.Contains(fakultas, "YAYASAN"):
			return "09"
		default:
			return "" // Kosongkan jika tidak dikenali
		}
	}

	for i, row := range rows {
		if i == 0 {
			// Skip header
			continue
		}

		// Pastikan row memiliki minimal 3 kolom (Nama, Unit Kerja, Tanggal Lahir)
		if len(row) < 3 {
			errors = append(errors, fmt.Sprintf("Baris %d: Data tidak lengkap - minimal harus ada 3 kolom (Nama, Unit Kerja, Tanggal Lahir)", i+1))
			continue
		}

		// Ambil data dengan aman sesuai urutan template:
		// Nama Anggota, Unit Kerja, Tanggal Lahir, NIK KTP, No Telepon, Jenis Kelamin, Fakultas, Gaji Bulanan, Alamat
		namaAnggota := getValue(row, 0)
		unitKerja := getValue(row, 1)
		tglLahir := getValue(row, 2)
		nikKTP := getValue(row, 3)
		noTelepon := getValue(row, 4)
		jenisKelamin := getValue(row, 5)
		fakultas := getValue(row, 6)
		gajiBulananStr := getValue(row, 7)
		alamat := getValue(row, 8)

		// Generate username otomatis dari nama (lowercase, tanpa spasi)
		username := strings.ToLower(strings.ReplaceAll(namaAnggota, " ", ""))

		// Status anggota default aktif
		statusAnggota := "Aktif"

		// Validasi data kosong untuk field penting
		if namaAnggota == "" {
			errors = append(errors, fmt.Sprintf("Baris %d: Nama Anggota tidak boleh kosong", i+1))
			continue
		}

		// Parse gaji bulanan dari Excel sebagai default
		var gajiBulanan int
		if gajiBulananStr != "" {
			gajiBulanan, _ = strconv.Atoi(strings.ReplaceAll(gajiBulananStr, ",", ""))
		}

		// PENTING: Cek apakah anggota sudah ada di database (berdasarkan NIK)
		// Jika sudah ada, SELALU gunakan sisa gaji dari database, BUKAN dari Excel
		if nikKTP != "" {
			var existingGaji int
			checkGajiQuery := "SELECT COALESCE(gaji_bulanan, 0) FROM anggota WHERE nik_ktp = $1 LIMIT 1"
			err := config.GetDB().QueryRow(checkGajiQuery, nikKTP).Scan(&existingGaji)
			if err == nil {
				// Anggota sudah ada - gunakan gaji dari database (ini adalah sisa gaji setelah pemotongan)
				gajiBulanan = existingGaji
				fmt.Printf("  Baris %d: Anggota sudah ada (NIK: %s), menggunakan sisa gaji dari database: Rp %d (mengabaikan Excel: %s)\n", i+1, nikKTP, gajiBulanan, gajiBulananStr)
			} else {
				// Anggota baru - gunakan gaji dari Excel
				fmt.Printf("  Baris %d: Anggota baru, menggunakan gaji dari Excel: Rp %d\n", i+1, gajiBulanan)
			}
		}

		fmt.Printf("  Baris %d: Nama=%s, Gaji Final=%d\n", i+1, namaAnggota, gajiBulanan)

		// Validasi format tanggal lahir jika ada (harus format date, bukan angka)
		if tglLahir != "" {
			// Cek jika tanggal lahir berupa angka murni (kemungkinan salah file - data gaji/lainnya)
			if _, err := strconv.ParseFloat(tglLahir, 64); err == nil && len(tglLahir) > 6 {
				errors = append(errors, fmt.Sprintf("Baris %d: Format tanggal lahir tidak valid (terdeteksi angka: %s). Gunakan format YYYY-MM-DD (contoh: 1990-05-15)", i+1, tglLahir))
				continue
			}
			// Validasi format tanggal YYYY-MM-DD atau DD/MM/YYYY
			if !strings.Contains(tglLahir, "-") && !strings.Contains(tglLahir, "/") {
				errors = append(errors, fmt.Sprintf("Baris %d: Format tanggal lahir harus YYYY-MM-DD atau DD/MM/YYYY (saat ini: %s)", i+1, tglLahir))
				continue
			}
		}

		// Validasi NIK jika ada (harus 16 digit)
		if nikKTP != "" && len(nikKTP) != 16 {
			errors = append(errors, fmt.Sprintf("Baris %d: NIK harus 16 digit (saat ini: %d digit)", i+1, len(nikKTP)))
			continue
		}

		// Mapping unit_kerja dan fakultas_code ke format 2 digit
		unitKerjaCode := mapUnitKerja(unitKerja)
		fakultasCode := mapFakultasCode(fakultas)

		// Hash password default
		defaultPassword := "12345678" // Password default
		if nikKTP != "" && len(nikKTP) == 16 {
			defaultPassword = nikKTP // Gunakan NIK jika valid
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Baris %d: Gagal hash password", i+1))
			continue
		}

		// Buat anggota baru
		anggota := models.Anggota{
			IDAnggota:     uuid.New().String(),
			NamaAnggota:   namaAnggota,
			Username:      username,
			Password:      string(hashedPassword),
			TglLahir:      tglLahir,
			JenisKelamin:  jenisKelamin,
			Alamat:        alamat,
			NikKTP:        nikKTP,
			NoTelepon:     noTelepon,
			UnitKerja:     unitKerjaCode,
			Fakultas:      fakultas,
			StatusAnggota: statusAnggota,
			Status:        "aktif",
			TglGabung:     time.Now(),
			FakultasCode:  fakultasCode,
			GajiBulanan:   gajiBulanan,
		}

		anggotaList = append(anggotaList, anggota)
	}

	fmt.Printf("Parsed %d valid records from %d total rows\n", len(anggotaList), len(rows)-1)
	fmt.Printf("Parse errors: %d\n", len(errors))

	// Cek apakah ada data yang valid untuk diimport
	if len(anggotaList) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "Tidak ada data valid untuk diimport. Periksa format data Anda.",
			"parseErrors":     errors,
			"hint":            "Minimal harus ada kolom: Nama Anggota | Username. Kolom opsional: Tanggal Lahir | Jenis Kelamin | Alamat | NIK (16 digit) | No. Telepon | Unit Kerja | Fakultas | Status Anggota",
			"detectedHeaders": rows[0],
		})
		return
	}

	// SIMPAN KE DATABASE - Data anggota akan disimpan ke tabel anggota
	db := config.GetDB()
	successCount := 0
	failedCount := 0
	var allErrors []string
	allErrors = append(allErrors, errors...)

	// Simpan setiap anggota ke database
	for _, anggota := range anggotaList {
		// Cek apakah NIK sudah ada (untuk update atau insert)
		var existingID string
		checkQuery := "SELECT id_anggota FROM anggota WHERE nik_ktp = $1 LIMIT 1"
		err := db.QueryRow(checkQuery, anggota.NikKTP).Scan(&existingID)

		if err == nil && existingID != "" {
			// Anggota sudah ada, lakukan UPDATE
			// PENTING: gaji_bulanan di anggota.GajiBulanan sudah berisi sisa gaji dari database
			// (diambil saat parsing Excel di atas)
			updateQuery := `
				UPDATE anggota SET
					nama_anggota = $1,
					tgl_lahir = $2,
					jenis_kelamin = $3,
					alamat = $4,
					no_telepon = $5,
					unit_kerja = $6,
					fakultas = $7,
					fakultas_code = $8,
					gaji_bulanan = $9,
					status_anggota = $10
				WHERE id_anggota = $11
			`

			_, err = db.Exec(
				updateQuery,
				anggota.NamaAnggota,
				anggota.TglLahir,
				anggota.JenisKelamin,
				anggota.Alamat,
				anggota.NoTelepon,
				anggota.UnitKerja,
				anggota.Fakultas,
				anggota.FakultasCode,
				anggota.GajiBulanan, // Sudah berisi sisa gaji dari database (diambil saat parsing)
				anggota.StatusAnggota,
				existingID,
			)

			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("Gagal update %s (NIK: %s): %v", anggota.NamaAnggota, anggota.NikKTP, err))
				failedCount++
			} else {
				successCount++
				fmt.Printf("✓ Updated anggota: %s (mempertahankan sisa gaji: Rp %d)\n", anggota.NamaAnggota, anggota.GajiBulanan)
			}
			continue
		}

		// Insert anggota baru ke database
		insertQuery := `
			INSERT INTO anggota (
				id_anggota, nama_anggota, username, password, tgl_lahir, 
				jenis_kelamin, alamat, nik_ktp, no_telepon, unit_kerja, 
				fakultas, fakultas_code, gaji_bulanan, status_anggota, status, tgl_gabung
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		`

		_, err = db.Exec(
			insertQuery,
			anggota.IDAnggota,
			anggota.NamaAnggota,
			anggota.Username,
			anggota.Password,
			anggota.TglLahir,
			anggota.JenisKelamin,
			anggota.Alamat,
			anggota.NikKTP,
			anggota.NoTelepon,
			anggota.UnitKerja,
			anggota.Fakultas,
			anggota.FakultasCode,
			anggota.GajiBulanan,
			anggota.StatusAnggota,
			"aktif", // Status langsung aktif untuk import agar bisa diproses pemotongan
			anggota.TglGabung,
		)

		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Gagal menyimpan %s: %v", anggota.NamaAnggota, err))
			failedCount++
		} else {
			successCount++
			fmt.Printf("✓ Inserted anggota baru: %s (gaji: Rp %d)\n", anggota.NamaAnggota, anggota.GajiBulanan)
		}
	}

	fmt.Printf("✓ Data saved to database: %d success, %d failed\n", successCount, failedCount)

	// Ambil semua data anggota untuk ditampilkan sebagai preview
	var importedData []gin.H
	for _, anggota := range anggotaList {
		importedData = append(importedData, gin.H{
			"nama_anggota":   anggota.NamaAnggota,
			"unit_kerja":     anggota.UnitKerja,
			"tgl_lahir":      anggota.TglLahir,
			"jenis_kelamin":  anggota.JenisKelamin,
			"alamat":         anggota.Alamat,
			"nik_ktp":        anggota.NikKTP,
			"no_telepon":     anggota.NoTelepon,
			"fakultas":       anggota.Fakultas,
			"gaji_bulanan":   anggota.GajiBulanan,
			"status_anggota": anggota.StatusAnggota,
		})
	}

	// Simpan riwayat import ke database
	session := sessions.Default(c)
	idPengelola := session.Get("user_id") // Gunakan "user_id" sesuai dengan key yang diset saat login
	username := session.Get("username")

	if idPengelola != nil {
		// Convert idPengelola ke int
		pengelolaID := 0
		if id, ok := idPengelola.(int); ok {
			pengelolaID = id
		} else if idStr, ok := idPengelola.(string); ok {
			pengelolaID, _ = strconv.Atoi(idStr)
		}

		fmt.Printf("=== Saving import history for pengelola ID: %d ===\n", pengelolaID)

		// Convert data ke JSON string
		importedDataJSON, _ := json.Marshal(importedData)
		parseErrorsJSON, _ := json.Marshal(allErrors)

		// Buat record import history
		importHistory := models.ImportHistory{
			IDImport:      uuid.New().String(),
			IDPengelola:   pengelolaID,
			Username:      fmt.Sprintf("%v", username),
			FileName:      file.Filename,
			TotalData:     len(anggotaList),
			SuccessCount:  successCount,
			FailedCount:   failedCount,
			ImportedData:  string(importedDataJSON),
			ParseErrors:   string(parseErrorsJSON),
			TanggalImport: time.Now(),
		}

		// Simpan ke database
		if err := repository.SaveImportHistory(db, importHistory); err != nil {
			fmt.Printf("❌ Error saving import history: %v\n", err)
		} else {
			fmt.Printf("✓ Import history saved successfully\n")
		}
	} else {
		fmt.Println("⚠️ Cannot save import history: No pengelola ID in session")
	}

	// Response
	c.JSON(http.StatusOK, gin.H{
		"message":      "Preview import selesai - data TIDAK disimpan ke database anggota",
		"success":      successCount,
		"failed":       failedCount,
		"total":        len(anggotaList),
		"parseErrors":  allErrors,
		"importedData": importedData,
	})
}

// BendaharaPreviewImportAnggota untuk preview data dari file Excel sebelum import
func BendaharaPreviewImportAnggota(c *gin.Context) {
	// Ambil file dari form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File tidak ditemukan",
		})
		return
	}

	// Validasi ekstensi file
	ext := filepath.Ext(file.Filename)
	if ext != ".xlsx" && ext != ".xls" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File harus berformat .xlsx atau .xls",
		})
		return
	}

	// Simpan file sementara
	tempPath := "./static/uploads/" + uuid.New().String() + ext
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan file",
		})
		return
	}

	// Hapus file temporary setelah selesai
	defer os.Remove(tempPath)

	// Buka file Excel
	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuka file Excel: " + err.Error(),
		})
		return
	}
	defer f.Close()

	// Ambil sheet pertama
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File Excel tidak memiliki sheet",
		})
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membaca data dari Excel",
		})
		return
	}

	if len(rows) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File Excel kosong",
		})
		return
	}

	// Ambil header (baris pertama)
	headers := rows[0]

	// Ambil sample data (maksimal 5 baris pertama setelah header)
	sampleData := [][]string{}
	maxSample := 5
	if len(rows) > maxSample+1 {
		sampleData = rows[1 : maxSample+1]
	} else if len(rows) > 1 {
		sampleData = rows[1:]
	}

	// Helper function untuk preview
	getValuePreview := func(row []string, index int) string {
		if index < len(row) {
			return row[index]
		}
		return ""
	}

	// Helper function untuk mapping unit kerja ke kode 2 digit (untuk preview)
	mapUnitKerjaPreview := func(unitKerja string) string {
		if len(unitKerja) <= 2 {
			return unitKerja
		}
		unitKerja = strings.ToLower(strings.TrimSpace(unitKerja))
		switch {
		case strings.Contains(unitKerja, "dosen"):
			return "01"
		case strings.Contains(unitKerja, "karyawan") || strings.Contains(unitKerja, "staff"):
			return "02"
		case strings.Contains(unitKerja, "mahasiswa"):
			return "03"
		default:
			return ""
		}
	}

	// Helper function untuk mapping fakultas ke kode 2 digit (untuk preview)
	mapFakultasCodePreview := func(fakultas string) string {
		if len(fakultas) <= 2 {
			return fakultas
		}
		fakultas = strings.ToUpper(strings.TrimSpace(fakultas))
		switch {
		case strings.Contains(fakultas, "FAI") || strings.Contains(fakultas, "AGAMA"):
			return "01"
		case strings.Contains(fakultas, "FE") || strings.Contains(fakultas, "EKONOMI"):
			return "02"
		case strings.Contains(fakultas, "FH") || strings.Contains(fakultas, "HUKUM"):
			return "03"
		case strings.Contains(fakultas, "FISIP") || strings.Contains(fakultas, "SOSIAL") || strings.Contains(fakultas, "POLITIK"):
			return "04"
		case strings.Contains(fakultas, "FKIP") || strings.Contains(fakultas, "KEGURUAN"):
			return "05"
		case strings.Contains(fakultas, "FKM") || strings.Contains(fakultas, "KESEHATAN MASYARAKAT"):
			return "06"
		case strings.Contains(fakultas, "FAPERTA") || strings.Contains(fakultas, "PERTANIAN"):
			return "07"
		case strings.Contains(fakultas, "FT") || strings.Contains(fakultas, "TEKNIK"):
			return "08"
		case strings.Contains(fakultas, "REKTORAT") || strings.Contains(fakultas, "YAYASAN"):
			return "09"
		default:
			return ""
		}
	}

	// Validasi format untuk memberikan feedback
	formatValid := true
	formatErrors := []string{}

	// Cek minimal 2 kolom (Nama dan Unit Kerja)
	if len(headers) < 2 {
		formatValid = false
		formatErrors = append(formatErrors, fmt.Sprintf("File harus memiliki minimal 2 kolom (Nama dan Unit Kerja), file Anda hanya memiliki %d kolom", len(headers)))
	}

	// Simulasi parsing untuk preview - cek data yang akan berhasil/gagal
	var previewValidCount int
	var previewErrors []string

	if formatValid && len(sampleData) > 0 {
		for i, row := range sampleData {
			rowNum := i + 2 // +1 untuk header, +1 untuk index 0

			// Validasi sama seperti import asli
			if len(row) < 2 {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Data tidak lengkap (minimal 2 kolom)", rowNum))
				continue
			}

			// Urutan sesuai template: Nama Anggota, Unit Kerja, Tanggal Lahir, NIK KTP, No Telepon, Jenis Kelamin, Fakultas, Gaji Bulanan, Alamat
			namaAnggota := getValuePreview(row, 0)
			unitKerja := getValuePreview(row, 1)
			tglLahir := getValuePreview(row, 2)
			nikKTP := getValuePreview(row, 3)
			_ = getValuePreview(row, 4) // noTelepon - tidak divalidasi di preview
			_ = getValuePreview(row, 5) // jenisKelamin - tidak divalidasi di preview
			fakultas := getValuePreview(row, 6)
			_ = getValuePreview(row, 7) // gajiBulanan - tidak divalidasi di preview

			if namaAnggota == "" {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Nama Anggota tidak boleh kosong", rowNum))
				continue
			}

			// Validasi format tanggal lahir jika ada
			if tglLahir != "" {
				// Cek jika tanggal lahir berupa angka murni (kemungkinan salah file)
				if _, err := strconv.ParseFloat(tglLahir, 64); err == nil && len(tglLahir) > 6 {
					previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: ⚠️ PERINGATAN - File ini sepertinya bukan file import anggota (kolom Tanggal Lahir berisi angka: %s). Download template yang benar!", rowNum, tglLahir))
					continue
				}
				if !strings.Contains(tglLahir, "-") && !strings.Contains(tglLahir, "/") {
					previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Format tanggal lahir harus YYYY-MM-DD atau DD/MM/YYYY (saat ini: %s)", rowNum, tglLahir))
					continue
				}
			}

			// Validasi NIK jika ada
			if nikKTP != "" && len(nikKTP) != 16 {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: NIK harus 16 digit (saat ini: %d digit) - NIK: %s", rowNum, len(nikKTP), nikKTP))
				continue
			}

			// Validasi unit_kerja dan fakultas (harus bisa dimapping atau sudah 2 digit)
			unitKerjaCode := mapUnitKerjaPreview(unitKerja)
			fakultasCode := mapFakultasCodePreview(fakultas)

			if unitKerja != "" && len(unitKerja) > 2 && unitKerjaCode == "" {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Unit Kerja '%s' tidak valid. Gunakan: Dosen, Karyawan/Staff, atau Mahasiswa", rowNum, unitKerja))
				continue
			}

			if fakultas != "" && len(fakultas) > 2 && fakultasCode == "" {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Fakultas '%s' tidak valid. Gunakan: FAI, FE, FH, FISIP, FKIP, FKM, FAPERTA, FT, atau Rektorat", rowNum, fakultas))
				continue
			}

			previewValidCount++
		}
	}

	// Return preview data dengan informasi validasi
	c.JSON(http.StatusOK, gin.H{
		"headers":           headers,
		"sampleData":        sampleData,
		"totalRows":         len(rows) - 1, // exclude header
		"columnCount":       len(headers),
		"filename":          file.Filename,
		"formatValid":       formatValid,
		"formatErrors":      formatErrors,
		"previewValidCount": previewValidCount,
		"previewErrorCount": len(previewErrors),
		"previewErrors":     previewErrors,
	})
}

// BendaharaClearImportHistory menghapus semua riwayat import anggota
func BendaharaClearImportHistory(c *gin.Context) {
	db := config.GetDB()

	// Ambil session untuk mendapatkan ID pengelola
	session := sessions.Default(c)
	idPengelola := session.Get("user_id")

	if idPengelola == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized - No user session",
		})
		return
	}

	// Convert ke int
	pengelolaID := 0
	if id, ok := idPengelola.(int); ok {
		pengelolaID = id
	} else if idStr, ok := idPengelola.(string); ok {
		pengelolaID, _ = strconv.Atoi(idStr)
	}

	fmt.Printf("=== Clearing all import history for pengelola ID: %d ===\n", pengelolaID)

	// Hapus semua import history untuk user ini
	err := repository.DeleteAllImportHistoryByPengelola(db, pengelolaID)
	if err != nil {
		fmt.Printf("❌ Error deleting import history: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menghapus riwayat import",
		})
		return
	}

	fmt.Printf("✓ All import history cleared successfully\n")

	c.JSON(http.StatusOK, gin.H{
		"message": "Semua riwayat import berhasil dihapus",
	})
}

// BendaharaTransaksiDataAnggota menampilkan semua jenis transaksi anggota
func BendaharaTransaksiDataAnggota(c *gin.Context) {
	db := config.GetDB()

	// Get filter parameters
	idAnggota := c.Query("id_anggota")
	tanggalMulai := c.Query("tanggal_mulai")
	tanggalAkhir := c.Query("tanggal_akhir")

	// Get LogoPath from context
	logoPath, exists := c.Get("LogoPath")
	if !exists {
		logoPath = "/static/images/placeholder.png"
	}

	// Fetch all members for filter dropdown
	var anggotas []models.Anggota
	queryAnggota := "SELECT id_anggota, nama_anggota, gaji_bulanan FROM anggota WHERE status = 'aktif' ORDER BY nama_anggota"
	rows, err := db.Query(queryAnggota)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var anggota models.Anggota
			if err := rows.Scan(&anggota.IDAnggota, &anggota.NamaAnggota, &anggota.GajiBulanan); err == nil {
				anggotas = append(anggotas, anggota)
			}
		}
	}

	// Ambil data simpanan wajib untuk semua anggota
	simpananWajib, err := repository.GetSimpananWajibAllAnggota()
	if err != nil {
		simpananWajib = make(map[string]float64) // Default ke map kosong jika error
	}

	// Ambil data pemotongan bulan ini untuk semua anggota
	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64) // Default ke map kosong jika error
	}

	// Hitung sisa gaji untuk setiap anggota
	sisaGaji := make(map[string]int)
	for _, anggota := range anggotas {
		potongan := int(potonganBulanIni[anggota.IDAnggota])
		sisaGaji[anggota.IDAnggota] = anggota.GajiBulanan - potongan
	}

	// Build query conditions
	whereConditions := []string{}
	queryParams := []interface{}{}
	paramIndex := 1

	if idAnggota != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("id_anggota = $%d", paramIndex))
		queryParams = append(queryParams, idAnggota)
		paramIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = " WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Fetch Simpanan data
	var simpanans []models.Detail
	querySimpanan := `
		SELECT d.id_detail, d.id_anggota, a.nama_anggota, d.id_simpanan, s.jenis_simpanan,
			   d.tgl_transaksi, d.jumlah_simpanan, d.total_simpanan, d.status
		FROM detail d
		JOIN anggota a ON d.id_anggota = a.id_anggota
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan` + whereClause

	if tanggalMulai != "" {
		if len(whereConditions) > 0 {
			querySimpanan += fmt.Sprintf(" AND d.tgl_transaksi >= $%d", paramIndex)
		} else {
			querySimpanan += fmt.Sprintf(" WHERE d.tgl_transaksi >= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalMulai)
		paramIndex++
	}
	if tanggalAkhir != "" {
		if len(whereConditions) > 0 || tanggalMulai != "" {
			querySimpanan += fmt.Sprintf(" AND d.tgl_transaksi <= $%d", paramIndex)
		} else {
			querySimpanan += fmt.Sprintf(" WHERE d.tgl_transaksi <= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalAkhir+" 23:59:59")
		paramIndex++
	}
	querySimpanan += " ORDER BY d.tgl_transaksi DESC"

	rowsSimpanan, err := db.Query(querySimpanan, queryParams...)
	if err == nil {
		defer rowsSimpanan.Close()
		for rowsSimpanan.Next() {
			var detail models.Detail
			err := rowsSimpanan.Scan(
				&detail.IDDetail, &detail.IDAnggota, &detail.NamaAnggota,
				&detail.IDSimpanan, &detail.Simpanan.JenisSimpanan,
				&detail.TglTransaksi, &detail.JumlahSimpanan, &detail.TotalSimpanan,
				&detail.Status,
			)
			if err == nil {
				simpanans = append(simpanans, detail)
			}
		}
	}

	// Reset query params for pinjaman
	queryParams = []interface{}{}
	paramIndex = 1
	if idAnggota != "" {
		queryParams = append(queryParams, idAnggota)
		paramIndex++
	}

	// Fetch Pinjaman data
	var pinjamans []models.Pinjaman
	queryPinjaman := `
		SELECT p.id_pinjaman, p.id_anggota, a.nama_anggota, p.tgl_pinjaman,
			   p.jumlah_pinjaman, p.jangka_waktu, p.bunga, p.metode_pencairan,
			   p.nomor_rekening, p.nama_bank, p.status
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota` + whereClause

	if tanggalMulai != "" {
		if len(whereConditions) > 0 {
			queryPinjaman += fmt.Sprintf(" AND p.tgl_pinjaman >= $%d", paramIndex)
		} else {
			queryPinjaman += fmt.Sprintf(" WHERE p.tgl_pinjaman >= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalMulai)
		paramIndex++
	}
	if tanggalAkhir != "" {
		if len(whereConditions) > 0 || tanggalMulai != "" {
			queryPinjaman += fmt.Sprintf(" AND p.tgl_pinjaman <= $%d", paramIndex)
		} else {
			queryPinjaman += fmt.Sprintf(" WHERE p.tgl_pinjaman <= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalAkhir+" 23:59:59")
		paramIndex++
	}
	queryPinjaman += " ORDER BY p.tgl_pinjaman DESC"

	rowsPinjaman, err := db.Query(queryPinjaman, queryParams...)
	if err == nil {
		defer rowsPinjaman.Close()
		for rowsPinjaman.Next() {
			var pinjaman models.Pinjaman
			err := rowsPinjaman.Scan(
				&pinjaman.IDPinjaman, &pinjaman.IDAnggota, &pinjaman.NamaAnggota,
				&pinjaman.TglPinjaman, &pinjaman.JumlahPinjaman, &pinjaman.JangkaWaktu,
				&pinjaman.Bunga, &pinjaman.MetodePencairan, &pinjaman.NomorRekening,
				&pinjaman.NamaBank, &pinjaman.Status,
			)
			if err == nil {
				pinjamans = append(pinjamans, pinjaman)
			}
		}
	}

	// Reset query params for angsuran
	queryParams = []interface{}{}
	paramIndex = 1
	if idAnggota != "" {
		queryParams = append(queryParams, idAnggota)
		paramIndex++
	}

	// Fetch Angsuran data
	var angsurans []models.Angsuran
	queryAngsuran := `
		SELECT ang.id_angsuran, ang.id_pinjaman, ang.id_anggota, a.nama_anggota,
			   ang.tgl_bayar, ang.sisa_pinjaman, ang.status
		FROM angsuran ang
		JOIN anggota a ON ang.id_anggota = a.id_anggota` + whereClause

	if tanggalMulai != "" {
		if len(whereConditions) > 0 {
			queryAngsuran += fmt.Sprintf(" AND ang.tgl_bayar >= $%d", paramIndex)
		} else {
			queryAngsuran += fmt.Sprintf(" WHERE ang.tgl_bayar >= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalMulai)
		paramIndex++
	}
	if tanggalAkhir != "" {
		if len(whereConditions) > 0 || tanggalMulai != "" {
			queryAngsuran += fmt.Sprintf(" AND ang.tgl_bayar <= $%d", paramIndex)
		} else {
			queryAngsuran += fmt.Sprintf(" WHERE ang.tgl_bayar <= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalAkhir+" 23:59:59")
		paramIndex++
	}
	queryAngsuran += " ORDER BY ang.tgl_bayar DESC"

	rowsAngsuran, err := db.Query(queryAngsuran, queryParams...)
	if err == nil {
		defer rowsAngsuran.Close()
		for rowsAngsuran.Next() {
			var angsuran models.Angsuran
			err := rowsAngsuran.Scan(
				&angsuran.IDAngsuran, &angsuran.IDPinjaman, &angsuran.IDAnggota,
				&angsuran.NamaAnggota, &angsuran.TglBayar, &angsuran.SisaPinjaman,
				&angsuran.Status,
			)
			if err == nil {
				angsurans = append(angsurans, angsuran)
			}
		}
	}

	// Reset query params for pengambilan
	queryParams = []interface{}{}
	paramIndex = 1
	if idAnggota != "" {
		queryParams = append(queryParams, idAnggota)
		paramIndex++
	}

	// Fetch Pengambilan Simpanan data
	var pengambilans []models.PengambilanSimpanan
	queryPengambilan := `
		SELECT ps.id_pengambilan, ps.id_anggota, a.nama_anggota, s.jenis_simpanan,
			   ps.tgl_pengajuan, ps.jumlah, ps.alasan, ps.status
		FROM pengambilan_simpanan ps
		JOIN anggota a ON ps.id_anggota = a.id_anggota
		JOIN simpanan s ON ps.id_simpanan = s.id_simpanan` + whereClause

	if tanggalMulai != "" {
		if len(whereConditions) > 0 {
			queryPengambilan += fmt.Sprintf(" AND ps.tgl_pengajuan >= $%d", paramIndex)
		} else {
			queryPengambilan += fmt.Sprintf(" WHERE ps.tgl_pengajuan >= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalMulai)
		paramIndex++
	}
	if tanggalAkhir != "" {
		if len(whereConditions) > 0 || tanggalMulai != "" {
			queryPengambilan += fmt.Sprintf(" AND ps.tgl_pengajuan <= $%d", paramIndex)
		} else {
			queryPengambilan += fmt.Sprintf(" WHERE ps.tgl_pengajuan <= $%d", paramIndex)
		}
		queryParams = append(queryParams, tanggalAkhir+" 23:59:59")
		paramIndex++
	}
	queryPengambilan += " ORDER BY ps.tgl_pengajuan DESC"

	rowsPengambilan, err := db.Query(queryPengambilan, queryParams...)
	if err == nil {
		defer rowsPengambilan.Close()
		for rowsPengambilan.Next() {
			var pengambilan models.PengambilanSimpanan
			err := rowsPengambilan.Scan(
				&pengambilan.IDPengambilan, &pengambilan.IDAnggota, &pengambilan.NamaAnggota,
				&pengambilan.JenisSimpanan, &pengambilan.TglPengajuan, &pengambilan.Jumlah,
				&pengambilan.Alasan, &pengambilan.Status,
			)
			if err == nil {
				pengambilans = append(pengambilans, pengambilan)
			}
		}
	}

	// Create combined transactions list
	type Transaction struct {
		ID          int
		IDAnggota   string
		NamaAnggota string
		Jenis       string
		Tanggal     time.Time
		Jumlah      float64
		Status      string
	}

	var allTransactions []Transaction

	// Add Simpanan to all transactions
	for _, s := range simpanans {
		allTransactions = append(allTransactions, Transaction{
			ID:          s.IDDetail,
			IDAnggota:   s.IDAnggota,
			NamaAnggota: s.NamaAnggota,
			Jenis:       "simpanan",
			Tanggal:     s.TglTransaksi,
			Jumlah:      s.JumlahSimpanan,
			Status:      s.Status,
		})
	}

	// Add Pinjaman to all transactions
	for _, p := range pinjamans {
		allTransactions = append(allTransactions, Transaction{
			ID:          p.IDPinjaman,
			IDAnggota:   p.IDAnggota,
			NamaAnggota: p.NamaAnggota,
			Jenis:       "pinjaman",
			Tanggal:     p.TglPinjaman,
			Jumlah:      p.JumlahPinjaman,
			Status:      p.Status,
		})
	}

	// Add Angsuran to all transactions
	for _, a := range angsurans {
		allTransactions = append(allTransactions, Transaction{
			ID:          a.IDAngsuran,
			IDAnggota:   a.IDAnggota,
			NamaAnggota: a.NamaAnggota,
			Jenis:       "angsuran",
			Tanggal:     a.TglBayar,
			Jumlah:      a.SisaPinjaman,
			Status:      a.Status,
		})
	}

	// Add Pengambilan to all transactions
	for _, p := range pengambilans {
		allTransactions = append(allTransactions, Transaction{
			ID:          p.IDPengambilan,
			IDAnggota:   p.IDAnggota,
			NamaAnggota: p.NamaAnggota,
			Jenis:       "pengambilan",
			Tanggal:     p.TglPengajuan,
			Jumlah:      p.Jumlah,
			Status:      p.Status,
		})
	}

	c.HTML(http.StatusOK, "bendahara_transaksi_data_anggota.html", gin.H{
		"ActivePage":       "transaksi-anggota",
		"LogoPath":         logoPath,
		"Title":            "Transaksi Data Anggota",
		"Anggotas":         anggotas,
		"Simpanans":        simpanans,
		"Pinjamans":        pinjamans,
		"Angsurans":        angsurans,
		"Pengambilans":     pengambilans,
		"AllTransactions":  allTransactions,
		"SimpananWajib":    simpananWajib,
		"PotonganBulanIni": potonganBulanIni,
		"SisaGaji":         sisaGaji,
	})
}

// BendaharaSettingSimpananWajib menampilkan halaman setting pemotongan simpanan wajib
func BendaharaSettingSimpananWajib(c *gin.Context) {
	logoPath, _ := c.Get("LogoPath")

	config, err := repository.GetKonfigurasiSimpananWajib()
	if err != nil {
		config = map[string]interface{}{
			"TanggalPotong":    1,
			"PersentasePotong": 5.0,
			"NominalTetap":     0.0,
			"TipePemotongan":   "persentase",
			"StatusAktif":      false,
		}
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage": "setting_simpanan_wajib",
		"LogoPath":   logoPath,
		"Title":      "Setting Simpanan Wajib",
		"Config":     config,
	})
}

// BendaharaSaveSettingSimpananWajib menyimpan konfigurasi pemotongan simpanan wajib
func BendaharaSaveSettingSimpananWajib(c *gin.Context) {
	logoPath, _ := c.Get("LogoPath")

	tanggalPotong, _ := strconv.Atoi(c.PostForm("TanggalPotong"))
	persentasePotong, _ := strconv.ParseFloat(c.PostForm("PersentasePotong"), 64)
	nominalTetap, _ := strconv.ParseFloat(c.PostForm("NominalTetap"), 64)
	tipePemotongan := c.PostForm("TipePemotongan")
	statusAktif := c.PostForm("StatusAktif") == "on"

	err := repository.SaveKonfigurasiSimpananWajib(tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif)

	config, _ := repository.GetKonfigurasiSimpananWajib()
	if config == nil {
		config = map[string]interface{}{
			"TanggalPotong":    tanggalPotong,
			"PersentasePotong": persentasePotong,
			"NominalTetap":     nominalTetap,
			"TipePemotongan":   tipePemotongan,
			"StatusAktif":      statusAktif,
		}
	}

	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_setting_simpanan_wajib.html", gin.H{
			"ActivePage": "setting_simpanan_wajib",
			"LogoPath":   logoPath,
			"Title":      "Setting Simpanan Wajib",
			"Config":     config,
			"error":      "Gagal menyimpan konfigurasi: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage": "setting_simpanan_wajib",
		"LogoPath":   logoPath,
		"Title":      "Setting Simpanan Wajib",
		"Config":     config,
		"success":    "Konfigurasi berhasil disimpan",
	})
}

// BendaharaProsesSimpananWajib melakukan proses pemotongan simpanan wajib manual
func BendaharaProsesSimpananWajib(c *gin.Context) {
	logoPath, _ := c.Get("LogoPath")

	successCount, failedCount, errors := repository.ProsesPemotonganSimpananWajib()

	config, _ := repository.GetKonfigurasiSimpananWajib()
	if config == nil {
		config = map[string]interface{}{
			"TanggalPotong":    1,
			"PersentasePotong": 5.0,
			"NominalTetap":     0.0,
			"TipePemotongan":   "persentase",
			"StatusAktif":      false,
		}
	}

	message := fmt.Sprintf("✓ Proses pemotongan selesai! Berhasil memotong %d anggota", successCount)
	if failedCount > 0 {
		message += fmt.Sprintf(", Gagal: %d anggota", failedCount)
	}

	if len(errors) > 0 {
		message += "<br><small>Error: " + errors[0] + "</small>"
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage": "setting_simpanan_wajib",
		"LogoPath":   logoPath,
		"Title":      "Setting Simpanan Wajib",
		"Config":     config,
		"success":    message,
	})
}
