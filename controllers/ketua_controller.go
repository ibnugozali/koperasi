package controllers

import (
	"database/sql"
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
	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// KetuaDetailAngsuran menampilkan detail angsuran berdasarkan IDAngsuran
func KetuaDetailAngsuran(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID angsuran tidak valid"})
		return
	}

	// Ambil data angsuran
	db := config.GetDB()
	var angsuran models.Angsuran
	var tglBayar sql.NullTime
	err = db.QueryRow(`
		SELECT a.id_angsuran, a.id_pinjaman, p.id_anggota, 
		       a.id_pengelola, 
		       a.tgl_bayar, 
		       COALESCE(a.sisa_pinjaman, 0), 
		       COALESCE(a.bukti_angsuran, ''), 
		       COALESCE(a.status_angsuran, ''), 
		       COALESCE(a.status, ''), 
		       ang.nama_anggota 
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		JOIN anggota ang ON p.id_anggota = ang.id_anggota
		WHERE a.id_angsuran = $1`, id).Scan(
		&angsuran.IDAngsuran, &angsuran.IDPinjaman, &angsuran.IDAnggota, &angsuran.IDPengelola,
		&tglBayar, &angsuran.SisaPinjaman, &angsuran.BuktiAngsuran,
		&angsuran.StatusAngsuran, &angsuran.Status, &angsuran.NamaAnggota,
	)
	if err != nil {
		c.HTML(http.StatusOK, "error.html", gin.H{"message": "Data angsuran tidak ditemukan: " + err.Error()})
		return
	}
	if tglBayar.Valid {
		angsuran.TglBayar = tglBayar.Time
	}

	// // Ambil data pinjaman terkait
	// var jumlahPinjaman float64
	// var angsuranKe int
	// var nomorRekening string
	// var metodePencairan string
	// var namaBank string
	// var namaPemilikRekening string
	// err = db.QueryRow(`SELECT jumlah_pinjaman, COALESCE(nomor_rekening, '-'), COALESCE(metode_pencairan, 'tunai'), COALESCE(nama_bank, '-'), COALESCE(nama_pemilik_rekening, '-') FROM pinjaman WHERE id_pinjaman = $1`, angsuran.IDPinjaman).Scan(&jumlahPinjaman, &nomorRekening, &metodePencairan, &namaBank, &namaPemilikRekening)
	// if err != nil {
	// 	jumlahPinjaman = 0
	// 	nomorRekening = "-"
	// 	metodePencairan = "tunai"
	// 	namaBank = "-"
	// 	namaPemilikRekening = "-"
	// }
	//Ambil data pinjaman terkait
	var jumlahPinjaman float64
	var angsuranKe int
	var nomorRekening string
	var metodePencairan string
	err = db.QueryRow(`SELECT jumlah_pinjaman, nomor_rekening, COALESCE(metode_pencairan, 'tunai') FROM pinjaman WHERE id_pinjaman = $1`, angsuran.IDPinjaman).Scan(&jumlahPinjaman, &nomorRekening, &metodePencairan)
	if err != nil {
		jumlahPinjaman = 0
		nomorRekening = "-"
		metodePencairan = "tunai"
	}
	// Hitung angsuran ke-berapa (berdasarkan urutan tgl_bayar)
	rows, _ := db.Query(`SELECT id_angsuran FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar ASC, id_angsuran ASC`, angsuran.IDPinjaman)
	defer rows.Close()
	idx := 1
	for rows.Next() {
		var tmpID int
		rows.Scan(&tmpID)
		if tmpID == angsuran.IDAngsuran {
			angsuranKe = idx
			break
		}
		idx++
	}

	// Ambil semua angsuran untuk riwayat
	angsurans := []models.Angsuran{}
	rows2, _ := db.Query(`SELECT id_angsuran, tgl_bayar, sisa_pinjaman, status, COALESCE(bukti_angsuran, '') FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar ASC, id_angsuran ASC`, angsuran.IDPinjaman)
	defer rows2.Close()
	for rows2.Next() {
		var a models.Angsuran
		rows2.Scan(&a.IDAngsuran, &a.TglBayar, &a.SisaPinjaman, &a.Status, &a.BuktiAngsuran)
		angsurans = append(angsurans, a)
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

	c.HTML(http.StatusOK, "ketua_detail_angsuran.html", gin.H{
		// 		"Anggota":              angsuran,
		// 		"JumlahPinjaman":       jumlahPinjaman,
		// 		"SisaPinjaman":         angsuran.SisaPinjaman,
		// 		"AngsuranKe":           angsuranKe,
		// 		"NomorRekening":        nomorRekening,
		// 		"MetodePencairan":      metodePencairan,
		// 		"NamaBank":             namaBank,
		// 		"NamaPemilikRekening":  namaPemilikRekening,
		// 		"Angsurans":            angsurans,
		// 		"CurrentLogo":          latestLogo,
		// 	})
		// }
		"Anggota":         angsuran,
		"JumlahPinjaman":  jumlahPinjaman,
		"SisaPinjaman":    angsuran.SisaPinjaman,
		"AngsuranKe":      angsuranKe,
		"NomorRekening":   nomorRekening,
		"MetodePencairan": metodePencairan,
		"Angsurans":       angsurans,
		"CurrentLogo":     latestLogo,
	})
}

// KetuaKonfirmasiTransaksiPost menangani konfirmasi/reject transaksi oleh ketua
func KetuaKonfirmasiTransaksiPost(c *gin.Context) {
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

// KetuaLihatPersyaratanPinjaman menampilkan halaman persyaratan ajukan pinjaman untuk anggota (read-only, mirip bendahara)
func KetuaLihatPersyaratanPinjaman(c *gin.Context) {
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

	// Render template persyaratan pinjaman khusus ketua
	c.HTML(http.StatusOK, "ketua/ketua_persyaratan_pinjaman.html", gin.H{
		"Anggota":       anggota,
		"TotalSimpanan": totalSimpanan,
		"LimitPinjaman": limitPinjaman,
		"JenisAnggota":  jenisAnggota,
		"Judul":         "Lihat Persyaratan Pengajuan Pinjaman",
		"Pinjaman":      pinjaman,
		"HasPinjaman":   hasPinjaman,
		"Bunga":         bungaTerkini,
		"CurrentLogo":   latestLogo,
	})
}

// KetuaDownloadLaporan handles download laporan for ketua
func KetuaDownloadLaporan(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	bulan := c.Query("bulan")
	tahun := c.Query("tahun")
	tipeLaporan := c.DefaultQuery("tipe_laporan", "bulanan")

	log.Printf("=== DOWNLOAD LAPORAN START === format=%s, bulan=%s, tahun=%s, tipeLaporan=%s", format, bulan, tahun, tipeLaporan)

	// Ambil path kop dari session (jika ada)
	session := sessions.Default(c)
	kopPath, _ := session.Get("kop_path").(string)
	absKopPath := kopPath
	if kopPath != "" && !filepath.IsAbs(kopPath) {
		absKopPath, _ = filepath.Abs(kopPath)
	}

	// Jika laporan tahunan, bulan tidak diperlukan
	bulanInt := 0
	if tipeLaporan == "bulanan" {
		bulanInt, _ = strconv.Atoi(bulan)
	}

	log.Printf("DEBUG: bulanInt=%d, format=%s", bulanInt, format)

	switch format {
	case "excel":
		log.Printf("DEBUG: Memulai generate Excel...")
		// Ambil data laporan keuangan
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
			log.Printf("ERROR: Gagal GetLaporanKeuangan: %v", err)
			c.String(http.StatusInternalServerError, "Gagal mengambil data laporan")
			return
		}

		f := excelize.NewFile()
		sheet := "Sheet1"
		// Jika ada kop PDF, skip image insert untuk Excel (PDF tidak bisa di embed ke Excel)
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
		if tipeLaporan == "tahunan" {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset), "LAPORAN KEUANGAN TAHUNAN KOPERASI")
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset), "LAPORAN KEUANGAN BULANAN KOPERASI")
		}
		// Tanggal cetak dan periode
		var waktuCetak time.Time
		var tanggalCetak string
		var jamCetak string
		var namaBulan string
		waktuCetak = time.Now()
		tanggalCetak = waktuCetak.Format("2 Januari 2006")
		jamCetak = waktuCetak.Format("15.04")
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+1), "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak)
		if tipeLaporan == "tahunan" {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+2), fmt.Sprintf("Periode: Tahun %d", tahunInt))
		} else {
			namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowOffset+2), fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt))
		}
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
		log.Printf("DEBUG Excel: GetAllAnggota err=%v, len(anggotas)=%d", err, len(anggotas))

		potonganBulanIni := make(map[string]float64)
		if err == nil {
			potonganBulanIni, _ = repository.GetPotonganBulanIniAllAnggota()
		}

		// Ambil laporan detail per anggota untuk tipe bulanan
		var laporanDetail []map[string]interface{}
		if tipeLaporan == "bulanan" {
			laporanDetail, _ = repository.GetLaporanBulananPerAnggota(bulanInt, tahunInt)
		}

		// Selalu buat tabel rincian meskipun tidak ada data anggota
		startRow := rowOffset + 5 + len(dataRows) + 2

		log.Printf("DEBUG Excel: tipeLaporan=%s, startRow=%d", tipeLaporan, startRow)
		log.Printf("DEBUG Excel: tipeLaporan=%s, startRow=%d", tipeLaporan, startRow)

		// Jika ada error atau tidak ada anggota, tetap buat struktur tabel
		if err != nil || len(anggotas) == 0 {
			log.Printf("DEBUG Excel: Membuat tabel dengan pesan 'Tidak ada data' (err=%v, len=%d)", err, len(anggotas))
			// Buat tabel kosong dengan header saja
			if tipeLaporan == "tahunan" {
				// Untuk tahunan, buat 5 tabel dengan pesan "Tidak ada data"
				tableNames := []string{
					"Rincian Simpanan Wajib Tahunan",
					"Rincian Simpanan Sukarela Tahunan",
					"Rincian Pinjaman Tahunan",
					"Rincian Angsuran Tahunan",
					"Rincian Pengambilan Tahunan",
				}
				headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}
				cols := []string{"A", "B", "C", "D", "E"}
				currentRow := startRow

				for _, tableName := range tableNames {
					f.SetCellValue(sheet, fmt.Sprintf("A%d", currentRow-1), tableName)
					for i, h := range headers {
						f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], currentRow), h)
					}
					// Baris pesan "Tidak ada data"
					f.MergeCell(sheet, fmt.Sprintf("A%d", currentRow+1), fmt.Sprintf("E%d", currentRow+1))
					f.SetCellValue(sheet, fmt.Sprintf("A%d", currentRow+1), "Tidak ada data anggota")
					currentRow += 4
				}
			} else {
				// Untuk bulanan, buat tabel Rincian Laporan Bulanan
				log.Printf("DEBUG Excel: Membuat tabel Rincian Laporan Bulanan (tanpa data) di startRow=%d", startRow)
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow-1), "Rincian Laporan Bulanan")
				headers1 := []string{"No", "Kode", "Nama", "Unit", "Pinjaman", "", "", "", "", "", "", "", "Simpanan", "", "", "", "", "", "", "Jumlah Pembayaran"}
				cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T"}

				for i, h := range headers1 {
					if h != "" {
						f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], startRow), h)
					}
				}
				f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("A%d", startRow+1))
				f.MergeCell(sheet, fmt.Sprintf("B%d", startRow), fmt.Sprintf("B%d", startRow+1))
				f.MergeCell(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("C%d", startRow+1))
				f.MergeCell(sheet, fmt.Sprintf("D%d", startRow), fmt.Sprintf("D%d", startRow+1))
				f.MergeCell(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("L%d", startRow))
				f.MergeCell(sheet, fmt.Sprintf("M%d", startRow), fmt.Sprintf("S%d", startRow))
				f.MergeCell(sheet, fmt.Sprintf("T%d", startRow), fmt.Sprintf("T%d", startRow+1))

				headers2 := []string{"", "", "", "", "Nominal Pinjaman", "Periode/Tenor", "Pokok Pinjaman", "Jasa", "Jumlah", "Angsuran ke-1", "Angsuran ke-2", "Sisa Pinjaman", "Pokok", "Wajib", "Jumlah Wajib", "Simpanan Hari Raya", "Jumlah Simpanan Hari Raya", "Sukarela", "Jumlah Sukarela", ""}
				for i := 4; i < len(headers2)-1; i++ {
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], startRow+1), headers2[i])
				}

				// Baris pesan "Tidak ada data"
				f.MergeCell(sheet, fmt.Sprintf("A%d", startRow+2), fmt.Sprintf("T%d", startRow+2))
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow+2), "Tidak ada data anggota")
			}
		} else {
			// Ada data anggota, buat tabel dengan data
			log.Printf("DEBUG Excel: Membuat tabel dengan data anggota (len=%d)", len(anggotas))
			// Hapus deklarasi ulang startRow yang tidak perlu
			// startRow sudah didefinisikan di atas

			// Jika laporan tahunan, buat 5 tabel terpisah
			if tipeLaporan == "tahunan" {
				// Tabel 1: Simpanan Wajib
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow-1), "Rincian Simpanan Wajib Tahunan")
				headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}
				cols := []string{"A", "B", "C", "D", "E"}
				for i, h := range headers {
					col := cols[i]
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow), h)
				}
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

				// Style untuk tabel simpanan wajib
				rincianHeaderStyle1, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0d6efd"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("E%d", startRow), rincianHeaderStyle1)

				// Tabel 2: Simpanan Sukarela
				startRow2 := startRow + len(anggotas) + 3
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow2-1), "Rincian Simpanan Sukarela Tahunan")
				for i, h := range headers {
					col := cols[i]
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow2), h)
				}
				for idx, anggota := range anggotas {
					row := startRow2 + 1 + idx
					nohp := anggota.NoTelepon
					if !strings.HasPrefix(nohp, "+62") {
						nohp = "+62" + strings.TrimLeft(nohp, "0")
					}
					sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
					f.SetCellValue(sheet, fmt.Sprintf("A%d", row), anggota.NamaAnggota)
					f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nohp)
					f.SetCellValue(sheet, fmt.Sprintf("C%d", row), repository.GetUnitKerjaName(anggota.UnitKerja))
					f.SetCellValue(sheet, fmt.Sprintf("D%d", row), int64(anggota.GajiBulanan))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", row), int64(sisaGaji))
				}
				rincianHeaderStyle2, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0dcaf0"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow2), fmt.Sprintf("E%d", startRow2), rincianHeaderStyle2)

				// Tabel 3: Pinjaman
				startRow3 := startRow2 + len(anggotas) + 3
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow3-1), "Rincian Pinjaman Tahunan")
				for i, h := range headers {
					col := cols[i]
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow3), h)
				}
				for idx, anggota := range anggotas {
					row := startRow3 + 1 + idx
					nohp := anggota.NoTelepon
					if !strings.HasPrefix(nohp, "+62") {
						nohp = "+62" + strings.TrimLeft(nohp, "0")
					}
					sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
					f.SetCellValue(sheet, fmt.Sprintf("A%d", row), anggota.NamaAnggota)
					f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nohp)
					f.SetCellValue(sheet, fmt.Sprintf("C%d", row), repository.GetUnitKerjaName(anggota.UnitKerja))
					f.SetCellValue(sheet, fmt.Sprintf("D%d", row), int64(anggota.GajiBulanan))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", row), int64(sisaGaji))
				}
				rincianHeaderStyle3, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#dc3545"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow3), fmt.Sprintf("E%d", startRow3), rincianHeaderStyle3)

				// Tabel 4: Angsuran
				startRow4 := startRow3 + len(anggotas) + 3
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow4-1), "Rincian Angsuran Tahunan")
				for i, h := range headers {
					col := cols[i]
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow4), h)
				}
				for idx, anggota := range anggotas {
					row := startRow4 + 1 + idx
					nohp := anggota.NoTelepon
					if !strings.HasPrefix(nohp, "+62") {
						nohp = "+62" + strings.TrimLeft(nohp, "0")
					}
					sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
					f.SetCellValue(sheet, fmt.Sprintf("A%d", row), anggota.NamaAnggota)
					f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nohp)
					f.SetCellValue(sheet, fmt.Sprintf("C%d", row), repository.GetUnitKerjaName(anggota.UnitKerja))
					f.SetCellValue(sheet, fmt.Sprintf("D%d", row), int64(anggota.GajiBulanan))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", row), int64(sisaGaji))
				}
				rincianHeaderStyle4, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#198754"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow4), fmt.Sprintf("E%d", startRow4), rincianHeaderStyle4)

				// Tabel 5: Pengambilan
				startRow5 := startRow4 + len(anggotas) + 3
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow5-1), "Rincian Pengambilan Tahunan")
				for i, h := range headers {
					col := cols[i]
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, startRow5), h)
				}
				for idx, anggota := range anggotas {
					row := startRow5 + 1 + idx
					nohp := anggota.NoTelepon
					if !strings.HasPrefix(nohp, "+62") {
						nohp = "+62" + strings.TrimLeft(nohp, "0")
					}
					sisaGaji := float64(anggota.GajiBulanan) - potonganBulanIni[anggota.IDAnggota]
					f.SetCellValue(sheet, fmt.Sprintf("A%d", row), anggota.NamaAnggota)
					f.SetCellValue(sheet, fmt.Sprintf("B%d", row), nohp)
					f.SetCellValue(sheet, fmt.Sprintf("C%d", row), repository.GetUnitKerjaName(anggota.UnitKerja))
					f.SetCellValue(sheet, fmt.Sprintf("D%d", row), int64(anggota.GajiBulanan))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", row), int64(sisaGaji))
				}
				rincianHeaderStyle5, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#000000"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ffc107"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow5), fmt.Sprintf("E%d", startRow5), rincianHeaderStyle5)
			} else {
				// Laporan bulanan - Tabel Rincian Laporan Bulanan
				log.Printf("DEBUG Excel: Membuat tabel Rincian Laporan Bulanan (dengan data %d anggota) di startRow=%d", len(anggotas), startRow)

				// Set lebar kolom yang proporsional
				f.SetColWidth(sheet, "A", "A", 5)  // No
				f.SetColWidth(sheet, "B", "B", 10) // Kode
				f.SetColWidth(sheet, "C", "C", 25) // Nama
				f.SetColWidth(sheet, "D", "D", 20) // Unit
				f.SetColWidth(sheet, "E", "E", 15) // Nominal Pinjaman
				f.SetColWidth(sheet, "F", "F", 10) // Tenor
				f.SetColWidth(sheet, "G", "G", 13) // Pokok Pinjaman
				f.SetColWidth(sheet, "H", "H", 13) // Jasa
				f.SetColWidth(sheet, "I", "I", 13) // Jumlah
				f.SetColWidth(sheet, "J", "J", 13) // Angsuran ke-1
				f.SetColWidth(sheet, "K", "K", 13) // Angsuran ke-2
				f.SetColWidth(sheet, "L", "L", 13) // Sisa Pinjaman
				f.SetColWidth(sheet, "M", "M", 12) // Pokok
				f.SetColWidth(sheet, "N", "N", 12) // Wajib
				f.SetColWidth(sheet, "O", "O", 13) // Jumlah Wajib
				f.SetColWidth(sheet, "P", "P", 15) // Simpanan Hari Raya
				f.SetColWidth(sheet, "Q", "Q", 18) // Jumlah Simpanan Hari Raya
				f.SetColWidth(sheet, "R", "R", 12) // Sukarela
				f.SetColWidth(sheet, "S", "S", 15) // Jumlah Sukarela
				f.SetColWidth(sheet, "T", "T", 15) // Jumlah Pembayaran

				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow-1), "Rincian Laporan Bulanan")
				// Header baris 1 (dengan merge cells untuk grup Pinjaman dan Simpanan)
				headers1 := []string{"No", "Kode", "Nama", "Unit", "Pinjaman", "", "", "", "", "", "", "", "Simpanan", "", "", "", "", "", "", "Jumlah Pembayaran"}
				cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T"}

				// Set header baris 1
				for i, h := range headers1 {
					if h != "" {
						f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], startRow), h)
					}
				}

				// Merge cells untuk kolom yang span
				f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("A%d", startRow+1)) // No
				f.MergeCell(sheet, fmt.Sprintf("B%d", startRow), fmt.Sprintf("B%d", startRow+1)) // Kode
				f.MergeCell(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("C%d", startRow+1)) // Nama
				f.MergeCell(sheet, fmt.Sprintf("D%d", startRow), fmt.Sprintf("D%d", startRow+1)) // Unit
				f.MergeCell(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("L%d", startRow))   // Pinjaman (colspan 8)
				f.MergeCell(sheet, fmt.Sprintf("M%d", startRow), fmt.Sprintf("S%d", startRow))   // Simpanan (colspan 7)
				f.MergeCell(sheet, fmt.Sprintf("T%d", startRow), fmt.Sprintf("T%d", startRow+1)) // Jumlah Pembayaran

				// Header baris 2 (detail kolom)
				headers2 := []string{"", "", "", "", "Nominal Pinjaman", "Periode/Tenor", "Pokok Pinjaman", "Jasa", "Jumlah", "Angsuran ke-1", "Angsuran ke-2", "Sisa Pinjaman", "Pokok", "Wajib", "Jumlah Wajib", "Simpanan Hari Raya", "Jumlah Simpanan Hari Raya", "Sukarela", "Jumlah Sukarela", ""}
				for i := 4; i < len(headers2)-1; i++ { // Skip kolom yang sudah di-merge (0-3 dan 19)
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], startRow+1), headers2[i])
				}

				// Style untuk data
				dataCenterStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				dataLeftStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				dataRightStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})

				// Data
				for idx, anggota := range anggotas {
					row := startRow + 2 + idx
					unitKerjaName := repository.GetUnitKerjaName(anggota.UnitKerja)

					// Ambil data detail untuk anggota ini (jika tersedia)
					var detail map[string]interface{}
					if idx < len(laporanDetail) {
						detail = laporanDetail[idx]
					}

					// Kolom identitas
					f.SetCellValue(sheet, fmt.Sprintf("A%d", row), idx+1)
					f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataCenterStyle)

					f.SetCellValue(sheet, fmt.Sprintf("B%d", row), anggota.IDAnggota)
					f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), dataCenterStyle)

					f.SetCellValue(sheet, fmt.Sprintf("C%d", row), anggota.NamaAnggota)
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), dataLeftStyle)

					f.SetCellValue(sheet, fmt.Sprintf("D%d", row), unitKerjaName)
					f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), dataLeftStyle)

					// Kolom Pinjaman (E-L) - menggunakan data dari laporanDetail
					if detail != nil {
						f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("Rp %.0f", detail["pinjaman_bulanan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("%v bulan", detail["jangka_waktu"]))
						f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataCenterStyle)

						f.SetCellValue(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("Rp %.0f", detail["pokok_per_bulan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("G%d", row), fmt.Sprintf("G%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("Rp %.0f", detail["jasa_per_bulan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("H%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("Rp %.0f", detail["jumlah_angsuran_per_bulan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("I%d", row), fmt.Sprintf("I%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("%v", detail["total_angsuran_dibayar"]))
						f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), dataCenterStyle)

						f.SetCellValue(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("%v", detail["sisa_angsuran"]))
						f.SetCellStyle(sheet, fmt.Sprintf("K%d", row), fmt.Sprintf("K%d", row), dataCenterStyle)

						f.SetCellValue(sheet, fmt.Sprintf("L%d", row), fmt.Sprintf("Rp %.0f", detail["sisa_pinjaman"]))
						f.SetCellStyle(sheet, fmt.Sprintf("L%d", row), fmt.Sprintf("L%d", row), dataRightStyle)

						// Kolom Simpanan (M-S)
						f.SetCellValue(sheet, fmt.Sprintf("M%d", row), fmt.Sprintf("Rp %.0f", detail["simpanan_pokok"]))
						f.SetCellStyle(sheet, fmt.Sprintf("M%d", row), fmt.Sprintf("M%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("N%d", row), fmt.Sprintf("Rp %.0f", detail["simpanan_wajib_bulanan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("N%d", row), fmt.Sprintf("N%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("O%d", row), fmt.Sprintf("Rp %.0f", detail["total_simpanan_wajib"]))
						f.SetCellStyle(sheet, fmt.Sprintf("O%d", row), fmt.Sprintf("O%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("P%d", row), fmt.Sprintf("Rp %.0f", detail["simpanan_hariraya_bulanan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("P%d", row), fmt.Sprintf("P%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), fmt.Sprintf("Rp %.0f", detail["total_simpanan_hariraya"]))
						f.SetCellStyle(sheet, fmt.Sprintf("Q%d", row), fmt.Sprintf("Q%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("R%d", row), fmt.Sprintf("Rp %.0f", detail["simpanan_sukarela_bulanan"]))
						f.SetCellStyle(sheet, fmt.Sprintf("R%d", row), fmt.Sprintf("R%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("S%d", row), fmt.Sprintf("Rp %.0f", detail["total_simpanan_sukarela"]))
						f.SetCellStyle(sheet, fmt.Sprintf("S%d", row), fmt.Sprintf("S%d", row), dataRightStyle)

						// Jumlah Pembayaran
						f.SetCellValue(sheet, fmt.Sprintf("T%d", row), fmt.Sprintf("Rp %.0f", detail["total_pembayaran"]))
						f.SetCellStyle(sheet, fmt.Sprintf("T%d", row), fmt.Sprintf("T%d", row), dataRightStyle)
					} else {
						// Default values jika tidak ada detail
						f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "Rp 0")
						f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), dataRightStyle)

						f.SetCellValue(sheet, fmt.Sprintf("F%d", row), "0 bulan")
						f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), dataCenterStyle)

						for col := 'G'; col <= 'L'; col++ {
							f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), "Rp 0")
							f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataRightStyle)
						}

						// Kolom Simpanan (M-S)
						for col := 'M'; col <= 'S'; col++ {
							f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), "Rp 0")
							f.SetCellStyle(sheet, fmt.Sprintf("%c%d", col, row), fmt.Sprintf("%c%d", col, row), dataRightStyle)
						}

						// Jumlah Pembayaran
						f.SetCellValue(sheet, fmt.Sprintf("T%d", row), "Rp 0")
						f.SetCellStyle(sheet, fmt.Sprintf("T%d", row), fmt.Sprintf("T%d", row), dataRightStyle)
					}
				}
				// Style header rincian
				rincianHeaderStyle, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF", Size: 10},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#17a2b8"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})

				// Set style untuk semua header
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("T%d", startRow+1), rincianHeaderStyle)

				// Set tinggi baris header
				f.SetRowHeight(sheet, startRow, 25)
				f.SetRowHeight(sheet, startRow+1, 25)

				rincianDataStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "left"},
					Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
				})
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("T%d", startRow+1), rincianHeaderStyle)
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow+2), fmt.Sprintf("T%d", startRow+1+len(anggotas)), rincianDataStyle)
				f.SetColWidth(sheet, "A", "T", 18)
			}
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
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data laporan")
			return
		}

		// Convert bulan to nama bulan
		var namaBulan string
		if tipeLaporan == "bulanan" && bulanInt > 0 && bulanInt <= 12 {
			namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
				"Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		}
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

		// Cari kop surat gambar terbaru (jpg, jpeg, png)
		kopDir := "static/uploads/kop/"
		files, err := os.ReadDir(kopDir)
		var kopPath string
		var latestTime int64
		for _, file := range files {
			if !file.IsDir() {
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
					info, _ := file.Info()
					if info.ModTime().Unix() > latestTime {
						latestTime = info.ModTime().Unix()
						kopPath = filepath.Join(kopDir, file.Name())
					}
				}
			}
		}

		// Set orientation based on report type
		var orientation string
		var pageWidth float64
		if tipeLaporan == "bulanan" {
			orientation = "L" // Landscape for monthly reports
			pageWidth = 277
		} else {
			orientation = "P" // Portrait for annual reports
			pageWidth = 190
		}

		pdf := gofpdf.New(orientation, "mm", "A4", "")
		pdf.AddPage()

		// Jika ada kop gambar, sisipkan di bagian atas
		if kopPath != "" {
			pdf.ImageOptions(kopPath, 10, 10, pageWidth, 0, false, gofpdf.ImageOptions{ImageType: ""}, 0, "")
			pdf.Ln(45) // Tambah jarak agar data tidak bertumpuk dengan kop surat
		}

		// PDF header, keterangan, dan data laporan
		pdf.SetFont("Arial", "B", 16)
		if tipeLaporan == "tahunan" {
			pdf.CellFormat(pageWidth, 10, "LAPORAN KEUANGAN TAHUNAN KOPERASI", "0", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(pageWidth, 10, "LAPORAN KEUANGAN BULANAN KOPERASI", "0", 1, "C", false, 0, "")
		}
		pdf.Ln(3)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(pageWidth, 6, "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak, "0", 1, "C", false, 0, "")
		if tipeLaporan == "tahunan" {
			pdf.CellFormat(pageWidth, 6, fmt.Sprintf("Periode: Tahun %d", tahunInt), "0", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(pageWidth, 6, fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt), "0", 1, "C", false, 0, "")
		}
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
		// Jika totalWidth < pageWidth, distribusikan sisa ke label
		if totalWidth < pageWidth {
			maxLabelWidth += pageWidth - totalWidth
			totalWidth = pageWidth
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
		potonganBulanIni := make(map[string]float64)
		if err == nil {
			potonganBulanIni, _ = repository.GetPotonganBulanIniAllAnggota()
		}

		// Selalu buat tabel rincian meskipun tidak ada data
		if err != nil || len(anggotas) == 0 {
			// Tidak ada data anggota
			if tipeLaporan == "tahunan" {
				tableNames := []string{
					"Rincian Simpanan Wajib Tahunan",
					"Rincian Simpanan Sukarela Tahunan",
					"Rincian Pinjaman Tahunan",
					"Rincian Angsuran Tahunan",
					"Rincian Pengambilan Tahunan",
				}
				headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}

				for _, tableName := range tableNames {
					pdf.Ln(6)
					pdf.SetFont("Arial", "B", 12)
					pdf.CellFormat(190, 8, tableName, "0", 1, "L", false, 0, "")
					pdf.Ln(1)
					pdf.SetFont("Arial", "B", 10)
					for _, h := range headers {
						pdf.CellFormat(38, 7, h, "1", 0, "C", true, 0, "")
					}
					pdf.Ln(-1)
					pdf.SetFont("Arial", "", 10)
					pdf.CellFormat(190, 7, "Tidak ada data anggota", "1", 1, "C", false, 0, "")
				}
			} else {
				// Laporan bulanan: Tabel Rincian Laporan Bulanan (already in landscape)
				pdf.Ln(10)
				pdf.SetFont("Arial", "B", 12)
				pdf.CellFormat(pageWidth, 8, "Rincian Laporan Bulanan", "0", 1, "L", false, 0, "")
				pdf.Ln(2)
				pdf.SetFont("Arial", "", 10)
				pdf.CellFormat(pageWidth, 7, "Tidak ada data anggota", "1", 1, "C", false, 0, "")
			}
		} else {
			// Ada data anggota

			if tipeLaporan == "tahunan" {
				// Laporan tahunan: 5 tabel terpisah
				headers := []string{"Anggota", "No. HP", "Jenis Unit", "Gaji Bulanan", "Sisa Gaji"}
				tableNames := []string{
					"Rincian Simpanan Wajib Tahunan",
					"Rincian Simpanan Sukarela Tahunan",
					"Rincian Pinjaman Tahunan",
					"Rincian Angsuran Tahunan",
					"Rincian Pengambilan Tahunan",
				}

				for _, tableName := range tableNames {
					pdf.Ln(6)
					pdf.SetFont("Arial", "B", 12)
					pdf.CellFormat(190, 8, tableName, "0", 1, "L", false, 0, "")
					pdf.Ln(1)
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
			} else {
				// Laporan bulanan: Tabel Rincian Laporan Bulanan (already in landscape mode)
				pdf.Ln(10)
				pdf.SetFont("Arial", "B", 14)
				pdf.CellFormat(pageWidth, 10, "Rincian Laporan Bulanan", "0", 1, "C", false, 0, "")
				pdf.Ln(2)

				// Header baris 1 dengan merged cells
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(23, 162, 184) // Info color
				pdf.SetTextColor(255, 255, 255)

				colWidths := []float64{8, 12, 30, 18, 14, 10, 13, 11, 13, 13, 13, 13, 11, 11, 13, 16, 20, 13, 16, 16}

				// Baris 1: Header dengan span
				headers1 := []string{"No", "Kode", "Nama", "Unit"}
				for i, h := range headers1 {
					pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
				}
				// Pinjaman (colspan 8)
				pinjamanWidth := colWidths[4] + colWidths[5] + colWidths[6] + colWidths[7] + colWidths[8] + colWidths[9] + colWidths[10] + colWidths[11]
				pdf.CellFormat(pinjamanWidth, 7, "Pinjaman", "1", 0, "C", true, 0, "")
				// Simpanan (colspan 7)
				simpananWidth := colWidths[12] + colWidths[13] + colWidths[14] + colWidths[15] + colWidths[16] + colWidths[17] + colWidths[18]
				pdf.CellFormat(simpananWidth, 7, "Simpanan", "1", 0, "C", true, 0, "")
				// Jumlah Pembayaran
				pdf.CellFormat(colWidths[19], 7, "Jumlah Bayar", "1", 1, "C", true, 0, "")

				// Baris 2: Detail headers
				// Skip 4 kolom pertama yang sudah di-set di baris 1
				pdf.CellFormat(colWidths[0], 0, "", "", 0, "", false, 0, "")
				pdf.CellFormat(colWidths[1], 0, "", "", 0, "", false, 0, "")
				pdf.CellFormat(colWidths[2], 0, "", "", 0, "", false, 0, "")
				pdf.CellFormat(colWidths[3], 0, "", "", 0, "", false, 0, "")

				// Detail Pinjaman
				pinjamanHeaders := []string{"Nominal", "Tenor", "Pokok", "Jasa", "Jumlah", "Angs-1", "Angs-2", "Sisa"}
				for i, h := range pinjamanHeaders {
					pdf.CellFormat(colWidths[4+i], 7, h, "1", 0, "C", true, 0, "")
				}

				// Detail Simpanan
				simpananHeaders := []string{"Pokok", "Wajib", "Jml Wajib", "Hari Raya", "Jml HR", "Sukarela", "Jml Sukarela"}
				for i, h := range simpananHeaders {
					pdf.CellFormat(colWidths[12+i], 7, h, "1", 0, "C", true, 0, "")
				}

				// Skip kolom Jumlah Pembayaran (sudah di-set di baris 1)
				pdf.CellFormat(colWidths[19], 0, "", "", 1, "", false, 0, "")

				// Data rows
				pdf.SetFont("Arial", "", 6)
				pdf.SetTextColor(0, 0, 0)
				for idx, anggota := range anggotas {
					pdf.CellFormat(colWidths[0], 5, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
					pdf.CellFormat(colWidths[1], 5, anggota.IDAnggota, "1", 0, "C", false, 0, "")
					pdf.CellFormat(colWidths[2], 5, anggota.NamaAnggota, "1", 0, "L", false, 0, "")
					pdf.CellFormat(colWidths[3], 5, repository.GetUnitKerjaName(anggota.UnitKerja), "1", 0, "L", false, 0, "")
					// Data Pinjaman (semua Rp 0)
					for i := 0; i < 8; i++ {
						if i == 1 { // Tenor
							pdf.CellFormat(colWidths[4+i], 5, "0", "1", 0, "C", false, 0, "")
						} else {
							pdf.CellFormat(colWidths[4+i], 5, "Rp 0", "1", 0, "R", false, 0, "")
						}
					}
					// Data Simpanan (semua Rp 0)
					for i := 0; i < 7; i++ {
						pdf.CellFormat(colWidths[12+i], 5, "Rp 0", "1", 0, "R", false, 0, "")
					}
					// Jumlah Pembayaran
					pdf.CellFormat(colWidths[19], 6, "Rp 0", "1", 1, "R", false, 0, "")
				}
			}
		}
		// Set header untuk download
		c.Header("Content-Type", "application/pdf")
		if tipeLaporan == "tahunan" {
			c.Header("Content-Disposition", "attachment; filename=laporan_koperasi_tahunan_"+fmt.Sprintf("%d", tahunInt)+".pdf")
		} else {
			c.Header("Content-Disposition", "attachment; filename=laporan_koperasi_"+fmt.Sprintf("%02d_%d", bulanInt, tahunInt)+".pdf")
		}
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

// ExportLaporanKeuangan handles export logic (stub)
func ExportLaporanKeuangan(c *gin.Context) {
	// ...existing code moved here...
}

// KetuaKonfirmasiTransaksi menampilkan halaman konfirmasi transaksi untuk ketua
func KetuaKonfirmasiTransaksi(c *gin.Context) {
	// Ambil data pending dari repository
	pendingSimpanan, errSimpanan := repository.GetPendingSimpanan()
	pendingPinjaman, errPinjaman := repository.GetPendingPinjaman()
	pendingAngsuran, errAngsuran := repository.GetPendingAngsuran()
	pendingPengambilan, errPengambilan := repository.GetPendingPengambilanSimpanan()

	if errSimpanan != nil || errPinjaman != nil || errAngsuran != nil || errPengambilan != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data konfirmasi transaksi"})
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

	c.HTML(http.StatusOK, "ketua_konfirmasi_transaksi.html", gin.H{
		"ActivePage":         "konfirmasi-transaksi",
		"PendingSimpanan":    pendingSimpanan,
		"PendingPinjaman":    pendingPinjaman,
		"PendingAngsuran":    pendingAngsuran,
		"PendingPengambilan": pendingPengambilan,
		"CurrentLogo":        latestLogo,
	})
}

// Menampilkan dashboard ketua dengan daftar calon anggota
func KetuaDashboard(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
		return
	}

	// Ambil data statistik seperti bendahara
	db := config.GetDB()
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

	// Ambil data aktivitas (riwayat simpanan & pinjaman per hari)
	riwayatSimpanan, _ := repository.GetRiwayatTotalSimpananPerHari(db)
	riwayatPinjaman, _ := repository.GetRiwayatTotalPinjamanPerHari(db)
	aktivitasData := []map[string]interface{}{}
	for _, r := range riwayatSimpanan {
		r["Jenis"] = "Simpanan"
		aktivitasData = append(aktivitasData, r)
	}
	for _, r := range riwayatPinjaman {
		r["Jenis"] = "Pinjaman"
		aktivitasData = append(aktivitasData, r)
	}
	// Fallback jika kosong
	if len(aktivitasData) == 0 {
		aktivitasData = []map[string]interface{}{
			{"Tanggal": time.Now(), "Jenis": "Simpanan", "Jumlah": totalSimpanan},
			{"Tanggal": time.Now(), "Jenis": "Pinjaman", "Jumlah": totalPinjaman},
		}
	}

	// Ambil konten dashboard anggota untuk form edit
	dashboardHalaman, err := repository.GetHalamanBySlug("dashboard_anggota")
	var dashboardKonten map[string]interface{}
	if err == nil {
		json.Unmarshal([]byte(dashboardHalaman.Konten), &dashboardKonten)
	} else {
		dashboardKonten = map[string]interface{}{
			"teks":    "Selamat datang di dashboard anggota.",
			"gambar":  "/static/images/placeholder.png",
			"welcome": "Selamat Datang di Koperasi Wirya",
			"slogan":  "Dari Anggota, Oleh Anggota, dan Untuk Anggota",
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

	c.HTML(http.StatusOK, "ketua_dashboard.html", gin.H{
		"PendingMembers":     pendingMembers,
		"DashboardKonten":    dashboardKonten,
		"ActivePage":         "dashboard",
		"CurrentLogo":        latestLogo,
		"TotalAnggota":       totalAnggota,
		"MenungguKonfirmasi": menungguKonfirmasi,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"TotalAngsuran":      totalAngsuran,
		"TotalPengambilan":   totalPengambilan,
		"AktivitasData":      aktivitasData,
	})
}

// Menampilkan halaman data anggota dengan status 'pending'
func KetuaDataAnggota(c *gin.Context) {
	// Ambil semua anggota (bukan hanya pending)
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"message": "Gagal mengambil data anggota"})
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

	c.HTML(http.StatusOK, "ketua_data_anggota.html", gin.H{
		"Anggotas":    anggotas,
		"ActivePage":  "anggota",
		"CurrentLogo": latestLogo,
	})
}

// KetuaListAnggotaKeluar menampilkan daftar anggota yang sudah keluar
func KetuaListAnggotaKeluar(c *gin.Context) {
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
		c.HTML(http.StatusOK, "ketua_data_anggota_keluar.html", gin.H{
			"Anggotas":    []models.Anggota{},
			"ActivePage":  "anggota_keluar",
			"CurrentLogo": latestLogo,
			"Title":       "Data Anggota Keluar",
			"Error":       "Gagal mengambil data anggota keluar",
		})
		return
	}

	// Render normal
	c.HTML(http.StatusOK, "ketua_data_anggota_keluar.html", gin.H{
		"Anggotas":    anggotas,
		"ActivePage":  "anggota_keluar",
		"CurrentLogo": latestLogo,
		"Title":       "Data Anggota Keluar",
	})
}

// KetuaViewAnggotaKeluar menampilkan detail anggota yang sudah keluar
func KetuaViewAnggotaKeluar(c *gin.Context) {
	idAnggota := c.Param("id")
	anggota, err := repository.GetAnggotaByID(idAnggota)
	if err != nil || anggota.Status != "keluar" {
		c.Redirect(http.StatusFound, "/ketua/anggota/keluar?error=Anggota keluar tidak ditemukan")
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

	c.HTML(http.StatusOK, "ketua_view_anggota_keluar.html", gin.H{
		"Anggota":     anggota,
		"ActivePage":  "anggota_keluar",
		"CurrentLogo": latestLogo,
		"Title":       "Detail Anggota Keluar",
	})
}

// KetuaViewAnggota menampilkan detail anggota untuk ketua
func KetuaViewAnggota(c *gin.Context) {
	idStr := c.Param("id")

	anggota, err := repository.GetAnggotaByID(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"message": "Anggota tidak ditemukan"})
		return
	}

	// Ambil detail simpanan per jenis
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

	// Hitung Total Simpanan dari semua simpanan yang ada
	totalSimpanan := simpananByJenis["pokok"] + simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

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

	c.HTML(http.StatusOK, "ketua_data_anggota_view.html", gin.H{
		"Anggota":           anggota,
		"ActivePage":        "anggota",
		"CurrentLogo":       latestLogo,
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

// Menampilkan halaman riwayat transaksi
func KetuaRiwayat(c *gin.Context) {
	// Ambil semua data riwayat transaksi dari database
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		// Log error detail ke console agar mudah debug
		fmt.Printf("[ERROR] Gagal mengambil data riwayat: %v\n", err)
		c.HTML(http.StatusInternalServerError, "ketua_riwayat_transaksi.html", gin.H{
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

	c.HTML(http.StatusOK, "ketua_riwayat_transaksi.html", gin.H{
		"ActivePage":  "riwayat",
		"Riwayats":    riwayats,
		"Anggotas":    anggotas,
		"CurrentLogo": latestLogo,
	})
}

// Menampilkan halaman laporan anggota
func KetuaLaporan(c *gin.Context) {
	// Ambil tipe laporan dari query parameter (default: bulanan)
	tipeLaporan := c.Query("tipe_laporan")
	if tipeLaporan == "" {
		tipeLaporan = "bulanan"
	}

	// Ambil bulan dan tahun dari query parameter, default bulan dan tahun saat ini
	currentTime := time.Now()
	bulan := int(currentTime.Month())
	tahun := currentTime.Year()

	// Jika laporan tahunan, set bulan ke 0 (tidak digunakan)
	if tipeLaporan == "tahunan" {
		bulan = 0
	} else {
		if b := c.Query("bulan"); b != "" {
			if parsed, err := strconv.Atoi(b); err == nil {
				bulan = parsed
			}
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
		c.HTML(http.StatusInternalServerError, "ketua_laporan.html", gin.H{
			"ActivePage":  "laporan",
			"Error":       "Gagal mengambil laporan",
			"CurrentLogo": latestLogo,
			"Bulan":       bulan,
			"Tahun":       tahun,
		})
		return
	}

	// Ambil data anggota aktif (untuk tabel tahunan)
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		log.Printf("Error getting anggotas: %v", err)
		anggotas = []models.Anggota{}
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

	// Ambil pesan success dari query parameter
	successMsg := c.Query("success")

	// Ambil laporan detail bulanan per anggota
	laporanDetail, err := repository.GetLaporanBulananPerAnggota(bulan, tahun)
	if err != nil {
		log.Printf("Error getting detailed report: %v", err)
		laporanDetail = []map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":       "laporan",
		"Report":           report,
		"Bulan":            bulan,
		"Tahun":            tahun,
		"TipeLaporan":      tipeLaporan,
		"CurrentLogo":      latestLogo,
		"Anggotas":         anggotas,
		"LaporanDetail":    laporanDetail,
		"SisaGaji":         sisaGaji,
		"GetUnitKerjaName": repository.GetUnitKerjaName,
		"success":          successMsg,
	})
}

// BendaharaPengaturan menampilkan halaman pengaturan bendahara
func KetuaPengaturan(c *gin.Context) {
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
		c.HTML(http.StatusInternalServerError, "ketua_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara: " + err.Error(),
			"LogoPath":   latestLogo,
		})
		return
	}

	c.HTML(http.StatusOK, "ketua_pengaturan.html", gin.H{
		"ActivePage":  "pengaturan",
		"Ketua":       bendahara,
		"LogoPath":    latestLogo,
		"CurrentLogo": latestLogo,
	})
}

// UpdateKetuaProfile memproses update username dan password ketua
func UpdateKetuaProfile(c *gin.Context) {

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
		fmt.Println("[UpdateKetuaProfile] Bind error:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid", "detail": err.Error()})
		return
	}

	// Trim spasi pada password dan konfirmasi
	request.Password = strings.TrimSpace(request.Password)
	request.ConfirmPassword = strings.TrimSpace(request.ConfirmPassword)
	// Jika hanya salah satu field password/konfirmasi diisi, tampilkan error
	if (request.Password != "" && request.ConfirmPassword == "") || (request.Password == "" && request.ConfirmPassword != "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password dan konfirmasi password harus diisi bersamaan untuk mengubah password"})
		return
	}
	// Jika keduanya diisi, validasi dan update password
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

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui", "username": request.Username})
}

