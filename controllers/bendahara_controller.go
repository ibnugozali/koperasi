package controllers

import (
	// "strconv" // Uncomment jika tahunKonfirmasi integer
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// BendaharaUploadKop menerima file kop surat dan menyimpannya ke folder static/uploads/kop/
func BendaharaUploadKop(c *gin.Context) {
	// Parse form file
	file, err := c.FormFile("kop_file")
	if err != nil {
		c.String(http.StatusBadRequest, "Gagal menerima file kop: %v", err)
		return
	}
	// Buat folder jika belum ada
	kopDir := "static/uploads/kop/"
	if err := os.MkdirAll(kopDir, os.ModePerm); err != nil {
		c.String(http.StatusInternalServerError, "Gagal membuat folder kop: %v", err)
		return
	}
	// Hanya izinkan file gambar (jpg, jpeg, png)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.String(http.StatusBadRequest, "Hanya file gambar (JPG, JPEG, PNG) yang diperbolehkan untuk kop surat!")
		return
	}
	filename := fmt.Sprintf("kop_%d%s", time.Now().Unix(), ext)
	savePath := filepath.Join(kopDir, filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.String(http.StatusInternalServerError, "Gagal menyimpan file kop: %v", err)
		return
	}
	// Simpan path kop ke session atau database jika perlu (sementara ke session)
	session := sessions.Default(c)
	session.Set("kop_path", savePath)
	session.Save()
	c.Redirect(http.StatusFound, "/bendahara/laporan?success=Kop surat berhasil diupload")
}

// BendaharaViewAnggotaKeluar menampilkan detail anggota keluar berdasarkan ID
func BendaharaViewAnggotaKeluar(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil || anggota.Status != "keluar" {
		c.Redirect(http.StatusFound, "/bendahara/anggota/keluar?error=Anggota keluar tidak ditemukan")
		return
	}

	// Ambil detail simpanan per jenis (seperti di anggota_controller)
	simpananByJenis, err := repository.GetDetailSimpananByJenis(idStr)
	if err != nil {
		simpananByJenis = map[string]float64{
			"pokok":     0,
			"wajib":     0,
			"sukarela":  0,
			"hari_raya": 0,
		}
	}

	// Samakan dengan halaman profil anggota dan detail ketua:
	// total simpanan tidak memasukkan simpanan pokok dari pendaftaran.
	totalSimpanan := simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"]
	profilSimpananRows := buildProfilSimpananRows(simpananByJenis)

	// Ambil total pinjaman
	_, totalPinjaman, _, err := repository.GetSaldoAnggota(idStr)
	if err != nil {
		totalPinjaman = 0
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

	c.HTML(http.StatusOK, "bendahara_data_anggota_keluar_view.html", gin.H{
		"Anggota":            anggota,
		"ActivePage":         "anggota_keluar",
		"CurrentLogo":        latestLogo,
		"Title":              "Detail Anggota Keluar",
		"ProfilSimpananRows": profilSimpananRows,
		"SimpananPokok":      simpananByJenis["pokok"],
		"SimpananWajib":      simpananByJenis["wajib"],
		"SimpananSukarela":   simpananByJenis["sukarela"],
		"SimpananHariRaya":   simpananByJenis["hari_raya"],
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
	})
}

// BendaharaListAnggotaKeluar menampilkan daftar anggota yang sudah keluar
func BendaharaListAnggotaKeluar(c *gin.Context) {
	// Cari logo terbaru di static/images
	dirFiles, errLogo := os.ReadDir("static/images")
	var latestLogo string
	var latestTime int64

	if errLogo == nil {
		for _, file := range dirFiles {
			name := file.Name()
			if (len(name) > 5 && name[:5] == "logo_" &&
				(name[len(name)-4:] == ".png" || name[len(name)-4:] == ".jpg")) ||
				name == "logo.png" {

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

	// Ambil data anggota keluar
	anggotas, err := repository.GetAnggotaByStatus("keluar")
	if err != nil {
		// ❗ TIDAK LAGI MEMANGGIL error.html
		c.HTML(http.StatusOK, "bendahara_data_anggota_keluar.html", gin.H{
			"Anggotas":    []models.Anggota{},
			"ActivePage":  "anggota_keluar",
			"CurrentLogo": latestLogo,
			"Title":       "Data Anggota Keluar",
			"Error":       "Gagal mengambil data anggota keluar",
		})
		return
	}

	// Render normal
	c.HTML(http.StatusOK, "bendahara_data_anggota_keluar.html", gin.H{
		"Anggotas":    anggotas,
		"ActivePage":  "anggota_keluar",
		"CurrentLogo": latestLogo,
		"Title":       "Data Anggota Keluar",
	})
}

// // BendaharaListAnggotaKeluar menampilkan daftar anggota yang sudah keluar
// func BendaharaListAnggotaKeluar(c *gin.Context) {
// 	// Cari logo terbaru di static/images
// 	dirFiles, errLogo := os.ReadDir("static/images")
// 	var latestLogo string
// 	var latestTime int64
// 	if errLogo == nil {
// 		for _, file := range dirFiles {
// 			name := file.Name()
// 			if (len(name) > 5 && name[:5] == "logo_" && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".jpg")) || name == "logo.png" {
// 				info, err := file.Info()
// 				if err == nil {
// 					modTime := info.ModTime().Unix()
// 					if modTime > latestTime {
// 						latestTime = modTime
// 						latestLogo = "/static/images/" + name
// 					}
// 				}
// 			}
// 		}
// 	}
// 	if latestLogo == "" {
// 		latestLogo = "/static/images/placeholder.png"
// 	}

// 	anggotas, err := repository.GetAnggotaByStatus("keluar")
// 	if err != nil {
// 		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota keluar"})
// 		return
// 	}

// 	c.HTML(http.StatusOK, "bendahara_data_anggota_keluar.html", gin.H{
// 		"Anggotas":    anggotas,
// 		"ActivePage":  "anggota_keluar",
// 		"CurrentLogo": latestLogo,
// 		"Title":       "Data Anggota Keluar",
// 	})

// }

// Menampilkan dashboard bendahara dengan data statistik
func BendaharaLihatDetailAngsuran(c *gin.Context) {
	id := c.Param("id")
	// Gunakan handler detail angsuran utama agar data konsisten dan tidak lagi tampil placeholder.
	c.Redirect(http.StatusMovedPermanently, "/bendahara/detail-angsuran/"+id)
}

// Handler untuk detail ajukan pengambilan simpanan
func BendaharaDetailAjukanPengambilan(c *gin.Context) {
	// Cari logo terbaru di static/images
	dirFiles, errLogo := os.ReadDir("./static/images")
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
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "ID tidak valid")
		return
	}
	pengambilan, err := repository.GetPengambilanSimpananByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Data penarikan tidak ditemukan")
		return
	}
	c.HTML(http.StatusOK, "bendahara_detail_ajukan_pengambilan.html", gin.H{
		"Anggota": map[string]interface{}{
			"NamaAnggota": pengambilan.NamaAnggota,
			"IDAnggota":   pengambilan.IDAnggota,
		},
		"JenisSimpanan":     pengambilan.JenisSimpanan,
		"Jumlah":            pengambilan.Jumlah,
		"MetodePengambilan": pengambilan.MetodePencairan,
		"Status":            pengambilan.Status,
		"NomorRekening":     pengambilan.NomorRekening,
		"NamaBank":          pengambilan.NamaBank,
		"NamaPemilik":       pengambilan.NamaPemilikRekening,
		"TglPengajuan":      pengambilan.TglPengajuan,
		"Alasan":            pengambilan.Alasan,
		"BuktiPengambilan":  pengambilan.BuktiPengambilan,
		"ActivePage":        "detail-ajukan-pengambilan",
		"CurrentLogo":       latestLogo,
	})
}

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

	// Cari logo terbaru di static/images
	dirFiles, errLogo := os.ReadDir("./static/images")
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
	// Data untuk template
	data := map[string]interface{}{
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"TotalAngsuran":      totalAngsuran,
		"TotalPengambilan":   totalPengambilan,
		"ActivePage":         "dashboard",
		"CurrentLogo":        latestLogo,
	}

	c.HTML(http.StatusOK, "bendahara_dashboard_content.html", data)
}

// Menampilkan halaman konfirmasi anggota----
// BendaharaEditRekeningRegister menampilkan halaman edit nomor rekening koperasi
// BendaharaKonfirmasi menampilkan halaman konfirmasi anggota
func BendaharaKonfirmasi(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal mengambil data anggota")
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

	// Ambil pesan error dari query string jika ada
	errorMsg := c.Query("error")
	c.HTML(http.StatusOK, "bendahara_anggota_konfirmasi.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage":     "konfirmasi_anggota",
		"CurrentLogo":    latestLogo,
		"Title":          "Konfirmasi Anggota",
		"ErrorMessage":   errorMsg,
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

	c.HTML(http.StatusOK, "bendahara_edit_rekening_register.html", gin.H{
		"NomorRekening":           nomorRekening,
		"NominalSimpanan":         nominalSimpanan,
		"KeteranganBuktiTransfer": keteranganBuktiTransfer,
		"ActivePage":              "edit-rekening-register",
		"CurrentLogo":             latestLogo,
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
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Nomor rekening harus diisi",
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan nomor rekening",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Nomor rekening berhasil disimpan",
		})
	case "simpanan":
		if nominalSimpanan == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Nominal simpanan harus diisi",
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan nominal simpanan",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Nominal simpanan berhasil disimpan",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Tipe field tidak valid",
		})
	}
}

