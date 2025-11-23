package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// AnggotaDashboard menampilkan halaman utama untuk anggota.
func AnggotaDashboard(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		// Jika user_id tidak ada di session, redirect ke login
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data lengkap anggota dari repository
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		// Handle jika data anggota tidak ditemukan
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Cek angsuran terlambat dan kirim pesan jika ada
	terlambats, err := repository.GetAngsuranTerlambat()
	if err == nil && len(terlambats) > 0 {
		for _, t := range terlambats {
			if t["nama_anggota"] == anggota.NamaAnggota {
				// Kirim pesan notifikasi (misal, tambahkan ke pesan)
				// Untuk sementara, tambahkan ke session atau tampilkan di dashboard
				c.Set("Notifikasi", "Anda memiliki angsuran yang terlambat. Silakan bayar segera.")
				break
			}
		}
	}

	// Ambil konten dashboard dari halaman
	halaman, err := repository.GetHalamanBySlug("dashboard_anggota")
	if err != nil {
		// Handle error, perhaps use default content
		halaman = models.Halaman{
			Konten: `{"teks": "Selamat datang di dashboard anggota.", "gambar": "/static/images/placeholder.png"}`,
		}
	}

	// Parse JSON konten
	var kontenParsed map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &kontenParsed); err != nil {
		// If parsing fails, use default
		kontenParsed = map[string]interface{}{
			"teks":   "Selamat datang di dashboard anggota.",
			"gambar": "/static/images/placeholder.png",
		}
	}

	// Render halaman dashboard dan kirim data anggota dan konten ke sana
	c.HTML(http.StatusOK, "anggota_dashboard.html", gin.H{
		"Anggota":      anggota,
		"KontenParsed": kontenParsed,
	})
}

// AnggotaProfil menampilkan halaman profil untuk anggota.
func AnggotaProfil(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		// Jika user_id tidak ada di session, redirect ke login
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data lengkap anggota dari repository
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		// Handle jika data anggota tidak ditemukan
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil data saldo
	totalSimpanan, totalPinjaman, _, err := repository.GetSaldoAnggota(userID)
	if err != nil {
		// Jika gagal ambil saldo, tetap tampilkan halaman dengan saldo 0
		totalSimpanan, totalPinjaman = 0, 0
	}

	// Render halaman profil dan kirim data anggota dan saldo ke sana
	c.HTML(http.StatusOK, "anggota_profil.html", gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"TotalPinjaman": totalPinjaman,
	})
}

// AnggotaPesan handles the /anggota/pesan route.
func AnggotaPesan(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		// Jika user_id tidak ada di session, redirect ke login
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data lengkap anggota dari repository
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		// Handle jika data anggota tidak ditemukan
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil daftar pesan untuk anggota
	pesans, err := repository.GetPesanByAnggotaID(userID)
	if err != nil {
		// Jika gagal ambil pesan, tetap tampilkan halaman dengan pesan kosong
		pesans = []models.Pesan{}
	}

	// Render halaman pesan dengan daftar pesan
	c.HTML(http.StatusOK, "anggota_pesan.html", gin.H{
		"Title":   "Pesan Saya",
		"Anggota": anggota,
		"Pesans":  pesans,
	})
}

// GantiPassword handles the /anggota/ganti-password route.
func GantiPassword(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		// Jika user_id tidak ada di session, redirect ke login
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data lengkap anggota dari repository
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		// Handle jika data anggota tidak ditemukan
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Render halaman ganti password dengan form
	c.HTML(http.StatusOK, "anggota_ganti_password.html", gin.H{
		"Title":   "Ganti Password",
		"Anggota": anggota,
	})
}