// KetuaKonfirmasiAnggota menampilkan halaman konfirmasi anggota
func KetuaKonfirmasiAnggota(c *gin.Context) {
	// Panggil repository untuk mendapatkan anggota dengan status 'pending'
	pendingMembers, err := repository.GetPendingAnggota()
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal mengambil data anggota")
		return
	}

	// Ambil anggota yang mengajukan keluar
	pendingKeluar, err := repository.GetPendingAnggotaKeluar()
	if err != nil {
		pendingKeluar = []models.Anggota{} // Set empty jika error
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
	successMsg := c.Query("success")
	c.HTML(http.StatusOK, "ketua_anggota_konfirmasi.html", gin.H{
		"PendingMembers": pendingMembers,
		"PendingKeluar":  pendingKeluar,
		"ActivePage":     "konfirmasi_anggota",
		"CurrentLogo":    latestLogo,
		"Title":          "Konfirmasi Anggota",
		"ErrorMessage":   errorMsg,
		"SuccessMessage": successMsg,
	})
}

// KetuaConfirmMembership mengkonfirmasi anggota
func KetuaConfirmMembership(c *gin.Context) {
	// Ambil id anggota dari URL (ini masih TEMP id)
	tempID := c.Param("id")

	// Ambil data anggota untuk mendapatkan informasi unit_kerja, fakultas, dan tahun
	anggota, err := repository.GetAnggotaByID(tempID)
	if err != nil {
		// Log error detail ke terminal/server log
		fmt.Printf("[ERROR] Gagal mengambil data anggota dengan id %s: %v\n", tempID, err)
		// Redirect ke halaman konfirmasi dengan pesan error
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal mengambil data anggota")
		return
	}

	// Generate ID anggota yang benar berdasarkan unit_kerja, fakultas_code, tahun konfirmasi, dan nomor urut
	db := config.GetDB()

	// Ambil tahun konfirmasi saat ini
	tahunKonfirmasi := time.Now().Format("2006")

	// Ambil nomor urut terakhir secara global (tidak direset per kombinasi)
	var lastNumber int
	query := `SELECT COALESCE(MAX(CAST(nomor_urut AS INTEGER)), 0) FROM anggota WHERE id_anggota NOT LIKE 'TEMP%'`
	err = db.QueryRow(query).Scan(&lastNumber)
	if err != nil {
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal generate nomor urut")
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
		// Log error detail ke terminal/server log
		fmt.Printf("[CONFIRM ANGGOTA ERROR] update anggota: %v\n", err)
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal mengkonfirmasi anggota")
		return
	}

	fmt.Printf("✓ Anggota dengan ID %s berhasil dikonfirmasi dan aktif\n", newIDAnggota)

	c.Redirect(http.StatusFound, "/ketua/konfirmasi?success=Anggota berhasil dikonfirmasi")
}

