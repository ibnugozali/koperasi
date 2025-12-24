package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// KetuaDownloadLaporan mengunduh laporan dalam format Excel atau PDF
func KetuaDownloadLaporan(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	bulan := c.Query("bulan")
	tahun := c.Query("tahun")

	switch format {
	case "excel":
		bulanInt, _ := strconv.Atoi(bulan)
		tahunInt, _ := strconv.Atoi(tahun)
		report, err := repository.GetLaporanKeuangan(bulanInt, tahunInt)
		if err != nil {
			c.String(http.StatusInternalServerError, "Gagal mengambil data laporan")
			return
		}

		f := excelize.NewFile()
		sheet := "Sheet1"
		f.SetCellValue(sheet, "A1", "LAPORAN KEUANGAN KOPERASI")
		waktuCetak := time.Now()
		tanggalCetak := waktuCetak.Format("2 Januari 2006")
		jamCetak := waktuCetak.Format("15.04")
		namaBulan := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		f.SetCellValue(sheet, "A2", "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak)
		f.SetCellValue(sheet, "A3", fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt))
		f.SetCellValue(sheet, "A5", "Keterangan")
		f.SetCellValue(sheet, "B5", "Jumlah")
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
			f.SetCellValue(sheet, fmt.Sprintf("A%d", 6+i), row.label)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", 6+i), row.value)
		}
		styleHeader, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2ecc71"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, "A5", "B5", styleHeader)
		styleData, _ := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "left"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, "A6", fmt.Sprintf("B%d", 6+len(dataRows)-1), styleData)
		f.SetColWidth(sheet, "A", "B", 25)
		anggotas, err := repository.GetAllAnggota()
		if err == nil && len(anggotas) > 0 {
			potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
			if err != nil {
				potonganBulanIni = make(map[string]float64)
			}
			startRow := 6 + len(dataRows) + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow-1), "Rincian Laporan Keuangan")
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
		namaBulan := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
		waktuCetak := time.Now()
		tanggalCetak := waktuCetak.Format("2 Januari 2006")
		jamCetak := waktuCetak.Format("15.04")
		totalPengeluaran := 0.0
		if pinjaman, ok := report["total_pinjaman"].(float64); ok {
			totalPengeluaran += pinjaman
		}
		if pengambilan, ok := report["total_pengambilan"].(float64); ok {
			totalPengeluaran += pengambilan
		}
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
		pdf.SetFont("Arial", "B", 16)
		pdf.CellFormat(190, 10, "LAPORAN KEUANGAN KOPERASI", "0", 1, "C", false, 0, "")
		pdf.Ln(3)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(190, 6, "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak, "0", 1, "C", false, 0, "")
		pdf.CellFormat(190, 6, fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt), "0", 1, "C", false, 0, "")
		pdf.Ln(10)
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
		maxLabelWidth += 10
		maxValueWidth += 10
		totalWidth := maxLabelWidth + maxValueWidth
		if totalWidth < 190 {
			maxLabelWidth += 190 - totalWidth
			totalWidth = 190
		}
		pdf.SetFont("Arial", "B", 11)
		pdf.SetFillColor(46, 204, 113)
		pdf.SetTextColor(255, 255, 255)
		headerLabelLines := pdf.SplitLines([]byte("Keterangan"), maxLabelWidth)
		headerValueLines := pdf.SplitLines([]byte("Jumlah"), maxValueWidth)
		headerLines := len(headerLabelLines)
		if len(headerValueLines) > headerLines {
			headerLines = len(headerValueLines)
		}
		headerHeight := float64(headerLines) * 6.0
		pdf.CellFormat(maxLabelWidth, headerHeight, "Keterangan", "1", 0, "C", true, 0, "")
		pdf.CellFormat(maxValueWidth, headerHeight, "Jumlah", "1", 1, "C", true, 0, "")
		pdf.SetFont("Arial", "", 10)
		pdf.SetTextColor(0, 0, 0)
		for _, row := range dataRows {
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
				unitKerjaName := repository.GetUnitKerjaName(anggota.UnitKerja)
				pdf.CellFormat(38, 7, anggota.NamaAnggota, "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, nohp, "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, unitKerjaName, "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, fmt.Sprintf("%d", anggota.GajiBulanan), "1", 0, "L", false, 0, "")
				pdf.CellFormat(38, 7, fmt.Sprintf("%.0f", sisaGaji), "1", 0, "L", false, 0, "")
				pdf.Ln(-1)
			}
		}
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

// Menampilkan halaman riwayat login
func KetuaRiwayat(c *gin.Context) {
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

	c.HTML(http.StatusOK, "ketua_riwayat_login.html", gin.H{
		"ActivePage":   "login_history",
		"LoginHistory": loginHistory,
		"CurrentLogo":  latestLogo,
	})
}

// Menampilkan halaman laporan anggota
func KetuaLaporan(c *gin.Context) {
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
		anggotas = []models.Anggota{}
	}

	// Hitung sisa gaji untuk setiap anggota
	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		totalSimpanan, totalPinjaman, _, _ := repository.GetSaldoAnggota(anggota.IDAnggota)
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - totalSimpanan - totalPinjaman
	}

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":       "laporan",
		"Report":           report,
		"Bulan":            bulan,
		"Tahun":            tahun,
		"CurrentLogo":      latestLogo,
		"Anggotas":         anggotas,
		"SisaGaji":         sisaGaji,
		"GetUnitKerjaName": repository.GetUnitKerjaName,
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

	c.HTML(http.StatusOK, "ketua_layout.html", gin.H{
		"ActivePage":  "pengaturan",
		"Bendahara":   bendahara,
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
