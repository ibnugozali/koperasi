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
	err = db.QueryRow(`SELECT id_angsuran, id_pinjaman, id_anggota, id_pengelola, tgl_bayar, sisa_pinjaman, bukti_angsuran, status_angsuran, status, nama_anggota FROM angsuran WHERE id_angsuran = $1`, id).Scan(
		&angsuran.IDAngsuran, &angsuran.IDPinjaman, &angsuran.IDAnggota, &angsuran.IDPengelola, &angsuran.TglBayar, &angsuran.SisaPinjaman, &angsuran.BuktiAngsuran, &angsuran.StatusAngsuran, &angsuran.Status, &angsuran.NamaAnggota,
	)
	if err != nil {
		c.HTML(http.StatusOK, "error.html", gin.H{"message": "Data angsuran tidak ditemukan"})
		return
	}

	// Ambil data pinjaman terkait
	var jumlahPinjaman float64
	var angsuranKe int
	var nomorRekening string
	err = db.QueryRow(`SELECT jumlah_pinjaman, nomor_rekening FROM pinjaman WHERE id_pinjaman = $1`, angsuran.IDPinjaman).Scan(&jumlahPinjaman, &nomorRekening)
	if err != nil {
		jumlahPinjaman = 0
		nomorRekening = "-"
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
	rows2, _ := db.Query(`SELECT id_angsuran, tgl_bayar, sisa_pinjaman, status FROM angsuran WHERE id_pinjaman = $1 ORDER BY tgl_bayar ASC, id_angsuran ASC`, angsuran.IDPinjaman)
	defer rows2.Close()
	for rows2.Next() {
		var a models.Angsuran
		rows2.Scan(&a.IDAngsuran, &a.TglBayar, &a.SisaPinjaman, &a.Status)
		angsurans = append(angsurans, a)
	}

	c.HTML(http.StatusOK, "ketua/ketua_detail_angsuran.html", gin.H{
		"Anggota":        angsuran,
		"JumlahPinjaman": jumlahPinjaman,
		"SisaPinjaman":   angsuran.SisaPinjaman,
		"AngsuranKe":     angsuranKe,
		"NomorRekening":  nomorRekening,
		"Angsurans":      angsurans,
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
	})
}

// KetuaDownloadLaporan handles download laporan for ketua
func KetuaDownloadLaporan(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	bulan := c.Query("bulan")
	tahun := c.Query("tahun")
	tipeLaporan := c.DefaultQuery("tipe_laporan", "bulanan")

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

	switch format {
	case "excel":
		// Ambil data laporan keuangan
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
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
		if err == nil && len(anggotas) > 0 {
			potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
			if err != nil {
				potonganBulanIni = make(map[string]float64)
			}
			startRow := rowOffset + 5 + len(dataRows) + 2

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
				// Laporan bulanan - 1 tabel saja
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
			}
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

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.AddPage()

		// Jika ada kop gambar, sisipkan di bagian atas
		if kopPath != "" {
			pdf.ImageOptions(kopPath, 10, 10, 190, 0, false, gofpdf.ImageOptions{ImageType: ""}, 0, "")
			pdf.Ln(45) // Tambah jarak agar data tidak bertumpuk dengan kop surat
		}

		// PDF header, keterangan, dan data laporan
		pdf.SetFont("Arial", "B", 16)
		if tipeLaporan == "tahunan" {
			pdf.CellFormat(190, 10, "LAPORAN KEUANGAN TAHUNAN KOPERASI", "0", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(190, 10, "LAPORAN KEUANGAN BULANAN KOPERASI", "0", 1, "C", false, 0, "")
		}
		pdf.Ln(3)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(190, 6, "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak, "0", 1, "C", false, 0, "")
		if tipeLaporan == "tahunan" {
			pdf.CellFormat(190, 6, fmt.Sprintf("Periode: Tahun %d", tahunInt), "0", 1, "C", false, 0, "")
		} else {
			pdf.CellFormat(190, 6, fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt), "0", 1, "C", false, 0, "")
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
				// Laporan bulanan: 1 tabel saja
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

	// Ambil data anggota aktif
	anggotas, err := repository.GetAllAnggota()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "ketua_laporan.html", gin.H{
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

	// Ambil pesan success dari query parameter
	successMsg := c.Query("success")

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":       "laporan",
		"Report":           report,
		"Bulan":            bulan,
		"Tahun":            tahun,
		"TipeLaporan":      tipeLaporan,
		"CurrentLogo":      latestLogo,
		"Anggotas":         anggotas,
		"SisaGaji":         sisaGaji,
		"GetUnitKerjaName": repository.GetUnitKerjaName,
		"success":          successMsg,
		"Error": func() string {
			if err != nil {
				return "Gagal mengambil data anggota"
			} else {
				return ""
			}
		}(),
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
	c.HTML(http.StatusOK, "ketua_anggota_konfirmasi.html", gin.H{
		"PendingMembers": pendingMembers,
		"ActivePage":     "konfirmasi_anggota",
		"CurrentLogo":    latestLogo,
		"Title":          "Konfirmasi Anggota",
		"ErrorMessage":   errorMsg,
	})
}

// KetuaConfirmMembership mengkonfirmasi anggota
func KetuaConfirmMembership(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	// Update status anggota menjadi 'approved'
	err = repository.UpdateAnggotaStatus(strconv.Itoa(id), "approved")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengonfirmasi anggota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Anggota berhasil dikonfirmasi"})
}

// KetuaRejectMembership menolak anggota
func KetuaRejectMembership(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	// Update status anggota menjadi 'rejected'
	err = repository.UpdateAnggotaStatus(strconv.Itoa(id), "rejected")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menolak anggota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Anggota berhasil ditolak"})
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

	userIDInt := userID.(int)
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

	userIDInt := userID.(int)
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