// KetuaRejectMembership menolak anggota
func KetuaRejectMembership(c *gin.Context) {
	// Ambil id anggota dari URL (ini masih TEMP id)
	tempID := c.Param("id")

	// Hapus anggota dari database
	db := config.GetDB()
	deleteQuery := `DELETE FROM anggota WHERE id_anggota = $1`

	_, err := db.Exec(deleteQuery, tempID)
	if err != nil {
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal menolak pendaftaran anggota")
		return
	}

	// Arahkan kembali ke halaman konfirmasi ketua
	c.Redirect(http.StatusFound, "/ketua/konfirmasi?success=Pendaftaran anggota berhasil ditolak")
}

// KetuaApproveAnggotaKeluar menyetujui permohonan anggota keluar
func KetuaApproveAnggotaKeluar(c *gin.Context) {
	idAnggota := c.Param("id")

	db := config.GetDB()
	// Update status anggota menjadi 'keluar' dan set tanggal keluar
	_, err := db.Exec("UPDATE anggota SET status = 'keluar', status_anggota = 'keluar', tgl_keluar = NOW() WHERE id_anggota = $1", idAnggota)
	if err != nil {
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal menyetujui permohonan keluar")
		return
	}

	c.Redirect(http.StatusFound, "/ketua/konfirmasi?success=Permohonan keluar berhasil disetujui")
}