// GantiPasswordPost handles the POST request for changing password and username.
func GantiPasswordPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data anggota
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Gagal mengambil data pengguna.",
		})
		return
	}

	// Ambil input dari form
	oldPassword := c.PostForm("old_password")
	newUsername := c.PostForm("new_username")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	// Validasi: pastikan semua field diisi
	if oldPassword == "" || newPassword == "" || confirmPassword == "" {
		c.HTML(http.StatusBadRequest, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Semua field harus diisi.",
		})
		return
	}

	// Validasi: password baru harus sama dengan konfirmasi
	if newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Password baru dan konfirmasi password tidak cocok.",
		})
		return
	}

	// Validasi: password lama harus cocok
	err = bcrypt.CompareHashAndPassword([]byte(anggota.Password), []byte(oldPassword))
	if err != nil {
		c.HTML(http.StatusUnauthorized, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Password lama salah.",
		})
		return
	}

	// Jika username baru kosong, gunakan username lama
	if newUsername == "" {
		newUsername = anggota.Username
	}

	// Hash password baru
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Gagal mengenkripsi password baru.",
		})
		return
	}

	// Update username dan password di database
	err = repository.UpdateAnggotaUsernamePassword(userID, newUsername, string(hashedPassword))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Gagal memperbarui username dan password.",
		})
		return
	}

	// Berhasil, redirect ke dashboard dengan pesan sukses
	c.HTML(http.StatusOK, "anggota_ganti_password.html", gin.H{
		"Title":   "Ganti Password",
		"Anggota": anggota,
		"Success": "Username dan password berhasil diubah.",
	})
}

// KeluarKoperasi handles the POST request for exiting the cooperative.
func KeluarKoperasi(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data anggota untuk error handling
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Update status anggota ke 'keluar'
	err = repository.UpdateAnggotaStatus(userID, "keluar")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_profil.html", gin.H{
			"Anggota": anggota,
			"Error":   "Gagal keluar dari koperasi.",
		})
		return
	}

	// Clear session dan redirect ke login
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login?message=Anda telah keluar dari koperasi.")
}

// AjukanPinjaman menampilkan form pengajuan pinjaman
func AjukanPinjaman(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Hitung total simpanan untuk menampilkan limit
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(userID)
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
	case "01", "02": // Dosen (01) atau Staff (02)
		jenisAnggota = "Dosen/Staff"
		limitPinjaman = 0 // Akan dihitung berdasarkan gaji di frontend
	default:
		jenisAnggota = "Tidak Diketahui"
		limitPinjaman = 0
	}

	c.HTML(http.StatusOK, "anggota_ajukan_pinjaman.html", gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"LimitPinjaman": limitPinjaman,
		"JenisAnggota":  jenisAnggota,
	})
}

// AjukanPinjamanPost memproses pengajuan pinjaman
func AjukanPinjamanPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data anggota untuk error handling
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Convert userID to int for pinjaman.IDAnggota
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	var pinjaman models.Pinjaman
	if err := c.ShouldBind(&pinjaman); err != nil {
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Data tidak valid. Pastikan semua field diisi dengan benar.",
		})
		return
	}

	// Validasi minimal pinjaman dihapus

	// Validasi jangka waktu
	if pinjaman.JangkaWaktu < 6 || pinjaman.JangkaWaktu > 36 {
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Jangka waktu harus antara 6-36 bulan.",
		})
		return
	}

	// Validasi bunga
	if pinjaman.Bunga < 0 || pinjaman.Bunga > 20 {
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Bunga harus antara 0-20%.",
		})
		return
	}

	// Hitung total simpanan
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Gagal menghitung total simpanan.",
		})
		return
	}

	// Hitung limit pinjaman berdasarkan jenis anggota
	var limitPinjaman float64
	var jenisAnggota string

	switch anggota.UnitKerja {
	case "03": // Mahasiswa
		jenisAnggota = "Mahasiswa"
		// Mahasiswa hanya bisa pinjam jika memiliki simpanan
		if totalSimpanan <= 0 {
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
				"Anggota": anggota,
				"Error":   "Mahasiswa tidak dapat mengajukan pinjaman karena belum memiliki simpanan. Silakan lakukan simpanan terlebih dahulu.",
			})
			return
		}
		limitPinjaman = 5 * totalSimpanan // 5x total simpanan
	case "01", "02": // Dosen (01) atau Staff (02)
		jenisAnggota = "Dosen/Staff"
		// Ambil gaji dari form
		gajiStr := c.PostForm("gaji_bulanan")
		if gajiStr == "" {
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
				"Anggota": anggota,
				"Error":   "Gaji bulanan wajib diisi untuk dosen/staff.",
			})
			return
		}
		// Parse gaji (asumsi dalam ribuan atau jutaan, sesuaikan dengan input)
		var gaji float64
		if _, err := fmt.Sscanf(gajiStr, "%f", &gaji); err != nil {
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
				"Anggota": anggota,
				"Error":   "Format gaji tidak valid.",
			})
			return
		}
		limitPinjaman = (0.4 * gaji * float64(pinjaman.JangkaWaktu)) + totalSimpanan // (40% * gaji * tenor) + total simpanan
	default:
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Jenis anggota tidak valid.",
		})
		return
	}

	// Validasi limit pinjaman
	if pinjaman.JumlahPinjaman > limitPinjaman {
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   fmt.Sprintf("Jumlah pinjaman melebihi limit maksimal Rp %.0f untuk %s.", limitPinjaman, jenisAnggota),
		})
		return
	}

	// Set bunga flat 2%
	pinjaman.Bunga = 2.0

	pinjaman.IDAnggota = userIDInt
	pinjaman.TglPinjaman = time.Now() // Set tanggal pengajuan otomatis
	pinjaman.Status = "pending"       // Status pending untuk konfirmasi bendahara

	err = repository.CreatePinjaman(pinjaman)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", gin.H{
			"Anggota": anggota,
			"Error":   "Gagal mengajukan pinjaman. Silakan coba lagi.",
		})
		return
	}

	// Berhasil, redirect ke riwayat untuk melihat data pinjaman
	c.Redirect(http.StatusFound, "/anggota/riwayat")
}