// Mengkonfirmasi keanggotaan
func BendaharaConfirmMembership(c *gin.Context) {
	// Ambil id anggota dari URL (ini masih TEMP id)
	tempID := c.Param("id")

	// Ambil data anggota untuk mendapatkan informasi unit_kerja, fakultas, dan tahun
	anggota, err := repository.GetAnggotaByID(tempID)
	if err != nil {
		// Log error detail ke terminal/server log
		log.Printf("[ERROR] BendaharaKonfirmasiAnggota ambil anggota gagal (id=%s): %v", tempID, err)
		// Redirect ke halaman konfirmasi dengan pesan error
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi?error=Gagal mengambil data anggota")
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

	// Ambil nomor urut terakhir secara global (tidak direset per kombinasi)
	var lastNumber int
	query := `SELECT COALESCE(MAX(CAST(nomor_urut AS INTEGER)), 0) FROM anggota WHERE id_anggota NOT LIKE 'TEMP%'`
	err = db.QueryRow(query).Scan(&lastNumber)
	if err != nil {
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi?error=Gagal generate nomor urut")
		return
	}

	// Nomor urut berikutnya (4 digit)
	newNumber := lastNumber + 1
	nomorUrut := fmt.Sprintf("%04d", newNumber)

	// Pastikan tahunKonfirmasi string, ambil 2 digit terakhir
	tahunKonfirmasiStr := tahunKonfirmasi
	tahun2Digit := tahunKonfirmasiStr
	if len(tahunKonfirmasiStr) == 4 {
		tahun2Digit = tahunKonfirmasiStr[2:]
	}
	newIDAnggota := fmt.Sprintf("%s%s%s%s", anggota.UnitKerja, anggota.FakultasCode, tahun2Digit, nomorUrut)
	// import (
	//   "strconv"
	//   "reflect"
	// )

	// Update id_anggota, status, tahun, dan nomor_urut
	updateQuery := `UPDATE anggota 
	                SET id_anggota = $1, status = $2, tahun = $3, nomor_urut = $4 
	                WHERE id_anggota = $5`

	_, err = db.Exec(updateQuery, newIDAnggota, "aktif", tahunKonfirmasi, nomorUrut, tempID)
	if err != nil {
		// Log error detail ke terminal/server log
		log.Printf("[ERROR] BendaharaKonfirmasiAnggota update anggota gagal (id=%s): %v", tempID, err)
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi?error="+url.QueryEscape("Gagal mengkonfirmasi anggota"))
		return
	}

	// Anggota berhasil dikonfirmasi dan akan muncul di halaman data anggota
	// Tidak ditambahkan ke import_history agar tidak muncul di halaman import anggota
	fmt.Printf("✓ Anggota dengan ID %s berhasil dikonfirmasi dan aktif\n", newIDAnggota)

	// FITUR BARU: Jika pemotongan otomatis aktif dan anggota punya gaji, proses simpanan wajib
	// PENTING: Hanya proses jika tanggal sekarang >= tanggal pemotongan yang disetting
	configData, err := repository.GetKonfigurasiSimpananWajib()
	if err == nil && configData != nil {
		statusAktif, ok := configData["StatusAktif"].(bool)
		if ok && statusAktif && anggota.GajiBulanan > 0 {
			// Cek apakah tanggal sekarang sudah melewati atau sama dengan tanggal pemotongan
			now := time.Now()
			tanggalSekarang := now.Day()
			tanggalPotong, _ := configData["TanggalPotong"].(int)

			// Hanya proses jika tanggal sekarang >= tanggal pemotongan
			if tanggalSekarang >= tanggalPotong {
				// Ambil data anggota yang sudah dikonfirmasi dengan ID baru
				confirmedAnggota, err := repository.GetAnggotaByID(newIDAnggota)
				if err == nil && confirmedAnggota.GajiBulanan > 0 {
					// Get nominal simpanan wajib dari konfigurasi
					nominalSimpananWajib := configData["PersentasePotong"].(float64)

					bulan := int(now.Month())
					tahun := now.Year()

					// Cek apakah anggota sudah punya log pemotongan bulan ini
					var exists bool
					checkQuery := "SELECT EXISTS(SELECT 1 FROM log_pemotongan_simpanan WHERE id_anggota = $1 AND bulan = $2 AND tahun = $3 AND status = 'berhasil')"
					db.QueryRow(checkQuery, newIDAnggota, bulan, tahun).Scan(&exists)

					// Cek apakah sudah ada simpanan wajib manual
					var hasSimpananWajib bool
					checkSimpananQuery := `SELECT EXISTS(SELECT 1 FROM detail WHERE id_anggota = $1 AND id_simpanan = 2 AND COALESCE(status, 'confirmed') = 'confirmed')`
					db.QueryRow(checkSimpananQuery, newIDAnggota).Scan(&hasSimpananWajib)

					// Jika belum ada log dan belum ada simpanan wajib, proses otomatis
					if !exists && !hasSimpananWajib {
						// Begin transaction
						tx, err := db.Begin()
						if err == nil {
							// Insert detail simpanan wajib
							detailQuery := `INSERT INTO detail (id_anggota, id_simpanan, id_pengelola, tgl_transaksi, jumlah_simpanan, status)
							                VALUES ($1, 2, 1, CURRENT_TIMESTAMP, $2, 'confirmed')`
							_, err = tx.Exec(detailQuery, newIDAnggota, nominalSimpananWajib)

							if err == nil {
								// Commit transaction
								err = tx.Commit()
								if err == nil {
									// Log success
									logQuery := `INSERT INTO log_pemotongan_simpanan (id_anggota, bulan, tahun, gaji_bulanan, jumlah_potong, status, keterangan)
									             VALUES ($1, $2, $3, $4, $5, 'berhasil', $6)`
									db.Exec(logQuery, newIDAnggota, bulan, tahun, float64(confirmedAnggota.GajiBulanan), nominalSimpananWajib,
										fmt.Sprintf("Pemotongan otomatis saat konfirmasi anggota sebesar Rp %.0f", nominalSimpananWajib))

									fmt.Printf("✓ Simpanan wajib otomatis berhasil diproses untuk anggota %s (Rp %.0f)\n", newIDAnggota, nominalSimpananWajib)
								} else {
									fmt.Printf("⚠️ Gagal commit simpanan wajib otomatis: %v\n", err)
								}
							} else {
								tx.Rollback()
								fmt.Printf("⚠️ Gagal insert simpanan wajib otomatis: %v\n", err)
							}
						} else {
							fmt.Printf("⚠️ Gagal memulai transaksi simpanan wajib otomatis: %v\n", err)
						}
					} else if hasSimpananWajib {
						fmt.Printf("ℹ️ Anggota %s sudah memiliki simpanan wajib manual\n", newIDAnggota)
					} else if exists {
						fmt.Printf("ℹ️ Anggota %s sudah diproses pemotongan simpanan wajib bulan ini\n", newIDAnggota)
					}
				}
			} else {
				fmt.Printf("ℹ️ Pemotongan otomatis belum waktunya. Tanggal sekarang: %d, Tanggal pemotongan: %d\n", tanggalSekarang, tanggalPotong)
			}
		}
	}

	// Arahkan kembali ke halaman konfirmasi bendahara
	c.Redirect(http.StatusFound, "/bendahara/konfirmasi")
}

// Menampilkan halaman pesan untuk bendahara
func BendaharaPesan(c *gin.Context) {
	db := config.GetDB()

	// Handle POST (kirim pesan) - AJAX JSON response
	if c.Request.Method == http.MethodPost {
		anggotaID := c.PostForm("anggota_id")
		judul := c.PostForm("judul")
		isi := c.PostForm("isi")

		if anggotaID == "" || judul == "" || isi == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Semua field harus diisi: Pilih Anggota, Judul, dan Isi Pesan",
			})
			return
		}

		// Ambil data anggota untuk pesan sukses dan notifikasi WA.
		var anggotaNama, anggotaNoTelepon string
		err := db.QueryRow("SELECT nama_anggota, COALESCE(no_telepon, '') FROM anggota WHERE id_anggota = $1 AND status = 'aktif'", anggotaID).Scan(&anggotaNama, &anggotaNoTelepon)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Anggota tidak ditemukan atau tidak aktif",
			})
			return
		}

		pesan := models.Pesan{
			IDAnggota: anggotaID,
			Judul:     judul,
			Isi:       isi,
			Status:    "unread",
		}

		if err := repository.CreatePesan(pesan); err != nil {
			log.Printf("Gagal create pesan: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal mengirim pesan. Silakan coba lagi.",
			})
			return
		}

		appBaseURL := resolveAppBaseURL(c, db)
		if errWA := sendAnggotaWhatsAppPesanNotification(anggotaNoTelepon, anggotaNama, judul, isi, appBaseURL); errWA != nil {
			log.Printf("[WA PESAN ANGGOTA] gagal kirim WA ke anggota %s (%s): %v", anggotaNama, anggotaID, errWA)
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": fmt.Sprintf("✅ Pesan berhasil dikirim ke <strong>%s</strong>", anggotaNama),
		})
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

	// Ambil daftar anggota untuk dropdown
	anggotaRows, err := db.Query("SELECT id_anggota, nama_anggota FROM anggota WHERE status = 'aktif'")
	var anggotaList []struct{ ID, Nama string }
	if err == nil {
		defer anggotaRows.Close()
		for anggotaRows.Next() {
			var id, nama string
			if scanErr := anggotaRows.Scan(&id, &nama); scanErr != nil {
				continue
			}
			anggotaList = append(anggotaList, struct{ ID, Nama string }{id, nama})
		}
	} else {
		log.Printf("[WARN] gagal memuat daftar anggota aktif: %v", err)
	}

	// Ambil daftar pesan terkirim (join nama anggota)
	pesanRows, err := db.Query(`
		SELECT p.judul, p.isi, p.tgl_kirim, a.nama_anggota
		FROM pesan p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		ORDER BY p.tgl_kirim DESC
	`)
	var pesanList []struct {
		Judul, Isi, NamaAnggota, Tanggal string
	}
	if err == nil {
		defer pesanRows.Close()
		for pesanRows.Next() {
			var judul, isi, tanggal, nama string
			if scanErr := pesanRows.Scan(&judul, &isi, &tanggal, &nama); scanErr != nil {
				continue
			}
			pesanList = append(pesanList, struct {
				Judul, Isi, NamaAnggota, Tanggal string
			}{judul, isi, nama, tanggal})
		}
	} else {
		log.Printf("[WARN] gagal memuat daftar pesan bendahara: %v", err)
	}

	c.HTML(http.StatusOK, "bendahara_pesan.html", gin.H{
		"AnggotaList": anggotaList,
		"PesanList":   pesanList,
		"Title":       "Pesan Bendahara",
		"ActivePage":  "pesan",
		"CurrentLogo": latestLogo,
	})
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

	// Gunakan template khusus untuk simpanan
	templateName := "bendahara_halaman_edit.html"
	activePage := "halaman"
	if slug == "simpanan" {
		templateName = "bendahara_halaman_edit_simpanan.html"
		activePage = "edit_simpanan"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Halaman":     halaman,
		"Konten":      konten,
		"CurrentLogo": latestLogo,
		"ActivePage":  activePage,
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
			log.Printf("[ERROR] BendaharaDashboard update halaman gagal (slug=%s): %v", slug, err)
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

	// Ambil nominal simpanan pokok dan petakan ke semua anggota aktif
	// (anggota pada list ini diambil dari GetAllAnggota yang hanya status aktif).
	simpananPokok := make(map[string]float64)
	var nominalSimpananPokok float64
	err = config.GetDB().QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
	if err != nil {
		nominalSimpananPokok = 100000
	}
	for _, anggota := range anggotas {
		simpananPokok[anggota.IDAnggota] = nominalSimpananPokok
	}

	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64)
	}
	potonganRegister, err := repository.GetPotonganRegisterPotongGajiBulanIniAllAnggota()
	if err != nil {
		potonganRegister = make(map[string]float64)
	}

	nominalSimpananWajib := 0.0
	if configSimpananWajib, err := repository.GetKonfigurasiSimpananWajib(); err == nil {
		if nominal, ok := configSimpananWajib["PersentasePotong"].(float64); ok {
			nominalSimpananWajib = nominal
		}
	}

	// Hitung sisa gaji untuk setiap anggota: Gaji Bulanan - Potongan Bulan Ini
	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potonganWajib := potonganBulanIni[anggota.IDAnggota]
		if potonganWajib <= 0 && nominalSimpananWajib > 0 {
			potonganWajib = nominalSimpananWajib
		}
		potongan := potonganWajib + potonganRegister[anggota.IDAnggota]
		// Sisa gaji = Gaji bulanan dikurangi potongan bulanan terjadwal
		// ditambah potongan simpanan pokok dari pendaftaran potong gaji bulan berjalan.
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan

		// Fallback tampilan: jika total simpanan wajib belum tercatat,
		// gunakan potongan bulan ini agar kolom "Simpanan Wajib" tidak kosong.
		if simpananWajib[anggota.IDAnggota] <= 0 && potonganWajib > 0 {
			simpananWajib[anggota.IDAnggota] = potonganWajib
		} else if simpananWajib[anggota.IDAnggota] <= 0 && nominalSimpananWajib > 0 {
			simpananWajib[anggota.IDAnggota] = nominalSimpananWajib
		}
	}

	c.HTML(http.StatusOK, "bendahara_data_anggota.html", gin.H{
		"Anggotas":         anggotas,
		"SimpananPokok":    simpananPokok,
		"SimpananWajib":    simpananWajib,
		"PotonganBulanIni": potonganBulanIni,
		"SisaGaji":         sisaGaji,
		"ActivePage":       "anggota",
		"CurrentLogo":      latestLogo,
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

	// Ambil detail simpanan per jenis (seperti di anggota_controller)
	simpananByJenis, err := repository.GetDetailSimpananByJenis(idStr)
	if err != nil {
		simpananByJenis = map[string]float64{
			"pokok":     0,
			"wajib":     0,
			"sukarela":  0,
			"hari_raya": 0,
		}
	}

	// Hitung Total Simpanan dari semua simpanan yang ada
	totalSimpanan := simpananByJenis["pokok"] + simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"]
	profilSimpananRows := buildProfilSimpananRows(simpananByJenis)

	// Ambil total pinjaman
	_, totalPinjaman, _, err := repository.GetSaldoAnggota(idStr)
	if err != nil {
		totalPinjaman = 0
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

	c.HTML(http.StatusOK, "bendahara_data_anggota_view.html", gin.H{
		"Anggota":            anggota,
		"ActivePage":         "anggota",
		"CurrentLogo":        latestLogo,
		"Title":              "Detail Anggota",
		"ProfilSimpananRows": profilSimpananRows,
		"SimpananPokok":      simpananByJenis["pokok"],
		"SimpananWajib":      simpananByJenis["wajib"],
		"SimpananSukarela":   simpananByJenis["sukarela"],
		"SimpananHariRaya":   simpananByJenis["hari_raya"],
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
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

	c.HTML(http.StatusOK, "bendahara_data_anggota_edit.html", gin.H{
		"Anggota":    anggota,
		"ActivePage": "anggota",
		// "LogoPath":   logoPath,
		"CurrentLogo": latestLogo,
		"Title":       "Edit Data Anggota",
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

	// Setelah hapus, redirect ke daftar anggota keluar
	c.Redirect(http.StatusFound, "/bendahara/anggota/keluar")
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
	// Fallback: jika ShouldBind gagal parse jumlah_simpanan, coba manual
	if detail.JumlahSimpanan == 0 {
		if jumlahStr := c.PostForm("jumlah_simpanan"); jumlahStr != "" {
			if jumlahParsed, err := strconv.ParseFloat(jumlahStr, 64); err == nil {
				detail.JumlahSimpanan = jumlahParsed
			}
		}
	}
	// Fallback tambahan: coba field lama "jumlah" jika "jumlah_simpanan" tidak ada
	if detail.JumlahSimpanan == 0 {
		if jumlahStr := c.PostForm("jumlah"); jumlahStr != "" {
			if jumlahParsed, err := strconv.ParseFloat(jumlahStr, 64); err == nil {
				detail.JumlahSimpanan = jumlahParsed
			}
		}
	}
	if detail.IDAnggota == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID anggota wajib diisi"})
		return
	}

	detail.IDPengelola = bendaharaID.(int)
	detail.TglTransaksi = time.Now()
	// Entri oleh bendahara disimpan sebagai pending, menunggu konfirmasi dari ketua.
	detail.Status = "pending"
	// Default metode pembayaran dari form (tunai untuk entri manual)
	if detail.MetodePembayaran == "" {
		detail.MetodePembayaran = "tunai"
	}

	// Ambil id_simpanan dari tabel simpanan berdasarkan nama (jenis_simpanan)
	jenisSimpanan := c.PostForm("jenis_simpanan")
	db := config.GetDB()
	var idSimpanan int
	err := db.QueryRow("SELECT id_simpanan FROM simpanan WHERE jenis_simpanan = $1", jenisSimpanan).Scan(&idSimpanan)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jenis simpanan tidak valid"})
		return
	}
	detail.IDSimpanan = idSimpanan

	// Hitung total simpanan (kumulatif)
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(detail.IDAnggota)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung total simpanan"})
		return
	}
	detail.TotalSimpanan = totalSimpanan + detail.JumlahSimpanan

	// VALIDASI: Entri manual tunai hanya boleh jika ada data pending yang cocok
	pendingList, err := repository.GetPendingSimpananByCriteria(detail.IDAnggota, detail.IDSimpanan, detail.JumlahSimpanan)
	if err != nil || len(pendingList) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak sesuai dengan simpanan pending. Entri manual tunai hanya diperbolehkan untuk data yang sudah ada di daftar simpanan pending."})
		return
	}

	// Update seluruh data pending yang cocok menjadi confirmed.
	// Tidak membuat record baru agar data bendahara -> ketua konsisten dan tidak duplikat.
	confirmedCount := 0
	for _, pending := range pendingList {
		err := repository.UpdateSimpananStatus(pending.IDDetail, "confirmed")
		if err != nil {
			log.Printf("[ERROR] Gagal update status simpanan pending (id_detail=%d): %v", pending.IDDetail, err)
			continue
		}
		confirmedCount++
	}

	if confirmedCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengonfirmasi data simpanan pending"})
		return
	}

	// Notifikasi WA ke ketua agar melakukan konfirmasi tahap ketua.
	anggota, errAnggota := repository.GetAnggotaByID(detail.IDAnggota)
	if errAnggota == nil {
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		nominal := fmt.Sprintf("%.2f", detail.JumlahSimpanan)
		if errWA := sendKetuaWhatsAppTransactionNotification("", anggota.NamaAnggota, "Simpanan", nominal, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF KETUA] gagal kirim dari entri simpanan bendahara: %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF KETUA] gagal ambil data anggota (%s) untuk entri simpanan: %v", detail.IDAnggota, errAnggota)
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
		// Log error detail ke console agar mudah debug
		log.Printf("[ERROR] BendaharaRiwayat ambil data riwayat gagal: %v", err)
		c.HTML(http.StatusInternalServerError, "bendahara_riwayat_content.html", gin.H{
			"ActivePage": "riwayat",
			"Error":      "Gagal mengambil data riwayat. Silakan hubungi admin.",
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
	c.HTML(http.StatusOK, "bendahara_riwayat_content.html", gin.H{
		"ActivePage":  "riwayat",
		"Riwayats":    riwayats,
		"Anggotas":    anggotas,
		"CurrentLogo": latestLogo,
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

	report, err := repository.GetLaporanKeuangan(bulan, tahun)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_laporan.html", gin.H{
			"ActivePage":  "laporan",
			"Error":       "Gagal mengambil laporan",
			"CurrentLogo": latestLogo,
			"Bulan":       bulan,
			"Tahun":       tahun,
		})
		return
	}

	// Ambil data anggota aktif
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "bendahara_laporan.html", gin.H{
			"ActivePage":  "laporan",
			"Error":       "Gagal mengambil data anggota",
			"CurrentLogo": latestLogo,
			"Bulan":       bulan,
			"Tahun":       tahun,
			"Report":      report,
		})
		return
	}

	// Ambil data potongan bulan ini untuk semua anggota
	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64)
	}

	// Hitung sisa gaji untuk setiap anggota: Gaji Bulanan - Potongan Bulan Ini
	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potongan := potonganBulanIni[anggota.IDAnggota]
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan
	}

	c.HTML(http.StatusOK, "bendahara_laporan.html", gin.H{
		"ActivePage":       "laporan",
		"Report":           report,
		"Bulan":            bulan,
		"Tahun":            tahun,
		"CurrentLogo":      latestLogo,
		"Anggotas":         anggotas,
		"PotonganBulanIni": potonganBulanIni,
		"SisaGaji":         sisaGaji,
		"GetUnitKerjaName": repository.GetUnitKerjaName,
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

	// Ambil data bendahara
	bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
	if err != nil {
		log.Printf("[ERROR] BendaharaPengaturan ambil data bendahara gagal (id=%v): %v", bendaharaID, err)
		c.HTML(http.StatusInternalServerError, "bendahara_pengaturan.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara",
			"LogoPath":   latestLogo,
		})
		return
	}

	c.HTML(http.StatusOK, "bendahara_pengaturan.html", gin.H{
		"ActivePage":  "pengaturan",
		"Bendahara":   bendahara,
		"LogoPath":    latestLogo,
		"CurrentLogo": latestLogo,
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

	// Jika hanya salah satu field password/konfirmasi diisi, abaikan perubahan password, update username saja
	if (request.Password != "" && request.ConfirmPassword == "") || (request.Password == "" && request.ConfirmPassword != "") {
		request.Password = ""
		request.ConfirmPassword = ""
	}
	// Validasi password jika diisi lengkap
	if request.Password != "" && request.ConfirmPassword != "" {
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
	plainPasswordToUpdate := ""
	if passwordToUpdate != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordToUpdate), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}
		plainPasswordToUpdate = request.Password
		passwordToUpdate = string(hashedPassword)
	} else {
		// Jika password kosong, ambil password lama
		bendahara, err := repository.GetPengelolaByID(bendaharaID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bendahara"})
			return
		}
		passwordToUpdate = bendahara.Password
		plainPasswordToUpdate = bendahara.PlainPassword
	}

	// Update username, password, dan plain_password
	err := repository.UpdatePengelolaUsernamePassword(bendaharaID.(int), request.Username, passwordToUpdate, plainPasswordToUpdate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// BendaharaKonfirmasiTransaksi menampilkan halaman konfirmasi transaksi
func BendaharaKonfirmasiTransaksi(c *gin.Context) {
	ensurePendingAngsuranPotongGaji()

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
		No     int
		Detail models.Detail
	}
	type numberedPinjaman struct {
		No       int
		Pinjaman models.Pinjaman
	}
	type numberedAngsuran struct {
		No            int
		Angsuran      models.Angsuran
		AngsuranKe    int
		PeriodeLabel  string
		StatusWaktu   string
		StatusWaktuBG string
	}
	type numberedPengambilan struct {
		No                  int
		PengambilanSimpanan models.PengambilanSimpanan
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
		angsuranKe := 0
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		jatuhTempo := time.Date(a.TglBayar.Year(), a.TglBayar.Month(), a.TglBayar.Day(), 0, 0, 0, 0, a.TglBayar.Location())
		selisihHari := int(jatuhTempo.Sub(today).Hours() / 24)

		statusWaktu := ""
		statusWaktuBG := "bg-light text-dark border"
		switch {
		case selisihHari > 0:
			statusWaktu = fmt.Sprintf("Tersisa %d hari", selisihHari)
			statusWaktuBG = "bg-info text-dark"
		case selisihHari == 0:
			statusWaktu = "Jatuh tempo hari ini"
			statusWaktuBG = "bg-warning text-dark"
		default:
			statusWaktu = fmt.Sprintf("Terlambat %d hari", -selisihHari)
			statusWaktuBG = "bg-danger"
		}

		periodeLabel := a.TglBayar.Format("January 2006")
		if riwayatAngsuran, err := repository.GetAngsuranByPinjamanID(a.IDPinjaman); err == nil {
			for idx := len(riwayatAngsuran) - 1; idx >= 0; idx-- {
				if riwayatAngsuran[idx].IDAngsuran == a.IDAngsuran {
					angsuranKe = len(riwayatAngsuran) - idx
					break
				}
			}
		}
		numberedAngsurans = append(numberedAngsurans, numberedAngsuran{
			No:            i + 1,
			Angsuran:      a,
			AngsuranKe:    angsuranKe,
			PeriodeLabel:  periodeLabel,
			StatusWaktu:   statusWaktu,
			StatusWaktuBG: statusWaktuBG,
		})
	}

	var numberedPengambilans []numberedPengambilan
	for i, ps := range pendingPengambilan {
		numberedPengambilans = append(numberedPengambilans, numberedPengambilan{No: i + 1, PengambilanSimpanan: ps})
	}

	// Get LogoPath from context
	// logoPath, _ := c.Get("LogoPath")
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

	// Ambil jenis simpanan dari halaman (slug: simpanan)
	halamanSimpanan, err := repository.GetHalamanBySlug("simpanan")
	var simpananTypes []map[string]interface{}
	if err == nil && len(halamanSimpanan.Konten) > 0 {
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &konten); err == nil {
			if jenisList, ok := konten["jenis_simpanan"].([]interface{}); ok {
				for _, item := range jenisList {
					if m, ok := item.(map[string]interface{}); ok {
						simpananTypes = append(simpananTypes, m)
					}
				}
			}
		}
	}

	// Cek apakah bukti transfer gaji sudah diupload untuk bulan & tahun ini
	now := time.Now()
	currentBulan := int(now.Month())
	currentTahun := now.Year()
	buktiTransferExists, _ := repository.CheckBuktiTransferGajiExists(currentBulan, currentTahun)
	var buktiTransfer *models.BuktiTransferGaji
	if buktiTransferExists {
		buktiTransfer, _ = repository.GetBuktiTransferGajiByBulanTahun(currentBulan, currentTahun)
	}

	// Cek apakah bukti transfer gaji sudah di-approve untuk bulan & tahun ini
	buktiTransferApproved, _ := repository.CheckBuktiTransferGajiApproved(currentBulan, currentTahun)

	// Ambil semua riwayat bukti transfer gaji untuk ditampilkan
	buktiList, _ := repository.GetAllBuktiTransferGaji()

	// Cek query parameter success/error untuk notifikasi
	successMsg := c.Query("success")
	errorMsg := c.Query("error")

	c.HTML(http.StatusOK, "bendahara_konfirmasi_transaksi.html", gin.H{
		"PendingSimpanan":       numberedSimpanan,
		"PendingPinjaman":       numberedPinjamans,
		"PendingAngsuran":       numberedAngsurans,
		"PendingPengambilan":    numberedPengambilans,
		"ActivePage":            "konfirmasi-transaksi",
		"CurrentLogo":           latestLogo,
		"Title":                 "Konfirmasi Transaksi",
		"SimpananTypes":         simpananTypes,
		"BuktiTransferExists":   buktiTransferExists,
		"BuktiTransfer":         buktiTransfer,
		"BuktiTransferApproved": buktiTransferApproved,
		"CurrentBulan":          currentBulan,
		"CurrentTahun":          currentTahun,
		"BuktiList":             buktiList,
		"SuccessMessage":        successMsg,
		"ErrorMessage":          errorMsg,
	})
}