// KetuaRejectAnggotaKeluar menolak permohonan anggota keluar
func KetuaRejectAnggotaKeluar(c *gin.Context) {
	idAnggota := c.Param("id")

	db := config.GetDB()
	// Kembalikan status_anggota menjadi kosong/null (batalkan permohonan keluar)
	_, err := db.Exec("UPDATE anggota SET status_anggota = NULL WHERE id_anggota = $1", idAnggota)
	if err != nil {
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal menolak permohonan keluar")
		return
	}

	c.Redirect(http.StatusFound, "/ketua/konfirmasi?success=Permohonan keluar berhasil ditolak")
}

// KetuaUploadKop handles kop surat upload for ketua
func KetuaUploadKop(c *gin.Context) {
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
	// Izinkan file gambar (jpg, jpeg, png) untuk kop surat
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
	c.Redirect(http.StatusFound, "/ketua/laporan?success=Kop surat berhasil diupload")
}

// KetuaSaveNeraca saves neraca data to database
func KetuaSaveNeraca(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.NeracaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)

	// Convert userID to int safely
	var userIDInt int
	switch v := userID.(type) {
	case int:
		userIDInt = v
	case string:
		// For ketua, user_id is stored as string (id_pengelola)
		// We can use a default value or fetch from pengelola table
		userIDInt = 1 // Default ketua ID
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	err := neracaRepo.SaveNeraca(&req, userIDInt)
	if err != nil {
		log.Printf("Error saving neraca: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save neraca: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data neraca berhasil disimpan ke database",
	})
}

