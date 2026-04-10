package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

func isAngsuranTerbayar(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "confirmed" || s == "lunas" || s == "diterima"
}

type pinjamanAngsuranInfo struct {
	Pinjaman    models.Pinjaman
	SisaPinjaman float64
	AngsuranKe   int
}

func getPinjamanAngsuranInfos(pinjamans []models.Pinjaman) ([]pinjamanAngsuranInfo, error) {
	var infos []pinjamanAngsuranInfo
	for i := range pinjamans {
		p := pinjamans[i]
		if strings.ToLower(strings.TrimSpace(p.Status)) != "aktif" {
			continue
		}

		angsurans, err := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
		if err != nil {
			return nil, err
		}

		totalAngsuranTerbayar := 0.0
		for _, a := range angsurans {
			if isAngsuranTerbayar(a.Status) {
				totalAngsuranTerbayar += a.SisaPinjaman
			}
		}

		sisaPinjaman := p.JumlahPinjaman - totalAngsuranTerbayar
		if sisaPinjaman < 0 {
			sisaPinjaman = 0
		}

		angsuranKe := 1
		if len(angsurans) > 0 {
			angsuranKe = len(angsurans) + 1
		}

		infos = append(infos, pinjamanAngsuranInfo{
			Pinjaman:     p,
			SisaPinjaman: sisaPinjaman,
			AngsuranKe:   angsuranKe,
		})
	}
	return infos, nil
}

func getPinjamanPrioritasAngsuran(pinjamans []models.Pinjaman) (*pinjamanAngsuranInfo, error) {
	infos, err := getPinjamanAngsuranInfos(pinjamans)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, nil
	}

	var prioritas *pinjamanAngsuranInfo
	for i := range infos {
		info := &infos[i]
		if info.SisaPinjaman <= 0 {
			continue
		}
		if prioritas == nil || info.Pinjaman.TglPinjaman.Before(prioritas.Pinjaman.TglPinjaman) {
			prioritas = info
		}
	}
	if prioritas != nil {
		return prioritas, nil
	}

	var terbaru *pinjamanAngsuranInfo
	for i := range infos {
		info := &infos[i]
		if terbaru == nil || info.Pinjaman.TglPinjaman.After(terbaru.Pinjaman.TglPinjaman) {
			terbaru = info
		}
	}
	return terbaru, nil
}

