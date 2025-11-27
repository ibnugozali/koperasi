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
// BendaharaKonfirmasi menampilkan halaman konfirmasi anggota dengan redirect ke konfirmasi-transaksi
func BendaharaKonfirmasi(c *gin.Context) {
	// Redirect ke halaman konfirmasi transaksi
	c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi")
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
	c.HTML(http.StatusOK, "bendahara_anggota_konfirmasi.html", gin.H{
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

	// Reuse admin anggota view template for bendahara
	c.HTML(http.StatusOK, "admin_anggota_view.html", gin.H{
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

	c.HTML(http.StatusOK, "bendahara_konfirmasi_transaksi.html", gin.H{
		"PendingSimpanan": numberedSimpanan,
		"PendingPinjaman": numberedPinjamans,
		"PendingAngsuran": numberedAngsurans,
		"ActivePage":      "konfirmasi-transaksi",
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
		       COALESCE(metode_pencairan, '') as metode_pencairan, COALESCE(nomor_rekening, '') as nomor_rekening
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