// AnggotaSimpanan menampilkan halaman simpanan untuk anggota.
func AnggotaSimpanan(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	c.HTML(http.StatusOK, "anggota_simpanan.html", gin.H{
		"Judul":   "Simpanan",
		"Anggota": anggota,
		"Now":     time.Now(),
	})
}

// AnggotaSimpananPost memproses pengajuan simpanan
func AnggotaSimpananPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data anggota untuk error handling
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil input dari form untuk berbagai jenis simpanan
	simpananWajibStr := c.PostForm("simpanan_wajib")
	simpananSukarelaStr := c.PostForm("simpanan_sukarela")
	simpananHariRayaStr := c.PostForm("simpanan_hari_raya")

	// Parse jumlah untuk setiap jenis simpanan
	var simpananWajib, simpananSukarela, simpananHariRaya float64

	if simpananWajibStr != "" {
		if _, err := fmt.Sscanf(simpananWajibStr, "%f", &simpananWajib); err != nil {
			c.HTML(http.StatusBadRequest, "anggota_simpanan.html", gin.H{
				"Judul":   "Simpanan",
				"Anggota": anggota,
				"Error":   "Format jumlah simpanan wajib tidak valid.",
			})
			return
		}
	}

	if simpananSukarelaStr != "" {
		if _, err := fmt.Sscanf(simpananSukarelaStr, "%f", &simpananSukarela); err != nil {
			c.HTML(http.StatusBadRequest, "anggota_simpanan.html", gin.H{
				"Judul":   "Simpanan",
				"Anggota": anggota,
				"Error":   "Format jumlah simpanan sukarela tidak valid.",
			})
			return
		}
	}

	if simpananHariRayaStr != "" {
		if _, err := fmt.Sscanf(simpananHariRayaStr, "%f", &simpananHariRaya); err != nil {
			c.HTML(http.StatusBadRequest, "anggota_simpanan.html", gin.H{
				"Judul":   "Simpanan",
				"Anggota": anggota,
				"Error":   "Format jumlah simpanan hari raya tidak valid.",
			})
			return
		}
	}

	// Validasi: setidaknya satu jenis simpanan harus memiliki jumlah positif
	if simpananWajib <= 0 && simpananSukarela <= 0 && simpananHariRaya <= 0 {
		c.HTML(http.StatusBadRequest, "anggota_simpanan.html", gin.H{
			"Judul":   "Simpanan",
			"Anggota": anggota,
			"Error":   "Setidaknya satu jenis simpanan harus memiliki jumlah lebih dari 0.",
		})
		return
	}

	// Handle file upload
	file, err := c.FormFile("bukti")
	if err != nil {
		c.HTML(http.StatusBadRequest, "anggota_simpanan.html", gin.H{
			"Judul":   "Simpanan",
			"Anggota": anggota,
			"Error":   "Bukti pembayaran wajib diupload.",
		})
		return
	}

	// Save the uploaded file
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	dst := "./static/uploads/" + filename
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_simpanan.html", gin.H{
			"Judul":   "Simpanan",
			"Anggota": anggota,
			"Error":   "Gagal menyimpan file bukti pembayaran.",
		})
		return
	}

	// Set tanggal pengajuan otomatis ke waktu sekarang
	tanggalPengajuan := time.Now()

	// Simpan setiap jenis simpanan yang memiliki jumlah positif
	simpananTypes := map[string]struct {
		jumlah float64
		id     int
	}{
		"wajib":     {simpananWajib, 2},
		"sukarela":  {simpananSukarela, 3},
		"hari_raya": {simpananHariRaya, 4},
	}

	for jenis, data := range simpananTypes {
		if data.jumlah > 0 {
			detail := models.Detail{
				IDAnggota:      userID,
				IDSimpanan:     data.id,
				IDPengelola:    1, // Default pengelola (bisa disesuaikan)
				TglTransaksi:   tanggalPengajuan,
				JumlahSimpanan: data.jumlah,
				TotalSimpanan:  data.jumlah, // Untuk sementara, total = jumlah (bisa dihitung ulang)
			}

			// Simpan ke database
			err = repository.CreateSimpanan(detail)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "anggota_simpanan.html", gin.H{
					"Judul":   "Simpanan",
					"Anggota": anggota,
					"Error":   fmt.Sprintf("Gagal menyimpan simpanan %s. Silakan coba lagi.", jenis),
				})
				return
			}
		}
	}

	// Berhasil, redirect ke riwayat
	c.Redirect(http.StatusFound, "/anggota/riwayat")
}

