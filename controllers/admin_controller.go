package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"

)

// Menampilkan dashboard admin dengan daftar calon anggota
func AdminDashboard(c *gin.Context) {
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

	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"PendingMembers": pendingMembers,
		"DashboardKonten": dashboardKonten,
		"ActivePage": "dashboard",
	})
}

// Menampilkan halaman konfirmasi anggota
func AdminKonfirmasi(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage": "konfirmasi",
	})
}

// Mengkonfirmasi keanggotaan
func ConfirmMembership(c *gin.Context) {
	// Ambil id anggota dari URL
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	// Buat kode anggota baru, contoh: KSPWIR-ID
	// ID yang digunakan adalah ID dari primary key yang auto-increment,
	// ini memastikan urutannya benar.
	newMemberCode := fmt.Sprintf("KSPWIR-%d", id)

	// Panggil repository untuk update status dan kode anggota
	err = repository.UpdateAnggotaStatusWithCode(id, "aktif", newMemberCode)
	if err != nil {
		// Handle error, mungkin tampilkan pesan kesalahan
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengkonfirmasi anggota"})
		return
	}

	// Arahkan kembali ke dashboard admin
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// ListHalaman menampilkan daftar semua halaman statis untuk di-edit.
func ListHalaman(c *gin.Context) {
	allHalaman, err := repository.GetAllHalaman()
	if err != nil {
		// Handle error
		c.String(http.StatusInternalServerError, "Gagal mengambil data halaman")
		return
	}
	c.HTML(http.StatusOK, "admin_halaman_list.html", gin.H{
		"AllHalaman": allHalaman,
		"ActivePage": "halaman",
	})
}

// ShowEditHalamanForm menampilkan form untuk mengedit halaman.
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

	c.HTML(http.StatusOK, "admin_halaman_edit.html", gin.H{
		"Halaman": halaman,
		"Konten":  konten,
	})
}

// UpdateHalaman memproses update konten halaman.
func UpdateHalaman(c *gin.Context) {
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
		c.Redirect(http.StatusFound, "/admin/dashboard")
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
	c.Redirect(http.StatusFound, "/admin/halaman")
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
func ListAllAnggota(c *gin.Context) {
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}
	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"Anggotas": anggotas,
		"ActivePage": "anggota",
	})
}

// ViewAnggota menampilkan detail anggota berdasarkan ID
func ViewAnggota(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID tidak valid"})
		return
	}

	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	c.HTML(http.StatusOK, "admin_anggota_view.html", gin.H{
		"Anggota": anggota,
	})
}

// EditAnggota menampilkan form edit anggota
func EditAnggota(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID tidak valid"})
		return
	}

	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	c.HTML(http.StatusOK, "admin_anggota_edit.html", gin.H{
		"Anggota": anggota,
	})
}

// UpdateAnggota memproses update data anggota
func UpdateAnggota(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

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
			no_telepon = $5, provinsi = $6, jenis_kelamin = $7, status_anggota = $8, fakultas = $9
		WHERE id_anggota = $10`
	_, err = db.Exec(query,
		anggota.NamaAnggota, anggota.Username, anggota.TglLahir, anggota.NikKTP,
		anggota.NoTelepon, anggota.Provinsi, anggota.JenisKelamin, anggota.StatusAnggota, anggota.Fakultas, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui anggota"})
		return
	}

	c.Redirect(http.StatusFound, "/admin/anggota/"+idStr)
}

// DeleteAnggota menghapus anggota
func DeleteAnggota(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	err = repository.DeleteAnggota(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus anggota"})
		return
	}

	c.Redirect(http.StatusFound, "/admin/anggota")
}

// AdminTransaksi menampilkan halaman transaksi admin
func AdminTransaksi(c *gin.Context) {
	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"ActivePage": "transaksi",
	})
}

// AdminLaporan menampilkan halaman laporan keuangan admin
func AdminLaporan(c *gin.Context) {
	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"ActivePage": "laporan",
	})
}

// AdminTentang menampilkan halaman tentang kami admin
func AdminTentang(c *gin.Context) {
	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"ActivePage": "tentang",
	})
}

// AdminPengaturan menampilkan halaman pengaturan admin
func AdminPengaturan(c *gin.Context) {
	// Ambil ID admin dari session
	session := sessions.Default(c)
	adminID := session.Get("user_id")
	if adminID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data admin
	admin, err := repository.GetPengelolaByID(adminID.(int))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "layout_admin.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data admin: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "layout_admin.html", gin.H{
		"ActivePage": "pengaturan",
		"Admin":      admin,
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
		admin, err := repository.GetPengelolaByID(adminID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data admin"})
			return
		}
		passwordToUpdate = admin.Password
	}

	// Update username dan password
	err := repository.UpdatePengelolaUsernamePassword(adminID.(int), request.Username, passwordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}
