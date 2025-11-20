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
func BendaharaKonfirmasi(c *gin.Context) {
	// Redirect ke halaman konfirmasi transaksi
	c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi")
}

// Mengkonfirmasi keanggotaan
func BendaharaConfirmMembership(c *gin.Context) {
	// Ambil id anggota dari URL
	idStr := c.Param("id")

	// Buat kode anggota baru, contoh: KSPWIR-ID
	// ID yang digunakan adalah ID dari primary key yang auto-increment,
	// ini memastikan urutannya benar.
	newMemberCode := fmt.Sprintf("KSPWIR-%s", idStr)

	// Panggil repository untuk update status dan kode anggota
	err := repository.UpdateAnggotaStatusWithCode(idStr, "aktif", newMemberCode)
	if err != nil {
		// Handle error, mungkin tampilkan pesan kesalahan
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengkonfirmasi anggota"})
		return
	}

	// Arahkan kembali ke dashboard bendahara
	c.Redirect(http.StatusFound, "/bendahara/dashboard")
}

// ListHalaman menampilkan daftar semua halaman statis untuk di-edit.
func BendaharaListHalaman(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
		return
	}
	c.HTML(http.StatusOK, "bendahara_halaman_list.html", gin.H{
		"AllHalaman": allHalaman,
		"ActivePage": "halaman",
	})
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

	c.HTML(http.StatusOK, "bendahara_halaman_edit.html", gin.H{
		"Halaman": halaman,
		"Konten":  konten,
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
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}
	c.HTML(http.StatusOK, "bendahara_anggota_list_content.html", gin.H{
		"Anggotas":   anggotas,
		"ActivePage": "anggota",
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

	c.HTML(http.StatusOK, "bendahara_anggota_view.html", gin.H{
		"Anggota": anggota,
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

	c.HTML(http.StatusOK, "bendahara_anggota_edit.html", gin.H{
		"Anggota": anggota,
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
	pendingSimpanan := []models.Detail{}

	// Ambil pending pinjaman
	pendingPinjaman, err := repository.GetPendingPinjaman()
	if err != nil {
		pendingPinjaman = []models.Pinjaman{}
	}

	// Ambil pending angsuran
	pendingAngsuran, err := repository.GetAllAngsurans()
	if err != nil {
		pendingAngsuran = []models.Angsuran{}
	}

	c.HTML(http.StatusOK, "bendahara_konfirmasi_transaksi.html", gin.H{
		"PendingSimpanan": pendingSimpanan,
		"PendingPinjaman": pendingPinjaman,
		"PendingAngsuran": pendingAngsuran,
		"ActivePage":      "konfirmasi-transaksi",
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
		// Simpanan tidak perlu update status, langsung return success
		err = nil
	case "pinjaman":
		if action == "confirm" {
			err = repository.UpdatePinjamanStatus(id, "aktif")
		} else {
			err = repository.UpdatePinjamanStatus(id, "rejected")
		}
	case "angsuran":
		if action == "confirm" {
			err = repository.UpdateAngsuranStatus(id, "confirmed")
		} else {
			err = repository.UpdateAngsuranStatus(id, "rejected")
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