// AnggotaAngsuran menampilkan halaman angsuran untuk anggota.
func AnggotaAngsuran(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil pinjaman aktif anggota
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}
	pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userIDInt)
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	// Hitung jumlah pinjaman dan sisa pinjaman
	var jumlahPinjaman float64
	var sisaPinjaman float64
	var angsuranKe int

	if len(pinjamans) > 0 {
		pinjaman := pinjamans[0] // Ambil pinjaman pertama yang aktif
		jumlahPinjaman = pinjaman.JumlahPinjaman

		// Hitung total angsuran yang sudah dibayar
		angsurans, err := repository.GetAngsuranByPinjamanID(pinjaman.IDPinjaman)
		if err == nil {
			var totalAngsuran float64
			for _, ang := range angsurans {
				totalAngsuran += ang.SisaPinjaman
			}
			sisaPinjaman = jumlahPinjaman - totalAngsuran
			angsuranKe = len(angsurans) + 1
		} else {
			sisaPinjaman = jumlahPinjaman
			angsuranKe = 1
		}
	}

	c.HTML(http.StatusOK, "anggota_angsuran.html", gin.H{
		"Judul":          "Angsuran",
		"Anggota":        anggota,
		"JumlahPinjaman": jumlahPinjaman,
		"SisaPinjaman":   sisaPinjaman,
		"AngsuranKe":     angsuranKe,
		"Pinjamans":      pinjamans,
	})
}