// KetuaGetNeraca retrieves neraca data from database
func KetuaGetNeraca(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)

	// Convert userID to int safely
	var userIDInt int
	switch v := userID.(type) {
	case int:
		userIDInt = v
	case string:
		userIDInt = 1 // Default ketua ID
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	neraca, err := neracaRepo.GetNeraca(userIDInt)
	if err != nil {
		log.Printf("Error getting neraca: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get neraca: " + err.Error()})
		return
	}

	if neraca == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    nil,
		})
		return
	}

	// Parse JSON strings back to objects
	var data2024, data2023, customItems map[string]interface{}
	var labels, noPerkiraan, headerData map[string]string
	var itemCounter map[string]int
	var deletedItems []string

	json.Unmarshal([]byte(neraca.Data2024), &data2024)
	json.Unmarshal([]byte(neraca.Data2023), &data2023)
	json.Unmarshal([]byte(neraca.Labels), &labels)
	json.Unmarshal([]byte(neraca.NoPerkiraan), &noPerkiraan)
	json.Unmarshal([]byte(neraca.CustomItems), &customItems)
	json.Unmarshal([]byte(neraca.ItemCounter), &itemCounter)
	json.Unmarshal([]byte(neraca.DeletedItems), &deletedItems)
	json.Unmarshal([]byte(neraca.HeaderData), &headerData)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"data_2024":        data2024,
			"data_2023":        data2023,
			"labels":           labels,
			"no_perkiraan":     noPerkiraan,
			"custom_items":     customItems,
			"item_counter":     itemCounter,
			"deleted_items":    deletedItems,
			"header_data":      headerData,
			"last_modified_at": neraca.LastModifiedAt,
		},
	})
}