func getTotalAngsuranAktif(pinjamans []models.Pinjaman) (float64, float64, error) {
	infos, err := getPinjamanAngsuranInfos(pinjamans)
	if err != nil {
		return 0, 0, err
	}

	var totalPinjaman float64
	var totalSisaPinjaman float64
	for _, info := range infos {
		totalPinjaman += info.Pinjaman.JumlahPinjaman
		totalSisaPinjaman += info.SisaPinjaman
	}
	return totalPinjaman, totalSisaPinjaman, nil
}

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

	// Render halaman dashboard dan kirim data anggota dan konten ke sana
	c.HTML(http.StatusOK, "anggota_dashboard.html", gin.H{
		"Anggota":      anggota,
		"KontenParsed": kontenParsed,
		"CurrentLogo":  latestLogo,
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

	// Ambil detail simpanan per jenis
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
	if err != nil {
		// Jika gagal, buat map kosong
		simpananByJenis = map[string]float64{
			"pokok":      0,
			"wajib":      0,
			"sukarela":   0,
			"hari_raya":  0,
			"umroh_haji": 0,
			"qurban":     0,
		}
	}

	// Hitung Total Simpanan dari semua simpanan yang ada di anggota_profil
	totalSimpanan := simpananByJenis["pokok"] + simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

	// Ambil total pinjaman
	_, totalPinjaman, _, err := repository.GetSaldoAnggota(userID)
	if err != nil {
		totalPinjaman = 0
	}

	// Kurangi total pinjaman dengan total angsuran yang statusnya diterima/confirmed
	angsurans, err := repository.GetRiwayatAngsuranByAnggotaID(userID, "")
	if err == nil {
		var totalAngsuranDiterima float64 = 0
		for _, a := range angsurans {
			if a.Status == "confirmed" || a.Status == "lunas" {
				totalAngsuranDiterima += a.SisaPinjaman
			}
		}
		totalPinjaman -= totalAngsuranDiterima
		if totalPinjaman < 0 {
			totalPinjaman = 0
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

	// Render halaman profil dan kirim data anggota, saldo, dan logo ke sana
	c.HTML(http.StatusOK, "anggota_profil.html", gin.H{
		"Anggota":           anggota,
		"TotalSimpanan":     totalSimpanan,
		"TotalPinjaman":     totalPinjaman,
		"SimpananPokok":     simpananByJenis["pokok"],
		"SimpananWajib":     simpananByJenis["wajib"],
		"SimpananSukarela":  simpananByJenis["sukarela"],
		"SimpananHariRaya":  simpananByJenis["hari_raya"],
		"SimpananUmrohHaji": simpananByJenis["umroh_haji"],
		"SimpananQurban":    simpananByJenis["qurban"],
		"CurrentLogo":       latestLogo,
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

	// Render halaman pesan dengan daftar pesan dan logo dinamis
	c.HTML(http.StatusOK, "anggota_pesan.html", gin.H{
		"Title":       "Pesan Saya",
		"Anggota":     anggota,
		"Pesans":      pesans,
		"CurrentLogo": latestLogo,
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

	// Render halaman ganti password dengan form dan logo dinamis
	c.HTML(http.StatusOK, "anggota_ganti_password.html", gin.H{
		"Title":       "Ganti Password",
		"Anggota":     anggota,
		"CurrentLogo": latestLogo,
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

	// Validasi: old password diperlukan jika akan mengubah password
	if (newPassword != "" || confirmPassword != "") && oldPassword == "" {
		c.HTML(http.StatusBadRequest, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Password lama harus diisi jika akan mengubah password.",
		})
		return
	}

	// Validasi: jika password baru diisi, konfirmasi juga harus diisi
	if newPassword != "" && confirmPassword == "" {
		c.HTML(http.StatusBadRequest, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Konfirmasi password baru harus diisi.",
		})
		return
	}

	// Validasi: jika konfirmasi password diisi, password baru juga harus diisi
	if confirmPassword != "" && newPassword == "" {
		c.HTML(http.StatusBadRequest, "anggota_ganti_password.html", gin.H{
			"Title":   "Ganti Password",
			"Anggota": anggota,
			"Error":   "Password baru harus diisi.",
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

	// Validasi: password lama harus cocok hanya jika mengubah password
	if newPassword != "" || confirmPassword != "" {
		// Cek apakah password tersimpan adalah hash bcrypt atau plain text
		var passwordMatch bool
		if strings.HasPrefix(anggota.Password, "$2a$") {
			// Password tersimpan adalah hash bcrypt
			err = bcrypt.CompareHashAndPassword([]byte(anggota.Password), []byte(oldPassword))
			passwordMatch = err == nil
		} else {
			// Password tersimpan adalah plain text
			passwordMatch = anggota.Password == oldPassword
		}

		if !passwordMatch {
			c.HTML(http.StatusUnauthorized, "anggota_ganti_password.html", gin.H{
				"Title":   "Ganti Password",
				"Anggota": anggota,
				"Error":   "Password lama salah.",
			})
			return
		}
	}

	// Jika username baru kosong, gunakan username lama
	if newUsername == "" {
		newUsername = anggota.Username
	}

	// Tentukan password yang akan disimpan
	var passwordToStore string
	if newPassword != "" {
		// Simpan password dalam bentuk plain text
		passwordToStore = newPassword
	} else {
		// Jika tidak mengubah password, gunakan password lama (yang sudah ada)
		passwordToStore = anggota.Password
	}

	// Update username dan password di database
	err = repository.UpdateAnggotaUsernamePassword(userID, newUsername, passwordToStore)
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

// KeluarKoperasi handles pengajuan keluar dari koperasi
func KeluarKoperasi(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Ambil data dari form
	simpananWajibStr := c.PostForm("simpanan_wajib")
	simpananLainnyaStr := c.PostForm("simpanan_lainnya")
	biayaAdminStr := c.PostForm("biaya_admin")
	alasanKeluar := c.PostForm("alasan_keluar")

	fmt.Printf("[DEBUG] Form data: simpanan_wajib=%s, simpanan_lainnya=%s, biaya_admin=%s, alasan=%s\n",
		simpananWajibStr, simpananLainnyaStr, biayaAdminStr, alasanKeluar)

	// Convert string to float64
	simpananWajib, err := strconv.ParseFloat(simpananWajibStr, 64)
	if err != nil {
		fmt.Printf("[ERROR] Parse simpanan_wajib: %v\n", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format simpanan wajib tidak valid")
		return
	}

	simpananLainnya, err := strconv.ParseFloat(simpananLainnyaStr, 64)
	if err != nil {
		fmt.Printf("[ERROR] Parse simpanan_lainnya: %v\n", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format simpanan lainnya tidak valid")
		return
	}

	biayaAdmin, err := strconv.ParseFloat(biayaAdminStr, 64)
	if err != nil {
		fmt.Printf("[ERROR] Parse biaya_admin: %v\n", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format biaya admin tidak valid")
		return
	}

	// Validasi alasan keluar tidak kosong
	if strings.TrimSpace(alasanKeluar) == "" {
		fmt.Printf("[ERROR] Alasan keluar kosong\n")
		c.Redirect(http.StatusFound, "/anggota/profil?error=Alasan keluar harus diisi")
		return
	}

	// Update status anggota menjadi 'pending_keluar' dan simpan data pengajuan
	db := config.GetDB()

	fmt.Printf("[DEBUG] Executing update query for user %s\n", userID)

	// Buat JSON string untuk data_keluar
	dataKeluarJSON := fmt.Sprintf(`{
		"simpanan_wajib": %.2f,
		"simpanan_lainnya": %.2f,
		"biaya_admin": %.2f,
		"alasan": "%s",
		"tanggal_pengajuan": "%s"
	}`, simpananWajib, simpananLainnya, biayaAdmin,
		strings.ReplaceAll(alasanKeluar, `"`, `\"`),
		time.Now().Format(time.RFC3339))

	// Update status dan simpan data pengembalian simpanan
	updateQuery := `
		UPDATE anggota 
		SET status_anggota = 'pending_keluar',
		    data_keluar = $1::jsonb
		WHERE id_anggota = $2
	`

	result, err := db.Exec(updateQuery, dataKeluarJSON, userID)
	if err != nil {
		fmt.Printf("[ERROR] Gagal update status keluar: %v\n", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Gagal mengajukan keluar dari koperasi: "+err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✓ Anggota %s mengajukan keluar dari koperasi (rows affected: %d)\n", userID, rowsAffected)

	// Redirect dengan pesan sukses
	c.Redirect(http.StatusFound, "/anggota/profil?success=Pengajuan keluar berhasil diajukan. Menunggu persetujuan Ketua.")
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
	case "01", "02": // Dosen (01) atau Tenaga Pendidikan (02)
		jenisAnggota = "Dosen/Tenaga Pendidikan"
		limitPinjaman = 0 // Akan dihitung berdasarkan gaji di frontend
	default:
		jenisAnggota = "Tidak Diketahui"
		limitPinjaman = 0
	}

	// Ambil bunga terkini dari database
	db := config.GetDB()
	var bungaTerkini float64
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&bungaTerkini)
	if err != nil {
		// Jika belum ada pengaturan, gunakan default 2.0
		bungaTerkini = 2.0
	}

	// Cari logo.png jika ada, jika tidak cari logo_ terbaru, jika tidak ada fallback ke placeholder.png
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

	c.HTML(http.StatusOK, "anggota_ajukan_pinjaman.html", gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"JenisAnggota":  jenisAnggota,
		"LimitPinjaman": limitPinjaman,
		"Bunga":         bungaTerkini,
		"CurrentLogo":   latestLogo,
		"GajiBulanan":   anggota.GajiBulanan,
	})
}

// getAjukanPinjamanTemplateData adalah helper function untuk mendapatkan data template yang konsisten
func getAjukanPinjamanTemplateData(userID string, anggota models.Anggota) gin.H {
	// Hitung total simpanan
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
		limitPinjaman = 5 * totalSimpanan
	case "01", "02": // Dosen/Tenaga Pendidikan
		jenisAnggota = "Dosen/Tenaga Pendidikan"
		limitPinjaman = 0 // Akan dihitung berdasarkan gaji di frontend
	default:
		jenisAnggota = "Tidak Diketahui"
		limitPinjaman = 0
	}

	// Ambil bunga terkini dari database
	db := config.GetDB()
	var bungaTerkini float64
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&bungaTerkini)
	if err != nil {
		bungaTerkini = 2.0
	}

	return gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"LimitPinjaman": limitPinjaman,
		"JenisAnggota":  jenisAnggota,
		"Bunga":         bungaTerkini,
		"GajiBulanan":   anggota.GajiBulanan, // Tambahkan gaji bulanan
	}
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

	var pinjaman models.Pinjaman
	bindErr := c.ShouldBind(&pinjaman)

	// Debug print form values and error
	formDebug := make(map[string][]string)
	for k, v := range c.Request.Form {
		formDebug[k] = v
	}

	// Log pinjaman struct after binding
	fmt.Printf("DEBUG: Pinjaman struct after binding: %+v\n", pinjaman)

	// Log important form fields
	jumlahPinjamanStr := c.PostForm("jumlah_pinjaman")
	jangkaWaktuStr := c.PostForm("jangka_waktu")
	bungaStr := c.PostForm("bunga")
	gajiBulananStr := c.PostForm("gaji_bulanan")

	fmt.Printf("DEBUG: Form Inputs - jumlah_pinjaman: %s, jangka_waktu: %s, bunga: %s, gaji_bulanan: %s\n",
		jumlahPinjamanStr, jangkaWaktuStr, bungaStr, gajiBulananStr)

	if bindErr != nil {
		errMsg := fmt.Sprintf("Data tidak valid. Pastikan semua field diisi dengan benar. Error: %v, Form Data: %v", bindErr, formDebug)
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = errMsg
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Validasi minimal pinjaman dihapus

	// Validasi jangka waktu
	if pinjaman.JangkaWaktu < 6 || pinjaman.JangkaWaktu > 36 {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Jangka waktu harus antara 6-36 bulan."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Validasi bunga
	if pinjaman.Bunga < 0 || pinjaman.Bunga > 20 {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Bunga harus antara 0-20%."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Cek apakah anggota sudah punya pinjaman aktif
	pinjamanAktif, err := repository.GetPinjamanAktifByAnggotaID(userID)
	if err != nil {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Gagal memeriksa status pinjaman aktif."
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	if len(pinjamanAktif) > 0 {
		// Ambil pinjaman aktif pertama
		p := pinjamanAktif[0]

		// Ambil semua angsuran
		angsurans, err := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
		if err != nil {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = "Gagal memeriksa status angsuran pinjaman aktif."
			c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
			return
		}

		// Hitung persentase berdasarkan nominal yang sudah dibayar, agar konsisten
		// dengan halaman angsuran (bukan berdasarkan jumlah cicilan/tenor).
		totalAngsuranTerbayar := 0.0
		for _, a := range angsurans {
			if a.Status == "lunas" || a.Status == "confirmed" || a.Status == "diterima" {
				totalAngsuranTerbayar += a.SisaPinjaman
			}
		}
		persentase := 0.0
		if p.JumlahPinjaman > 0 {
			persentase = (totalAngsuranTerbayar / p.JumlahPinjaman) * 100
			if persentase > 100 {
				persentase = 100
			}
		}
		// Syarat: minimal 50% nominal pinjaman harus sudah terbayar
		if persentase < 50 {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = fmt.Sprintf(
				"Anda baru melunasi %.2f%% pinjaman. Minimal harus 50%% untuk mengajukan pinjaman baru.",
				persentase,
			)
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
			return
		}
	}

	// // Cek apakah anggota sudah punya pinjaman aktif
	// pinjamanAktif, err := repository.GetPinjamanAktifByAnggotaID(userID)
	// if err != nil {
	// 	templateData := getAjukanPinjamanTemplateData(userID, anggota)
	// 	templateData["Error"] = "Gagal memeriksa status pinjaman aktif."
	// 	c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
	// 	return
	// }
	// if len(pinjamanAktif) > 0 {
	// 	// Cek apakah pinjaman aktif sudah lunas >= 60% angsuran
	// 	// Ambil pinjaman aktif pertama (asumsi hanya satu aktif)
	// 	p := pinjamanAktif[0]
	// 	angsurans, err := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
	// 	if err != nil {
	// 		templateData := getAjukanPinjamanTemplateData(userID, anggota)
	// 		templateData["Error"] = "Gagal memeriksa status angsuran pinjaman aktif."
	// 		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
	// 		return
	// 	}
	// 	totalAngsuran := p.JangkaWaktu
	// 	angsuranLunas := 0
	// 	for _, a := range angsurans {
	// 		// Anggap status 'lunas' atau 'diterima' sebagai angsuran sah
	// 		if a.Status == "lunas" || a.Status == "diterima" {
	// 			angsuranLunas++
	// 		}
	// 	}
	// 	persentase := float64(angsuranLunas) / float64(totalAngsuran) * 100
	// 	if persentase < 60 {
	// 		templateData := getAjukanPinjamanTemplateData(userID, anggota)
	// 		templateData["Error"] = "Anda hanya dapat mengajukan pinjaman baru jika angsuran pinjaman sebelumnya sudah lunas minimal 60%."
	// 		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
	// 		return
	// 	}
	// }

	// Hitung total simpanan
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(userID)
	if err != nil {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Gagal menghitung total simpanan."
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Get database connection and bunga for limit calculation
	db := config.GetDB()
	var bungaTerkini float64
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'bunga_pinjaman'").Scan(&bungaTerkini)
	if err != nil {
		bungaTerkini = 2.0
	}

	// Hitung limit pinjaman berdasarkan jenis anggota
	var limitPinjaman float64
	var jenisAnggota string

	switch anggota.UnitKerja {
	case "03": // Mahasiswa
		jenisAnggota = "Mahasiswa"
		// Mahasiswa hanya bisa pinjam jika memiliki simpanan
		if totalSimpanan <= 0 {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = "Mahasiswa tidak dapat mengajukan pinjaman karena belum memiliki simpanan. Silakan lakukan simpanan terlebih dahulu."
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
			return
		}
		limitPinjaman = 5 * totalSimpanan // 5x total simpanan
	case "01", "02": // Dosen (01) atau Tenaga Pendidikan (02)
		jenisAnggota = "Dosen/Tenaga Pendidikan"
		// Ambil gaji dari form
		gajiStr := c.PostForm("gaji_bulanan")
		if gajiStr == "" {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = "Gaji bulanan wajib diisi untuk dosen/tenaga pendidikan."
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
			return
		}
		// Parse gaji (asumsi dalam ribuan atau jutaan, sesuaikan dengan input)
		var gaji float64
		if _, err := fmt.Sscanf(gajiStr, "%f", &gaji); err != nil {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = "Format gaji tidak valid."
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
			return
		}
		// Kemampuan bayar: 0.4 x gaji x tenor
		kemampuanBayar := 0.4 * gaji * float64(pinjaman.JangkaWaktu)
		limitPinjaman = kemampuanBayar

	default:
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Jenis anggota tidak valid."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Validasi menggunakan kemampuan bayar (Langkah 1), bukan limit pinjaman (Langkah 3)
	var maxLimit float64
	if anggota.UnitKerja == "03" { // Mahasiswa
		maxLimit = limitPinjaman
	} else { // Dosen/Staff - gunakan kemampuan bayar
		// Hitung kemampuan bayar untuk Dosen/Staff
		gajiStr := c.PostForm("gaji_bulanan")
		var gaji float64
		fmt.Sscanf(gajiStr, "%f", &gaji)
		maxLimit = 0.4 * gaji * float64(pinjaman.JangkaWaktu)
	}

	if pinjaman.JumlahPinjaman > maxLimit {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = fmt.Sprintf("Jumlah pinjaman melebihi limit maksimal Rp %.0f untuk %s (berdasarkan kemampuan bayar).", maxLimit, jenisAnggota)
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Set bunga from previously fetched value
	pinjaman.Bunga = bungaTerkini

	pinjaman.IDAnggota = userID
	pinjaman.NamaAnggota = anggota.NamaAnggota
	// Capture metode pencairan from the form (transfer_bank / tunai)
	pinjaman.MetodePencairan = c.PostForm("metode_pencairan")
	pinjaman.NomorRekening = c.PostForm("no_rekening")
	pinjaman.NamaBank = c.PostForm("nama_bank")
	pinjaman.NamaPemilikRekening = c.PostForm("nama_pemilik")

	// Capture gaji bulanan dari form
	gajiStr := c.PostForm("gaji_bulanan")
	if gajiStr != "" {
		var gaji float64
		if _, err := fmt.Sscanf(gajiStr, "%f", &gaji); err == nil {
			pinjaman.GajiBulanan = gaji
		}
	}
	// Capture tujuan pinjaman dari form (multiple checkboxes + text)
	tujuanList := c.PostFormArray("tujuan_pinjaman")
	tujuanLain := c.PostForm("tujuan_lain")
	if tujuanLain != "" {
		tujuanList = append(tujuanList, "Lain-lain: "+tujuanLain)
	}
	if len(tujuanList) > 0 {
		pinjaman.TujuanPinjaman = strings.Join(tujuanList, ", ")
	}

	// Validasi metode pencairan
	if pinjaman.MetodePencairan != "transfer_bank" && pinjaman.MetodePencairan != "tunai" {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Metode pencairan harus dipilih."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Validasi data rekening jika metode transfer bank
	if pinjaman.MetodePencairan == "transfer_bank" {
		if pinjaman.NomorRekening == "" || pinjaman.NamaBank == "" || pinjaman.NamaPemilikRekening == "" {
			templateData := getAjukanPinjamanTemplateData(userID, anggota)
			templateData["Error"] = "Data rekening bank harus dilengkapi jika memilih metode transfer bank."
			c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
			return
		}
	}

	pinjaman.TglPinjaman = time.Now() // Set tanggal pengajuan otomatis
	pinjaman.Status = "proses"        // Status proses untuk konfirmasi bendahara

	// Log the final pinjaman struct before creating to help debugging
	fmt.Printf("DEBUG: Pinjaman ready to create: %+v\n", pinjaman)

	err = repository.CreatePinjaman(pinjaman)
	if err != nil {
		// Log CreatePinjaman error
		fmt.Printf("DEBUG: CreatePinjaman error: %s\n", err.Error())
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Gagal mengajukan pinjaman. Silakan coba lagi."
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
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

	db := config.GetDB()

	// Ambil nomor rekening dari database
	var nomorRekening string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening)
	if err != nil {
		nomorRekening = "1234567890 (Bank ABC)" // Default jika belum diset
	}

	// Ambil detail simpanan per jenis untuk cek apakah sudah ada data
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
	if err != nil {
		simpananByJenis = map[string]float64{
			"pokok":     0,
			"wajib":     0,
			"sukarela":  0,
			"hari_raya": 0,
		}
	}

	// Cek apakah simpanan wajib sudah ada (lebih dari 0)
	hasSimpananWajib := simpananByJenis["wajib"] > 0

	// Ambil konfigurasi simpanan wajib untuk validasi
	configSimpananWajib, err := repository.GetKonfigurasiSimpananWajib()
	var nominalSimpananWajib float64 = 0

	if err == nil && configSimpananWajib != nil {
		// PersentasePotong sekarang menyimpan NOMINAL TETAP (bukan persentase lagi)
		// Karena form sudah diubah menjadi input nominal langsung dalam Rupiah
		if nominal, ok := configSimpananWajib["PersentasePotong"].(float64); ok {
			nominalSimpananWajib = nominal
		}
	}

	// Cek apakah anggota sudah melakukan simpanan wajib bulan ini
	// Logika: Cek apakah ada transaksi simpanan wajib (id_simpanan=2) di bulan dan tahun ini
	now := time.Now()
	bulanIni := int(now.Month())
	tahunIni := now.Year()

	var countSimpananBulanIni int
	querySimpananBulanIni := `
		SELECT COUNT(*) FROM detail 
		WHERE id_anggota = $1 
		AND id_simpanan = 2 
		AND EXTRACT(MONTH FROM tgl_transaksi) = $2 
		AND EXTRACT(YEAR FROM tgl_transaksi) = $3
	`
	err = db.QueryRow(querySimpananBulanIni, userID, bulanIni, tahunIni).Scan(&countSimpananBulanIni)
	if err != nil {
		countSimpananBulanIni = 0
	}

	sudahSimpananBulanIni := countSimpananBulanIni > 0

	// Cek apakah simpanan wajib sudah sesuai dengan konfigurasi
	// isLocked akan TRUE jika:
	// 1. Sudah melakukan simpanan wajib bulan ini, ATAU
	// 2. Total simpanan wajib sudah >= target DAN sudah pernah simpan bulan ini
	isLocked := sudahSimpananBulanIni

	// Ambil konten halaman simpanan dari database
	halaman, err := repository.GetHalamanBySlug("simpanan")
	var kontenData map[string]interface{}
	if err == nil && halaman.Konten != "" {
		json.Unmarshal([]byte(halaman.Konten), &kontenData)
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

	c.HTML(http.StatusOK, "anggota_simpanan.html", gin.H{
		"Judul":                "Simpanan",
		"Anggota":              anggota,
		"Now":                  time.Now(),
		"NomorRekening":        nomorRekening,
		"Konten":               kontenData,
		"HasSimpananWajib":     hasSimpananWajib,
		"SimpananWajib":        simpananByJenis["wajib"],
		"IsLocked":             isLocked,
		"NominalSimpananWajib": nominalSimpananWajib,
		"CurrentLogo":          latestLogo,
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

	// Ambil input dari form (template mengirim beberapa field: simpanan_wajib, simpanan_sukarela, simpanan_hari_raya, simpanan_umroh_haji, simpanan_qurban, total_simpanan)
	wajibStr := c.PostForm("simpanan_wajib")
	sukarelaStr := c.PostForm("simpanan_sukarela")
	hariRayaStr := c.PostForm("simpanan_hari_raya")
	umrohHajiStr := c.PostForm("simpanan_umroh_haji")
	qurbanStr := c.PostForm("simpanan_qurban")
	totalStr := c.PostForm("total_simpanan")

	// Set tanggal pengajuan otomatis ke waktu sekarang (atau gunakan yang dikirim jika ada)
	tanggalPengajuan := time.Now()
	if t := c.PostForm("tanggal_pengajuan"); t != "" {
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			// Combine parsed date with current time-of-day so timestamp reflects submission time
			now := time.Now()
			tanggalPengajuan = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
		}
	}

	// Parse values (toleran terhadap empty)
	var wajib, sukarela, hariRaya, umrohHaji, qurban float64
	if wajibStr != "" {
		fmt.Sscanf(wajibStr, "%f", &wajib)
	}
	if sukarelaStr != "" {
		fmt.Sscanf(sukarelaStr, "%f", &sukarela)
	}
	if hariRayaStr != "" {
		fmt.Sscanf(hariRayaStr, "%f", &hariRaya)
	}
	if umrohHajiStr != "" {
		fmt.Sscanf(umrohHajiStr, "%f", &umrohHaji)
	}
	if qurbanStr != "" {
		fmt.Sscanf(qurbanStr, "%f", &qurban)
	}
	var total float64
	if totalStr != "" {
		fmt.Sscanf(totalStr, "%f", &total)
	} else {
		total = wajib + sukarela + hariRaya + umrohHaji + qurban
	}

	if wajib <= 0 && sukarela <= 0 && hariRaya <= 0 && umrohHaji <= 0 && qurban <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Minimal salah satu nilai simpanan harus lebih dari 0.",
		})
		return
	}

	// Handle file upload
	file, err := c.FormFile("bukti")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Bukti pembayaran wajib diupload.",
		})
		return
	}

	// Save the uploaded file
	filename := time.Now().Format("20060102150405") + "_" + file.Filename
	dst := "./static/uploads/" + filename
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan file bukti pembayaran.",
		})
		return
	}

	// Buat entri untuk setiap jenis simpanan yang > 0
	// IDSimpanan mapping: pokok(1), wajib(2), sukarela(3), hari_raya(4), umroh_haji(5), qurban(6)
	var errs []error
	if wajib > 0 {
		d := models.Detail{
			IDAnggota:       userID,
			IDSimpanan:      2,
			IDPengelola:     1,
			TglTransaksi:    tanggalPengajuan,
			JumlahSimpanan:  wajib,
			TotalSimpanan:   total,
			BuktiPembayaran: filename,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if sukarela > 0 {
		d := models.Detail{
			IDAnggota:       userID,
			IDSimpanan:      3,
			IDPengelola:     1,
			TglTransaksi:    tanggalPengajuan,
			JumlahSimpanan:  sukarela,
			TotalSimpanan:   total,
			BuktiPembayaran: filename,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if hariRaya > 0 {
		d := models.Detail{
			IDAnggota:       userID,
			IDSimpanan:      4,
			IDPengelola:     1,
			TglTransaksi:    tanggalPengajuan,
			JumlahSimpanan:  hariRaya,
			TotalSimpanan:   total,
			BuktiPembayaran: filename,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if umrohHaji > 0 {
		d := models.Detail{
			IDAnggota:       userID,
			IDSimpanan:      5,
			IDPengelola:     1,
			TglTransaksi:    tanggalPengajuan,
			JumlahSimpanan:  umrohHaji,
			TotalSimpanan:   total,
			BuktiPembayaran: filename,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if qurban > 0 {
		d := models.Detail{
			IDAnggota:       userID,
			IDSimpanan:      6,
			IDPengelola:     1,
			TglTransaksi:    tanggalPengajuan,
			JumlahSimpanan:  qurban,
			TotalSimpanan:   total,
			BuktiPembayaran: filename,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}

	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan beberapa data simpanan. Silakan coba lagi.",
		})
		return
	}

	// Berhasil, return JSON
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Simpanan berhasil diajukan!",
		"redirect": "/anggota/riwayat",
	})
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
	pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userID)
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	// Hitung jumlah pinjaman dan sisa pinjaman
	var jumlahPinjaman float64
	var sisaPinjaman float64
	var angsuranKe int

	totalPinjamanAktif, totalSisaAktif, errTotal := getTotalAngsuranAktif(pinjamans)
	pinjamanInfo, err := getPinjamanPrioritasAngsuran(pinjamans)
	if errTotal == nil {
		jumlahPinjaman = totalPinjamanAktif
		sisaPinjaman = totalSisaAktif
	}
	if err == nil && pinjamanInfo != nil {
		angsuranKe = pinjamanInfo.AngsuranKe
	}

	db := config.GetDB()

	// Ambil nomor rekening dari database
	var nomorRekening string
	err = db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = 'nomor_rekening'").Scan(&nomorRekening)
	if err != nil {
		nomorRekening = "1234567890 (Bank ABC)" // Default jika belum diset
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
	// Hitung persentase pelunasan
	persentasePelunasan := 0.0
	if jumlahPinjaman > 0 {
		persentasePelunasan = float64(jumlahPinjaman-sisaPinjaman) / float64(jumlahPinjaman) * 100
		if persentasePelunasan < 0 {
			persentasePelunasan = 0
		}
	}
	c.HTML(http.StatusOK, "anggota_angsuran.html", gin.H{
		"Judul":               "Angsuran",
		"Anggota":             anggota,
		"JumlahPinjaman":      jumlahPinjaman,
		"SisaPinjaman":        sisaPinjaman,
		"AngsuranKe":          angsuranKe,
		"Pinjamans":           pinjamans,
		"TotalPinjaman":       jumlahPinjaman,
		"NomorRekening":       nomorRekening,
		"CurrentLogo":         latestLogo,
		"PersentasePelunasan": persentasePelunasan,
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
		// compute safe defaults to pass to template
		_, totalPinjaman, _, _ := repository.GetSaldoAnggota("")
		c.HTML(http.StatusInternalServerError, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Gagal mengambil data pengguna.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  totalPinjaman,
		})
		return
	}

	// helper to render error with recomputed loan totals so template shows correct state
	renderWithTotals := func(status int, msg string) {
		pinjamans, _ := repository.GetPinjamanAktifByAnggotaID(userID)

		var jumlahPinjaman float64
		var sisaPinjaman float64
		var angsuranKe int

		totalPinjamanAktif, totalSisaAktif, errTotal := getTotalAngsuranAktif(pinjamans)
		pinjamanInfo, err := getPinjamanPrioritasAngsuran(pinjamans)
		if errTotal == nil {
			jumlahPinjaman = totalPinjamanAktif
			sisaPinjaman = totalSisaAktif
		}
		if err == nil && pinjamanInfo != nil {
			angsuranKe = pinjamanInfo.AngsuranKe
		}

		c.HTML(status, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          msg,
			"JumlahPinjaman": jumlahPinjaman,
			"SisaPinjaman":   sisaPinjaman,
			"AngsuranKe":     angsuranKe,
			"Pinjamans":      pinjamans,
			"TotalPinjaman":  jumlahPinjaman,
		})
	}

	// Ambil input dari form
	jumlahAngsuranStr := c.PostForm("jumlah_angsuran")
	tanggalPembayaranStr := c.PostForm("tanggal_pembayaran")
	metodePembayaran := c.PostForm("metode_pembayaran")

	// Fallback tanggal pembayaran ke hari ini jika tidak dikirim dari form
	if tanggalPembayaranStr == "" {
		tanggalPembayaranStr = time.Now().Format("2006-01-02")
	}

	// Validasi input
	if jumlahAngsuranStr == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Jumlah angsuran wajib diisi.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}

	if tanggalPembayaranStr == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Tanggal pembayaran wajib diisi.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}

	if metodePembayaran == "" {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Metode pembayaran wajib dipilih.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}

	// Parse jumlah angsuran
	var jumlahAngsuran float64
	if _, err := fmt.Sscanf(jumlahAngsuranStr, "%f", &jumlahAngsuran); err != nil {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Format jumlah angsuran tidak valid.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}

	if jumlahAngsuran <= 0 {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Jumlah angsuran harus lebih dari 0.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}

	// Parse tanggal pembayaran and combine with current time-of-day so timestamp reflects submission time
	parsedDate, err := time.Parse("2006-01-02", tanggalPembayaranStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
			"Judul":          "Angsuran",
			"Anggota":        anggota,
			"Error":          "Format tanggal pembayaran tidak valid.",
			"JumlahPinjaman": 0.0,
			"SisaPinjaman":   0.0,
			"AngsuranKe":     0,
			"TotalPinjaman":  0.0,
		})
		return
	}
	now := time.Now()
	tanggalPembayaran := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())

	// Handle file upload: only required for transfer method
	var filename string
	if strings.ToLower(metodePembayaran) == "transfer" {
		file, err := c.FormFile("bukti")
		if err != nil {
			renderWithTotals(http.StatusBadRequest, "Bukti pembayaran wajib diupload.")
			return
		}

		// Save the uploaded file
		filename = time.Now().Format("20060102150405") + "_" + file.Filename
		dst := "./static/uploads/" + filename
		if err := c.SaveUploadedFile(file, dst); err != nil {
			renderWithTotals(http.StatusInternalServerError, "Gagal menyimpan file bukti pembayaran.")
			return
		}
	} else {
		// not a transfer, bukti optional
		filename = ""
	}

	// Ambil ID pinjaman dari form (jika ada) atau gunakan pinjaman aktif pertama
	idPinjamanStr := c.PostForm("id_pinjaman")
	var idPinjaman int
	if idPinjamanStr != "" {
		if parsedID, err := strconv.Atoi(idPinjamanStr); err == nil {
			idPinjaman = parsedID
		} else {
			// fallback: cari pinjaman aktif
			pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userID)
			pinjamanInfo, errInfo := getPinjamanPrioritasAngsuran(pinjamans)
			if err != nil || errInfo != nil || pinjamanInfo == nil {
				c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
					"Judul":          "Angsuran",
					"Anggota":        anggota,
					"Error":          "Tidak ada pinjaman aktif.",
					"JumlahPinjaman": 0.0,
					"SisaPinjaman":   0.0,
					"AngsuranKe":     0,
					"TotalPinjaman":  0.0,
				})
				return
			}
			idPinjaman = pinjamanInfo.Pinjaman.IDPinjaman
		}
	} else {
		// Jika tidak ada ID pinjaman di form, ambil pinjaman aktif pertama
		pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userID)
		pinjamanInfo, errInfo := getPinjamanPrioritasAngsuran(pinjamans)
		if err != nil || errInfo != nil || pinjamanInfo == nil {
			c.HTML(http.StatusBadRequest, "anggota_angsuran.html", gin.H{
				"Judul":          "Angsuran",
				"Anggota":        anggota,
				"Error":          "Tidak ada pinjaman aktif.",
				"JumlahPinjaman": 0.0,
				"SisaPinjaman":   0.0,
				"AngsuranKe":     0,
				"TotalPinjaman":  0.0,
			})
			return
		}
		idPinjaman = pinjamanInfo.Pinjaman.IDPinjaman
	}

	// Buat angsuran baru
	angsuran := models.Angsuran{
		IDPinjaman:    idPinjaman,
		IDPengelola:   sql.NullInt64{Int64: 1, Valid: true}, // Default pengelola
		TglBayar:      tanggalPembayaran,
		SisaPinjaman:  jumlahAngsuran, // Untuk sementara, sisa pinjaman = jumlah angsuran
		BuktiAngsuran: filename,       // Simpan nama file sebagai string
		Status:        "",             // Status akan diset ke pending oleh repository
		NamaAnggota:   anggota.NamaAnggota,
	}

	// Simpan ke database
	err = repository.CreateAngsuran(angsuran)
	if err != nil {
		fmt.Printf("CreateAngsuran error: %v\nAngsuran: %+v\n", err, angsuran)
		renderWithTotals(http.StatusInternalServerError, "Gagal menyimpan angsuran. Silakan coba lagi.")
		return
	}

	// Berhasil, redirect ke halaman riwayat sehingga angsuran baru muncul di sana
	c.Redirect(http.StatusFound, "/anggota/riwayat")
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

	// // Selalu gunakan logo.png jika ada, jika tidak pakai placeholder
	// latestLogo := "/static/images/logo.png"
	// if _, err := os.Stat("static/images/logo.png"); os.IsNotExist(err) {
	// 	latestLogo = "/static/images/placeholder.png"
	// }
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

	c.HTML(http.StatusOK, "anggota_sejarah.html", gin.H{
		"Judul":       halaman.Judul,
		"Konten":      konten,
		"Anggota":     anggota,
		"CurrentLogo": latestLogo,
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

	c.HTML(http.StatusOK, "anggota_visi_misi.html", gin.H{
		"Judul":       halaman.Judul,
		"Konten":      konten,
		"Anggota":     anggota,
		"CurrentLogo": latestLogo,
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

// AjukanPengambilanSimpanan menampilkan halaman form pengajuan pengambilan simpanan
func AjukanPengambilanSimpanan(c *gin.Context) {
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

	// Hitung total saldo simpanan anggota
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(userID)
	if err != nil {
		totalSimpanan = 0
	}

	// Ambil detail simpanan per jenis
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
	if err != nil {
		simpananByJenis = make(map[string]float64)
	}

	// Ambil daftar jenis simpanan
	db := config.GetDB()
	rows, err := db.Query("SELECT id_simpanan, jenis_simpanan FROM simpanan ORDER BY id_simpanan")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data jenis simpanan"})
		return
	}
	defer rows.Close()

	type JenisSimpanan struct {
		ID    int    `json:"id"`
		Jenis string `json:"jenis"`
	}
	var jenisSimpananList []JenisSimpanan
	for rows.Next() {
		var js JenisSimpanan
		if err := rows.Scan(&js.ID, &js.Jenis); err == nil {
			jenisSimpananList = append(jenisSimpananList, js)
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

	c.HTML(http.StatusOK, "anggota_ajukan_pengambilan_simpanan.html", gin.H{
		"Anggota":           anggota,
		"TotalSimpanan":     totalSimpanan,
		"SimpananByJenis":   simpananByJenis,
		"JenisSimpananList": jenisSimpananList,
		"CurrentLogo":       latestLogo,
	})
}

// AjukanPengambilanSimpananPost memproses pengajuan pengambilan simpanan
func AjukanPengambilanSimpananPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Anda harus login"})
		return
	}

	// Ambil data dari form
	jumlahStr := c.PostForm("jumlah")
	alasan := c.PostForm("alasan")
	idSimpananStr := c.PostForm("jenis_simpanan")
	metodePengambilan := c.PostForm("metode_pengambilan")
	noRekening := c.PostForm("no_rekening")
	namaBank := c.PostForm("nama_bank")
	namaPemilik := c.PostForm("nama_pemilik")

	// Konversi jumlah ke float
	jumlah, err := strconv.ParseFloat(jumlahStr, 64)
	if err != nil || jumlah <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah tidak valid"})
		return
	}

	// Validasi metode pengambilan
	if metodePengambilan != "transfer_bank" && metodePengambilan != "tunai" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Metode penarikan harus dipilih"})
		return
	}

	// Validasi data rekening jika transfer bank
	if metodePengambilan == "transfer_bank" {
		if noRekening == "" || namaBank == "" || namaPemilik == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data rekening bank harus dilengkapi"})
			return
		}
	}

	// Validasi jenis simpanan
	idSimpanan, err := strconv.Atoi(idSimpananStr)
	if err != nil || idSimpanan <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenis simpanan harus dipilih"})
		return
	}

	// Ambil nama jenis simpanan
	db := config.GetDB()
	var jenisNama string
	err = db.QueryRow("SELECT jenis_simpanan FROM simpanan WHERE id_simpanan = $1", idSimpanan).Scan(&jenisNama)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenis simpanan tidak valid"})
		return
	}

	// Cek saldo per jenis simpanan
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil saldo"})
		return
	}

	saldoJenis, exists := simpananByJenis[jenisNama]
	if !exists || saldoJenis <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada saldo untuk jenis simpanan ini"})
		return
	}

	if jumlah > saldoJenis {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah penarikan melebihi saldo jenis simpanan yang dipilih"})
		return
	}

	// Simpan pengajuan pengambilan simpanan ke database
	query := `INSERT INTO pengambilan_simpanan (id_anggota, id_simpanan, jumlah, alasan, metode_pengambilan, no_rekening, nama_bank, nama_pemilik, tgl_pengajuan, status) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, 'pending')`

	_, err = db.Exec(query, userID, idSimpanan, jumlah, alasan, metodePengambilan, noRekening, namaBank, namaPemilik)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pengajuan: " + err.Error()})
		return
	}

	// Berhasil, kirim response sukses (frontend akan redirect ke riwayat)
	c.JSON(http.StatusOK, gin.H{"message": "Pengajuan penarikan simpanan berhasil disubmit. Menunggu persetujuan bendahara."})
}

// AnggotaRiwayatPage menampilkan halaman riwayat anggota dengan logo terbaru dan data riwayat gabungan
func AnggotaRiwayatPage(c *gin.Context) {
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

	// Ambil riwayat transaksi anggota
	riwayat, err := repository.GetRiwayatTransaksiByAnggotaID(userID)
	if err != nil {
		riwayat = []models.Riwayat{}
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

	c.HTML(http.StatusOK, "anggota_riwayat.html", gin.H{
		"Judul":       "Riwayat Transaksi",
		"Anggota":     anggota,
		"Riwayat":     riwayat,
		"CurrentLogo": latestLogo,
	})
}