// AnggotaAngsuranPost memproses pembayaran angsuran
func AnggotaAngsuranPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data anggota untuk error handling
	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Gagal mengambil data pengguna.",
		})
		return
	}

	// Ambil input dari form
	jumlahAngsuranStr := c.PostForm("jumlah_angsuran")
	tanggalPembayaranStr := c.PostForm("tanggal_pembayaran")
	metodePembayaran := c.PostForm("metode_pembayaran")

	// Validasi input
	if jumlahAngsuranStr == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Jumlah angsuran wajib diisi.",
		})
		return
	}

	if tanggalPembayaranStr == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Tanggal pembayaran wajib diisi.",
		})
		return
	}

	if metodePembayaran == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Metode pembayaran wajib dipilih.",
		})
		return
	}

	// Parse jumlah angsuran
	var jumlahAngsuran float64
	if _, err := fmt.Sscanf(jumlahAngsuranStr, "%f", &jumlahAngsuran); err != nil {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Format jumlah angsuran tidak valid.",
		})
		return
	}

	if jumlahAngsuran <= 0 {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Jumlah angsuran harus lebih dari 0.",
		})
		return
	}

	// Parse tanggal pembayaran
	tanggalPembayaran, err := time.Parse("2006-01-02", tanggalPembayaranStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Format tanggal pembayaran tidak valid.",
		})
		return
	}

	// Handle file upload
	file, err := c.FormFile("bukti")
	if err != nil {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Bukti pembayaran wajib diupload.",
		})
		return
	}

	// Save the uploaded file
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	dst := "./static/uploads/" + filename
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Gagal menyimpan file bukti pembayaran.",
		})
		return
	}

	// Ambil ID pinjaman dari form (jika ada) atau gunakan pinjaman aktif pertama
	idPinjamanStr := c.PostForm("id_pinjaman")
	var idPinjaman int
	if idPinjamanStr != "" {
		if parsedID, err := strconv.Atoi(idPinjamanStr); err == nil {
			idPinjaman = parsedID
		} else {
			c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
				"Judul":   "Angsuran",
				"Anggota": anggota,
				"Error":   "ID pinjaman tidak valid.",
			})
			return
		}
	} else {
		// Jika tidak ada ID pinjaman di form, ambil pinjaman aktif pertama
		userIDInt, err := strconv.Atoi(userID)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "anggota_angsuran.html", gin.H{
				"Judul":   "Angsuran",
				"Anggota": anggota,
				"Error":   "Gagal mengambil data pengguna.",
			})
			return
		}
		pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userIDInt)
		if err != nil || len(pinjamans) == 0 {
			c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
				"Judul":   "Angsuran",
				"Anggota": anggota,
				"Error":   "Tidak ada pinjaman aktif.",
			})
			return
		}
		idPinjaman = pinjamans[0].IDPinjaman
	}

	// Buat angsuran baru
	angsuran := models.Angsuran{
		IDPinjaman:    idPinjaman,
		IDPengelola:   sql.NullInt64{Int64: 1, Valid: true}, // Default pengelola
		TglBayar:      tanggalPembayaran,
		SisaPinjaman:  jumlahAngsuran, // Untuk sementara, sisa pinjaman = jumlah angsuran
		BuktiAngsuran: filename,       // Simpan nama file sebagai string
		Status:        "valid",        // Status valid
		NamaAnggota:   anggota.NamaAnggota,
	}

	// Simpan ke database
	err = repository.CreateAngsuran(angsuran)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "anggota_angsuran.html", gin.H{
			"Judul":   "Angsuran",
			"Anggota": anggota,
			"Error":   "Gagal menyimpan angsuran. Silakan coba lagi.",
		})
		return
	}

	// Berhasil, tampilkan pesan sukses
	c.HTML(http.StatusOK, "anggota_angsuran.html", gin.H{
		"Judul":   "Angsuran",
		"Anggota": anggota,
		"Success": "Angsuran berhasil dikirim. Silakan tunggu konfirmasi dari admin.",
	})
}

// AnggotaSejarah menampilkan halaman sejarah untuk anggota.
func AnggotaSejarah(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil data dari database
	halaman, err := repository.GetHalamanBySlug("sejarah")
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Parse konten JSON
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
		konten = map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "anggota_sejarah.html", gin.H{
		"Judul":   halaman.Judul,
		"Konten":  konten,
		"Anggota": anggota,
	})
}

// AnggotaVisiMisi menampilkan halaman visi misi untuk anggota.
func AnggotaVisiMisi(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil data dari database
	halaman, err := repository.GetHalamanBySlug("visi-misi")
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Parse konten JSON
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
		konten = map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "anggota_visi_misi.html", gin.H{
		"Judul":   halaman.Judul,
		"Konten":  konten,
		"Anggota": anggota,
	})
}

// AnggotaStruktur menampilkan halaman struktur untuk anggota.
func AnggotaStruktur(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"error": "Gagal mengambil data pengguna."})
		return
	}

	// Ambil data dari database
	halaman, err := repository.GetHalamanBySlug("struktur")
	if err != nil {
		c.String(http.StatusNotFound, "Halaman tidak ditemukan")
		return
	}

	// Parse konten JSON
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halaman.Konten), &konten); err != nil {
		konten = map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "anggota_struktur.html", gin.H{
		"Judul":   halaman.Judul,
		"Konten":  konten,
		"Anggota": anggota,
	})
}