// KetuaLihatDetailSimpanan menampilkan detail simpanan pending anggota (read-only)
func KetuaLihatDetailSimpanan(c *gin.Context) {
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
	var totalWajib, totalSukarela, totalHariRaya, totalUmrohHaji, totalQurban, grandTotal float64
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
		d.BuktiPembayaran = bukti
		detailSimpanan = append(detailSimpanan, d)

		// Ambil bukti pembayaran pertama yang ada
		if buktiPembayaran == "" && bukti != "" {
			buktiPembayaran = bukti
		}

		// Hitung total per jenis
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

	c.HTML(http.StatusOK, "ketua_detail_simpanan.html", gin.H{
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
		"Judul":           "Detail Simpanan Pending",
		"CurrentLogo":     latestLogo,
	})
}

// KetuaDetailAjukanPengambilan menampilkan detail pengajuan pengambilan simpanan
func KetuaDetailAjukanPengambilan(c *gin.Context) {
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

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "ID tidak valid")
		return
	}

	pengambilan, err := repository.GetPengambilanSimpananByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Data pengambilan tidak ditemukan")
		return
	}

	c.HTML(http.StatusOK, "ketua_detail_ajukan_pengambilan.html", gin.H{
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
		"ActivePage":        "konfirmasi-transaksi",
		"CurrentLogo":       latestLogo,
	})
}