// BendaharaLihatDetailSimpanan menampilkan detail simpanan pending untuk anggota
func BendaharaLihatDetailSimpanan(c *gin.Context) {
	id := c.Param("id")

	// Cek apakah ID adalah angka dan valid sebagai ID detail
	if idNum, err := strconv.Atoi(id); err == nil {
		// Cek apakah idNum benar-benar ID detail yang ada di tabel detail
		db := config.GetDB()
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM detail WHERE id_detail = $1)", idNum).Scan(&exists)
		if err == nil && exists {
			// Redirect ke view detail simpanan jika memang ID detail valid
			c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/bendahara/view-detail-simpanan/%d", idNum))
			return
		}
		// Jika tidak ditemukan sebagai ID detail, lanjutkan sebagai ID anggota
	}

	// Ambil data anggota
	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan atau ID tidak valid. Silakan cek kembali tautan atau gunakan menu yang benar."})
		return
	}

	// Ambil semua simpanan anggota ini (semua status)
	db := config.GetDB()
	query := `
		SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.tgl_transaksi, 
		       d.jumlah_simpanan, d.total_simpanan, s.jenis_simpanan,
		       COALESCE(d.status, 'confirmed') as status,
		       COALESCE(d.bukti_pembayaran, '') as bukti_pembayaran
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1
		ORDER BY d.tgl_transaksi DESC
	`

	rows, err := db.Query(query, id)
	var detailSimpanan []models.Detail
	var totalWajib, totalSukarela, totalHariRaya, totalUmrohHaji, totalQurban, grandTotal float64
	var buktiPembayaran string
	if err == nil {
		defer rows.Close()
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
			d.BuktiPembayaran = bukti
			detailSimpanan = append(detailSimpanan, d)

			if buktiPembayaran == "" && bukti != "" {
				buktiPembayaran = bukti
			}

			switch s.JenisSimpanan {
			case "pokok":
				totalWajib += d.JumlahSimpanan
			case "wajib":
				totalWajib += d.JumlahSimpanan
			case "sukarela":
				totalSukarela += d.JumlahSimpanan
			case "hari_raya":
				totalHariRaya += d.JumlahSimpanan
			case "umroh_haji":
				totalUmrohHaji += d.JumlahSimpanan
			case "qurban":
				totalQurban += d.JumlahSimpanan
			}
			grandTotal += d.JumlahSimpanan
		}
	}

	// Ambil nomor rekening koperasi
	nomorRekening, _ := repository.GetNomorRekening("simpanan")

	infoMsg := ""
	if len(detailSimpanan) == 0 {
		infoMsg = "Tidak ada data simpanan untuk anggota ini."
	}
	c.HTML(http.StatusOK, "bendahara_detail_simpanan.html", gin.H{
		"Anggota":         anggota,
		"DetailSimpanan":  detailSimpanan,
		"TotalWajib":      totalWajib,
		"TotalSukarela":   totalSukarela,
		"TotalHariRaya":   totalHariRaya,
		"TotalUmrohHaji":  totalUmrohHaji,
		"TotalQurban":     totalQurban,
		"GrandTotal":      grandTotal,
		"NomorRekening":   nomorRekening,
		"BuktiPembayaran": buktiPembayaran,
		"Judul":           "Rekapitulasi Simpanan Anggota",
		"InfoMsg":         infoMsg,
	})
}

