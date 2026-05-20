package controllers

import (
	// "strconv" // Uncomment jika tahunKonfirmasi integer
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
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
	"github.com/skip2/go-qrcode"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

type buktiTransferGajiView struct {
	models.BuktiTransferGaji
	CountdownText string
	IsExpired     bool
}

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
			err = repository.UpdateSimpananStatus(id, "diterima")
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
			err = repository.UpdateAngsuranStatus(id, "diterima")
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

// getFloat extracts a float64 value from a map[string]interface{} by key, returns 0 if not found or not convertible
func getFloat(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return f
			}
		}
	}
	return 0
}

// getInt extracts an int value from a map[string]interface{} by key, returns 0 if not found or not convertible
func getInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case float32:
			return int(v)
		case string:
			i, err := strconv.Atoi(v)
			if err == nil {
				return i
			}
		}
	}
	return 0
}

func getLatestSignatureFilePath(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return ""
	}

	signDir := "static/uploads/signatures"
	files, err := os.ReadDir(signDir)
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	prefix := "ttd_" + role + "_"
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := strings.ToLower(file.Name())
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		mod := info.ModTime().Unix()
		if mod > latestTime {
			latestTime = mod
			latestPath = filepath.Join(signDir, file.Name())
		}
	}

	return latestPath
}

func isSupportedImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

func applyKopToExcelEveryPage(f *excelize.File, sheet, kopPath string) {
	if strings.TrimSpace(kopPath) == "" || !isSupportedImageFile(kopPath) {
		return
	}

	data, err := os.ReadFile(kopPath)
	if err != nil {
		return
	}

	ext := strings.ToLower(filepath.Ext(kopPath))
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	if err := f.AddHeaderFooterImage(sheet, &excelize.HeaderFooterImageOptions{
		Position:  excelize.HeaderFooterImagePositionCenter,
		File:      data,
		Extension: ext,
		Width:     "500pt",
		Height:    "70pt",
	}); err != nil {
		return
	}

	_ = f.SetHeaderFooter(sheet, &excelize.HeaderFooterOptions{
		OddHeader:   "&C&G",
		FirstHeader: "&C&G",
	})
}

func formatTanggalIndonesia(t time.Time) string {
	bulan := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	b := bulan[int(t.Month())]
	return fmt.Sprintf("%d %s %d", t.Day(), b, t.Year())
}

func getLatestLogoFilePath() string {
	files, err := os.ReadDir("static/images")
	if err != nil {
		return ""
	}

	var latestPath string
	var latestTime int64
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := strings.ToLower(file.Name())
		if !(name == "logo.png" || strings.HasPrefix(name, "logo_")) {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		mod := info.ModTime().Unix()
		if mod > latestTime {
			latestTime = mod
			latestPath = filepath.Join("static/images", file.Name())
		}
	}
	return latestPath
}

