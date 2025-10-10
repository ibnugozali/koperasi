package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository" // Ganti dengan path repository Anda
)

// Menampilkan dashboard admin dengan daftar calon anggota
func AdminDashboard(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"PendingMembers": pendingMembers,
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
	err = repository.UpdateAnggotaStatus(id, "aktif", newMemberCode)
	if err != nil {
		// Handle error, mungkin tampilkan pesan kesalahan
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengkonfirmasi anggota"})
		return
	}

	// Arahkan kembali ke dashboard admin
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// Di file: controllers/admin_controller.go

// ... (fungsi-fungsi admin lainnya)

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
	c.HTML(http.StatusOK, "admin_halaman_edit.html", gin.H{
		"Halaman": halaman,
	})
}

// UpdateHalaman memproses update konten halaman.
func UpdateHalaman(c *gin.Context) {
	var halaman models.Halaman
	if err := c.ShouldBind(&halaman); err != nil {
		c.String(http.StatusBadRequest, "Data tidak valid")
		return
	}
	halaman.Slug = c.Param("slug") // Ambil slug dari URL

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