// BendaharaViewDetailSimpanan menampilkan detail transaksi simpanan berdasarkan ID detail (untuk riwayat)
func BendaharaViewDetailSimpanan(c *gin.Context) {
	idDetail := c.Param("id")

	db := config.GetDB()

	// Ambil data detail simpanan berdasarkan ID detail
	var d models.Detail
	var s models.Simpanan
	var a models.Anggota
	var bukti string

	query := `
		SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.tgl_transaksi, 
		       d.jumlah_simpanan, d.total_simpanan, s.jenis_simpanan,
		       COALESCE(d.status, 'confirmed') as status,
		       COALESCE(d.bukti_pembayaran, '') as bukti_pembayaran,
		       a.nama_anggota, a.no_telepon
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		JOIN anggota a ON d.id_anggota = a.id_anggota
		WHERE d.id_detail = $1
	`

	err := db.QueryRow(query, idDetail).Scan(
		&d.IDDetail, &d.IDAnggota, &d.IDSimpanan, &d.TglTransaksi,
		&d.JumlahSimpanan, &d.TotalSimpanan, &s.JenisSimpanan,
		&d.Status, &bukti,
		&a.NamaAnggota, &a.NoTelepon,
	)

	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Data simpanan tidak ditemukan"})
		return
	}

	d.Simpanan = s
	d.BuktiPembayaran = bukti
	a.IDAnggota = d.IDAnggota

	// Ambil data anggota lengkap (alamat, unit kerja, fakultas, dsb)
	anggotaLengkap, err := repository.GetAnggotaByID(d.IDAnggota)
	if err == nil {
		a = anggotaLengkap
	}

	// Ambil nomor rekening koperasi
	nomorRekening, _ := repository.GetNomorRekening("simpanan")

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

	// Ambil semua simpanan anggota ini untuk total per jenis
	dbAll := config.GetDB()
	rows, err := dbAll.Query(`
		SELECT d.jumlah_simpanan, s.jenis_simpanan
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE d.id_anggota = $1 AND COALESCE(d.status, 'confirmed') = 'confirmed'
	`, d.IDAnggota)
	var totalWajib, totalSukarela, totalHariRaya, totalUmrohHaji, totalQurban, grandTotal float64
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var jumlah float64
			var jenis string
			if err := rows.Scan(&jumlah, &jenis); err == nil {
				switch jenis {
				case "pokok":
					totalWajib += jumlah
				case "wajib":
					totalWajib += jumlah
				case "sukarela":
					totalSukarela += jumlah
				case "hari_raya":
					totalHariRaya += jumlah
				case "umroh_haji":
					totalUmrohHaji += jumlah
				case "qurban":
					totalQurban += jumlah
				}
				grandTotal += jumlah
			}
		}
	}

	c.HTML(http.StatusOK, "bendahara_view_detail_simpanan.html", gin.H{
		"Anggota":          a,
		"Detail":           d,
		"JenisSimpanan":    s.JenisSimpanan,
		"Jumlah":           d.JumlahSimpanan,
		"TanggalTransaksi": d.TglTransaksi,
		"Status":           d.Status,
		"BuktiPembayaran":  bukti,
		"NomorRekening":    nomorRekening,
		"CurrentLogo":      latestLogo,
		"ActivePage":       "riwayat",
		"TotalWajib":       totalWajib,
		"TotalSukarela":    totalSukarela,
		"TotalHariRaya":    totalHariRaya,
		"TotalUmrohHaji":   totalUmrohHaji,
		"TotalQurban":      totalQurban,
		"GrandTotal":       grandTotal,
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
	case "01", "02": // Dosen/Tenaga Pendidikan
		jenisAnggota = "Dosen/Tenaga Pendidikan"
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
		&pinjaman.NamaBank,
		&pinjaman.Status,
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
	// Cek session login
	session := sessions.Default(c)
	user := session.Get("user_id")
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Session expired, silakan login ulang"})
		return
	}

	transactionType := c.Param("type")
	idStr := c.Param("id")
	action := c.PostForm("action")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("[ERROR] ID tidak valid: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "ID tidak valid"})
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
			if err == nil {
				if createErr := createPendingAngsuranPotongGajiAwal(id); createErr != nil {
					log.Printf("[ERROR] Gagal membuat cicilan pending otomatis untuk pinjaman %d: %v", id, createErr)
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Pinjaman dikonfirmasi, tetapi gagal membuat cicilan pending otomatis"})
					return
				}
			}
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
		log.Printf("[ERROR] Tipe transaksi tidak valid: %s", transactionType)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Tipe transaksi tidak valid"})
		return
	}

	if err != nil {
		log.Printf("[ERROR] Gagal update status transaksi: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal memproses transaksi"})
		return
	}

	// Notifikasi ke ketua hanya saat transaksi diterima bendahara.
	if action == "confirm" {
		anggotaNama, jenisLabel, nominal, errInfo := getKetuaTransactionNotifInfo(transactionType, id)
		if errInfo != nil {
			log.Printf("[WA NOTIF KETUA] gagal ambil data transaksi type=%s id=%d: %v", transactionType, id, errInfo)
		} else {
			appBaseURL := resolveAppBaseURL(c, config.GetDB())
			if errWA := sendKetuaWhatsAppTransactionNotification("", anggotaNama, jenisLabel, nominal, appBaseURL); errWA != nil {
				log.Printf("[WA NOTIF KETUA] gagal kirim notifikasi konfirmasi bendahara: %v", errWA)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Transaksi berhasil diproses"})
}

func ensurePendingAngsuranPotongGaji() {
	db := config.GetDB()

	rows, err := db.Query(`
		SELECT p.id_pinjaman
		FROM pinjaman p
		WHERE p.status IN ('proses', 'aktif')
		  AND REPLACE(LOWER(COALESCE(p.metode_angsuran, '')), ' ', '_') = 'potong_gaji'
	`)
	if err != nil {
		log.Printf("[WARN] Gagal sinkronisasi cicilan pending potong gaji: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var idPinjaman int
		if err := rows.Scan(&idPinjaman); err != nil {
			log.Printf("[WARN] Gagal membaca pinjaman untuk sinkronisasi cicilan pending: %v", err)
			continue
		}
		if err := ensureScheduledAngsuranPotongGajiForPinjaman(idPinjaman, time.Now()); err != nil {
			log.Printf("[WARN] Gagal membuat cicilan pending sinkronisasi untuk pinjaman %d: %v", idPinjaman, err)
		}
	}
}

func createPendingAngsuranPotongGajiAwal(idPinjaman int) error {
	return ensureScheduledAngsuranPotongGajiForPinjaman(idPinjaman, time.Now())
}

func countJatuhTempoAngsuran(tglPinjaman, now time.Time, jangkaWaktu int) int {
	if jangkaWaktu <= 0 {
		return 0
	}

	dueCount := 0
	for i := 0; i < jangkaWaktu; i++ {
		jatuhTempo := tglPinjaman.AddDate(0, i, 0)
		if jatuhTempo.After(now) {
			break
		}
		dueCount++
	}
	return dueCount
}

func ensureScheduledAngsuranPotongGajiForPinjaman(idPinjaman int, now time.Time) error {
	db := config.GetDB()

	var pinjaman models.Pinjaman
	err := db.QueryRow(`
		SELECT id_pinjaman, id_anggota, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status,
		       COALESCE(metode_angsuran, '')
		FROM pinjaman
		WHERE id_pinjaman = $1
	`, idPinjaman).Scan(
		&pinjaman.IDPinjaman,
		&pinjaman.IDAnggota,
		&pinjaman.TglPinjaman,
		&pinjaman.JumlahPinjaman,
		&pinjaman.JangkaWaktu,
		&pinjaman.Bunga,
		&pinjaman.Status,
		&pinjaman.MetodeAngsuran,
	)
	if err != nil {
		return err
	}

	metode := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(pinjaman.MetodeAngsuran, " ", "_")))
	if metode != "potong_gaji" {
		return nil
	}

	if pinjaman.JangkaWaktu <= 0 || pinjaman.JumlahPinjaman <= 0 {
		return nil
	}

	angsuranList, err := repository.GetAngsuranByPinjamanID(idPinjaman)
	if err != nil {
		return err
	}

	jatuhTempoSampaiHariIni := countJatuhTempoAngsuran(pinjaman.TglPinjaman, now, pinjaman.JangkaWaktu)
	existingCount := len(angsuranList)
	if existingCount >= jatuhTempoSampaiHariIni {
		return nil
	}

	bungaNominal := pinjaman.JumlahPinjaman * pinjaman.Bunga / 100
	totalKewajiban := pinjaman.JumlahPinjaman + bungaNominal
	jumlahAngsuran := totalKewajiban / float64(pinjaman.JangkaWaktu)
	if jumlahAngsuran <= 0 {
		return nil
	}

	for angsuranKe := existingCount + 1; angsuranKe <= jatuhTempoSampaiHariIni; angsuranKe++ {
		jatuhTempo := pinjaman.TglPinjaman.AddDate(0, angsuranKe-1, 0)
		sisaSetelah := totalKewajiban - (jumlahAngsuran * float64(angsuranKe))
		if sisaSetelah < 0 {
			sisaSetelah = 0
		}

		err = repository.CreateAngsuran(models.Angsuran{
			IDPinjaman:     idPinjaman,
			IDPengelola:    sql.NullInt64{},
			TglBayar:       jatuhTempo,
			JumlahAngsuran: jumlahAngsuran,
			SisaPinjaman:   sisaSetelah,
			BuktiAngsuran:  fmt.Sprintf("POTONG_GAJI_AUTO_%02d", angsuranKe),
			Status:         "pending",
		})
		if err != nil {
			return err
		}
	}
	return nil
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

	bunga := GetCurrentBunga()
	c.HTML(http.StatusOK, "bendahara_edit_bunga.html", gin.H{
		"CurrentBunga": fmt.Sprintf("%.2f", bunga),
		"ActivePage":   "edit-bunga",

		"CurrentLogo": latestLogo,
	})
}

func BendaharaUpdateBunga(c *gin.Context) {
	bungaStr := c.PostForm("bunga")

	if bungaStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Nilai bunga harus diisi",
		})
		return
	}

	bungaVal, err := strconv.ParseFloat(bungaStr, 64)
	if err != nil || bungaVal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Nilai bunga tidak valid",
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Gagal menyimpan bunga ke database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bunga berhasil disimpan",
	})
}