func resizeImageNN(src image.Image, targetW, targetH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW <= 0 || srcH <= 0 || targetW <= 0 || targetH <= 0 {
		return dst
	}

	for y := 0; y < targetH; y++ {
		sy := srcBounds.Min.Y + (y*srcH)/targetH
		for x := 0; x < targetW; x++ {
			sx := srcBounds.Min.X + (x*srcW)/targetW
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func buildPublicStaticURL(c *gin.Context, filePath string) string {
	raw := strings.TrimSpace(filePath)
	if raw == "" {
		return ""
	}

	normalized := strings.ReplaceAll(raw, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "./")

	if strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://") {
		return normalized
	}

	urlPath := normalized
	if strings.HasPrefix(normalized, "static/") {
		urlPath = "/" + normalized
	} else if !strings.HasPrefix(normalized, "/") {
		urlPath = "/" + normalized
	}

	proto := "http"
	if c.GetHeader("X-Forwarded-Proto") != "" {
		proto = strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]
		proto = strings.TrimSpace(proto)
	} else if c.Request != nil && c.Request.TLS != nil {
		proto = "https"
	}

	host := ""
	if c.Request != nil {
		host = c.Request.Host
	}
	if strings.TrimSpace(host) == "" {
		host = "localhost:8081"
	}

	return fmt.Sprintf("%s://%s%s", proto, host, urlPath)
}

func generateSignatureQRCodePath(role, signerName, imageURL, logoPath string) string {
	if strings.TrimSpace(imageURL) == "" {
		return ""
	}

	payload := strings.TrimSpace(imageURL)
	logoKey := ""
	if strings.TrimSpace(logoPath) != "" {
		if info, err := os.Stat(logoPath); err == nil {
			logoKey = logoPath + "|" + strconv.FormatInt(info.ModTime().Unix(), 10)
		}
	}
	cacheKey := payload + "|" + strings.TrimSpace(role) + "|" + strings.TrimSpace(signerName) + "|" + logoKey
	sum := sha1.Sum([]byte(cacheKey))
	fileName := "kopma_ttd_qr_" + hex.EncodeToString(sum[:]) + ".png"
	qrPath := filepath.Join(os.TempDir(), fileName)

	if _, err := os.Stat(qrPath); err == nil {
		return qrPath
	}

	qrBytes, err := qrcode.Encode(payload, qrcode.Medium, 260)
	if err != nil {
		return ""
	}

	qrImg, _, err := image.Decode(bytes.NewReader(qrBytes))
	if err != nil {
		return ""
	}

	dst := image.NewRGBA(qrImg.Bounds())
	draw.Draw(dst, dst.Bounds(), qrImg, qrImg.Bounds().Min, draw.Src)

	if strings.TrimSpace(logoPath) != "" {
		if logoFile, err := os.Open(logoPath); err == nil {
			defer logoFile.Close()
			if logoImg, _, err := image.Decode(logoFile); err == nil {
				qrW := dst.Bounds().Dx()
				qrH := dst.Bounds().Dy()
				logoSize := qrW / 5
				if qrH < qrW {
					logoSize = qrH / 5
				}
				if logoSize > 0 {
					resizedLogo := resizeImageNN(logoImg, logoSize, logoSize)
					x := (qrW - logoSize) / 2
					y := (qrH - logoSize) / 2

					padding := logoSize / 8
					bg := image.Rect(x-padding, y-padding, x+logoSize+padding, y+logoSize+padding).Intersect(dst.Bounds())
					draw.Draw(dst, bg, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
					draw.Draw(dst, image.Rect(x, y, x+logoSize, y+logoSize), resizedLogo, image.Point{}, draw.Over)
				}
			}
		}
	}

	out, err := os.Create(qrPath)
	if err != nil {
		return ""
	}
	defer out.Close()
	if err := png.Encode(out, dst); err != nil {
		return ""
	}

	return qrPath
}

func getSignatureDisplayNames() map[string]string {
	db := config.GetDB()
	getValue := func(key, def string) string {
		var value string
		err := db.QueryRow("SELECT nilai FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&value)
		if err != nil || strings.TrimSpace(value) == "" {
			return def
		}
		return value
	}
	return map[string]string{
		"ketua":      getValue("ttd_nama_ketua", "Ketua KOPMA"),
		"bendahara":  getValue("ttd_nama_bendahara", "Bendahara"),
		"sekretaris": getValue("ttd_nama_sekretaris", "Sekretaris"),
	}
}

func addSignatureBlockToExcel(f *excelize.File, sheet string, startRow int, tipeLaporan string, signatures map[string]string, names map[string]string) {
	if startRow < 1 {
		startRow = 1
	}

	dateStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	labelStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	nameStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Underline: "single"},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Gunakan blok kolom C-N agar sejajar dengan tabel bulanan yang sudah dicenter.
	startCol := "C"
	endCol := "N"

	tanggalLokasi := "Indramayu, " + formatTanggalIndonesia(time.Now())
	f.SetCellValue(sheet, fmt.Sprintf("M%d", startRow), tanggalLokasi)
	f.MergeCell(sheet, fmt.Sprintf("M%d", startRow), fmt.Sprintf("N%d", startRow))
	f.SetCellStyle(sheet, fmt.Sprintf("M%d", startRow), fmt.Sprintf("N%d", startRow), dateStyle)

	f.SetCellValue(sheet, fmt.Sprintf("%s%d", startCol, startRow+1), "Mengetahui,")
	f.MergeCell(sheet, fmt.Sprintf("%s%d", startCol, startRow+1), fmt.Sprintf("%s%d", endCol, startRow+1))
	f.SetCellStyle(sheet, fmt.Sprintf("%s%d", startCol, startRow+1), fmt.Sprintf("%s%d", endCol, startRow+1), titleStyle)

	jabatanRow := startRow + 3
	imgRow := startRow + 4
	nameRow := startRow + 8

	addImage := func(cell string, imagePath string) {
		if imagePath == "" || !isSupportedImageFile(imagePath) {
			return
		}
		_ = f.AddPicture(sheet, cell, imagePath, &excelize.GraphicOptions{
			AutoFit: false,
			ScaleX:  0.32,
			ScaleY:  0.32,
			OffsetX: 12,
			OffsetY: 2,
		})
	}

	if tipeLaporan == "tahunan" {
		f.SetCellValue(sheet, fmt.Sprintf("C%d", jabatanRow), "Ketua KOPMA,")
		f.SetCellValue(sheet, fmt.Sprintf("G%d", jabatanRow), "Bendahara,")
		f.SetCellValue(sheet, fmt.Sprintf("K%d", jabatanRow), "Sekretaris,")
		f.MergeCell(sheet, fmt.Sprintf("C%d", jabatanRow), fmt.Sprintf("F%d", jabatanRow))
		f.MergeCell(sheet, fmt.Sprintf("G%d", jabatanRow), fmt.Sprintf("J%d", jabatanRow))
		f.MergeCell(sheet, fmt.Sprintf("K%d", jabatanRow), fmt.Sprintf("N%d", jabatanRow))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", jabatanRow), fmt.Sprintf("N%d", jabatanRow), labelStyle)

		addImage(fmt.Sprintf("C%d", imgRow), signatures["ketua"])
		addImage(fmt.Sprintf("G%d", imgRow), signatures["bendahara"])
		addImage(fmt.Sprintf("K%d", imgRow), signatures["sekretaris"])

		f.SetCellValue(sheet, fmt.Sprintf("C%d", nameRow), names["ketua"])
		f.SetCellValue(sheet, fmt.Sprintf("G%d", nameRow), names["bendahara"])
		f.SetCellValue(sheet, fmt.Sprintf("K%d", nameRow), names["sekretaris"])
		f.MergeCell(sheet, fmt.Sprintf("C%d", nameRow), fmt.Sprintf("F%d", nameRow))
		f.MergeCell(sheet, fmt.Sprintf("G%d", nameRow), fmt.Sprintf("J%d", nameRow))
		f.MergeCell(sheet, fmt.Sprintf("K%d", nameRow), fmt.Sprintf("N%d", nameRow))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", nameRow), fmt.Sprintf("N%d", nameRow), nameStyle)
	} else {
		f.SetCellValue(sheet, fmt.Sprintf("C%d", jabatanRow), "Ketua KOPMA,")
		f.MergeCell(sheet, fmt.Sprintf("C%d", jabatanRow), fmt.Sprintf("F%d", jabatanRow))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", jabatanRow), fmt.Sprintf("F%d", jabatanRow), labelStyle)

		// Rapikan area QR agar center dan tidak terlalu besar
		_ = f.SetRowHeight(sheet, imgRow, 20)
		_ = f.SetRowHeight(sheet, imgRow+1, 20)
		// Geser QR ke tengah area C-F (kolom E lebih center)
		addImage(fmt.Sprintf("E%d", imgRow), signatures["ketua"])

		f.SetCellValue(sheet, fmt.Sprintf("C%d", nameRow), names["ketua"])
		f.MergeCell(sheet, fmt.Sprintf("C%d", nameRow), fmt.Sprintf("F%d", nameRow))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", nameRow), fmt.Sprintf("F%d", nameRow), nameStyle)
	}
}

func addSignatureBlockToPDF(pdf *gofpdf.Fpdf, tipeLaporan string, signatures map[string]string, names map[string]string) {
	if pdf.GetY() > 230 {
		pdf.AddPage()
	}

	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 6, "Indramayu, "+formatTanggalIndonesia(time.Now()), "0", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(190, 7, "Mengetahui,", "0", 1, "C", false, 0, "")
	pdf.Ln(2)

	addSignArea := func(x, y, w float64, title, imgPath, caption string) {
		pdf.SetXY(x, y)
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(w, 6, title, "0", 1, "C", false, 0, "")

		if imgPath != "" && isSupportedImageFile(imgPath) {
			pdf.ImageOptions(imgPath, x+(w-28)/2, y+6, 28, 0, false, gofpdf.ImageOptions{ImageType: ""}, 0, "")
		}

		pdf.SetXY(x, y+34)
		pdf.SetFont("Arial", "BU", 9)
		pdf.CellFormat(w, 6, caption, "0", 1, "C", false, 0, "")
	}

	baseY := pdf.GetY()
	if tipeLaporan == "tahunan" {
		pageLeft := 10.0
		pageWidth := 190.0
		sectionWidth := pageWidth / 3.0

		addSignArea(pageLeft+(sectionWidth*0), baseY, sectionWidth, "Ketua KOPMA,", signatures["ketua"], names["ketua"])
		addSignArea(pageLeft+(sectionWidth*1), baseY, sectionWidth, "Bendahara,", signatures["bendahara"], names["bendahara"])
		addSignArea(pageLeft+(sectionWidth*2), baseY, sectionWidth, "Sekretaris,", signatures["sekretaris"], names["sekretaris"])
		pdf.SetY(baseY + 42)
	} else {
		addSignArea(10, baseY, 56, "Ketua KOPMA,", signatures["ketua"], names["ketua"])
		pdf.SetY(baseY + 42)
	}
}

func resolveNeracaOwnerID(c *gin.Context) int {
	// Untuk halaman/admin endpoint, pakai akun ketua aktif agar tampilan identik dengan ketua.
	if strings.HasPrefix(c.Request.URL.Path, "/admin/") {
		db := config.GetDB()
		var ketuaID int
		err := db.QueryRow("SELECT id_pengelola FROM pengelola WHERE level = 'ketua' AND status = 'aktif' ORDER BY id_pengelola ASC LIMIT 1").Scan(&ketuaID)
		if err == nil && ketuaID > 0 {
			return ketuaID
		}
		return 1
	}

	session := sessions.Default(c)
	userID := session.Get("user_id")
	switch v := userID.(type) {
	case int:
		return v
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 1
}

// KetuaDetailAngsuran menampilkan detail angsuran berdasarkan IDAngsuran
func KetuaDetailAngsuran(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "ID angsuran tidak valid"})
		return
	}

	db := config.GetDB()
	var angsuran models.Angsuran
	var tglBayar sql.NullTime
	err = db.QueryRow(`
		       SELECT a.id_angsuran, a.id_pinjaman, p.id_anggota, a.id_pengelola, a.tgl_bayar, a.jumlah_angsuran, COALESCE(a.sisa_pinjaman, 0), COALESCE(a.bukti_angsuran, ''), COALESCE(a.status_angsuran, ''), COALESCE(a.status, ''), ang.nama_anggota
		       FROM angsuran a
		       JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		       JOIN anggota ang ON p.id_anggota = ang.id_anggota
		       WHERE a.id_angsuran = $1`, id).Scan(
		&angsuran.IDAngsuran, &angsuran.IDPinjaman, &angsuran.IDAnggota, &angsuran.IDPengelola,
		&tglBayar, &angsuran.JumlahAngsuran, &angsuran.SisaPinjaman, &angsuran.BuktiAngsuran,
		&angsuran.StatusAngsuran, &angsuran.Status, &angsuran.NamaAnggota,
	)
	if err != nil {
		log.Printf("[ERROR] KetuaDetailAngsuran data tidak ditemukan (id=%d): %v", id, err)
		c.HTML(http.StatusOK, "error.html", gin.H{"message": "Data angsuran tidak ditemukan"})
		return
	}
	if tglBayar.Valid {
		angsuran.TglBayar = tglBayar.Time
	}

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
	signatures := map[string]string{
		"ketua":      getLatestSignatureFilePath("ketua"),
		"bendahara":  getLatestSignatureFilePath("bendahara"),
		"sekretaris": getLatestSignatureFilePath("sekretaris"),
	}
	signatureNames := getSignatureDisplayNames()
	signatureDisplayImages := map[string]string{
		"ketua":      signatures["ketua"],
		"bendahara":  signatures["bendahara"],
		"sekretaris": signatures["sekretaris"],
	}
	logoPath := getLatestLogoFilePath()
	for _, role := range []string{"ketua", "bendahara", "sekretaris"} {
		imageURL := buildPublicStaticURL(c, signatures[role])
		if qrPath := generateSignatureQRCodePath(role, signatureNames[role], imageURL, logoPath); qrPath != "" {
			signatureDisplayImages[role] = qrPath
		}
	}

	// Jika laporan tahunan, bulan tidak diperlukan
	bulanInt := 0
	if tipeLaporan == "bulanan" {
		bulanInt, _ = strconv.Atoi(bulan)
	}

	// Untuk laporan tahunan, prioritaskan angka dari data Neraca yang disimpan user.
	type summaryRow struct {
		Label string
		Value float64
	}
	type neracaItem struct {
		No    string
		Label string
		V2024 float64
		V2023 float64
	}
	neracaSummaryRows := []summaryRow{}
	neracaAsetLancar := []neracaItem{}
	neracaAsetTetap := []neracaItem{}
	neracaKewajibanLancar := []neracaItem{}
	neracaEkuitas := []neracaItem{}
	neracaTotalAsetLancar2024, neracaTotalAsetLancar2023 := 0.0, 0.0
	neracaTotalAsetTetap2024, neracaTotalAsetTetap2023 := 0.0, 0.0
	neracaTotalKewajibanLancar2024, neracaTotalKewajibanLancar2023 := 0.0, 0.0
	neracaTotalEkuitas2024, neracaTotalEkuitas2023 := 0.0, 0.0
	useNeracaSummary := false

	toFloat := func(v interface{}) float64 {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		case string:
			clean := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(n, "Rp", ""), ".", ""), ",", "")
			clean = strings.TrimSpace(clean)
			f, _ := strconv.ParseFloat(clean, 64)
			return f
		default:
			return 0
		}
	}

	if tipeLaporan == "tahunan" {
		userIDInt := resolveNeracaOwnerID(c)

		db := config.GetDB()
		neracaRepo := repository.NewNeracaRepository(db)
		neraca, err := neracaRepo.GetNeraca(userIDInt)
		if err != nil {
			log.Printf("WARN: gagal ambil neraca untuk download tahunan: %v", err)
		}
		if neraca != nil {
			var data2024 map[string]interface{}
			var data2023 map[string]interface{}
			var customItems map[string]interface{}
			var deletedItems []string
			var labels map[string]string
			var noPerkiraan map[string]string

			_ = json.Unmarshal([]byte(neraca.Data2024), &data2024)
			_ = json.Unmarshal([]byte(neraca.Data2023), &data2023)
			_ = json.Unmarshal([]byte(neraca.CustomItems), &customItems)
			_ = json.Unmarshal([]byte(neraca.DeletedItems), &deletedItems)
			_ = json.Unmarshal([]byte(neraca.Labels), &labels)
			_ = json.Unmarshal([]byte(neraca.NoPerkiraan), &noPerkiraan)
			if labels == nil {
				labels = map[string]string{}
			}
			if noPerkiraan == nil {
				noPerkiraan = map[string]string{}
			}

			deletedSet := map[string]bool{}
			for _, id := range deletedItems {
				deletedSet[id] = true
			}

			defaultFields := map[string][]string{
				"asetLancar":      {"kas", "bank", "piutangAnggota", "perlengkapan"},
				"asetTetap":       {"tanah", "bangunan", "kendaraan", "peralatan"},
				"kewajibanLancar": {"hutangUsaha", "hutangBunga", "simpananAnggota"},
				"ekuitas":         {"simpananPokok", "simpananWajib", "cadangan", "shuTahunBerjalan"},
			}
			defaultLabelMap := map[string]string{
				"kas":              "Kas",
				"bank":             "Bank",
				"piutangAnggota":   "Piutang Anggota USP",
				"perlengkapan":     "Perlengkapan",
				"tanah":            "Tanah",
				"bangunan":         "Bangunan",
				"kendaraan":        "Kendaraan",
				"peralatan":        "Peralatan",
				"hutangUsaha":      "Hutang Usaha",
				"hutangBunga":      "Hutang Bunga",
				"simpananAnggota":  "Simpanan Anggota",
				"simpananPokok":    "Simpanan Pokok",
				"simpananWajib":    "Simpanan Wajib",
				"cadangan":         "Cadangan",
				"shuTahunBerjalan": "SHU Tahun Berjalan",
			}

			collectCategoryItems := func(kategori string) []neracaItem {
				items := []neracaItem{}
				for _, field := range defaultFields[kategori] {
					if deletedSet[field] {
						continue
					}
					lbl := strings.TrimSpace(labels[field])
					if lbl == "" {
						lbl = defaultLabelMap[field]
					}
					if strings.TrimSpace(lbl) == "" {
						lbl = field
					}
					no := strings.TrimSpace(noPerkiraan[field])
					v24 := toFloat(data2024[field])
					v23 := toFloat(data2023[field])
					hasMeta := strings.TrimSpace(no) != "" || (strings.TrimSpace(labels[field]) != "" && strings.TrimSpace(labels[field]) != defaultLabelMap[field])
					if v24 == 0 && v23 == 0 && !hasMeta {
						continue
					}
					items = append(items, neracaItem{
						No:    no,
						Label: lbl,
						V2024: v24,
						V2023: v23,
					})
				}
				if raw, ok := customItems[kategori]; ok {
					if arr, ok := raw.([]interface{}); ok {
						for _, item := range arr {
							m, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							id, _ := m["id"].(string)
							if id != "" && deletedSet[id] {
								continue
							}
							lbl := strings.TrimSpace(fmt.Sprintf("%v", m["label"]))
							if lbl == "" || lbl == "<nil>" {
								lbl = strings.TrimSpace(labels[id])
							}
							if lbl == "" || lbl == "<nil>" {
								lbl = id
							}
							no := strings.TrimSpace(fmt.Sprintf("%v", m["no"]))
							if no == "" || no == "<nil>" {
								no = strings.TrimSpace(noPerkiraan[id])
							}
							v24 := toFloat(m["value"])
							v23 := toFloat(m["value2023"])
							if strings.TrimSpace(lbl) == "" && v24 == 0 && v23 == 0 {
								continue
							}
							items = append(items, neracaItem{
								No:    no,
								Label: lbl,
								V2024: v24,
								V2023: v23,
							})
						}
					}
				}
				return items
			}
			neracaAsetLancar = collectCategoryItems("asetLancar")
			neracaAsetTetap = collectCategoryItems("asetTetap")
			neracaKewajibanLancar = collectCategoryItems("kewajibanLancar")
			neracaEkuitas = collectCategoryItems("ekuitas")
			sumItems := func(items []neracaItem) (float64, float64) {
				a, b := 0.0, 0.0
				for _, it := range items {
					a += it.V2024
					b += it.V2023
				}
				return a, b
			}
			neracaTotalAsetLancar2024, neracaTotalAsetLancar2023 = sumItems(neracaAsetLancar)
			neracaTotalAsetTetap2024, neracaTotalAsetTetap2023 = sumItems(neracaAsetTetap)
			neracaTotalKewajibanLancar2024, neracaTotalKewajibanLancar2023 = sumItems(neracaKewajibanLancar)
			neracaTotalEkuitas2024, neracaTotalEkuitas2023 = sumItems(neracaEkuitas)
			totalAset := neracaTotalAsetLancar2024 + neracaTotalAsetTetap2024
			totalKewajibanEkuitas := neracaTotalKewajibanLancar2024 + neracaTotalEkuitas2024
			neracaSummaryRows = []summaryRow{
				{Label: "Total Aset Lancar", Value: neracaTotalAsetLancar2024},
				{Label: "Total Aset Tetap", Value: neracaTotalAsetTetap2024},
				{Label: "Total Kewajiban Lancar", Value: neracaTotalKewajibanLancar2024},
				{Label: "Total Ekuitas", Value: neracaTotalEkuitas2024},
				{Label: "TOTAL ASET", Value: totalAset},
				{Label: "TOTAL KEWAJIBAN & EKUITAS", Value: totalKewajibanEkuitas},
			}
			// Tetap gunakan format laporan tahunan klasik (ringkasan + rincian anggota)
			// agar output download sesuai format yang diharapkan.
			useNeracaSummary = false
		}
	}

	switch format {
	case "excel":
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
		applyKopToExcelEveryPage(f, sheet, absKopPath)
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
		// Header judul (dipusatkan seperti PDF)
		titleStartCol := "C"
		titleEndCol := "N"
		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		})
		if tipeLaporan == "tahunan" {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset), "LAPORAN KEUANGAN TAHUNAN KOPERASI")
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset), "LAPORAN KEUANGAN BULANAN KOPERASI")
		}
		f.MergeCell(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset), fmt.Sprintf("%s%d", titleEndCol, rowOffset))
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset), fmt.Sprintf("%s%d", titleEndCol, rowOffset), titleStyle)
		// Tanggal cetak dan periode
		var waktuCetak time.Time
		var tanggalCetak string
		var jamCetak string
		var namaBulan string
		waktuCetak = time.Now()
		tanggalCetak = waktuCetak.Format("2 Januari 2006")
		jamCetak = waktuCetak.Format("15.04")
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+1), "Dicetak pada: "+tanggalCetak+" pukul "+jamCetak)
		f.MergeCell(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+1), fmt.Sprintf("%s%d", titleEndCol, rowOffset+1))
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+1), fmt.Sprintf("%s%d", titleEndCol, rowOffset+1), titleStyle)
		if tipeLaporan == "tahunan" {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+2), fmt.Sprintf("Periode: Tahun %d", tahunInt))
		} else {
			namaBulan = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}[bulanInt]
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+2), fmt.Sprintf("Periode: %s %d", namaBulan, tahunInt))
		}
		f.MergeCell(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+2), fmt.Sprintf("%s%d", titleEndCol, rowOffset+2))
		f.SetCellStyle(sheet, fmt.Sprintf("%s%d", titleStartCol, rowOffset+2), fmt.Sprintf("%s%d", titleEndCol, rowOffset+2), titleStyle)
		// Header tabel
		f.SetColWidth(sheet, "A", "B", 2)
		f.SetColWidth(sheet, "C", "C", 28)
		f.SetColWidth(sheet, "D", "D", 18)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowOffset+4), "Keterangan")
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowOffset+4), "Jumlah")
		// Data
		totalPengeluaran := 0.0
		if pinjaman, ok := report["total_pinjaman"].(float64); ok {
			totalPengeluaran = pinjaman
		}
		dataRows := []struct {
			label string
			value interface{}
		}{}
		if useNeracaSummary {
			for _, r := range neracaSummaryRows {
				dataRows = append(dataRows, struct {
					label string
					value interface{}
				}{label: r.Label, value: r.Value})
			}
		} else {
			dataRows = []struct {
				label string
				value interface{}
			}{
				{"Total Simpanan", report["total_simpanan"]},
				{"Total Pinjaman", report["total_pinjaman"]},
				{"Total Angsuran", report["total_angsuran"]},
				{"Total Pengeluaran", totalPengeluaran},
			}
		}
		for i, row := range dataRows {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowOffset+5+i), row.label)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowOffset+5+i), row.value)
		}
		// Style header tabel
		styleHeader, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#2ecc71"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowOffset+4), fmt.Sprintf("D%d", rowOffset+4), styleHeader)
		// Style data
		styleData, _ := f.NewStyle(&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "left"},
			Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
		})
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowOffset+5), fmt.Sprintf("D%d", rowOffset+5+len(dataRows)-1), styleData)
		// Set lebar kolom ringkasan
		f.SetColWidth(sheet, "C", "D", 24)

		// Untuk laporan tahunan, samakan format rincian dengan tampilan halaman ketua/laporan (rincianTahunan).
		if tipeLaporan == "tahunan" && !useNeracaSummary {
			laporanDetail, _ := repository.GetLaporanBulananPerAnggota(0, tahunInt)
			startRow := rowOffset + 5 + len(dataRows) + 2

			tableBorderStyle, _ := f.NewStyle(&excelize.Style{
				Alignment: &excelize.Alignment{Horizontal: "left"},
				Border: []excelize.Border{
					{Type: "left", Color: "#000000", Style: 1},
					{Type: "right", Color: "#000000", Style: 1},
					{Type: "top", Color: "#000000", Style: 1},
					{Type: "bottom", Color: "#000000", Style: 1},
				},
			})
			tableLeftStyle, _ := f.NewStyle(&excelize.Style{
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
				Border: []excelize.Border{
					{Type: "left", Color: "#000000", Style: 1},
					{Type: "right", Color: "#000000", Style: 1},
					{Type: "top", Color: "#000000", Style: 1},
					{Type: "bottom", Color: "#000000", Style: 1},
				},
			})
			tableCenterStyle, _ := f.NewStyle(&excelize.Style{
				Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
				Border: []excelize.Border{
					{Type: "left", Color: "#000000", Style: 1},
					{Type: "right", Color: "#000000", Style: 1},
					{Type: "top", Color: "#000000", Style: 1},
					{Type: "bottom", Color: "#000000", Style: 1},
				},
			})
			tableRightStyle, _ := f.NewStyle(&excelize.Style{
				Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
				Border: []excelize.Border{
					{Type: "left", Color: "#000000", Style: 1},
					{Type: "right", Color: "#000000", Style: 1},
					{Type: "top", Color: "#000000", Style: 1},
					{Type: "bottom", Color: "#000000", Style: 1},
				},
			})
			makeHeaderStyle := func(color string) int {
				styleID, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1},
						{Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1},
						{Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				return styleID
			}

			writeTable := func(title string, headers []string, rows [][]string, headerColor string, row int) int {
				cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
				lastCol := cols[len(headers)-1]
				f.SetCellValue(sheet, fmt.Sprintf("A%d", row), title)
				headerRow := row + 1
				for i, h := range headers {
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], headerRow), h)
				}
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", headerRow), fmt.Sprintf("%s%d", lastCol, headerRow), makeHeaderStyle(headerColor))
				_ = f.SetRowHeight(sheet, headerRow, 24)

				dataStart := headerRow + 1
				if len(rows) == 0 {
					f.MergeCell(sheet, fmt.Sprintf("A%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart))
					f.SetCellValue(sheet, fmt.Sprintf("A%d", dataStart), "Tidak ada data")
					f.SetCellStyle(sheet, fmt.Sprintf("A%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart), tableBorderStyle)
					return dataStart + 2
				}

				for i, r := range rows {
					cur := dataStart + i
					for j, v := range r {
						cell := fmt.Sprintf("%s%d", cols[j], cur)
						f.SetCellValue(sheet, cell, v)
						switch {
						case j == 0 || j == 1:
							f.SetCellStyle(sheet, cell, cell, tableCenterStyle)
						case j == 2:
							f.SetCellStyle(sheet, cell, cell, tableLeftStyle)
						case j == 3:
							f.SetCellStyle(sheet, cell, cell, tableCenterStyle)
						case strings.HasPrefix(strings.TrimSpace(v), "Rp "):
							f.SetCellStyle(sheet, cell, cell, tableRightStyle)
						default:
							f.SetCellStyle(sheet, cell, cell, tableLeftStyle)
						}
					}
					_ = f.SetRowHeight(sheet, cur, 20)
				}
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart+len(rows)-1), tableBorderStyle)
				return dataStart + len(rows) + 2
			}

			toText := func(v interface{}) string {
				if v == nil {
					return "-"
				}
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s == "" || s == "<nil>" {
					return "-"
				}
				return s
			}
			toRupiah := func(v float64) string {
				return fmt.Sprintf("Rp %.0f", v)
			}

			// Neraca format 2 sisi: Aset (kiri) vs Kewajiban & Ekuitas (kanan)
			if len(neracaAsetLancar)+len(neracaAsetTetap)+len(neracaKewajibanLancar)+len(neracaEkuitas) > 0 {
				headerDark, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1f232a"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				headerBlue, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1f6feb"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				headerGreen, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#198754"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				sectionBlue, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#dbeafe"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "left"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				sectionGreen, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#dcfce7"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "left"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				totalStyle, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Bold: true},
					Fill:      excelize.Fill{Type: "pattern", Color: []string{"#fef3c7"}, Pattern: 1},
					Alignment: &excelize.Alignment{Horizontal: "left"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				numStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "right"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				centerStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "center"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})
				textStyle, _ := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: "left"},
					Border: []excelize.Border{
						{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1},
						{Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1},
					},
				})

				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), "Neraca Koperasi Simpan Pinjam")
				f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("H%d", startRow))
				startRow++

				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), "NoPerkiraan")
				f.SetCellValue(sheet, fmt.Sprintf("B%d", startRow), "Perkiraan")
				f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), "ASET")
				f.MergeCell(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("D%d", startRow))
				f.SetCellValue(sheet, fmt.Sprintf("E%d", startRow), "NoPerkiraan")
				f.SetCellValue(sheet, fmt.Sprintf("F%d", startRow), "Perkiraan")
				f.SetCellValue(sheet, fmt.Sprintf("G%d", startRow), "KEWAJIBAN & EKUITAS")
				f.MergeCell(sheet, fmt.Sprintf("G%d", startRow), fmt.Sprintf("H%d", startRow))
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("B%d", startRow), headerDark)
				f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("D%d", startRow), headerBlue)
				f.SetCellStyle(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("F%d", startRow), headerDark)
				f.SetCellStyle(sheet, fmt.Sprintf("G%d", startRow), fmt.Sprintf("H%d", startRow), headerGreen)

				startRow++
				f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), "2024 (Rp)")
				f.SetCellValue(sheet, fmt.Sprintf("D%d", startRow), "2023 (Rp)")
				f.SetCellValue(sheet, fmt.Sprintf("G%d", startRow), "2024 (Rp)")
				f.SetCellValue(sheet, fmt.Sprintf("H%d", startRow), "2023 (Rp)")
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("B%d", startRow), headerDark)
				f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("D%d", startRow), headerBlue)
				f.SetCellStyle(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("F%d", startRow), headerDark)
				f.SetCellStyle(sheet, fmt.Sprintf("G%d", startRow), fmt.Sprintf("H%d", startRow), headerGreen)

				writePair := func(leftTitle string, left []neracaItem, lTotal24, lTotal23 float64, rightTitle string, right []neracaItem, rTotal24, rTotal23 float64) {
					startRow++
					f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), leftTitle)
					f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("D%d", startRow))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", startRow), rightTitle)
					f.MergeCell(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("H%d", startRow))
					f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("D%d", startRow), sectionBlue)
					f.SetCellStyle(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("H%d", startRow), sectionGreen)

					maxLen := len(left)
					if len(right) > maxLen {
						maxLen = len(right)
					}
					for i := 0; i < maxLen; i++ {
						startRow++
						if i < len(left) {
							f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), left[i].No)
							f.SetCellValue(sheet, fmt.Sprintf("B%d", startRow), left[i].Label)
							f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), toRupiah(left[i].V2024))
							f.SetCellValue(sheet, fmt.Sprintf("D%d", startRow), toRupiah(left[i].V2023))
						}
						if i < len(right) {
							f.SetCellValue(sheet, fmt.Sprintf("E%d", startRow), right[i].No)
							f.SetCellValue(sheet, fmt.Sprintf("F%d", startRow), right[i].Label)
							f.SetCellValue(sheet, fmt.Sprintf("G%d", startRow), toRupiah(right[i].V2024))
							f.SetCellValue(sheet, fmt.Sprintf("H%d", startRow), toRupiah(right[i].V2023))
						}
						f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("A%d", startRow), centerStyle)
						f.SetCellStyle(sheet, fmt.Sprintf("B%d", startRow), fmt.Sprintf("B%d", startRow), textStyle)
						f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("D%d", startRow), numStyle)
						f.SetCellStyle(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("E%d", startRow), centerStyle)
						f.SetCellStyle(sheet, fmt.Sprintf("F%d", startRow), fmt.Sprintf("F%d", startRow), textStyle)
						f.SetCellStyle(sheet, fmt.Sprintf("G%d", startRow), fmt.Sprintf("H%d", startRow), numStyle)
					}

					startRow++
					f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), "Total "+leftTitle)
					f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("B%d", startRow))
					f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), toRupiah(lTotal24))
					f.SetCellValue(sheet, fmt.Sprintf("D%d", startRow), toRupiah(lTotal23))
					f.SetCellValue(sheet, fmt.Sprintf("E%d", startRow), "Total "+rightTitle)
					f.MergeCell(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("F%d", startRow))
					f.SetCellValue(sheet, fmt.Sprintf("G%d", startRow), toRupiah(rTotal24))
					f.SetCellValue(sheet, fmt.Sprintf("H%d", startRow), toRupiah(rTotal23))
					f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("D%d", startRow), sectionBlue)
					f.SetCellStyle(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("H%d", startRow), sectionGreen)
				}

				// Samakan dengan tampilan web: jika baris kanan kurang, item ekuitas awal bisa "naik"
				// ke blok Kewajiban Lancar agar sejajar dengan Aset Lancar.
				rightKewajiban := append([]neracaItem{}, neracaKewajibanLancar...)
				rightEkuitas := append([]neracaItem{}, neracaEkuitas...)
				totalKewajiban24, totalKewajiban23 := neracaTotalKewajibanLancar2024, neracaTotalKewajibanLancar2023
				totalEkuitas24, totalEkuitas23 := neracaTotalEkuitas2024, neracaTotalEkuitas2023
				if len(rightKewajiban) < len(neracaAsetLancar) && len(rightEkuitas) > 0 {
					need := len(neracaAsetLancar) - len(rightKewajiban)
					if need > len(rightEkuitas) {
						need = len(rightEkuitas)
					}
					for i := 0; i < need; i++ {
						it := rightEkuitas[i]
						rightKewajiban = append(rightKewajiban, it)
						totalKewajiban24 += it.V2024
						totalKewajiban23 += it.V2023
						totalEkuitas24 -= it.V2024
						totalEkuitas23 -= it.V2023
					}
					rightEkuitas = rightEkuitas[need:]
				}

				writePair(
					"Aset Lancar", neracaAsetLancar, neracaTotalAsetLancar2024, neracaTotalAsetLancar2023,
					"Kewajiban Lancar", rightKewajiban, totalKewajiban24, totalKewajiban23,
				)
				writePair(
					"Aset Tetap", neracaAsetTetap, neracaTotalAsetTetap2024, neracaTotalAsetTetap2023,
					"Ekuitas/Modal", rightEkuitas, totalEkuitas24, totalEkuitas23,
				)

				startRow++
				totalAset24 := neracaTotalAsetLancar2024 + neracaTotalAsetTetap2024
				totalAset23 := neracaTotalAsetLancar2023 + neracaTotalAsetTetap2023
				totalKE24 := totalKewajiban24 + totalEkuitas24
				totalKE23 := totalKewajiban23 + totalEkuitas23
				f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow), "TOTAL ASET")
				f.MergeCell(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("B%d", startRow))
				f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), toRupiah(totalAset24))
				f.SetCellValue(sheet, fmt.Sprintf("D%d", startRow), toRupiah(totalAset23))
				f.SetCellValue(sheet, fmt.Sprintf("E%d", startRow), "TOTAL KEWAJIBAN & EKUITAS")
				f.MergeCell(sheet, fmt.Sprintf("E%d", startRow), fmt.Sprintf("F%d", startRow))
				f.SetCellValue(sheet, fmt.Sprintf("G%d", startRow), toRupiah(totalKE24))
				f.SetCellValue(sheet, fmt.Sprintf("H%d", startRow), toRupiah(totalKE23))
				f.SetCellStyle(sheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("H%d", startRow), totalStyle)
				f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("D%d", startRow), numStyle)
				f.SetCellStyle(sheet, fmt.Sprintf("G%d", startRow), fmt.Sprintf("H%d", startRow), numStyle)

				startRow += 2
			}

			// 1) Daftar Simpanan Anggota
			rowsSimpanan := [][]string{}
			for i, d := range laporanDetail {
				rowsSimpanan = append(rowsSimpanan, []string{
					fmt.Sprintf("%d", i+1),
					toText(d["id_anggota"]),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRupiah(getFloat(d, "simpanan_pokok")),
					toRupiah(getFloat(d, "total_simpanan_wajib")),
					toRupiah(getFloat(d, "total_simpanan_hariraya")),
					"Rp 0",
					"Rp 0",
					toRupiah(getFloat(d, "total_simpanan_sukarela")),
				})
			}
			startRow = writeTable(
				"Daftar Simpanan Anggota",
				[]string{"No", "ID Anggota", "Nama", "Unit", "Simpanan Pokok", "Simpanan Wajib", "Simpanan Hari Raya", "Simpanan Umroh/Haji", "Simpanan Qurban", "Simpanan Sukarela"},
				rowsSimpanan,
				"#0d6efd",
				startRow,
			)

			// 2) Daftar Piutang Anggota
			rowsPiutang := [][]string{}
			for i, d := range laporanDetail {
				rowsPiutang = append(rowsPiutang, []string{
					fmt.Sprintf("%d", i+1),
					toText(d["id_anggota"]),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRupiah(getFloat(d, "sisa_pinjaman")),
				})
			}
			startRow = writeTable(
				"Daftar Piutang Anggota",
				[]string{"No", "ID Anggota", "Nama", "Unit", "Sisa Piutang"},
				rowsPiutang,
				"#0dcaf0",
				startRow,
			)

			// 3) Daftar Piutang Macet Anggota
			rowsMacet := [][]string{}
			for _, d := range laporanDetail {
				if getFloat(d, "sisa_pinjaman") > 0 {
					rowsMacet = append(rowsMacet, []string{
						fmt.Sprintf("%d", len(rowsMacet)+1),
						toText(d["id_anggota"]),
						toText(d["nama_anggota"]),
						repository.GetUnitKerjaName(toText(d["unit_kerja"])),
						toRupiah(getFloat(d, "sisa_pinjaman")),
					})
				}
			}
			startRow = writeTable(
				"Daftar Piutang Macet Anggota",
				[]string{"No", "ID Anggota", "Nama", "Unit", "Jumlah Piutang"},
				rowsMacet,
				"#dc3545",
				startRow,
			)

			// 4) Daftar SHU Anggota
			rowsSHU := [][]string{}
			for i, d := range laporanDetail {
				shuPinjaman := getFloat(d, "shu_pinjaman")
				shuSimpanan := getFloat(d, "shu_simpanan")
				jumlahSHU := getFloat(d, "jumlah_shu")
				rowsSHU = append(rowsSHU, []string{
					fmt.Sprintf("%d", i+1),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRupiah(shuPinjaman),
					toRupiah(shuSimpanan),
					toRupiah(jumlahSHU),
				})
			}
			_ = writeTable(
				"Daftar SHU Anggota",
				[]string{"No", "Nama", "Unit", "SHU Pinjaman", "SHU Simpanan", "Jumlah SHU"},
				rowsSHU,
				"#198754",
				startRow,
			)

			// Lebar kolom menyesuaikan tabel tahunan
			f.SetColWidth(sheet, "A", "A", 8)
			f.SetColWidth(sheet, "B", "B", 16)
			f.SetColWidth(sheet, "C", "C", 24)
			f.SetColWidth(sheet, "D", "D", 16)
			f.SetColWidth(sheet, "E", "J", 18)
			rowsForSign, _ := f.GetRows(sheet)
			addSignatureBlockToExcel(f, sheet, len(rowsForSign)+3, tipeLaporan, signatureDisplayImages, signatureNames)

			c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			c.Header("Content-Disposition", "attachment; filename=laporan_koperasi.xlsx")
			c.Header("Content-Transfer-Encoding", "binary")
			if err := f.Write(c.Writer); err != nil {
				c.String(http.StatusInternalServerError, "Gagal membuat file Excel")
			}
			return
		}

		// Untuk tahunan berbasis neraca, jangan tampilkan rincian anggota agar output konsisten dengan data neraca.
		if !(tipeLaporan == "tahunan" && useNeracaSummary) {
			// Ambil data anggota aktif dan potongan/sisa gajian bulan ini untuk semua anggota, agar bisa buat tabel rincian per anggota di bagian bawah.
			anggotas, err := repository.GetAllAnggota()

			potonganBulanIni := make(map[string]float64)
			if err == nil {
				potonganBulanIni, _ = repository.GetPotonganBulanIniAllAnggota()
			}
			// Tambahan agar laporanDetail terdefinisi
			var laporanDetail []map[string]interface{}
			if tipeLaporan == "bulanan" {
				laporanDetail, _ = repository.GetLaporanBulananPerAnggota(bulanInt, tahunInt)
			}

			// Selalu buat tabel rincian meskipun tidak ada data
			startRow := rowOffset + 5 + len(dataRows) + 2

			// Jika ada error atau tidak ada anggota, tetap buat struktur tabel
			if err != nil || len(anggotas) == 0 {
				// Buat tabel kosong dengan header saja
				if tipeLaporan == "tahunan" {
					// Untuk tahunan, buat 5 tabel dengan pesan "Tidak ada data"
					tableNames := []string{
						"Rincian Simpanan Wajib Tahunan",
						"Rincian Simpanan Sukarela Tahunan",
						"Rincian Pinjaman Tahunan",
						"Rincian Angsuran Tahunan",
						"Rincian Penarikan Tahunan",
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
					// Untuk bulanan, tampilkan 2 tabel kosong (Pinjaman + Simpanan) dengan posisi center
					f.SetColWidth(sheet, "A", "B", 2)
					f.SetColWidth(sheet, "C", "C", 8)
					f.SetColWidth(sheet, "D", "D", 14)
					f.SetColWidth(sheet, "E", "E", 24)
					f.SetColWidth(sheet, "F", "F", 16)
					f.SetColWidth(sheet, "G", "N", 14)

					titleStyle, _ := f.NewStyle(&excelize.Style{
						Font:      &excelize.Font{Bold: true},
						Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
					})
					headerStyle, _ := f.NewStyle(&excelize.Style{
						Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
						Fill:      excelize.Fill{Type: "pattern", Color: []string{"#17a2b8"}, Pattern: 1},
						Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
						Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
					})
					headerStyleBlue, _ := f.NewStyle(&excelize.Style{
						Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
						Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0d6efd"}, Pattern: 1},
						Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
						Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
					})
					borderStyle, _ := f.NewStyle(&excelize.Style{
						Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
						Border:    []excelize.Border{{Type: "left", Color: "#000000", Style: 1}, {Type: "right", Color: "#000000", Style: 1}, {Type: "top", Color: "#000000", Style: 1}, {Type: "bottom", Color: "#000000", Style: 1}},
					})

					cols := []string{"C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"}
					lastCol := "N"
					// Rincian Pinjaman
					f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow), "Rincian Pinjaman")
					f.MergeCell(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("%s%d", lastCol, startRow))
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow), fmt.Sprintf("%s%d", lastCol, startRow), titleStyle)
					headersPinjam := []string{"No", "Kode", "Nama", "Unit", "Nominal", "Tenor", "Pokok", "Jasa", "Jumlah", "Angs.", "S.A.", "S.P."}
					for i, h := range headersPinjam {
						f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], startRow+1), h)
					}
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow+1), fmt.Sprintf("%s%d", lastCol, startRow+1), headerStyle)
					f.MergeCell(sheet, fmt.Sprintf("C%d", startRow+2), fmt.Sprintf("%s%d", lastCol, startRow+2))
					f.SetCellValue(sheet, fmt.Sprintf("C%d", startRow+2), "Tidak ada data anggota")
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", startRow+2), fmt.Sprintf("%s%d", lastCol, startRow+2), borderStyle)

					nextRow := startRow + 4
					// Rincian Simpanan
					f.SetCellValue(sheet, fmt.Sprintf("C%d", nextRow), "Rincian Simpanan")
					f.MergeCell(sheet, fmt.Sprintf("C%d", nextRow), fmt.Sprintf("%s%d", lastCol, nextRow))
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", nextRow), fmt.Sprintf("%s%d", lastCol, nextRow), titleStyle)
					headersSimpan := []string{"No", "Kode", "Nama", "Unit", "Pokok", "Wajib", "Jml Wajib", "Simp. HR", "Jml HR", "Suk.", "Jml Suk.", "Jml Bayar"}
					for i, h := range headersSimpan {
						f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], nextRow+1), h)
					}
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", nextRow+1), fmt.Sprintf("%s%d", lastCol, nextRow+1), headerStyleBlue)
					f.MergeCell(sheet, fmt.Sprintf("C%d", nextRow+2), fmt.Sprintf("%s%d", lastCol, nextRow+2))
					f.SetCellValue(sheet, fmt.Sprintf("C%d", nextRow+2), "Tidak ada data anggota")
					f.SetCellStyle(sheet, fmt.Sprintf("C%d", nextRow+2), fmt.Sprintf("%s%d", lastCol, nextRow+2), borderStyle)
				}
			} else {
				// Ada data anggota, buat tabel dengan data
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
					f.SetCellValue(sheet, fmt.Sprintf("A%d", startRow5-1), "Rincian Penarikan Tahunan")
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
					// Laporan bulanan - pecah jadi 2 tabel (Pinjaman + Simpanan) agar sama dengan PDF

					tableHeaderStyle := func(color string) int {
						styleID, _ := f.NewStyle(&excelize.Style{
							Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
							Fill:      excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
							Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
							Border: []excelize.Border{
								{Type: "left", Color: "#000000", Style: 1},
								{Type: "right", Color: "#000000", Style: 1},
								{Type: "top", Color: "#000000", Style: 1},
								{Type: "bottom", Color: "#000000", Style: 1},
							},
						})
						return styleID
					}
					tableBorderStyle, _ := f.NewStyle(&excelize.Style{
						Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
						Border: []excelize.Border{
							{Type: "left", Color: "#000000", Style: 1},
							{Type: "right", Color: "#000000", Style: 1},
							{Type: "top", Color: "#000000", Style: 1},
							{Type: "bottom", Color: "#000000", Style: 1},
						},
					})
					tableLeftStyle, _ := f.NewStyle(&excelize.Style{
						Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
						Border: []excelize.Border{
							{Type: "left", Color: "#000000", Style: 1},
							{Type: "right", Color: "#000000", Style: 1},
							{Type: "top", Color: "#000000", Style: 1},
							{Type: "bottom", Color: "#000000", Style: 1},
						},
					})
					tableCenterStyle, _ := f.NewStyle(&excelize.Style{
						Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
						Border: []excelize.Border{
							{Type: "left", Color: "#000000", Style: 1},
							{Type: "right", Color: "#000000", Style: 1},
							{Type: "top", Color: "#000000", Style: 1},
							{Type: "bottom", Color: "#000000", Style: 1},
						},
					})
					tableRightStyle, _ := f.NewStyle(&excelize.Style{
						Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
						Border: []excelize.Border{
							{Type: "left", Color: "#000000", Style: 1},
							{Type: "right", Color: "#000000", Style: 1},
							{Type: "top", Color: "#000000", Style: 1},
							{Type: "bottom", Color: "#000000", Style: 1},
						},
					})

					writeTable := func(title string, headers []string, rows [][]string, headerColor string, row int) int {
						cols := []string{"C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N"}
						lastCol := cols[len(headers)-1]
						titleStyle, _ := f.NewStyle(&excelize.Style{
							Font:      &excelize.Font{Bold: true},
							Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
						})
						f.SetCellValue(sheet, fmt.Sprintf("C%d", row), title)
						f.MergeCell(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%s%d", lastCol, row))
						f.SetCellStyle(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%s%d", lastCol, row), titleStyle)
						_ = f.SetRowHeight(sheet, row, 20)
						headerRow := row + 1
						for i, h := range headers {
							f.SetCellValue(sheet, fmt.Sprintf("%s%d", cols[i], headerRow), h)
						}
						f.SetCellStyle(sheet, fmt.Sprintf("C%d", headerRow), fmt.Sprintf("%s%d", lastCol, headerRow), tableHeaderStyle(headerColor))
						_ = f.SetRowHeight(sheet, headerRow, 24)

						dataStart := headerRow + 1
						if len(rows) == 0 {
							f.MergeCell(sheet, fmt.Sprintf("C%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart))
							f.SetCellValue(sheet, fmt.Sprintf("C%d", dataStart), "Tidak ada data")
							f.SetCellStyle(sheet, fmt.Sprintf("C%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart), tableBorderStyle)
							return dataStart + 2
						}

						for i, r := range rows {
							cur := dataStart + i
							for j, v := range r {
								cell := fmt.Sprintf("%s%d", cols[j], cur)
								f.SetCellValue(sheet, cell, v)
								switch {
								case j == 0:
									f.SetCellStyle(sheet, cell, cell, tableCenterStyle)
								case j == 1:
									f.SetCellStyle(sheet, cell, cell, tableCenterStyle)
								case j == 2:
									f.SetCellStyle(sheet, cell, cell, tableLeftStyle)
								case j == 3:
									f.SetCellStyle(sheet, cell, cell, tableCenterStyle)
								case strings.HasPrefix(strings.TrimSpace(v), "Rp "):
									f.SetCellStyle(sheet, cell, cell, tableRightStyle)
								default:
									f.SetCellStyle(sheet, cell, cell, tableLeftStyle)
								}
							}
							_ = f.SetRowHeight(sheet, cur, 20)
						}
						f.SetCellStyle(sheet, fmt.Sprintf("C%d", dataStart), fmt.Sprintf("%s%d", lastCol, dataStart+len(rows)-1), tableBorderStyle)
						return dataStart + len(rows) + 2
					}

					toText := func(v interface{}) string {
						if v == nil {
							return "-"
						}
						s := strings.TrimSpace(fmt.Sprintf("%v", v))
						if s == "" || s == "<nil>" {
							return "-"
						}
						return s
					}
					toRp := func(v float64) string {
						return fmt.Sprintf("Rp %.0f", v)
					}

					// Tabel 1: Rincian Pinjaman
					pinjamRows := [][]string{}
					for i, d := range laporanDetail {
						pinjamRows = append(pinjamRows, []string{
							fmt.Sprintf("%d", i+1),
							toText(d["id_anggota"]),
							toText(d["nama_anggota"]),
							repository.GetUnitKerjaName(toText(d["unit_kerja"])),
							toRp(getFloat(d, "pinjaman_bulanan")),
							fmt.Sprintf("%d", getInt(d, "jangka_waktu")),
							toRp(getFloat(d, "pokok_per_bulan")),
							toRp(getFloat(d, "jasa_per_bulan")),
							toRp(getFloat(d, "jumlah_angsuran_per_bulan")),
							fmt.Sprintf("%d", getInt(d, "total_angsuran_dibayar")),
							fmt.Sprintf("%d", getInt(d, "sisa_angsuran")),
							toRp(getFloat(d, "sisa_pinjaman")),
						})
					}
					f.SetColWidth(sheet, "A", "B", 2)
					f.SetColWidth(sheet, "C", "C", 8)
					f.SetColWidth(sheet, "D", "D", 14)
					f.SetColWidth(sheet, "E", "E", 24)
					f.SetColWidth(sheet, "F", "F", 16)
					f.SetColWidth(sheet, "G", "N", 14)
					nextRow := writeTable(
						"Rincian Pinjaman",
						[]string{"No", "Kode", "Nama", "Unit", "Nominal", "Tenor", "Pokok", "Jasa", "Jumlah", "Angs.", "S.A.", "S.P."},
						pinjamRows,
						"#17a2b8",
						startRow,
					)

					// Tabel 2: Rincian Simpanan
					simpanRows := [][]string{}
					for i, d := range laporanDetail {
						simpanRows = append(simpanRows, []string{
							fmt.Sprintf("%d", i+1),
							toText(d["id_anggota"]),
							toText(d["nama_anggota"]),
							repository.GetUnitKerjaName(toText(d["unit_kerja"])),
							toRp(getFloat(d, "simpanan_pokok")),
							toRp(getFloat(d, "simpanan_wajib_bulanan")),
							toRp(getFloat(d, "total_simpanan_wajib")),
							toRp(getFloat(d, "simpanan_hariraya_bulanan")),
							toRp(getFloat(d, "total_simpanan_hariraya")),
							toRp(getFloat(d, "simpanan_sukarela_bulanan")),
							toRp(getFloat(d, "total_simpanan_sukarela")),
							toRp(getFloat(d, "total_pembayaran")),
						})
					}
					f.SetColWidth(sheet, "A", "B", 2)
					f.SetColWidth(sheet, "C", "C", 8)
					f.SetColWidth(sheet, "D", "D", 14)
					f.SetColWidth(sheet, "E", "E", 24)
					f.SetColWidth(sheet, "F", "F", 16)
					f.SetColWidth(sheet, "G", "N", 14)
					_ = writeTable(
						"Rincian Simpanan",
						[]string{"No", "Kode", "Nama", "Unit", "Pokok", "Wajib", "Jml Wajib", "Simp. HR", "Jml HR", "Suk.", "Jml Suk.", "Jml Bayar"},
						simpanRows,
						"#0d6efd",
						nextRow,
					)
				}
			}
		}
		// Set header untuk download
		rowsForSign, _ := f.GetRows(sheet)
		addSignatureBlockToExcel(f, sheet, len(rowsForSign)+3, tipeLaporan, signatureDisplayImages, signatureNames)
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
			totalPengeluaran = pinjaman
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

		// Gunakan portrait untuk semua jenis laporan agar output konsisten.
		orientation := "P"
		pageWidth := 190.0

		pdf := gofpdf.New(orientation, "mm", "A4", "")
		if kopPath != "" && isSupportedImageFile(kopPath) {
			kopForHeader := kopPath
			pdf.SetHeaderFuncMode(func() {
				pdf.ImageOptions(kopForHeader, 10, 8, pageWidth, 0, false, gofpdf.ImageOptions{ImageType: ""}, 0, "")
				pdf.SetY(50)
			}, true)
			pdf.SetTopMargin(52)
		}
		pdf.AddPage()

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
		}{}
		if useNeracaSummary {
			for _, r := range neracaSummaryRows {
				dataRows = append(dataRows, struct {
					label string
					value float64
				}{label: r.Label, value: r.Value})
			}
		} else {
			dataRows = []struct {
				label string
				value float64
			}{
				{"Total Simpanan", report["total_simpanan"].(float64)},
				{"Total Pinjaman", report["total_pinjaman"].(float64)},
				{"Total Angsuran", report["total_angsuran"].(float64)},
				{"Total Pengeluaran", totalPengeluaran},
			}
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

		// Untuk laporan tahunan, samakan format rincian dengan tampilan halaman ketua/laporan (rincianTahunan).
		if tipeLaporan == "tahunan" && !useNeracaSummary {
			laporanDetail, _ := repository.GetLaporanBulananPerAnggota(0, tahunInt)
			toText := func(v interface{}) string {
				if v == nil {
					return "-"
				}
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s == "" || s == "<nil>" {
					return "-"
				}
				return s
			}
			toRp := func(v float64) string {
				return formatRupiah(v)
			}

			// Neraca format 2 sisi: Aset (kiri) vs Kewajiban & Ekuitas (kanan)
			if len(neracaAsetLancar)+len(neracaAsetTetap)+len(neracaKewajibanLancar)+len(neracaEkuitas) > 0 {
				if pdf.GetY()+110 > 270 {
					pdf.AddPage()
				}
				pdf.Ln(6)
				pdf.SetFont("Arial", "B", 12)
				pdf.CellFormat(190, 8, "Neraca Koperasi Simpan Pinjam", "0", 1, "L", false, 0, "")

				// Lebar kolom diseimbangkan agar judul total di sisi kanan tidak terpotong.
				w := []float64{13, 40, 21, 21, 13, 40, 21, 21}

				// Header row 1
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(31, 35, 42)
				pdf.SetTextColor(255, 255, 255)
				pdf.CellFormat(w[0], 7, "NoPerkiraan", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[1], 7, "Perkiraan", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(31, 111, 235)
				pdf.CellFormat(w[2]+w[3], 7, "ASET", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(31, 35, 42)
				pdf.CellFormat(w[4], 7, "NoPerkiraan", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[5], 7, "Perkiraan", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(25, 135, 84)
				pdf.CellFormat(w[6]+w[7], 7, "KEWAJIBAN & EKUITAS", "1", 1, "C", true, 0, "")

				// Header row 2
				pdf.SetFillColor(31, 35, 42)
				pdf.CellFormat(w[0], 7, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[1], 7, "", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(31, 111, 235)
				pdf.CellFormat(w[2], 7, "2024 (Rp)", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[3], 7, "2023 (Rp)", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(31, 35, 42)
				pdf.CellFormat(w[4], 7, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[5], 7, "", "1", 0, "C", true, 0, "")
				pdf.SetFillColor(25, 135, 84)
				pdf.CellFormat(w[6], 7, "2024 (Rp)", "1", 0, "C", true, 0, "")
				pdf.CellFormat(w[7], 7, "2023 (Rp)", "1", 1, "C", true, 0, "")

				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 7)

				writePairPDF := func(leftTitle string, left []neracaItem, lTotal24, lTotal23 float64, rightTitle string, right []neracaItem, rTotal24, rTotal23 float64) {
					pdf.SetFillColor(219, 234, 254)
					pdf.SetFont("Arial", "B", 7)
					pdf.CellFormat(w[0]+w[1]+w[2]+w[3], 7, leftTitle, "1", 0, "L", true, 0, "")
					pdf.SetFillColor(220, 252, 231)
					pdf.CellFormat(w[4]+w[5]+w[6]+w[7], 7, rightTitle, "1", 1, "L", true, 0, "")
					pdf.SetFont("Arial", "", 7)

					maxLen := len(left)
					if len(right) > maxLen {
						maxLen = len(right)
					}
					for i := 0; i < maxLen; i++ {
						if i < len(left) {
							pdf.CellFormat(w[0], 7, left[i].No, "1", 0, "C", false, 0, "")
							pdf.CellFormat(w[1], 7, left[i].Label, "1", 0, "L", false, 0, "")
							pdf.CellFormat(w[2], 7, toRp(left[i].V2024), "1", 0, "R", false, 0, "")
							pdf.CellFormat(w[3], 7, toRp(left[i].V2023), "1", 0, "R", false, 0, "")
						} else {
							pdf.CellFormat(w[0], 7, "", "1", 0, "C", false, 0, "")
							pdf.CellFormat(w[1], 7, "", "1", 0, "L", false, 0, "")
							pdf.CellFormat(w[2], 7, "", "1", 0, "R", false, 0, "")
							pdf.CellFormat(w[3], 7, "", "1", 0, "R", false, 0, "")
						}
						if i < len(right) {
							pdf.CellFormat(w[4], 7, right[i].No, "1", 0, "C", false, 0, "")
							pdf.CellFormat(w[5], 7, right[i].Label, "1", 0, "L", false, 0, "")
							pdf.CellFormat(w[6], 7, toRp(right[i].V2024), "1", 0, "R", false, 0, "")
							pdf.CellFormat(w[7], 7, toRp(right[i].V2023), "1", 1, "R", false, 0, "")
						} else {
							pdf.CellFormat(w[4], 7, "", "1", 0, "C", false, 0, "")
							pdf.CellFormat(w[5], 7, "", "1", 0, "L", false, 0, "")
							pdf.CellFormat(w[6], 7, "", "1", 0, "R", false, 0, "")
							pdf.CellFormat(w[7], 7, "", "1", 1, "R", false, 0, "")
						}
					}

					pdf.SetFont("Arial", "B", 7)
					pdf.SetFillColor(219, 234, 254)
					pdf.CellFormat(w[0]+w[1], 7, "Total "+leftTitle, "1", 0, "L", true, 0, "")
					pdf.CellFormat(w[2], 7, toRp(lTotal24), "1", 0, "R", true, 0, "")
					pdf.CellFormat(w[3], 7, toRp(lTotal23), "1", 0, "R", true, 0, "")
					pdf.SetFillColor(220, 252, 231)
					pdf.CellFormat(w[4]+w[5], 7, "Total "+rightTitle, "1", 0, "L", true, 0, "")
					pdf.CellFormat(w[6], 7, toRp(rTotal24), "1", 0, "R", true, 0, "")
					pdf.CellFormat(w[7], 7, toRp(rTotal23), "1", 1, "R", true, 0, "")
					pdf.SetFont("Arial", "", 7)
				}

				// Samakan dengan tampilan web: item ekuitas bisa naik mengisi blok Kewajiban Lancar.
				rightKewajiban := append([]neracaItem{}, neracaKewajibanLancar...)
				rightEkuitas := append([]neracaItem{}, neracaEkuitas...)
				totalKewajiban24, totalKewajiban23 := neracaTotalKewajibanLancar2024, neracaTotalKewajibanLancar2023
				totalEkuitas24, totalEkuitas23 := neracaTotalEkuitas2024, neracaTotalEkuitas2023
				if len(rightKewajiban) < len(neracaAsetLancar) && len(rightEkuitas) > 0 {
					need := len(neracaAsetLancar) - len(rightKewajiban)
					if need > len(rightEkuitas) {
						need = len(rightEkuitas)
					}
					for i := 0; i < need; i++ {
						it := rightEkuitas[i]
						rightKewajiban = append(rightKewajiban, it)
						totalKewajiban24 += it.V2024
						totalKewajiban23 += it.V2023
						totalEkuitas24 -= it.V2024
						totalEkuitas23 -= it.V2023
					}
					rightEkuitas = rightEkuitas[need:]
				}

				writePairPDF(
					"Aset Lancar", neracaAsetLancar, neracaTotalAsetLancar2024, neracaTotalAsetLancar2023,
					"Kewajiban Lancar", rightKewajiban, totalKewajiban24, totalKewajiban23,
				)
				writePairPDF(
					"Aset Tetap", neracaAsetTetap, neracaTotalAsetTetap2024, neracaTotalAsetTetap2023,
					"Ekuitas/Modal", rightEkuitas, totalEkuitas24, totalEkuitas23,
				)

				totalAset24 := neracaTotalAsetLancar2024 + neracaTotalAsetTetap2024
				totalAset23 := neracaTotalAsetLancar2023 + neracaTotalAsetTetap2023
				totalKE24 := totalKewajiban24 + totalEkuitas24
				totalKE23 := totalKewajiban23 + totalEkuitas23
				pdf.SetFont("Arial", "B", 8)
				pdf.SetFillColor(254, 243, 199)
				pdf.CellFormat(w[0]+w[1], 8, "TOTAL ASET", "1", 0, "L", true, 0, "")
				pdf.CellFormat(w[2], 8, toRp(totalAset24), "1", 0, "R", true, 0, "")
				pdf.CellFormat(w[3], 8, toRp(totalAset23), "1", 0, "R", true, 0, "")
				pdf.CellFormat(w[4]+w[5], 8, "TOTAL KEWAJIBAN & EKUITAS", "1", 0, "L", true, 0, "")
				pdf.CellFormat(w[6], 8, toRp(totalKE24), "1", 0, "R", true, 0, "")
				pdf.CellFormat(w[7], 8, toRp(totalKE23), "1", 1, "R", true, 0, "")
			}

			writePDFTable := func(title string, headers []string, widths []float64, rows [][]string, r, g, b int) {
				neededRows := len(rows)
				if neededRows == 0 {
					neededRows = 1
				}
				neededHeight := 6.0 + 8.0 + 7.0 + (7.0 * float64(neededRows))
				if pdf.GetY()+neededHeight > 280 {
					pdf.AddPage()
				}
				pdf.Ln(4)
				pdf.SetFont("Arial", "B", 12)
				pdf.CellFormat(190, 8, title, "0", 1, "L", false, 0, "")
				pdf.SetFont("Arial", "B", 8)
				pdf.SetFillColor(r, g, b)
				pdf.SetTextColor(255, 255, 255)
				for i, h := range headers {
					pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
				}
				pdf.Ln(-1)

				pdf.SetFont("Arial", "", 8)
				pdf.SetTextColor(0, 0, 0)
				if len(rows) == 0 {
					pdf.CellFormat(190, 7, "Tidak ada data", "1", 1, "C", false, 0, "")
					return
				}
				for _, row := range rows {
					for i, v := range row {
						align := "L"
						if i == 0 || i == 1 {
							align = "C"
						} else if i >= 4 {
							align = "R"
						}
						pdf.CellFormat(widths[i], 7, v, "1", 0, align, false, 0, "")
					}
					pdf.Ln(-1)
				}
			}

			rowsSimpanan := [][]string{}
			for i, d := range laporanDetail {
				rowsSimpanan = append(rowsSimpanan, []string{
					fmt.Sprintf("%d", i+1),
					" " + toText(d["id_anggota"]),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRp(getFloat(d, "simpanan_pokok")),
					toRp(getFloat(d, "total_simpanan_wajib")),
					toRp(getFloat(d, "total_simpanan_hariraya")),
					"Rp 0",
					"Rp 0",
					toRp(getFloat(d, "total_simpanan_sukarela")),
				})
			}
			writePDFTable(
				"Daftar Simpanan Anggota",
				[]string{"No", "ID", "Nama", "Unit", "Pokok", "Wajib", "HR", "Umroh", "Qurban", "Sukarela"},
				[]float64{6, 20, 27, 12, 18, 18, 18, 18, 18, 35},
				rowsSimpanan,
				13, 110, 253,
			)

			rowsPiutang := [][]string{}
			for i, d := range laporanDetail {
				rowsPiutang = append(rowsPiutang, []string{
					fmt.Sprintf("%d", i+1),
					toText(d["id_anggota"]),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRp(getFloat(d, "sisa_pinjaman")),
				})
			}
			writePDFTable(
				"Daftar Piutang Anggota",
				[]string{"No", "ID Anggota", "Nama", "Unit", "Sisa Piutang"},
				[]float64{10, 30, 55, 45, 50},
				rowsPiutang,
				13, 202, 240,
			)

			rowsMacet := [][]string{}
			for _, d := range laporanDetail {
				if getFloat(d, "sisa_pinjaman") > 0 {
					rowsMacet = append(rowsMacet, []string{
						fmt.Sprintf("%d", len(rowsMacet)+1),
						toText(d["id_anggota"]),
						toText(d["nama_anggota"]),
						repository.GetUnitKerjaName(toText(d["unit_kerja"])),
						toRp(getFloat(d, "sisa_pinjaman")),
					})
				}
			}
			writePDFTable(
				"Daftar Piutang Macet Anggota",
				[]string{"No", "ID Anggota", "Nama", "Unit", "Jumlah Piutang"},
				[]float64{10, 30, 55, 45, 50},
				rowsMacet,
				220, 53, 69,
			)

			rowsSHU := [][]string{}
			for i, d := range laporanDetail {
				shuPinjaman := getFloat(d, "shu_pinjaman")
				shuSimpanan := getFloat(d, "shu_simpanan")
				jumlahSHU := getFloat(d, "jumlah_shu")
				rowsSHU = append(rowsSHU, []string{
					fmt.Sprintf("%d", i+1),
					toText(d["nama_anggota"]),
					repository.GetUnitKerjaName(toText(d["unit_kerja"])),
					toRp(shuPinjaman),
					toRp(shuSimpanan),
					toRp(jumlahSHU),
				})
			}
			writePDFTable(
				"Daftar SHU Anggota",
				[]string{"No", "Nama", "Unit", "SHU Pinjaman", "SHU Simpanan", "Jumlah SHU"},
				[]float64{10, 55, 35, 30, 30, 30},
				rowsSHU,
				25, 135, 84,
			)
			addSignatureBlockToPDF(pdf, tipeLaporan, signatureDisplayImages, signatureNames)

			c.Header("Content-Type", "application/pdf")
			c.Header("Content-Disposition", "attachment; filename=laporan_koperasi_tahunan_"+fmt.Sprintf("%d", tahunInt)+".pdf")
			err = pdf.Output(c.Writer)
			if err != nil {
				c.String(http.StatusInternalServerError, "Gagal membuat file PDF")
			}
			return
		}

		// Untuk tahunan berbasis neraca, jangan tampilkan rincian anggota agar output konsisten dengan data neraca.
		if !(tipeLaporan == "tahunan" && useNeracaSummary) {
			// Ambil data anggota aktif dan potongan/sisa gaji
			anggotas, err := repository.GetAllAnggota()
			potonganBulanIni := make(map[string]float64)
			if err == nil {
				potonganBulanIni, _ = repository.GetPotonganBulanIniAllAnggota()
			}
			// Tambahan agar laporanDetail terdefinisi
			var laporanDetail []map[string]interface{}
			if tipeLaporan == "bulanan" {
				laporanDetail, _ = repository.GetLaporanBulananPerAnggota(bulanInt, tahunInt)
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
						"Rincian Penarikan Tahunan",
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
					// Laporan bulanan: Tabel Rincian Laporan Bulanan
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
						"Rincian Penarikan Tahunan",
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
					// Laporan bulanan: 2 tabel matching web (Pinjaman + Simpanan) dengan lebar kolom konsisten.
					pdf.Ln(10)

					asText := func(v interface{}) string {
						if v == nil {
							return ""
						}
						return fmt.Sprintf("%v", v)
					}
					toCompactRp := func(v float64) string {
						return strings.TrimPrefix(formatRupiah(v), "Rp ")
					}
					normalizeCell := func(text string) string {
						if strings.TrimSpace(text) == "" {
							return "-"
						}
						return text
					}
					shortUnit := func(unit string) string {
						u := strings.TrimSpace(strings.ToLower(unit))
						switch u {
						case "tenaga pendidikan":
							return "Tendik"
						case "mahasiswa":
							return "Mhs"
						default:
							return unit
						}
					}
					fitText := func(text string, width float64) string {
						t := normalizeCell(text)
						maxWidth := width - 1.2
						if maxWidth <= 1 {
							return t
						}
						if pdf.GetStringWidth(t) <= maxWidth {
							return t
						}
						runes := []rune(t)
						if len(runes) == 0 {
							return t
						}
						suffix := "..."
						for len(runes) > 1 {
							candidate := string(runes) + suffix
							if pdf.GetStringWidth(candidate) <= maxWidth {
								return candidate
							}
							runes = runes[:len(runes)-1]
						}
						return string(runes)
					}
					writeTableHeader := func(headers []string, widths []float64) {
						pdf.SetFont("Arial", "B", 7)
						pdf.SetFillColor(23, 162, 184)
						pdf.SetTextColor(255, 255, 255)
						for i, h := range headers {
							pdf.CellFormat(widths[i], 6, h, "1", 0, "C", true, 0, "")
						}
						pdf.Ln(-1)
						pdf.SetFont("Arial", "", 7)
						pdf.SetTextColor(0, 0, 0)
					}

					writeCompactRow := func(values []string, widths []float64, aligns []string) {
						rowHeight := 6.0
						if pdf.GetY()+rowHeight > 280 {
							pdf.AddPage()
						}
						for i, v := range values {
							align := "L"
							if i < len(aligns) {
								align = aligns[i]
							}
							pdf.CellFormat(widths[i], rowHeight, fitText(v, widths[i]), "1", 0, align, false, 0, "")
						}
						pdf.Ln(-1)
					}
					// TABLE 1: RINCIAN PINJAMAN
					pdf.SetFont("Arial", "B", 12)
					pdf.CellFormat(pageWidth, 8, "Rincian Pinjaman", "0", 1, "L", false, 0, "")
					pdf.Ln(2)

					pinjamColWidths := []float64{8, 30, 34, 15, 18, 11, 17, 13, 17, 9, 9, 9} // total 190 (portrait)
					pinjamHeaders := []string{"No", "Kode", "Nama", "Unit", "Nominal", "Tenor", "Pokok", "Jasa", "Jumlah", "Angs.", "S.A.", "S.P."}
					pinjamAligns := []string{"C", "C", "L", "C", "R", "C", "R", "R", "R", "C", "C", "R"}
					writeTableHeader(pinjamHeaders, pinjamColWidths)
					for idx, detail := range laporanDetail {
						row := []string{
							fmt.Sprintf("%d", idx+1),
							asText(detail["id_anggota"]),
							asText(detail["nama_anggota"]),
							shortUnit(repository.GetUnitKerjaName(asText(detail["unit_kerja"]))),
							toCompactRp(getFloat(detail, "pinjaman_bulanan")),
							fmt.Sprintf("%d", getInt(detail, "jangka_waktu")),
							toCompactRp(getFloat(detail, "pokok_per_bulan")),
							toCompactRp(getFloat(detail, "jasa_per_bulan")),
							toCompactRp(getFloat(detail, "jumlah_angsuran_per_bulan")),
							fmt.Sprintf("%d", getInt(detail, "total_angsuran_dibayar")),
							fmt.Sprintf("%d", getInt(detail, "sisa_angsuran")),
							toCompactRp(getFloat(detail, "sisa_pinjaman")),
						}
						writeCompactRow(row, pinjamColWidths, pinjamAligns)
					}

					// TABLE 2: RINCIAN SIMPANAN
					if pdf.GetY()+70 > 270 {
						pdf.AddPage()
					}
					pdf.Ln(8)
					pdf.SetFont("Arial", "B", 12)
					pdf.CellFormat(pageWidth, 8, "Rincian Simpanan", "0", 1, "L", false, 0, "")
					pdf.Ln(2)

					simpanColWidths := []float64{8, 30, 30, 14, 12, 12, 14, 14, 14, 12, 14, 16} // total 190 (portrait)
					simpanHeaders := []string{"No", "Kode", "Nama", "Unit", "Pokok", "Wajib", "Jml Wajib", "Simp. HR", "Jml HR", "Suk.", "Jml Suk.", "Jml Bayar"}
					simpanAligns := []string{"C", "C", "L", "C", "R", "R", "R", "R", "R", "R", "R", "R"}
					writeTableHeader(simpanHeaders, simpanColWidths)
					for idx, detail := range laporanDetail {
						row := []string{
							fmt.Sprintf("%d", idx+1),
							asText(detail["id_anggota"]),
							asText(detail["nama_anggota"]),
							shortUnit(repository.GetUnitKerjaName(asText(detail["unit_kerja"]))),
							toCompactRp(getFloat(detail, "simpanan_pokok")),
							toCompactRp(getFloat(detail, "simpanan_wajib_bulanan")),
							toCompactRp(getFloat(detail, "total_simpanan_wajib")),
							toCompactRp(getFloat(detail, "simpanan_hariraya_bulanan")),
							toCompactRp(getFloat(detail, "total_simpanan_hariraya")),
							toCompactRp(getFloat(detail, "simpanan_sukarela_bulanan")),
							toCompactRp(getFloat(detail, "total_simpanan_sukarela")),
							toCompactRp(getFloat(detail, "total_pembayaran")),
						}
						writeCompactRow(row, simpanColWidths, simpanAligns)
					}
				}
			}
		}
		// Set header untuk download
		addSignatureBlockToPDF(pdf, tipeLaporan, signatureDisplayImages, signatureNames)
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
	pendingSimpanan, errSimpanan := repository.GetConfirmedSimpanan()
	pendingPinjaman, errPinjaman := repository.GetPendingPinjaman() // pinjaman tetap status 'proses'
	pendingAngsuran, errAngsuran := repository.GetConfirmedAngsuran()
	pendingPengambilan, errPengambilan := repository.GetPendingPengambilanSimpanan()

	if errSimpanan != nil {
		log.Printf("[WARN] KetuaKonfirmasiTransaksi: gagal mengambil simpanan terkonfirmasi: %v", errSimpanan)
		pendingSimpanan = []models.Detail{}
	}
	if errPinjaman != nil {
		log.Printf("[WARN] KetuaKonfirmasiTransaksi: gagal mengambil pinjaman pending: %v", errPinjaman)
		pendingPinjaman = []models.Pinjaman{}
	}
	if errAngsuran != nil {
		log.Printf("[WARN] KetuaKonfirmasiTransaksi: gagal mengambil angsuran terkonfirmasi: %v", errAngsuran)
		pendingAngsuran = []models.Angsuran{}
	}
	if errPengambilan != nil {
		log.Printf("[WARN] KetuaKonfirmasiTransaksi: gagal mengambil pengambilan simpanan pending: %v", errPengambilan)
		pendingPengambilan = []models.PengambilanSimpanan{}
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

	// Ambil data simpanan wajib untuk semua anggota
	simpananWajib, err := repository.GetSimpananWajibAllAnggota()
	if err != nil {
		simpananWajib = make(map[string]float64)
	}

	// Ambil nominal simpanan pokok dan petakan ke semua anggota aktif
	simpananPokok := make(map[string]float64)
	var nominalSimpananPokok float64
	err = config.GetDB().QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
	if err != nil {
		nominalSimpananPokok = 100000
	}
	for _, anggota := range anggotas {
		simpananPokok[anggota.IDAnggota] = nominalSimpananPokok
	}

	// Ambil konfigurasi simpanan wajib
	potonganBulanIni, err := repository.GetPotonganBulanIniAllAnggota()
	if err != nil {
		potonganBulanIni = make(map[string]float64)
	}
	potonganRegister, err := repository.GetPotonganRegisterPotongGajiBulanIniAllAnggota()
	if err != nil {
		potonganRegister = make(map[string]float64)
	}

	// Hitung sisa gaji untuk setiap anggota: Gaji Bulanan - Potongan Bulan Ini
	sisaGaji := make(map[string]float64)
	for _, anggota := range anggotas {
		potongan := potonganBulanIni[anggota.IDAnggota] + potonganRegister[anggota.IDAnggota]
		sisaGaji[anggota.IDAnggota] = float64(anggota.GajiBulanan) - potongan

		// Fallback tampilan jika total simpanan wajib belum tercatat.
		if simpananWajib[anggota.IDAnggota] <= 0 && potonganBulanIni[anggota.IDAnggota] > 0 {
			simpananWajib[anggota.IDAnggota] = potonganBulanIni[anggota.IDAnggota]
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

	c.HTML(http.StatusOK, "ketua_data_anggota.html", gin.H{
		"Anggotas":         anggotas,
		"SimpananPokok":    simpananPokok,
		"SimpananWajib":    simpananWajib,
		"PotonganBulanIni": potonganBulanIni,
		"SisaGaji":         sisaGaji,
		"ActivePage":       "anggota",
		"CurrentLogo":      latestLogo,
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

	// Samakan dengan halaman profil anggota:
	// total simpanan tidak memasukkan simpanan pokok dari pendaftaran.
	totalSimpanan := simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]
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

	c.HTML(http.StatusOK, "ketua_data_anggota_view.html", gin.H{
		"Anggota":            anggota,
		"ActivePage":         "anggota",
		"CurrentLogo":        latestLogo,
		"Title":              "Detail Anggota",
		"ProfilSimpananRows": profilSimpananRows,
		"SimpananPokok":      simpananByJenis["pokok"],
		"SimpananWajib":      simpananByJenis["wajib"],
		"SimpananSukarela":   simpananByJenis["sukarela"],
		"SimpananHariRaya":   simpananByJenis["hari_raya"],
		"SimpananUmrohHaji":  simpananByJenis["umroh_haji"],
		"SimpananQurban":     simpananByJenis["qurban"],
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
	})
}

// Menampilkan halaman riwayat transaksi
func KetuaRiwayat(c *gin.Context) {
	// Ambil semua data riwayat transaksi dari database
	riwayats, err := repository.GetAllRiwayat()
	if err != nil {
		// Log error detail ke console agar mudah debug
		log.Printf("[ERROR] KetuaRiwayat ambil data riwayat gagal: %v", err)
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
			"ActivePage":      "laporan",
			"Error":           "Gagal mengambil laporan",
			"CurrentLogo":     latestLogo,
			"Bulan":           bulan,
			"Tahun":           tahun,
			"LaporanBasePath": "/ketua/laporan",
			"UseAdminLayout":  false,
			"ReadOnlyMode":    false,
		})
		return
	}

	// Ambil data neraca dari repository
	userIDInt := resolveNeracaOwnerID(c)
	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)
	neraca, _ := neracaRepo.GetNeraca(userIDInt)
	var data2024, data2023 map[string]interface{}
	if neraca != nil {
		json.Unmarshal([]byte(neraca.Data2024), &data2024)
		json.Unmarshal([]byte(neraca.Data2023), &data2023)
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
	labelByKey, customSimpananColumns := getLaporanSimpananColumns()
	hydrateCustomSimpananValuesToLaporanDetail(laporanDetail, customSimpananColumns, bulan, tahun)

	c.HTML(http.StatusOK, "ketua_laporan.html", gin.H{
		"ActivePage":            "laporan",
		"Report":                report,
		"Bulan":                 bulan,
		"Tahun":                 tahun,
		"TipeLaporan":           tipeLaporan,
		"CurrentLogo":           latestLogo,
		"Anggotas":              anggotas,
		"LaporanDetail":         laporanDetail,
		"SisaGaji":              sisaGaji,
		"SimpananLabelPokok":    labelByKey["simpanan_pokok"],
		"SimpananLabelWajib":    labelByKey["simpanan_wajib"],
		"SimpananLabelHariRaya": labelByKey["simpanan_hari_raya"],
		"SimpananLabelSukarela": labelByKey["simpanan_sukarela"],
		"CustomSimpananColumns": customSimpananColumns,
		"GetUnitKerjaName":      repository.GetUnitKerjaName,
		"success":               successMsg,
		"NeracaData2024":        data2024,
		"NeracaData2023":        data2023,
		"LaporanBasePath":       "/ketua/laporan",
		"UseAdminLayout":        false,
		"ReadOnlyMode":          false,
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
		log.Printf("[ERROR] KetuaPengaturan ambil data bendahara gagal (id=%v): %v", bendaharaID, err)
		c.HTML(http.StatusInternalServerError, "ketua_layout.html", gin.H{
			"ActivePage": "pengaturan",
			"Error":      "Gagal mengambil data bendahara",
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
		log.Printf("[ERROR] UpdateKetuaProfile bind request gagal: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
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
		log.Printf("[ERROR] KetuaKonfirmasiAnggota ambil anggota gagal (id=%s): %v", tempID, err)
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
		fmt.Printf("[CONFIRM ANGGOTA ERROR] update anggota: %v\n", err)
		c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Gagal mengkonfirmasi anggota")
		return
	}

	fmt.Printf("✓ Anggota dengan ID %s berhasil dikonfirmasi dan aktif\n", newIDAnggota)

	if strings.EqualFold(strings.TrimSpace(anggota.BuktiTransfer), "POTONG_GAJI") {
		var nominalSimpananPokok float64
		err = db.QueryRow("SELECT COALESCE(CAST(nilai AS NUMERIC), 100000) FROM pengaturan WHERE nama_pengaturan = 'nominal_simpanan'").Scan(&nominalSimpananPokok)
		if err != nil {
			nominalSimpananPokok = 100000
		}

		var sudahAda bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1
				FROM detail
				WHERE id_anggota = $1 AND id_simpanan = 1 AND COALESCE(status, 'confirmed') IN ('confirmed', 'diterima', 'lunas')
			)
		`, newIDAnggota).Scan(&sudahAda)
		if err != nil {
			log.Printf("[WARN] gagal cek simpanan pokok potong gaji untuk anggota %s: %v", newIDAnggota, err)
		}

		if !sudahAda {
			_, err = db.Exec(`
				INSERT INTO detail (
					id_anggota, id_simpanan, id_pengelola, tgl_transaksi,
					jumlah_simpanan, total_simpanan, status, bukti_pembayaran, metode_pembayaran
				) VALUES ($1, 1, NULL, CURRENT_TIMESTAMP, $2, $2, 'confirmed', 'POTONG_GAJI', 'potong_gaji')
			`, newIDAnggota, nominalSimpananPokok)
			if err != nil {
				log.Printf("[ERROR] gagal mencatat simpanan pokok potong gaji untuk anggota %s: %v", newIDAnggota, err)
				c.Redirect(http.StatusFound, "/ketua/konfirmasi?error=Anggota aktif, tetapi gagal mencatat simpanan pokok potong gaji")
				return
			}
		}
	}

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
		log.Printf("[ERROR] KetuaSaveNeraca bind json gagal: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := config.GetDB()
	neracaRepo := repository.NewNeracaRepository(db)

	userIDInt := resolveNeracaOwnerID(c)

	err := neracaRepo.SaveNeraca(&req, userIDInt)
	if err != nil {
		log.Printf("Error saving neraca: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save neraca"})
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

	userIDInt := resolveNeracaOwnerID(c)

	neraca, err := neracaRepo.GetNeraca(userIDInt)
	if err != nil {
		log.Printf("Error getting neraca: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get neraca"})
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
		c.String(http.StatusNotFound, "Data penarikan tidak ditemukan")
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

// KetuaUploadBuktiTransferGaji menampilkan form upload bukti transfer gaji
func KetuaUploadBuktiTransferGaji(c *gin.Context) {
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

	// Ambil riwayat upload
	buktiList, err := repository.GetAllBuktiTransferGaji()
	if err != nil {
		buktiList = []models.BuktiTransferGaji{}
	}
	now := time.Now()
	currentMonth := int(now.Month())
	currentYear := now.Year()
	buktiListView := make([]buktiTransferGajiView, 0, len(buktiList))
	for i := range buktiList {
		buktiList[i].Status = strings.ToLower(strings.TrimSpace(buktiList[i].Status))

		view := buktiTransferGajiView{
			BuktiTransferGaji: buktiList[i],
			CountdownText:     "-",
			IsExpired:         false,
		}

		// Countdown mengikuti periode BULAN BERJALAN secara otomatis.
		if buktiList[i].Bulan == currentMonth && buktiList[i].Tahun == currentYear {
			location := now.Location()
			deadline := time.Date(currentYear, time.Month(currentMonth)+1, 1, 0, 0, 0, 0, location)
			remaining := deadline.Sub(now)
			if remaining <= 0 {
				view.IsExpired = true
				view.CountdownText = "Waktu habis"
			} else {
				totalHours := int(remaining.Hours())
				days := totalHours / 24
				hours := totalHours % 24
				view.CountdownText = fmt.Sprintf("%d hari %d jam", days, hours)
			}
		} else if buktiList[i].Tahun < currentYear || (buktiList[i].Tahun == currentYear && buktiList[i].Bulan < currentMonth) {
			view.IsExpired = true
			view.CountdownText = "Periode selesai"
		} else {
			view.CountdownText = "Belum periode berjalan"
		}
		buktiListView = append(buktiListView, view)
	}

	// Kirim notifikasi WA pengingat jika bukti transfer bulan berjalan belum Approved (maksimal 1x per hari).
	sendCurrentMonthBuktiTransferReminderIfNeeded(c, buktiListView, currentMonth, currentYear)

	// Ambil pesan dari query parameter
	successMsg := c.Query("success")
	errorMsg := c.Query("error")

	c.HTML(http.StatusOK, "ketua_upload_bukti_transfer_gaji.html", gin.H{
		"CurrentLogo":    latestLogo,
		"BuktiList":      buktiListView,
		"CurrentYear":    now.Year(),
		"SuccessMessage": successMsg,
		"ErrorMessage":   errorMsg,
	})
}

func sendCurrentMonthBuktiTransferReminderIfNeeded(c *gin.Context, buktiList []buktiTransferGajiView, currentMonth, currentYear int) {
	hasCurrentMonthApproved := false
	for i := range buktiList {
		item := buktiList[i]
		if item.Bulan == currentMonth && item.Tahun == currentYear && item.Status == "approved" {
			hasCurrentMonthApproved = true
			break
		}
	}
	if hasCurrentMonthApproved {
		return
	}

	db := config.GetDB()
	today := time.Now().Format("2006-01-02")
	key := "wa_notif_bukti_transfer_current_month_last_sent"

	var lastSent string
	_ = db.QueryRow("SELECT COALESCE(nilai, '') FROM pengaturan WHERE nama_pengaturan = $1", key).Scan(&lastSent)
	expectedMarker := fmt.Sprintf("%s|%02d|%d", today, currentMonth, currentYear)
	if strings.TrimSpace(lastSent) == expectedMarker {
		return
	}

	appBaseURL := resolveAppBaseURL(c, db)
	if err := sendKetuaWhatsAppTransactionNotification(
		"",
		"Dokumen Bukti Transfer Gaji",
		"Pengingat Upload/Approval Bulan Berjalan",
		fmt.Sprintf("Periode %02d/%d belum Approved", currentMonth, currentYear),
		appBaseURL,
	); err != nil {
		log.Printf("[WA REMINDER BUKTI TRANSFER] gagal kirim notifikasi: %v", err)
		return
	}

	_, _ = db.Exec(`
		INSERT INTO pengaturan (nama_pengaturan, nilai, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (nama_pengaturan)
		DO UPDATE SET nilai = EXCLUDED.nilai, updated_at = NOW()
	`, key, expectedMarker)
}

// KetuaUploadBuktiTransferGajiPost memproses upload bukti transfer gaji
func KetuaUploadBuktiTransferGajiPost(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id")
	if userID == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	bulan, err := strconv.Atoi(c.PostForm("bulan"))
	if err != nil || bulan < 1 || bulan > 12 {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Bulan tidak valid")
		return
	}

	tahun, err := strconv.Atoi(c.PostForm("tahun"))
	if err != nil || tahun < 2000 || tahun > 2100 {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Tahun tidak valid")
		return
	}
	currentYear := time.Now().Year()
	if tahun != currentYear {
		c.Redirect(http.StatusFound, fmt.Sprintf("/ketua/upload-bukti-transfer-gaji?error=Tahun upload harus tahun berjalan (%d), tidak boleh tahun sebelumnya atau tahun mendatang", currentYear))
		return
	}

	catatan := strings.TrimSpace(c.PostForm("catatan"))

	// Handle file upload
	file, err := c.FormFile("file")
	if err != nil {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=File tidak ditemukan")
		return
	}

	// Validasi ukuran file (max 5MB)
	if file.Size > 5*1024*1024 {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Ukuran file maksimal 5MB")
		return
	}

	// Validasi ekstensi file
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Format file tidak didukung. Gunakan PDF, JPG, JPEG, atau PNG")
		return
	}

	// Generate nama file unik
	timestamp := time.Now().Format("20060102_150405")
	newFileName := fmt.Sprintf("bukti_transfer_gaji_%s_%d_%d%s", timestamp, bulan, tahun, ext)
	uploadPath := filepath.Join("static", "uploads", newFileName)

	// Simpan file
	if err := c.SaveUploadedFile(file, uploadPath); err != nil {
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Gagal menyimpan file")
		return
	}

	// Simpan ke database
	var userIDInt int
	switch v := userID.(type) {
	case int:
		userIDInt = v
	case string:
		userIDInt, _ = strconv.Atoi(v)
	}

	bukti := &models.BuktiTransferGaji{
		Bulan:        bulan,
		Tahun:        tahun,
		NamaFile:     file.Filename,
		PathFile:     "/" + filepath.ToSlash(uploadPath),
		DiuploadOleh: userIDInt,
		Status:       "pending",
		Catatan:      catatan,
	}

	if err := repository.SaveBuktiTransferGaji(bukti); err != nil {
		// Hapus file jika gagal simpan ke DB
		os.Remove(uploadPath)
		c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?error=Gagal menyimpan data ke database")
		return
	}

	c.Redirect(http.StatusFound, "/ketua/upload-bukti-transfer-gaji?success=Bukti transfer gaji berhasil diupload")
}