// BendaharaLoginHistory menampilkan halaman riwayat login
// BendaharaDownloadLaporan mengunduh laporan dalam format Excel atau PDF
func BendaharaDownloadLaporan(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	bulan := c.Query("bulan")
	tahun := c.Query("tahun")

	// Ambil path kop dari session (jika ada)
	session := sessions.Default(c)
	kopPath, _ := session.Get("kop_path").(string)
	absKopPath := kopPath
	if kopPath != "" && !filepath.IsAbs(kopPath) {
		absKopPath, _ = filepath.Abs(kopPath)
	}

	switch format {
	case "excel":
		// Ambil data laporan keuangan
		bulanInt, _ := strconv.Atoi(bulan)
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data laporan")
			return
		}

		f := excelize.NewFile()
		sheet := "Sheet1"
		// Jika ada kop gambar (jpg/png), masukkan ke Excel
		rowOffset := 1
		if absKopPath != "" && (strings.HasSuffix(strings.ToLower(absKopPath), ".jpg") || strings.HasSuffix(strings.ToLower(absKopPath), ".jpeg") || strings.HasSuffix(strings.ToLower(absKopPath), ".png")) {
			// Masukkan gambar kop di baris 1, tanpa AutoFit, dan data mulai baris 15
			if err := f.AddPicture(sheet, "A1", absKopPath, &excelize.GraphicOptions{
				AutoFit: false,
				OffsetY: 0,
				OffsetX: 0,
				ScaleX:  1.0,
				ScaleY:  1.0,
			}); err == nil {
				rowOffset = 15
			} else {
				log.Printf("Gagal menambahkan gambar kop ke Excel: %v", err)
				rowOffset = 1
			}
		}
		// Header judul
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset), "LAPORAN KEUANGAN KOPERASI")
		// Tanggal cetak dan periode
		var waktuCetak time.Time
		var tanggalCetak string
		var jamCetak string
		var namaBulan string
		waktuCetak = time.Now()
		tanggalCetak = waktuCetak.Format("2 Januari 2006")
		jamCetak = waktuCetak.Format("15.04")
		namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+1), "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+2), fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt))
		// Header tabel
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+4), "Keterangan")
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowOffset+4), "Jumlah")
		// Data
		totalPengeluaran := 0.0
		if pinjaman, ok := report["total_pinjaman"].(float64); ok {
			totalPengeluaran += pinjaman
		}
		if pengambilan, ok := report["total_pengambilan"].(float64); ok {
			totalPengeluaran += pengambilan
		}
		dataRows := []struct {
			label string
			value interface{}
		}{
			{"Total Simpanan", report["total_simpanan"]},
			{"Total Pinjaman", report["total_pinjaman"]},
			{"Total Angsuran", report["total_angsuran"]},
			{"Total Pengeluaran", totalPengeluaran},
		}
		for i, row := range dataRows {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+5+i), row.label)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowOffset+5+i), row.value)
		}
		// Style header tabel
		styleHeader, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2ecc71"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowOffset+4), fmt.Sprintf("B%d", rowOffset+4), styleHeader)
		// Style data
		styleData, _ := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "left"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowOffset+5), fmt.Sprintf("B%d", rowOffset+5+len(dataRows)-1), styleData)
		// Set lebar kolom otomatis
		f.SetColWidth(sheet, "A", "B", 25)
		// Ambil data anggota aktif dan potongan/sisa gaji
		anggotas, err := repository.GetAllAnggota()
		if err == nil && len(anggotas) > 0 {
			potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
			if err != nil {
				potonganBulanIni = make(map[string]float64)
			}
			startRow := rowOffset + 5 + len(dataRows) + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow-1), "Rincian Laporan Keuangan")
			// Header
			headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}
			cols := []string{"A", "B", "C", "D", "E"}
			for i, h := range headers {
				col := cols[i]
				f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow), h)
			}
			// Data
			for idx, anggota := range anggotas {
				row := startRow + 1 + idx
				nohp := anggota.NoTelepon
				if !strings.HasPrefix(nohp, "+62") {
					nohp = "+62" + strings.TrimLeft(nohp, "0")
				}
				sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
				unitKerjaName := repository.GetUnitKerjaName(anggota.UnitKerja)
				f.SetCellValue(sheet, fmt.Sprintf("A%d", row), anggota.NamaAnggota)
				f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nohp)
				f.SetCellValue(sheet, fmt.Sprintf("C%d", row), unitKerjaName)
				f.SetCellValue(sheet, fmt.Sprintf("D%d", row), int64(anggota.GajiBulanan))
				f.SetCellValue(sheet, fmt.Sprintf("E%d", row), int64(sisaGaji))
			}
			// Style header rincian
			rincianHeaderStyle, _ := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
				Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2ecc71"}, Pattern: 1},
				Alignment: &excelize.Alignment{Horizontal: "center"},
				Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
			})
			rincianDataStyle, _ := f.NewStyle(&excelize.Style{
				Alignment: &excelize.Alignment{Horizontal: "left"},
				Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
			})
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("E%d", startRow), rincianHeaderStyle)
			f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow+1), fmt.Sprintf("E%d", startRow+len(anggotas)), rincianDataStyle)
			f.SetColWidth(sheet, "A", "E", 22)
		}
		// Set header untuk download
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=laporan_koperasi.xlsx")
		c.Header("Content-Transfer-Encoding", "binary")
		if err := f.Write(c.Writer); err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat file Excel")
		}
		return
	case "pdf":
		bulanInt, _ := strconv.Atoi(bulan)
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data laporan")
			return
		}

		// Convert bulan to nama bulan
		namaBulan := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
			"Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		// Format tanggal cetak
		waktuCetak := time.Now()
		tanggalCetak := waktuCetak.Format("2 Januari 2006")
		jamCetak := waktuCetak.Format("15.04")

		// Hitung total pengeluaran (pinjaman + pengambilan)
		totalPengeluaran := 0.0
		if pinjaman, ok := report["total_pinjaman"].(float64); ok {
			totalPengeluaran += pinjaman
		}
		if pengambilan, ok := report["total_pengambilan"].(float64); ok {
			totalPengeluaran += pengambilan
		}

		// Helper function untuk format Rupiah dengan pemisah ribuan (tanpa dobel 'Rp')
		formatRupiah := func(amount float64) string {
			amountStr := fmt.Sprintf("%.0f", amount)
			var result []string
			for i, digit := range amountStr {
				if i > 0 && (len(amountStr)-i)%3 == 0 {
					result = append(result, ".")
				}
				result = append(result, string(digit))
			}
			return "Rp " + strings.Join(result, "")
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()
		// Tambahkan kop surat jika ada
		kopDir := "static/uploads/kop/"
		files, err := os.ReadDir(kopDir)
		var kopPath string
		var latestTime int64
		for _, file := range files {
			if !file.IsDir() && (strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") || strings.HasSuffix(strings.ToLower(file.Name()), ".jpeg") || strings.HasSuffix(strings.ToLower(file.Name()), ".png")) {
				info, _ := file.Info()
				if info.ModTime().Unix() > latestTime {
					latestTime = info.ModTime().Unix()
					kopPath = filepath.Join(kopDir, file.Name())
				}
			}
		}
		if kopPath != "" {
			pdf.ImageOptions(kopPath, 10, 10, 190, 0, false, gofpdf.ImageOptions{ImageType: ""}, 0, "")
			pdf.Ln(45) // Tambah jarak agar data tidak bertumpuk dengan kop surat
		}
		// PDF header, keterangan, dan data laporan
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(190, 10, "LAPORAN KEUANGAN KOPERASI", "0", 1, "C", false, 0, "")
		pdf.Ln(3)
		waktuCetak = time.Now()
		tanggalCetak = waktuCetak.Format("2 Januari 2006")
		jamCetak = waktuCetak.Format("15.04")
		namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(190, 6, "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak, "0", 1, "C", false, 0, "")
		pdf.CellFormat(190, 6, fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt), "0", 1, "C", false, 0, "")
		pdf.Ln(10)

		// Data rows dan header, lebar kolom dinamis
		pdf.SetFont("Arial", "", 10)
		dataRows := []struct {
			label string
			value float64
		}{
			{"Total Simpanan", report["total_simpanan"].(float64)},
			{"Total Pinjaman", report["total_pinjaman"].(float64)},
			{"Total Angsuran", report["total_angsuran"].(float64)},
			{"Total Pengeluaran", totalPengeluaran},
		}
		// Hitung lebar kolom maksimal
		pdf.SetFont("Arial", "B", 11)
		maxLabelWidth := pdf.GetStringWidth("Keterangan")
		maxValueWidth := pdf.GetStringWidth("Jumlah")
		pdf.SetFont("Arial", "", 10)
		for _, row := range dataRows {
			lw := pdf.GetStringWidth(row.label)
			vw := pdf.GetStringWidth(formatRupiah(row.value))
			if lw > maxLabelWidth {
				maxLabelWidth = lw
			}
			if vw > maxValueWidth {
				maxValueWidth = vw
			}
		}
		// Tambahkan padding
		maxLabelWidth += 10
		maxValueWidth += 10
		totalWidth := maxLabelWidth + maxValueWidth
		// Jika totalWidth < 190, distribusikan sisa ke label
		if totalWidth < 190 {
			maxLabelWidth += 190 - totalWidth
			totalWidth = 190
		}
		// Header tabel
		pdf.SetFont("Arial", "B", 11)
		pdf.SetFillColor(46, 204, 113)
		pdf.SetTextColor(255, 255, 255)
		// Hitung tinggi baris header secara dinamis
		headerLabelLines := pdf.SplitLines([]byte("Keterangan"), maxLabelWidth)
		headerValueLines := pdf.SplitLines([]byte("Jumlah"), maxValueWidth)
		headerLines := len(headerLabelLines)
		if len(headerValueLines) > headerLines {
			headerLines = len(headerValueLines)
		}
		headerHeight := float64(headerLines) * 6.0
		pdf.CellFormat(maxLabelWidth, headerHeight, "Keterangan", "1", 0, "C", true, 0, "")
		pdf.CellFormat(maxValueWidth, headerHeight, "Jumlah", "1", 1, "C", true, 0, "")

		// Data rows dengan tinggi baris dinamis
		pdf.SetFont("Arial", "", 10)
		pdf.SetTextColor(0, 0, 0)
		for _, row := range dataRows {
			// Hitung tinggi baris berdasarkan jumlah baris teks (jika multiline)
			labelLines := pdf.SplitLines([]byte(row.label), maxLabelWidth)
			valueLines := pdf.SplitLines([]byte(formatRupiah(row.value)), maxValueWidth)
			nLines := len(labelLines)
			if len(valueLines) > nLines {
				nLines = len(valueLines)
			}
			rowHeight := float64(nLines) * 6.0
			pdf.CellFormat(maxLabelWidth, rowHeight, row.label, "1", 0, "L", false, 0, "")
			pdf.CellFormat(maxValueWidth, rowHeight, formatRupiah(row.value), "1", 1, "L", false, 0, "")
		}

		// Ambil data anggota aktif dan potongan/sisa gaji
		anggotas, err := repository.GetAllAnggota()
		if err == nil && len(anggotas) > 0 {
			potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
			if err != nil {
				potonganBulanIni = make(map[string]float64)
			}
			pdf.Ln(6)
			pdf.SetFont("Arial", "B", 12)
			pdf.CellFormat(190, 8, "Rincian Laporan Keuangan", "0", 1, "L", false, 0, "")
			pdf.Ln(1)
			headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}
			pdf.SetFont("Arial", "B", 10)
			for _, h := range headers {
				pdf.CellFormat(38, 7, h, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Arial", "", 10)
			for _, anggota := range anggotas {
				nohp := anggota.NoTelepon
				if !strings.HasPrefix(nohp, "+62") {
					nohp = "+62" + strings.TrimLeft(nohp, "0")
				}
				sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
				pdf.CellFormat(38, 7, anggota.NamaAnggota, "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, nohp, "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, repository.GetUnitKerjaName(anggota.UnitKerja), "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, fmt.Sprintf("%d", anggota.GajiBulanan), "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, fmt.Sprintf("%.0f", sisaGaji), "1", 1, "L", false, 0, "")
			}
		}
		// Set header untuk download
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename=laporan_koperasi_"+fmt.Sprintf("%02d_%d", bulanInt, tahunInt)+".pdf")
		err = pdf.Output(c.Writer)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal membuat file PDF")
		}
		return
	default:
		c.String(http.StatusBadRequest, "Format file tidak didukung")
		return
	}
}
func BendaharaLoginHistory(c *gin.Context) {
	// Ambil data riwayat login dari database
	loginHistory, err := repository.GetLoginHistory()
	if err != nil {
		loginHistory = []models.LoginHistory{} // Default kosong jika error
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

	c.HTML(http.StatusOK, "bendahara_login_history.html", gin.H{
		"ActivePage":   "login_history",
		"LoginHistory": loginHistory,
		"CurrentLogo":  latestLogo,
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
	var allImportedData []map[string]interface{} // Data dari latest import
	var parseErrors []string
	var totalSuccessCount int
	var totalFailedCount int
	var err error

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

		fmt.Printf("=== Loading latest import history for pengelola ID: %d ===\n", pengelolaID)

		// Ambil HANYA riwayat import terbaru (bukan semua)
		latestImport, err = repository.GetLatestImportHistory(db, pengelolaID)
		if err != nil {
			fmt.Printf("❌ Error loading import history: %v\n", err)
		} else if latestImport != nil {
			fmt.Printf("✓ Found latest import: %s (Date: %v)\n", latestImport.FileName, latestImport.TanggalImport)

			totalSuccessCount = latestImport.SuccessCount
			totalFailedCount = latestImport.FailedCount

			// Parse imported data dari latest import
			if latestImport.ImportedData != "" {
				if err := json.Unmarshal([]byte(latestImport.ImportedData), &allImportedData); err != nil {
					fmt.Printf("❌ Error parsing ImportedData: %v\n", err)
				} else {
					fmt.Printf("✓ Loaded %d records from latest import\n", len(allImportedData))
				}
			} else {
				fmt.Printf("⚠️ No ImportedData in latest import\n")
			}

			// Parse errors dari latest import
			if latestImport.ParseErrors != "" {
				if err := json.Unmarshal([]byte(latestImport.ParseErrors), &parseErrors); err != nil {
					fmt.Printf("❌ Error parsing ParseErrors: %v\n", err)
				}
			}

			fmt.Printf("✓ Total data: %d records (Success: %d, Failed: %d)\n",
				len(allImportedData), totalSuccessCount, totalFailedCount)
		} else {
			fmt.Printf("ℹ️ No import history found for pengelola ID: %d (database is empty)\n", pengelolaID)
		}
	} else {
		fmt.Println("⚠️ No pengelola ID found in session - user not logged in?")
	}

	fmt.Printf("=== Rendering template with %d total records from import history ===\n", len(allImportedData))

	c.HTML(http.StatusOK, "bendahara_import_anggota.html", gin.H{
		"ActivePage":        "import_anggota",
		"LogoPath":          logoPath,
		"LatestImport":      latestImport,
		"ImportedData":      allImportedData, // Kirim data dari import history, bukan dari database anggota
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
		log.Printf("[ERROR] BendaharaImportDataAnggota simpan file gagal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan file",
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
		log.Printf("[ERROR] BendaharaImportDataAnggota buka file excel gagal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file Excel"})
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
		case strings.Contains(fakultas, "PASKASARJANA") || strings.Contains(fakultas, "PASCASARJANA"):
			return "10"
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

		// PENTING: C
		// Jika sudah ada, SELALU gunakan sisa gaji dari database (Gaji Bulanan - Potongan Bulan Ini)
		if nikKTP != "" {
			var idAnggotaExisting string
			var gajiBulananDB int

			// Ambil ID anggota dan gaji bulanan dari database
			checkAnggotaQuery := "SELECT id_anggota, COALESCE(gaji_bulanan, 0) FROM anggota WHERE nik_ktp = $1 LIMIT 1"
			err := config.GetDB().QueryRow(checkAnggotaQuery, nikKTP).Scan(&idAnggotaExisting, &gajiBulananDB)

			if err == nil {
				// Anggota sudah ada - hitung sisa gaji (Gaji Bulanan - Potongan Bulan Ini)
				potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
				if err != nil {
					potonganBulanIni = make(map[string]float64)
				}

				potongan := int(potonganBulanIni[idAnggotaExisting])
				sisaGaji := gajiBulananDB - potongan

				// Gunakan sisa gaji sebagai gaji bulanan yang akan di-update
				gajiBulanan = sisaGaji
				fmt.Printf("  Baris %d: Anggota sudah ada (NIK: %s), Gaji DB: Rp %d, Potongan: Rp %d, Sisa Gaji: Rp %d (mengabaikan Excel: %s)\n",
					i+1, nikKTP, gajiBulananDB, potongan, sisaGaji, gajiBulananStr)
			} else {
				// Anggota baru - gunakan gaji dari Excel
				fmt.Printf("  Baris %d: Anggota baru, menggunakan gaji dari Excel: Rp %d\n", i+1, gajiBulanan)
			}
		}

		fmt.Printf("  Baris %d: Nama=%s, Gaji Final=%d\n", i+1, namaAnggota, gajiBulanan)

		// Validasi format tanggal lahir jika diisi
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

		// Mapping fakultas_code ke format 2 digit (unit_kerja tetap gunakan nama lengkap)
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
			UnitKerja:     unitKerja,
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
			// PENTING: gaji_bulanan di anggota.GajiBulanan sudah berisi sisa gaji dari database (diambil saat parsing Excel di atas)
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
		log.Printf("[ERROR] BendaharaImportSimpananWajibExcel buka file excel gagal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuka file Excel",
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
		case strings.Contains(fakultas, "PASKASARJANA") || strings.Contains(fakultas, "PASCASARJANA"):
			return "10"
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
			_ = getValuePreview(row, 1) // unitKerja - tidak divalidasi di preview
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
					previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Format tanggal lahir tidak valid (terdeteksi angka: %s). Gunakan format YYYY-MM-DD (contoh: 1990-05-15)", rowNum, tglLahir))
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

			// Mapping fakultas_code (unit_kerja tetap gunakan nama lengkap)
			fakultasCode := mapFakultasCodePreview(fakultas)

			// Validasi fakultas (harus bisa dimapping atau sudah 2 digit)
			if fakultas != "" && len(fakultas) > 2 && fakultasCode == "" {
				previewErrors = append(previewErrors, fmt.Sprintf("Baris %d: Fakultas '%s' tidak valid. Gunakan: FAI, FE, FH, FISIP, FKIP, FKM, FAPERTA, FT, Paskasarjana, atau Rektorat", rowNum, fakultas))
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

// BendaharaUpdateImportData memperbarui data import history
func BendaharaUpdateImportData(c *gin.Context) {
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

	// Parse request body
	var requestData struct {
		AllImportedData []map[string]interface{} `json:"allImportedData"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	fmt.Printf("=== Updating import data for pengelola ID: %d ===\n", pengelolaID)
	fmt.Printf("Received %d records to update\n", len(requestData.AllImportedData))

	// Ambil latest import history
	latestImport, err := repository.GetLatestImportHistory(db, pengelolaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil riwayat import",
		})
		return
	}

	// Update imported_data field dengan data baru
	importedDataJSON, _ := json.Marshal(requestData.AllImportedData)
	latestImport.ImportedData = string(importedDataJSON)
	latestImport.SuccessCount = len(requestData.AllImportedData)
	latestImport.TotalData = len(requestData.AllImportedData)

	// Save ke database
	err = repository.UpdateImportHistory(db, latestImport)
	if err != nil {
		fmt.Printf("❌ Error updating import history: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memperbarui data import",
		})
		return
	}

	fmt.Printf("✓ Import data updated successfully\n")

	c.JSON(http.StatusOK, gin.H{
		"message": "Data berhasil diperbarui",
		"count":   len(requestData.AllImportedData),
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

	// Hitung sisa gaji untuk setiap anggota: Gaji Bulanan - Potongan Bulan Ini
	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potongan := potonganBulanIni[anggota.IDAnggota]
		// Sisa gaji = Gaji bulanan dikurangi potongan bulan ini
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan
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
			   ang.tgl_bayar, ang.sisa_pinjaman, ang.Status
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
			   ps.tgl_pengajuan, ps.jumlah, ps.alasan, ps.Status
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

	config, err := repository.GetKonfigurasiSimpananWajib()
	if err != nil {
		log.Printf("⚠️ Error mengambil konfigurasi: %v, menggunakan default", err)

		// Jika tidak ada konfigurasi, coba ambil dari data simpanan wajib yang sudah ada
		simpananWajibData, errSimpanan := repository.GetSimpananWajibAllAnggota()
		var avgSimpananWajib float64 = 50000.0 // Default 50k

		if errSimpanan == nil && len(simpananWajibData) > 0 {
			var total float64
			var count int
			for _, nilai := range simpananWajibData {
				if nilai > 0 {
					total += nilai
					count++
				}
			}
			if count > 0 {
				avgSimpananWajib = total / float64(count)
			}
		}

		config = map[string]interface{}{
			"TanggalPotong":    1,
			"PersentasePotong": avgSimpananWajib,
			"NominalTetap":     0.0,
			"TipePemotongan":   "persentase",
			"StatusAktif":      false,
		}
	} else {
		log.Printf("📖 Menampilkan konfigurasi: TanggalPotong=%v, Status=%v, Tipe=%v",
			config["TanggalPotong"], config["StatusAktif"], config["TipePemotongan"])
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage":  "setting_simpanan_wajib",
		"CurrentLogo": latestLogo,
		"Title":       "Setting Simpanan Wajib",
		"Config":      config,
	})
}

// BendaharaSaveSettingSimpananWajib menyimpan konfigurasi pemotongan simpanan wajib
func BendaharaSaveSettingSimpananWajib(c *gin.Context) {
	// logoPath, _ := c.Get("LogoPath")
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

	tanggalPotong, _ := strconv.Atoi(c.PostForm("TanggalPotong"))
	persentasePotong, _ := strconv.ParseFloat(c.PostForm("PersentasePotong"), 64)
	nominalTetap := 0.0            // Set ke 0 karena tidak digunakan lagi
	tipePemotongan := "persentase" // Set default karena field dihapus dari form
	statusAktif := c.PostForm("StatusAktif") == "1"

	// Log data yang akan disimpan
	log.Printf("💾 Menyimpan konfigurasi simpanan wajib:")
	log.Printf("   - Tanggal Potong: %d", tanggalPotong)
	log.Printf("   - Nominal Simpanan Wajib: Rp %.2f", persentasePotong)
	log.Printf("   - Status Aktif: %v", statusAktif)

	err := repository.SaveKonfigurasiSimpananWajib(tanggalPotong, persentasePotong, nominalTetap, tipePemotongan, statusAktif)

	if err != nil {
		log.Printf("❌ ERROR menyimpan konfigurasi: %v", err)
		// Get config untuk ditampilkan di form
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
		c.HTML(http.StatusInternalServerError, "bendahara_setting_simpanan_wajib.html", gin.H{
			"ActivePage": "setting_simpanan_wajib",
			// "LogoPath":   logoPath,
			"CurrentLogo": latestLogo,
			"Title":       "Setting Simpanan Wajib",
			"Config":      config,
			"error":       "Gagal menyimpan konfigurasi",
		})
		return
	}

	// Berhasil simpan, ambil data terbaru dari database
	log.Printf("✅ Konfigurasi berhasil disimpan ke database")
	config, err := repository.GetKonfigurasiSimpananWajib()
	if err != nil {
		log.Printf("⚠️ Warning: Gagal membaca kembali data: %v", err)
	} else {
		log.Printf("📋 Data tersimpan & diverifikasi: TanggalPotong=%v, Status=%v, NominalSimpananWajib=%v",
			config["TanggalPotong"], config["StatusAktif"], config["PersentasePotong"])
	}

	// Pastikan config tidak nil
	if config == nil {
		config = map[string]interface{}{
			"TanggalPotong":    tanggalPotong,
			"PersentasePotong": persentasePotong,
			"NominalTetap":     nominalTetap,
			"TipePemotongan":   tipePemotongan,
			"StatusAktif":      statusAktif,
		}
	}

	// Log data yang berhasil disimpan
	log.Printf("📋 Data yang akan ditampilkan: TanggalPotong=%v, Status=%v", config["TanggalPotong"], config["StatusAktif"])

	successMsg := fmt.Sprintf("Konfigurasi berhasil disimpan. Simpanan wajib otomatis aktif setiap tanggal %d.", tanggalPotong)
	if statusAktif {
		successMsg = fmt.Sprintf("✓ Konfigurasi berhasil disimpan! Pemotongan otomatis AKTIF setiap tanggal %d", tanggalPotong)
	} else {
		successMsg = "✓ Konfigurasi berhasil disimpan! Pemotongan otomatis NONAKTIF"
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage": "setting_simpanan_wajib",
		// "LogoPath":   logoPath,
		"CurrentLogo": latestLogo,
		"Title":       "Setting Simpanan Wajib",
		"Config":      config,
		"success":     successMsg,
	})
}

// BendaharaProsesSimpananWajib melakukan proses pemotongan simpanan wajib manual
func BendaharaProsesSimpananWajib(c *gin.Context) {
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

	successCount, failedCount, errors := repository.ProsesPemotonganSimpananWajib()

	config, err := repository.GetKonfigurasiSimpananWajib()
	if err != nil {
		log.Printf("⚠️ Error mengambil konfigurasi: %v, menggunakan default", err)
		config = map[string]interface{}{
			"TanggalPotong":    1,
			"PersentasePotong": 5.0,
			"NominalTetap":     0.0,
			"TipePemotongan":   "persentase",
			"StatusAktif":      false,
		}
	} else {
		log.Printf("📖 Menampilkan konfigurasi: TanggalPotong=%v, Status=%v, Tipe=%v",
			config["TanggalPotong"], config["StatusAktif"], config["TipePemotongan"])
	}

	message := fmt.Sprintf("✓ Proses pemotongan selesai! Berhasil memotong %d anggota", successCount)
	if failedCount > 0 {
		message += fmt.Sprintf(", Gagal: %d anggota", failedCount)
	}

	if len(errors) > 0 {
		message += "<br><small>Error: " + errors[0] + "</small>"
	}

	c.HTML(http.StatusOK, "bendahara_setting_simpanan_wajib.html", gin.H{
		"ActivePage":  "setting_simpanan_wajib",
		"CurrentLogo": latestLogo,

		"Title":   "Setting Simpanan Wajib",
		"Config":  config,
		"success": message,
	})
}

// BendaharaApproveBuktiTransferGaji menyetujui bukti transfer gaji
func BendaharaApproveBuktiTransferGaji(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?error=ID tidak valid")
		return
	}

	if err := repository.UpdateBuktiTransferGajiStatus(id, "approved"); err != nil {
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?error=Gagal menyetujui bukti transfer gaji")
		return
	}

	c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?success=Bukti transfer gaji disetujui")
}

// BendaharaRejectBuktiTransferGaji menolak bukti transfer gaji
func BendaharaRejectBuktiTransferGaji(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?error=ID tidak valid")
		return
	}

	if err := repository.UpdateBuktiTransferGajiStatus(id, "rejected"); err != nil {
		c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?error=Gagal menolak bukti transfer gaji")
		return
	}

	c.Redirect(http.StatusFound, "/bendahara/konfirmasi-transaksi?success=Bukti transfer gaji ditolak")
}

// BendaharaImportPotongGajiExcel memproses upload file Excel untuk tambah massal transaksi potong gaji.
func BendaharaImportPotongGajiExcel(c *gin.Context) {
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired, silakan login ulang"})
		return
	}

	// Validasi: cek apakah bukti transfer gaji sudah di-approve untuk bulan & tahun ini
	now := time.Now()
	currentBulan := int(now.Month())
	currentTahun := now.Year()
	buktiTransferApproved, _ := repository.CheckBuktiTransferGajiApproved(currentBulan, currentTahun)
	if !buktiTransferApproved {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Bukti transfer gaji belum disetujui untuk periode ini. Silakan minta Ketua untuk mengupload bukti transfer gaji, lalu approve melalui menu Konfirmasi Transaksi.",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak ditemukan."})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file harus .xlsx atau .xls"})
		return
	}
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 10MB"})
		return
	}

	tempPath := "./static/uploads/" + uuid.New().String() + ext
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file upload"})
		return
	}
	defer os.Remove(tempPath)

	f, err := excelize.OpenFile(tempPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak bisa dibaca"})
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File Excel tidak memiliki sheet"})
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data Excel kosong atau tidak valid"})
		return
	}

	// Cari baris header secara dinamis
	headerRowIdx := -1
	headerMap := map[string]int{}
	for r, row := range rows {
		tmp := map[string]int{}
		for i, h := range row {
			norm := normalizeHeader(h)
			if norm != "" {
				tmp[norm] = i
			}
		}
		idxID := findHeaderIndex(tmp, "id anggota", "idanggota", "id", "kode anggota")
		idxNama := findHeaderIndex(tmp, "nama anggota", "nama", "namaanggota")
		if idxID >= 0 || idxNama >= 0 {
			headerRowIdx = r
			headerMap = tmp
			break
		}
	}

	if headerRowIdx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Header tidak ditemukan di file Excel. Pastikan ada kolom 'ID Anggota' atau 'Nama Anggota'."})
		return
	}

	idxIDAnggota := findHeaderIndex(headerMap, "id anggota", "idanggota", "id", "kode anggota")
	idxNama := findHeaderIndex(headerMap, "nama anggota", "nama", "namaanggota")
	idxJenis := findHeaderIndex(headerMap, "jenis transaksi", "jenis", "jenistransaksi", "tipe")
	idxPendingID := findHeaderIndex(headerMap, "id pending", "idpending", "id referensi", "idreferensi", "id transaksi pending", "idtransaksipending")
	idxDetail := findHeaderIndex(headerMap, "jenis simpanan", "detail", "jenissimpanan", "simpanan", "id pinjaman", "idpinjaman")
	idxJumlah := findHeaderIndex(headerMap, "jumlah", "nominal", "amount", "total")

	db := config.GetDB()

	// Cache id_simpanan berdasarkan nama jenis
	idSimpananCache := make(map[string]int)
	rowsSimpanan, err := db.Query("SELECT id_simpanan, LOWER(jenis_simpanan) FROM simpanan")
	if err == nil && rowsSimpanan != nil {
		defer rowsSimpanan.Close()
		for rowsSimpanan.Next() {
			var id int
			var jenis string
			if err := rowsSimpanan.Scan(&id, &jenis); err == nil {
				idSimpananCache[jenis] = id
			}
		}
	} else if err != nil {
		log.Printf("[WARN] gagal memuat master jenis simpanan: %v", err)
	}

	successCount := 0
	failedCount := 0
	var parseErrors []string

	normalizeMetodePotongGaji := func(value string) string {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.ReplaceAll(value, " ", "_")
		return value
	}
	isPotongGajiEligible := func(value string) bool {
		normalized := normalizeMetodePotongGaji(value)
		return normalized == "potong_gaji" || normalized == ""
	}

	for i, row := range rows[headerRowIdx+1:] {
		rowNum := headerRowIdx + i + 2

		idAnggota := getCell(row, idxIDAnggota)
		namaAnggota := getCell(row, idxNama)
		jenisTransaksi := strings.ToLower(getCell(row, idxJenis))
		pendingIDStr := getCell(row, idxPendingID)
		detail := strings.ToLower(getCell(row, idxDetail))
		jumlahStr := getCell(row, idxJumlah)

		// Skip baris kosong
		if idAnggota == "" && namaAnggota == "" && jumlahStr == "" {
			continue
		}

		// Cari ID anggota jika hanya nama yang diisi
		if idAnggota == "" && namaAnggota != "" {
			var foundID string
			err := db.QueryRow("SELECT id_anggota FROM anggota WHERE LOWER(nama_anggota) = $1 AND status = 'aktif' LIMIT 1", strings.ToLower(namaAnggota)).Scan(&foundID)
			if err == nil {
				idAnggota = foundID
			} else {
				failedCount++
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: tidak dapat menemukan anggota dengan nama '%s'", rowNum, namaAnggota))
				continue
			}
		}

		// Validasi anggota aktif
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM anggota WHERE id_anggota = $1 AND status = 'aktif')", idAnggota).Scan(&exists)
		if err != nil || !exists {
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: anggota '%s' tidak ditemukan atau tidak aktif", rowNum, idAnggota))
			continue
		}

		// Parse jumlah
		jumlahStr = strings.ReplaceAll(jumlahStr, ".", "")
		jumlahStr = strings.ReplaceAll(jumlahStr, ",", "")
		jumlah, err := strconv.ParseFloat(jumlahStr, 64)
		if err != nil || jumlah <= 0 {
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: jumlah tidak valid '%s'", rowNum, getCell(row, idxJumlah)))
			continue
		}

		// Default jenis transaksi ke simpanan jika tidak diisi
		if jenisTransaksi == "" {
			jenisTransaksi = "simpanan"
		}
		if jenisTransaksi == "cicilan" {
			jenisTransaksi = "angsuran"
		}

		switch jenisTransaksi {
		case "simpanan":
			if pendingIDStr != "" {
				pendingID, convErr := strconv.Atoi(pendingIDStr)
				if convErr != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: ID Pending simpanan tidak valid '%s'", rowNum, pendingIDStr))
					continue
				}

				var pending models.Detail
				err := db.QueryRow(`
					SELECT d.id_detail, d.id_anggota, d.id_simpanan, d.jumlah_simpanan, COALESCE(d.metode_pembayaran, '')
					FROM detail d
					WHERE d.id_detail = $1 AND d.status = 'pending'
				`, pendingID).Scan(&pending.IDDetail, &pending.IDAnggota, &pending.IDSimpanan, &pending.JumlahSimpanan, &pending.MetodePembayaran)
				if err != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: data simpanan pending dengan ID %d tidak ditemukan", rowNum, pendingID))
					continue
				}
				if !isPotongGajiEligible(pending.MetodePembayaran) {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: simpanan pending ID %d bukan metode potong gaji", rowNum, pendingID))
					continue
				}
				if err := repository.UpdateSimpananStatus(pendingID, "confirmed"); err != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal mengonfirmasi simpanan pending ID %d", rowNum, pendingID))
					continue
				}
				successCount++
				continue
			}

			// Default jenis simpanan ke wajib jika tidak diisi
			if detail == "" {
				detail = "wajib"
			}
			idSimpanan, ok := idSimpananCache[detail]
			if !ok {
				// Coba cari dengan mengandung substring
				found := false
				for k, v := range idSimpananCache {
					if strings.Contains(k, detail) || strings.Contains(detail, k) {
						idSimpanan = v
						found = true
						break
					}
				}
				if !found {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: jenis simpanan '%s' tidak ditemukan", rowNum, detail))
					continue
				}
			}

			pendingList, err := repository.GetPendingSimpananByCriteria(idAnggota, idSimpanan, jumlah)
			if err != nil {
				failedCount++
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal mengambil simpanan pending (%v)", rowNum, err))
				continue
			}

			confirmedCount := 0
			for _, pending := range pendingList {
				if !isPotongGajiEligible(pending.MetodePembayaran) {
					continue
				}
				if err := repository.UpdateSimpananStatus(pending.IDDetail, "confirmed"); err != nil {
					log.Printf("[ERROR] Gagal update status simpanan pending via import potong gaji (id_detail=%d): %v", pending.IDDetail, err)
					continue
				}
				confirmedCount++
			}

			if confirmedCount == 0 {
				failedCount++
				var metodeList []string
				for _, pending := range pendingList {
					metodeList = append(metodeList, pending.MetodePembayaran)
				}
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: tidak ada simpanan pending metode potong gaji yang cocok (metode ditemukan: %s)", rowNum, strings.Join(metodeList, ", ")))
				continue
			}

		case "angsuran":
			if pendingIDStr != "" {
				pendingID, convErr := strconv.Atoi(pendingIDStr)
				if convErr != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: ID Pending cicilan tidak valid '%s'", rowNum, pendingIDStr))
					continue
				}

				var pending models.Angsuran
				err := db.QueryRow(`
					SELECT a.id_angsuran, p.id_anggota, a.id_pinjaman, a.jumlah_angsuran, COALESCE(p.metode_angsuran, '')
					FROM angsuran a
					JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
					WHERE a.id_angsuran = $1 AND a.status = 'pending'
				`, pendingID).Scan(&pending.IDAngsuran, &pending.IDAnggota, &pending.IDPinjaman, &pending.JumlahAngsuran, &pending.MetodeAngsuran)
				if err != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: data cicilan pending dengan ID %d tidak ditemukan", rowNum, pendingID))
					continue
				}
				if !isPotongGajiEligible(pending.MetodeAngsuran) {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: cicilan pending ID %d bukan metode potong gaji", rowNum, pendingID))
					continue
				}
				if err := repository.UpdateAngsuranStatus(pendingID, "confirmed"); err != nil {
					failedCount++
					parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal mengonfirmasi cicilan pending ID %d", rowNum, pendingID))
					continue
				}
				successCount++
				continue
			}

			// Ambil pinjaman aktif anggota
			pinjamans, err := repository.GetPinjamanAktifByAnggotaID(idAnggota)
			if err != nil || len(pinjamans) == 0 {
				failedCount++
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: tidak ada pinjaman aktif untuk anggota '%s'", rowNum, idAnggota))
				continue
			}

			pinjaman := pinjamans[0]
			idPinjaman := pinjaman.IDPinjaman

			// Jika detail diisi dan berupa angka, gunakan sebagai id_pinjaman
			if detail != "" {
				if parsedID, err := strconv.Atoi(detail); err == nil {
					// Validasi bahwa pinjaman tersebut milik anggota ini
					var valid bool
					db.QueryRow("SELECT EXISTS(SELECT 1 FROM pinjaman WHERE id_pinjaman = $1 AND id_anggota = $2)", parsedID, idAnggota).Scan(&valid)
					if valid {
						idPinjaman = parsedID
					}
				}
			}

			pendingAngsuranList, err := repository.GetPendingAngsuranByCriteria(idAnggota, idPinjaman, jumlah)
			if err != nil {
				failedCount++
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: gagal mengambil cicilan pending (%v)", rowNum, err))
				continue
			}

			confirmedCount := 0
			for _, pending := range pendingAngsuranList {
				if !isPotongGajiEligible(pending.MetodeAngsuran) {
					continue
				}
				if err := repository.UpdateAngsuranStatus(pending.IDAngsuran, "confirmed"); err != nil {
					log.Printf("[ERROR] Gagal update status angsuran pending via import potong gaji (id_angsuran=%d): %v", pending.IDAngsuran, err)
					continue
				}
				confirmedCount++
			}

			if confirmedCount == 0 {
				failedCount++
				var metodeList []string
				for _, pending := range pendingAngsuranList {
					metodeList = append(metodeList, pending.MetodeAngsuran)
				}
				parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: tidak ada cicilan pending metode potong gaji yang cocok (metode ditemukan: %s)", rowNum, strings.Join(metodeList, ", ")))
				continue
			}

		default:
			failedCount++
			parseErrors = append(parseErrors, fmt.Sprintf("Baris %d: jenis transaksi '%s' tidak valid (gunakan Simpanan/Angsuran/Cicilan)", rowNum, jenisTransaksi))
			continue
		}

		successCount++
	}

	if successCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Tidak ada transaksi yang berhasil diimport",
			"success":     successCount,
			"failed":      failedCount,
			"parseErrors": parseErrors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Import transaksi potong gaji berhasil diproses",
		"success":     successCount,
		"failed":      failedCount,
		"parseErrors": parseErrors,
	})
}

// BendaharaDownloadTemplatePotongGajiExcel membuat template Excel dinamis dari simpanan
// pending dan cicilan pending yang menggunakan metode potong gaji.
func BendaharaDownloadTemplatePotongGajiExcel(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] panic saat generate template potong gaji: %v", r)
			c.String(http.StatusInternalServerError, "Gagal membuat template potong gaji")
		}
	}()

	normalizeMetodePotongGaji := func(value string) string {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.ReplaceAll(value, " ", "_")
		return value
	}
	isPotongGajiEligible := func(value string) bool {
		normalized := normalizeMetodePotongGaji(value)
		return normalized == "potong_gaji" || normalized == ""
	}

	pendingSimpanan, err := repository.GetPendingSimpanan()
	if err != nil {
		log.Printf("[ERROR] download template potong gaji: gagal ambil simpanan pending: %v", err)
		c.String(http.StatusInternalServerError, "Gagal mengambil data simpanan pending")
		return
	}

	pendingAngsuran, err := repository.GetPendingAngsuran()
	if err != nil {
		log.Printf("[ERROR] download template potong gaji: gagal ambil cicilan pending: %v", err)
		c.String(http.StatusInternalServerError, "Gagal mengambil data cicilan pending")
		return
	}

	f := excelize.NewFile()
	const sheetName = "Template Potong Gaji"
	defaultSheet := f.GetSheetName(0)
	if defaultSheet != sheetName {
		f.SetSheetName(defaultSheet, sheetName)
	}

	headers := []string{
		"ID Anggota",
		"Nama Anggota",
		"Jenis Transaksi",
		"ID Pending",
		"Detail",
		"Jumlah",
		"Metode",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			log.Printf("[ERROR] download template potong gaji: gagal set header %q di %s: %v", header, cell, err)
			c.String(http.StatusInternalServerError, "Gagal menyusun header template")
			return
		}
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0D6EFD"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err := f.SetCellStyle(sheetName, "A1", "G1", headerStyle); err != nil {
		log.Printf("[WARN] download template potong gaji: gagal set style header: %v", err)
		c.String(http.StatusInternalServerError, "Gagal memformat header template")
		return
	}

	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 4,
	})

	rowIdx := 2
	for _, item := range pendingSimpanan {
		if !isPotongGajiEligible(item.MetodePembayaran) {
			continue
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), item.IDAnggota); err != nil {
			log.Printf("[ERROR] download template potong gaji: gagal set simpanan row %d: %v", rowIdx, err)
			c.String(http.StatusInternalServerError, "Gagal mengisi data template")
			return
		}
		_ = f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), item.NamaAnggota)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), "simpanan")
		_ = f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), item.IDDetail)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), item.Simpanan.JenisSimpanan)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), item.JumlahSimpanan)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), "potong gaji")
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), currencyStyle)
		rowIdx++
	}

	for _, item := range pendingAngsuran {
		if !isPotongGajiEligible(item.MetodeAngsuran) {
			continue
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowIdx), item.IDAnggota); err != nil {
			log.Printf("[ERROR] download template potong gaji: gagal set angsuran row %d: %v", rowIdx, err)
			c.String(http.StatusInternalServerError, "Gagal mengisi data template")
			return
		}
		_ = f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), item.NamaAnggota)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), "angsuran")
		_ = f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), item.IDAngsuran)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), item.IDPinjaman)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), item.JumlahAngsuran)
		_ = f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), "potong gaji")
		_ = f.SetCellStyle(sheetName, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), currencyStyle)
		rowIdx++
	}

	if rowIdx == 2 {
		_ = f.SetCellValue(sheetName, "A2", "Tidak ada data pending dengan metode potong gaji")
	}

	_ = f.SetColWidth(sheetName, "A", "A", 18)
	_ = f.SetColWidth(sheetName, "B", "B", 28)
	_ = f.SetColWidth(sheetName, "C", "C", 18)
	_ = f.SetColWidth(sheetName, "D", "D", 24)
	_ = f.SetColWidth(sheetName, "E", "E", 24)
	_ = f.SetColWidth(sheetName, "F", "F", 16)
	_ = f.SetColWidth(sheetName, "G", "G", 16)

	filename := fmt.Sprintf("template_potong_gaji_%s.xlsx", time.Now().Format("20060102_150405"))
	buf, err := f.WriteToBuffer()
	if err != nil {
		log.Printf("[ERROR] download template potong gaji: gagal write workbook ke buffer: %v", err)
		c.String(http.StatusInternalServerError, "Gagal menghasilkan file template")
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// BendaharaCekDanProsesPemotonganOtomatis endpoint untuk mengecek dan menjalankan pemotongan otomatis
func BendaharaCekDanProsesPemotonganOtomatis(c *gin.Context) {
	now := time.Now()
	tanggalSekarang := now.Day()
	bulan := int(now.Month())
	tahun := now.Year()

	// Ambil konfigurasi
	configData, err := repository.GetKonfigurasiSimpananWajib()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"shouldRun":   false,
			"message":     "Gagal mengambil konfigurasi",
			"tanggal":     tanggalSekarang,
			"statusAktif": false,
		})
		return
	}

	// Cek apakah status aktif
	statusAktif, ok := configData["StatusAktif"].(bool)
	if !ok || !statusAktif {
		c.JSON(http.StatusOK, gin.H{
			"shouldRun":   false,
			"message":     "Pemotongan otomatis tidak aktif",
			"tanggal":     tanggalSekarang,
			"statusAktif": false,
		})
		return
	}

	// Cek tanggal pemotongan
	tanggalPotong, ok := configData["TanggalPotong"].(int)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"shouldRun":     false,
			"message":       "Tanggal pemotongan tidak valid",
			"tanggal":       tanggalSekarang,
			"tanggalPotong": 0,
			"statusAktif":   true,
		})
		return
	}

	// HANYA proses jika tanggal sekarang SAMA DENGAN tanggal pemotongan yang disetting
	// Ini memastikan pemotongan HANYA terjadi di tanggal yang tepat
	if tanggalSekarang != tanggalPotong {
		c.JSON(http.StatusOK, gin.H{
			"shouldRun":     false,
			"message":       fmt.Sprintf("Belum waktunya pemotongan. Pemotongan akan dilakukan pada tanggal %d (sekarang tanggal %d)", tanggalPotong, tanggalSekarang),
			"tanggal":       tanggalSekarang,
			"tanggalPotong": tanggalPotong,
			"statusAktif":   true,
		})
		return
	}

	// Cek apakah ada anggota yang belum diproses bulan ini
	// PERBAIKAN: Cek apakah ada anggota aktif dengan gaji > 0 yang belum diproses DAN belum punya simpanan wajib
	db := config.GetDB()

	// Hitung anggota yang perlu diproses (punya gaji, belum diproses, dan belum punya simpanan wajib)
	var anggotaBelumDiproses int
	countBelumProsesQuery := `
		SELECT COUNT(*) 
		FROM anggota a
		WHERE a.status = 'aktif' 
		  AND a.gaji_bulanan > 0
		  AND NOT EXISTS (
		      SELECT 1 FROM log_pemotongan_simpanan l 
		      WHERE l.id_anggota = a.id_anggota 
		        AND l.bulan = $1 
		        AND l.tahun = $2 
		        AND l.status = 'berhasil'
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM detail d
		      WHERE d.id_anggota = a.id_anggota
		        AND d.id_simpanan = 2
		        AND COALESCE(d.status, 'confirmed') = 'confirmed'
		  )`
	err = db.QueryRow(countBelumProsesQuery, bulan, tahun).Scan(&anggotaBelumDiproses)

	// Jika tidak ada anggota yang perlu diproses, skip
	if err == nil && anggotaBelumDiproses == 0 {
		// Hitung total yang sudah diproses untuk info
		var totalSudahDiproses int
		db.QueryRow(`SELECT COUNT(*) FROM log_pemotongan_simpanan 
		             WHERE bulan = $1 AND tahun = $2 AND status = 'berhasil' AND id_anggota != 'SYSTEM'`,
			bulan, tahun).Scan(&totalSudahDiproses)

		c.JSON(http.StatusOK, gin.H{
			"shouldRun": false,

			"message":          fmt.Sprintf("Semua anggota sudah diproses atau tidak ada yang perlu diproses (%d anggota)", totalSudahDiproses),
			"tanggal":          tanggalSekarang,
			"tanggalPotong":    tanggalPotong,
			"statusAktif":      true,
			"alreadyProcessed": true,
			"processedCount":   totalSudahDiproses,
		})
		return
	}

	// Log jumlah anggota yang akan diproses
	fmt.Printf("🔍 Ditemukan %d anggota yang perlu diproses simpanan wajib\n", anggotaBelumDiproses)

	// Tanggal sekarang = tanggal pemotongan DAN ada anggota yang belum diproses
	// Jalankan proses pemotongan
	fmt.Printf("🤖 Menjalankan pemotongan otomatis untuk bulan %d tahun %d (tanggal: %d, setting: %d)\n",
		bulan, tahun, tanggalSekarang, tanggalPotong)
	successCount, failedCount, errors := repository.ProsesPemotonganSimpananWajib()

	var errorMessage string
	var message string

	if len(errors) > 0 {
		errorMessage = errors[0]
	}

	// Customize message berdasarkan hasil
	if successCount == 1 && failedCount == 0 && errorMessage == "" {
		// Bisa jadi ini hasil dari "tidak ada yang perlu diproses"
		message = fmt.Sprintf("Pemotongan otomatis selesai dicek. Berhasil: %d, Gagal: %d", successCount, failedCount)
	} else if successCount > 0 {
		message = fmt.Sprintf("Pemotongan otomatis berhasil dijalankan! Berhasil: %d, Gagal: %d", successCount, failedCount)
	} else if failedCount > 0 {
		message = fmt.Sprintf("Pemotongan otomatis selesai dengan error. Berhasil: %d, Gagal: %d", successCount, failedCount)
	} else {
		message = "Pemotongan otomatis dijalankan, tidak ada anggota yang perlu diproses"
	}

	c.JSON(http.StatusOK, gin.H{
		"shouldRun":     true,
		"processed":     true,
		"message":       message,
		"tanggal":       tanggalSekarang,
		"tanggalPotong": tanggalPotong,
		"statusAktif":   true,
		"successCount":  successCount,
		"failedCount":   failedCount,
		"errorMessage":  errorMessage,
	})
}

// BendaharaDetailAngsuran menampilkan detail angsuran anggota
func BendaharaDetailAngsuran(c *gin.Context) {
	id := c.Param("id")

	// Cek apakah ID adalah angka (id_angsuran) atau string ID anggota
	if idNum, err := strconv.Atoi(id); err == nil {
		// Ini adalah ID angsuran, redirect ke view detail angsuran
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/bendahara/view-detail-angsuran/%d", idNum))
		return
	}

	anggota, err := repository.GetAnggotaByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Ambil data saldo (total pinjaman yang belum lunas)
	_, totalPinjaman, _, err := repository.GetSaldoAnggota(id)
	if err != nil {
		totalPinjaman = 0
	}

	// Ambil pinjaman aktif anggota
	pinjamans, err := repository.GetPinjamanAktifByAnggotaID(id)
	if err != nil {
		pinjamans = []models.Pinjaman{} // Default kosong jika error
	}

	var jumlahPinjaman float64
	var sisaPinjaman float64
	var angsuranKe int
	var angsurans []models.Angsuran

	if len(pinjamans) > 0 {
		pinjaman := pinjamans[0]
		jumlahPinjaman = pinjaman.JumlahPinjaman
		angsurans, _ = repository.GetAngsuranByPinjamanID(pinjaman.IDPinjaman)
		sisaPinjaman = pinjaman.JumlahPinjaman
		for _, a := range angsurans {
			if a.Status == "confirmed" {
				sisaPinjaman -= a.SisaPinjaman
			}
		}
		if sisaPinjaman < 0 {
			sisaPinjaman = 0 //perbaiki bendahara_laporan.html agar untuk kop nya ada diatas data bukan seperti ini slide 1 sedangkan di slide2 itu datanya seharusnya
		}
		angsuranKe = len(angsurans) + 1
	} else if totalPinjaman > 0 {
		jumlahPinjaman = totalPinjaman
		sisaPinjaman = totalPinjaman
		angsuranKe = 1
	}

	db := config.GetDB()
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

	c.HTML(http.StatusOK, "bendahara_detail_angsuran.html", gin.H{
		"Judul":          "Detail Angsuran Anggota",
		"Anggota":        anggota,
		"JumlahPinjaman": jumlahPinjaman,
		"SisaPinjaman":   sisaPinjaman,
		"AngsuranKe":     angsuranKe,
		"TotalPinjaman":  totalPinjaman,
		"NomorRekening":  nomorRekening,
		"Angsurans":      angsurans,
		"CurrentLogo":    latestLogo,
	})
}

// BendaharaViewDetailPinjaman menampilkan detail pinjaman berdasarkan ID pinjaman (untuk riwayat)
func BendaharaViewDetailPinjaman(c *gin.Context) {
	idPinjaman := c.Param("id")

	db := config.GetDB()
	var p models.Pinjaman
	var a models.Anggota

	query := `
		SELECT p.id_pinjaman, p.id_anggota, p.tgl_pinjaman, p.jumlah_pinjaman, 
		       p.jangka_waktu, p.bunga, p.status,
		       COALESCE(p.metode_pencairan, '') as metode_pencairan,
		       COALESCE(p.nomor_rekening, '') as nomor_rekening,
		       COALESCE(p.nama_bank, '') as nama_bank,
		       COALESCE(p.nama_pemilik_rekening, '') as nama_pemilik_rekening,
		       COALESCE(p.gaji_bulanan, 0) as gaji_bulanan,
		       COALESCE(p.tujuan_pinjaman, '') as tujuan_pinjaman,
		       a.nama_anggota, a.no_telepon, a.nik_ktp, a.username, a.alamat, a.unit_kerja
		FROM pinjaman p
		JOIN anggota a ON p.id_anggota = a.id_anggota
		WHERE p.id_pinjaman = $1
	`

	err := db.QueryRow(query, idPinjaman).Scan(
		&p.IDPinjaman, &p.IDAnggota, &p.TglPinjaman, &p.JumlahPinjaman,
		&p.JangkaWaktu, &p.Bunga, &p.Status,
		&p.MetodePencairan, &p.NomorRekening, &p.NamaBank, &p.NamaPemilikRekening,
		&p.GajiBulanan, &p.TujuanPinjaman,
		&a.NamaAnggota, &a.NoTelepon, &a.NikKTP, &a.Username, &a.Alamat, &a.UnitKerja,
	)

	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Data pinjaman tidak ditemukan"})
		return
	}

	a.IDAnggota = p.IDAnggota
	p.NamaAnggota = a.NamaAnggota

	// Hitung total simpanan untuk menampilkan limit
	totalSimpanan, _, _, err := repository.GetSaldoAnggota(p.IDAnggota)
	if err != nil {
		totalSimpanan = 0
	}

	// Hitung limit pinjaman berdasarkan jenis anggota
	var jenisAnggota string
	switch a.UnitKerja {
	case "03": // Mahasiswa
		jenisAnggota = "Mahasiswa"
	case "01", "02": // Dosen/Tenaga Pendidikan
		jenisAnggota = "Dosen/Tenaga Pendidikan"
	default:
		jenisAnggota = "Tidak Diketahui"
	}

	// Cari logo terbaru
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

	c.HTML(http.StatusOK, "bendahara_view_detail_pinjaman.html", gin.H{
		"Anggota":       a,
		"Pinjaman":      p,
		"TotalSimpanan": totalSimpanan,
		"JenisAnggota":  jenisAnggota,
		"CurrentLogo":   latestLogo,
		"ActivePage":    "riwayat",
	})
}

// BendaharaViewDetailAngsuran menampilkan detail angsuran berdasarkan ID angsuran (untuk riwayat)
func BendaharaViewDetailAngsuran(c *gin.Context) {
	idAngsuran := c.Param("id")

	db := config.GetDB()
	var angsuran models.Angsuran
	var tglBayar sql.NullTime

	query := `
		SELECT a.id_angsuran, a.id_pinjaman, p.id_anggota,
		       a.tgl_bayar, a.jumlah_angsuran, a.sisa_pinjaman,
		       COALESCE(a.bukti_angsuran, '') as bukti_angsuran,
		       COALESCE(a.status, '') as status,
		       ang.nama_anggota, ang.no_telepon,
		       p.jumlah_pinjaman, p.jangka_waktu,
		       COALESCE(p.metode_pencairan, '') as metode_pencairan,
		       COALESCE(p.metode_angsuran, '') as metode_angsuran
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		JOIN anggota ang ON p.id_anggota = ang.id_anggota
		WHERE a.id_angsuran = $1
	`

	var namaAnggota, noTelepon string
	var jumlahPinjaman float64
	var jangkaWaktu int
	var metodePencairan, metodeAngsuran string

	err := db.QueryRow(query, idAngsuran).Scan(
		&angsuran.IDAngsuran, &angsuran.IDPinjaman, &angsuran.IDAnggota,
		&tglBayar, &angsuran.JumlahAngsuran, &angsuran.SisaPinjaman, &angsuran.BuktiAngsuran,
		&angsuran.Status, &namaAnggota, &noTelepon,
		&jumlahPinjaman, &jangkaWaktu, &metodePencairan, &metodeAngsuran,
	)

	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Data angsuran tidak ditemukan"})
		return
	}

	if tglBayar.Valid {
		angsuran.TglBayar = tglBayar.Time
	}
	angsuran.NamaAnggota = namaAnggota
	angsuran.MetodeAngsuran = metodeAngsuran

	// Hitung angsuran ke berapa
	var angsuranKe int
	rows, err := db.Query(`SELECT id_angsuran FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar ASC, id_angsuran ASC`, angsuran.IDPinjaman)
	if err == nil {
		defer rows.Close()
		idx := 1
		for rows.Next() {
			var tmpID int
			if scanErr := rows.Scan(&tmpID); scanErr != nil {
				continue
			}
			if tmpID == angsuran.IDAngsuran {
				angsuranKe = idx
				break
			}
			idx++
		}
	}

	// Ambil semua angsuran untuk riwayat
	angsurans := []models.Angsuran{}
	rows2, err := db.Query(`SELECT id_angsuran, tgl_bayar, jumlah_angsuran, sisa_pinjaman, status, COALESCE(bukti_angsuran, '') FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar ASC, id_angsuran ASC`, angsuran.IDPinjaman)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var a models.Angsuran
			if scanErr := rows2.Scan(&a.IDAngsuran, &a.TglBayar, &a.JumlahAngsuran, &a.SisaPinjaman, &a.Status, &a.BuktiAngsuran); scanErr != nil {
				continue
			}
			angsurans = append(angsurans, a)
		}
	}

	nomorRekening, _ := repository.GetNomorRekening("angsuran")

	// Cari logo terbaru
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

	c.HTML(http.StatusOK, "bendahara_view_detail_angsuran.html", gin.H{
		"Anggota": map[string]interface{}{
			"NamaAnggota": namaAnggota,
			"IDAnggota":   angsuran.IDAnggota,
			"NoTelepon":   noTelepon,
		},
		"JumlahPinjaman":  jumlahPinjaman,
		"Angsuran":        angsuran,
		"SisaPinjaman":    angsuran.SisaPinjaman,
		"AngsuranKe":      angsuranKe,
		"MetodePencairan": metodePencairan,
		"MetodeAngsuran":  metodeAngsuran,
		"NomorRekening":   nomorRekening,
		"Angsurans":       angsurans,
		"CurrentLogo":     latestLogo,
		"ActivePage":      "riwayat",
	})
}

// ApiJenisAngsuran mengembalikan daftar pinjaman aktif/proses milik anggota
func ApiJenisAngsuran(c *gin.Context) {
	idAnggota := c.Query("id_anggota")
	pinjamans, err := repository.GetPinjamanAktifByAnggota(idAnggota)
	var result []map[string]interface{}
	if err != nil || len(pinjamans) == 0 {
		c.JSON(200, gin.H{"jenis_angsuran": result})
		return
	}
	for _, p := range pinjamans {
		label := fmt.Sprintf("Pinjaman #%d - %d bulan - Rp %.0f", p.IDPinjaman, p.JangkaWaktu, p.JumlahPinjaman)
		result = append(result, map[string]interface{}{
			"key":  p.IDPinjaman,
			"nama": label,
		})
	}
	c.JSON(200, gin.H{"jenis_angsuran": result})
}

// BendaharaCatatAngsuran memproses pencatatan angsuran baru oleh bendahara
func BendaharaCatatAngsuran(c *gin.Context) {
	// ...existing code...
	// ...existing code...
	var angsuran models.Angsuran
	var err error
	var pinjaman models.Pinjaman
	session := sessions.Default(c)
	bendaharaID := session.Get("user_id")
	if bendaharaID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if err = c.ShouldBind(&angsuran); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
		return
	}

	// Ambil id_pinjaman dari jenis_angsuran (dropdown)
	idPinjamanStr := c.PostForm("jenis_angsuran")
	idPinjaman, err := strconv.Atoi(idPinjamanStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pinjaman tidak valid"})
		return
	}
	angsuran.IDPinjaman = idPinjaman
	angsuran.IDPengelola.Int64 = int64(bendaharaID.(int))
	// Pastikan status angsuran selalu confirmed (bendahara = validasi langsung)
	angsuran.Status = "confirmed"
	// Default metode angsuran dari form (tunai untuk entri manual)
	if angsuran.MetodeAngsuran == "" {
		angsuran.MetodeAngsuran = "tunai"
	}
	// Ambil data pinjaman untuk mengisi IDAnggota
	pinjaman, err = repository.GetPinjamanByID(idPinjaman)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pinjaman"})
		return
	}
	angsuran.IDAnggota = pinjaman.IDAnggota

	jumlahAngsuranStr := c.PostForm("jumlah_angsuran")
	jumlahAngsuran, err := strconv.ParseFloat(jumlahAngsuranStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah angsuran tidak valid"})
		return
	}
	angsuran.JumlahAngsuran = jumlahAngsuran

	// Ambil sisa pinjaman terakhir dari DB
	pinjaman, err = repository.GetPinjamanByID(idPinjaman)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pinjaman"})
		return
	}
	// Ambil angsuran terakhir yang statusnya confirmed/lunas/diterima (urut ASC)
	angsuransSebelum, _ := repository.GetAngsuranByPinjamanID(idPinjaman)
	sisaSebelum := pinjaman.JumlahPinjaman
	if len(angsuransSebelum) > 0 {
		for i := len(angsuransSebelum) - 1; i >= 0; i-- {
			a := angsuransSebelum[i]
			if a.Status == "confirmed" || a.Status == "lunas" || a.Status == "diterima" {
				sisaSebelum = a.SisaPinjaman
				break
			}
		}
	}
	sisaSetelah := sisaSebelum - jumlahAngsuran
	if sisaSetelah < 0 {
		sisaSetelah = 0
	}
	angsuran.SisaPinjaman = sisaSetelah
	// Entri oleh bendahara dianggap validasi tahap bendahara selesai.
	angsuran.Status = "confirmed"
	// Entri manual Tunai langsung confirmed (tidak perlu konfirmasi lagi)
	// Cari pending angsuran yang cocok, lalu konfirmasi tanpa membuat record baru
	pendingAngsuranList, err := repository.GetPendingAngsuranByCriteria(angsuran.IDAnggota, angsuran.IDPinjaman, angsuran.JumlahAngsuran)
	if err != nil || len(pendingAngsuranList) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak sesuai dengan angsuran pending. Entri manual tunai hanya diperbolehkan untuk data yang sudah ada di daftar angsuran pending."})
		return
	}

	confirmedAngsuranCount := 0
	for _, pending := range pendingAngsuranList {
		if err := repository.UpdateAngsuranStatus(pending.IDAngsuran, "confirmed"); err != nil {
			log.Printf("[ERROR] Gagal update status angsuran pending (id_angsuran=%d): %v", pending.IDAngsuran, err)
			continue
		}
		confirmedAngsuranCount++
	}

	if confirmedAngsuranCount == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengonfirmasi data angsuran pending"})
		return
	}

	// Notifikasi WA ke ketua agar melakukan konfirmasi tahap ketua.
	anggota, errAnggota := repository.GetAnggotaByID(angsuran.IDAnggota)
	if errAnggota == nil {
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		nominal := fmt.Sprintf("%.2f", angsuran.JumlahAngsuran)
		if errWA := sendKetuaWhatsAppTransactionNotification("", anggota.NamaAnggota, "Angsuran", nominal, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF KETUA] gagal kirim dari entri angsuran bendahara: %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF KETUA] gagal ambil data anggota (%s) untuk entri angsuran: %v", angsuran.IDAnggota, errAnggota)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Angsuran berhasil dicatat"})
}

func getKetuaTransactionNotifInfo(transactionType string, id int) (anggotaNama, jenisLabel, nominal string, err error) {
	db := config.GetDB()
	var nominalFloat float64
	switch transactionType {
	case "simpanan":
		jenisLabel = "Simpanan"
		err = db.QueryRow(`
			SELECT COALESCE(a.nama_anggota, ''), COALESCE(d.jumlah_simpanan, 0)
			FROM detail d
			JOIN anggota a ON a.id_anggota = d.id_anggota
			WHERE d.id_detail = $1
			LIMIT 1
		`, id).Scan(&anggotaNama, &nominalFloat)
		if err == nil {
			nominal = fmt.Sprintf("%.2f", nominalFloat)
		}
	case "angsuran":
		jenisLabel = "Angsuran"
		err = db.QueryRow(`
			SELECT COALESCE(ag.nama_anggota, ''), COALESCE(an.jumlah_angsuran, 0)
			FROM angsuran an
			JOIN pinjaman p ON p.id_pinjaman = an.id_pinjaman
			JOIN anggota ag ON ag.id_anggota = p.id_anggota
			WHERE an.id_angsuran = $1
			LIMIT 1
		`, id).Scan(&anggotaNama, &nominalFloat)
		if err == nil {
			nominal = fmt.Sprintf("%.2f", nominalFloat)
		}
	case "pinjaman":
		jenisLabel = "Pinjaman"
		err = db.QueryRow(`
			SELECT COALESCE(a.nama_anggota, ''), COALESCE(p.jumlah_pinjaman, 0)
			FROM pinjaman p
			JOIN anggota a ON a.id_anggota = p.id_anggota
			WHERE p.id_pinjaman = $1
			LIMIT 1
		`, id).Scan(&anggotaNama, &nominalFloat)
		if err == nil {
			nominal = fmt.Sprintf("%.2f", nominalFloat)
		}
	case "pengambilan":
		jenisLabel = "Pengambilan Simpanan"
		err = db.QueryRow(`
			SELECT COALESCE(a.nama_anggota, ''), COALESCE(ps.jumlah, 0)
			FROM pengambilan_simpanan ps
			JOIN anggota a ON a.id_anggota = ps.id_anggota
			WHERE ps.id_pengambilan = $1
			LIMIT 1
		`, id).Scan(&anggotaNama, &nominalFloat)
		if err == nil {
			nominal = fmt.Sprintf("%.2f", nominalFloat)
		}
	default:
		return "", "", "", fmt.Errorf("tipe transaksi tidak didukung: %s", transactionType)
	}

	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(anggotaNama) == "" {
		anggotaNama = "-"
	}
	if strings.TrimSpace(nominal) == "" {
		nominal = "0.00"
	}
	return anggotaNama, jenisLabel, nominal, nil
}

func mustParseFloat(v string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}
