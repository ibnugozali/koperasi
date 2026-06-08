package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/models"
	"koperasi-simpan-pinjam/repository"
)

// // Fungsi untuk menggabungkan seluruh pinjaman aktif/proses menjadi satu resume gabungan
// func getResumePinjamanGabungan(userID string) *resumePinjamanInfo {
// 	   // Ambil data pinjaman aktif dari repository
// 	   pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userID)
// 	   if err != nil || len(pinjamans) == 0 {
// 		   // Fallback: cari pinjaman status 'proses' (baru diajukan)
// 		   db := config.GetDB()
// 		   row := db.QueryRow(`SELECT id_pinjaman, tgl_pinjaman, jumlah_pinjaman, jangka_waktu, bunga, status, metode_pencairan, metode_angsuran FROM pinjaman WHERE id_anggota = $1 AND status = 'proses' ORDER BY tgl_pinjaman DESC, id_pinjaman DESC LIMIT 1`, userID)
// 		   var idPinjaman int
// 		   var tglPinjamanGabungan time.Time
// 		   var totalPinjaman, bungaVal float64
// 		   var totalJangkaWaktu int
// 		   var statusGabungan, metodePencairanGabungan, metodeAngsuranGabungan string
// 		   err := row.Scan(&idPinjaman, &tglPinjamanGabungan, &totalPinjaman, &totalJangkaWaktu, &bungaVal, &statusGabungan, &metodePencairanGabungan, &metodeAngsuranGabungan)
// 		   if err != nil {
// 			   return nil
// 		   }
// 		   // Jika bunga kosong/0, pakai default 4
// 		   if bungaVal == 0 {
// 			   bungaVal = 4
// 		   }
// 		   bungaNominal := totalPinjaman * bungaVal / 100
// 		   info := &resumePinjamanInfo{
// 			   Status:             statusGabungan,
// 			   TglPinjaman:        tglPin
// 			   Bunga:              bungaNominal,
// 			   MetodePencairan:    metodePencairanGabungan,
// 			   MetodeAngsuran:     metodeAngsuranGabungan,
// 		   }
// 		   fmt.Printf("[DEBUG] Fallback assignment: TotalTerbayar=%v, SisaPokok=%v, PersentaseTerbayar=%v\n", info.TotalTerbayar, info.SisaPokok, info.PersentaseTerbayar)
// 		   return info
// 	   }

// 	   // Ambil sisa pinjaman terbaru dari GetSaldoAnggota
// 	   _, totalSisaPinjaman, _, _ := repository.GetSaldoAnggota(userID)

// 	   // Ambil data resume dari pinjaman aktif pertama (jika ada)
// 	   p := pinjamans[0]
// 	   statusGabungan := p.Status
// 	   tglPinjamanGabungan := p.TglPinjaman
// 	   metodePencairanGabungan := p.MetodePencairan
// 	   metodeAngsuranGabungan := p.MetodeAngsuran
// 	   totalPinjaman := p.JumlahPinjaman
// 	   totalJangkaWaktu := p.JangkaWaktu
// 	   bungaNominal := p.JumlahPinjaman * p.Bunga / 100

// 	   // Hitung angsuran terbayar dan sisa angsuran
// 	   angsurans, _ := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
// 	   angsuranTerbayar := 0
// 	   for _, a := range angsurans {
// 		   if isAngsuranTerbayar(a.Status) {
// 			   angsuranTerbayar++
// 		   }r _, a := range angsurans {
// 		   if isAngsuranTerbayar(a.Status) {
// 			   angsuranTerbayar++
// 		   }angsuranTerbayar++
// 		   }r _, a := range angsurans {
// 		   if isAngsuranTerbayar(a.Status) {
// 			   angsuranTerbayar++
// 		   }angsuranTerbayar++
// 		   }
// 	   }
// 	   sisaAngsuran := p.JangkaWaktu - angsuranTerbayar
// 	   if sisaAngsuran < 0 {
// 		   sisaAngsuran = 0
// 	   }

// 	   persentaseGabungan := 0.0
// 	   if totalPinjaman > 0 {
// 		   persentaseGabungan = (totalPinjaman - totalSisaPinjaman) / totalPinjaman * 100
// 	   }

// 	   // Syarat pengajuan baru: minimal 50% sudah terbayar
// 	   bisaAjukanLagi := persentaseGabungan >= 50

// 	   info := &resumePinjamanInfo{
// 		   Status:             statusGabungan,
// 		   TglPinjaman:        tglPinjamanGabungan,
// 		   JumlahPinjaman:     totalPinjaman,
// 		   JangkaWaktu:        totalJangkaWaktu,
// 		   AngsuranTerbayar:   angsuranTerbayar,
// 		   SisaAngsuran:       sisaAngsuran,
// 		   TotalTerbayar:      totalPinjaman - totalSisaPinjaman,
// 		   SisaPokok:          totalSisaPinjaman,
// 		   PersentaseTerbayar: persentaseGabungan,
// 		   BisaAjukanLagi:     bisaAjukanLagi,
// 		   Bunga:              bungaNominal,
// 		   MetodePencairan:    metodePencairanGabungan,
// 		   MetodeAngsuran:     metodeAngsuranGabungan,
// 	   }
// 	   return info
// }

// // import (
// // 	"database/sql"
// // 	"encoding/json"
// // 	"fmt"
// // 	"net/http"
// // 	"os"
// // 	"strconv"
// // 	"strings"
// // 	"time"

// // 	"github.com/gin-contrib/sessions"
// // 	"github.com/gin-gonic/gin"
// // 	"github.com/lib/pq"
// // 	"golang.org/x/crypto/bcrypt"

// // 	"koperasi-simpan-pinjam/config"
// // 	"koperasi-simpan-pinjam/models"
// // 	"koperasi-simpan-pinjam/repository"
// // )

func isAngsuranTerbayar(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "confirmed" || s == "lunas" || s == "diterima"
}

type pinjamanAngsuranInfo struct {
	Pinjaman     models.Pinjaman
	SisaPinjaman float64
	AngsuranKe   int
}

type profilSimpananRow struct {
	Key    string
	Label  string
	Amount float64
}

type resumePinjamanInfo struct {
	IDPinjaman                  int
	Status                      string
	TglPinjaman                 time.Time
	JumlahPinjaman              float64
	JangkaWaktu                 int
	AngsuranTerbayar            int
	SisaAngsuran                int
	TotalTerbayar               float64
	SisaPokok                   float64
	AngsuranPerBulan            float64
	PersentaseTerbayar          float64
	BisaAjukanLagi              bool
	MetodePencairan             string
	MetodeAngsuran              string
	Bunga                       float64
	SisaPinjamanSebelumnya      float64
	TotalPinjamanDenganSisaLama float64
	NomorResume                 int
	IDPinjamanSebelumnya        int
}

func hitungAngsuranPerBulan(jumlahPinjaman, bungaNominal float64, jangkaWaktu int) float64 {
	if jangkaWaktu <= 0 {
		return 0
	}
	return (jumlahPinjaman + bungaNominal) / float64(jangkaWaktu)
}

type laporanSimpananColumn struct {
	Key          string
	Label        string
	BulananField string
	TotalField   string
	Jenis        string
}

// func buildProfilSimpananRows(simpananByJenis map[string]float64) []profilSimpananRow {
// 	keyToAmount := map[string]float64{
// 		"simpanan_pokok":      simpananByJenis["pokok"],
// 		"simpanan_wajib":      simpananByJenis["wajib"],
// 		"simpanan_sukarela":   simpananByJenis["sukarela"],
// 		"simpanan_hari_raya":  simpananByJenis["hari_raya"],
// 		"simpanan_umroh_haji": simpananByJenis["umroh_haji"],
// 		"simpanan_qurban":     simpananByJenis["qurban"],
// 	}
// 	defaultLabelByKey := map[string]string{
// 		"simpanan_pokok":      "Simpanan Pokok",
// 		"simpanan_wajib":      "Simpanan Wajib",
// 		"simpanan_sukarela":   "Simpanan Sukarela",
// 		"simpanan_hari_raya":  "Simpanan Hari Raya",
// 		"simpanan_umroh_haji": "Simpanan Umroh/Haji",
// 		"simpanan_qurban":     "Simpanan Qurban",
// 	}

// 	rows := []profilSimpananRow{
// 		{Key: "simpanan_pokok", Label: defaultLabelByKey["simpanan_pokok"], Amount: keyToAmount["simpanan_pokok"]},
// 	}
// 	addedKeys := map[string]bool{"simpanan_pokok": true}

// 	halamanSimpanan, errHalaman := repository.GetHalamanBySlug("simpanan")
// 	if errHalaman == nil && strings.TrimSpace(halamanSimpanan.Konten) != "" {
// 		var kontenData map[string]interface{}
// 		if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &kontenData); err == nil {
// 			if rawRows, ok := kontenData["formulir_simpanan"].([]interface{}); ok {
// 				for _, rawRow := range rawRows {
// 					rowMap, ok := rawRow.(map[string]interface{})
// 					if !ok {
// 						continue
// 					}
// 					key, _ := rowMap["key"].(string)
// 					key = strings.TrimSpace(key)
// 					if key == "" || addedKeys[key] || key == "total_simpanan" || key == "bukti" {
// 						continue
// 					}

// 					amount := 0.0
// 					if mappedAmount, known := keyToAmount[key]; known {
// 						amount = mappedAmount
// 					}

// 					label, _ := rowMap["label"].(string)
// 					label = strings.TrimSpace(label)
// 					if label == "" {
// 						label = defaultLabelByKey[key]
// 					}
// 					if label == "" {
// 						prettyKey := strings.ReplaceAll(key, "_", " ")
// 						label = strings.Title(prettyKey)
// 					}

// 					rows = append(rows, profilSimpananRow{
// 						Key:    key,
// 						Label:  label,
// 						Amount: amount,
// 					})
// 					addedKeys[key] = true
// 				}
// 			}
// 		}
// 	}

// 	fallbackOrder := []string{
// 		"simpanan_wajib",
// 		"simpanan_sukarela",
// 		"simpanan_hari_raya",
// 		"simpanan_umroh_haji",
// 		"simpanan_qurban",
// 	}
// 	for _, key := range fallbackOrder {
// 		if addedKeys[key] {
// 			continue
// 		}
// 		rows = append(rows, profilSimpananRow{
// 			Key:    key,
// 			Label:  defaultLabelByKey[key],
// 			Amount: keyToAmount[key],
// 		})
// 	}

//		return rows
//	}
func buildProfilSimpananRows(simpananByJenis map[string]float64) []profilSimpananRow {
	keyToAmount := map[string]float64{
		"simpanan_pokok":      simpananByJenis["pokok"],
		"simpanan_wajib":      simpananByJenis["wajib"],
		"simpanan_sukarela":   simpananByJenis["sukarela"],
		"simpanan_hari_raya":  simpananByJenis["hari_raya"],
		"simpanan_umroh_haji": simpananByJenis["umroh_haji"],
		"simpanan_qurban":     simpananByJenis["qurban"],
	}
	defaultLabelByKey := map[string]string{
		"simpanan_pokok":      "Simpanan Pokok",
		"simpanan_wajib":      "Simpanan Wajib",
		"simpanan_sukarela":   "Simpanan Sukarela",
		"simpanan_hari_raya":  "Simpanan Hari Raya",
		"simpanan_umroh_haji": "Simpanan Umroh/Haji",
		"simpanan_qurban":     "Simpanan Qurban",
	}

	rows := []profilSimpananRow{
		{Key: "simpanan_pokok", Label: defaultLabelByKey["simpanan_pokok"], Amount: keyToAmount["simpanan_pokok"]},
	}
	addedKeys := map[string]bool{"simpanan_pokok": true}

	halamanSimpanan, errHalaman := repository.GetHalamanBySlug("simpanan")
	if errHalaman == nil && strings.TrimSpace(halamanSimpanan.Konten) != "" {
		var kontenData map[string]interface{}
		if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &kontenData); err == nil {
			if rawRows, ok := kontenData["formulir_simpanan"].([]interface{}); ok {
				for _, rawRow := range rawRows {
					rowMap, ok := rawRow.(map[string]interface{})
					if !ok {
						continue
					}
					key, _ := rowMap["key"].(string)
					key = strings.TrimSpace(key)
					if key == "" || addedKeys[key] || key == "total_simpanan" || key == "bukti" {
						continue
					}

					amount := 0.0
					if mappedAmount, known := keyToAmount[key]; known {
						amount = mappedAmount
					}

					label, _ := rowMap["label"].(string)
					label = strings.TrimSpace(label)
					if label == "" {
						label = defaultLabelByKey[key]
					}
					if label == "" {
						prettyKey := strings.ReplaceAll(key, "_", " ")
						label = strings.Title(prettyKey)
					}

					rows = append(rows, profilSimpananRow{
						Key:    key,
						Label:  label,
						Amount: amount,
					})
					addedKeys[key] = true
				}
			}
		}
	}

	fallbackOrder := []string{
		"simpanan_wajib",
		"simpanan_sukarela",
		"simpanan_hari_raya",
		"simpanan_umroh_haji",
		"simpanan_qurban",
	}
	for _, key := range fallbackOrder {
		if addedKeys[key] {
			continue
		}
		rows = append(rows, profilSimpananRow{
			Key:    key,
			Label:  defaultLabelByKey[key],
			Amount: keyToAmount[key],
		})
	}

	return rows
}

// Fungsi untuk menggabungkan seluruh pinjaman aktif/proses menjadi satu resume gabungan
func getResumePinjamanGabungan(userID string, includeProses bool) *resumePinjamanInfo {
	// Prioritaskan pinjaman proses jika ada
	var p models.Pinjaman
	var err error

	if includeProses {
		p, err = repository.GetLatestPinjamanByStatusAndAnggotaID(userID, "proses")
		if err != nil && err != sql.ErrNoRows {
			return nil
		}
		if err == nil {
			// Temukan pinjaman proses terbaru dan langsung gunakan
			resume := buildResumePinjamanInfo(&p)
			tambahkanSisaPinjamanSebelumnya(userID, resume, p.IDPinjaman)
			return resume
		}
	}

	pList, err := repository.GetPinjamanAktifByAnggotaID(userID)
	if err != nil || len(pList) == 0 {
		return nil
	}

	var pAktif *models.Pinjaman
	for i := range pList {
		status := strings.ToLower(strings.TrimSpace(pList[i].Status))
		if status == "aktif" {
			pAktif = &pList[i]
			break
		}
	}
	if pAktif == nil {
		return nil
	}
	resume := buildResumePinjamanInfo(pAktif)
	tambahkanSisaPinjamanSebelumnya(userID, resume, pAktif.IDPinjaman)
	return resume
}

func tambahkanSisaPinjamanSebelumnya(userID string, resume *resumePinjamanInfo, currentPinjamanID int) {
	if resume == nil {
		return
	}

	totalKewajibanSekarang := resume.JumlahPinjaman + resume.Bunga
	resume.TotalPinjamanDenganSisaLama = totalKewajibanSekarang
	resume.NomorResume = 1

	pList, err := repository.GetPinjamanAktifByAnggotaID(userID)
	if err != nil {
		return
	}

	totalSisaSebelumnya := 0.0
	jumlahPinjamanSebelumnya := 0
	idPinjamanSebelumnya := 0
	var tglPinjamanSebelumnya time.Time
	for i := range pList {
		pinjamanSebelumnya := &pList[i]
		status := strings.ToLower(strings.TrimSpace(pinjamanSebelumnya.Status))
		if pinjamanSebelumnya.IDPinjaman == currentPinjamanID || status != "aktif" {
			continue
		}
		if !isPinjamanSebelumnyaUntukResume(pinjamanSebelumnya, resume, currentPinjamanID) {
			continue
		}

		resumeSebelumnya := buildResumePinjamanInfo(pinjamanSebelumnya)
		if resumeSebelumnya != nil && resumeSebelumnya.SisaPokok > 0 {
			totalSisaSebelumnya += resumeSebelumnya.SisaPokok
			jumlahPinjamanSebelumnya++
			if idPinjamanSebelumnya == 0 || pinjamanSebelumnya.TglPinjaman.After(tglPinjamanSebelumnya) {
				idPinjamanSebelumnya = pinjamanSebelumnya.IDPinjaman
				tglPinjamanSebelumnya = pinjamanSebelumnya.TglPinjaman
			}
		}
	}
	resume.NomorResume = jumlahPinjamanSebelumnya + 1

	if totalSisaSebelumnya <= 0 {
		return
	}

	resume.SisaPinjamanSebelumnya = totalSisaSebelumnya
	resume.IDPinjamanSebelumnya = idPinjamanSebelumnya
	resume.TotalPinjamanDenganSisaLama = totalKewajibanSekarang + totalSisaSebelumnya
	resume.SisaPokok += totalSisaSebelumnya
	resume.AngsuranPerBulan = hitungAngsuranPerBulan(resume.JumlahPinjaman, resume.Bunga+totalSisaSebelumnya, resume.JangkaWaktu)
}

func isPinjamanSebelumnyaUntukResume(pinjaman *models.Pinjaman, resume *resumePinjamanInfo, currentPinjamanID int) bool {
	if pinjaman.TglPinjaman.Before(resume.TglPinjaman) {
		return true
	}
	return pinjaman.TglPinjaman.Equal(resume.TglPinjaman) && pinjaman.IDPinjaman < currentPinjamanID
}

func buildResumePinjamanInfo(p *models.Pinjaman) *resumePinjamanInfo {
	statusGabungan := strings.ToLower(strings.TrimSpace(p.Status))
	tglPinjamanGabungan := p.TglPinjaman
	metodePencairanGabungan := p.MetodePencairan
	metodeAngsuranGabungan := p.MetodeAngsuran
	totalPinjaman := p.JumlahPinjaman
	totalJangkaWaktu := p.JangkaWaktu
	bungaNominal := p.JumlahPinjaman * p.Bunga / 100
	totalKewajiban := totalPinjaman + bungaNominal
	angsuranPerBulan := hitungAngsuranPerBulan(totalPinjaman, bungaNominal, totalJangkaWaktu)

	angsurans, _ := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
	angsuranTerbayar := 0
	totalAngsuranTerbayar := 0.0
	for _, a := range angsurans {
		if isAngsuranTerbayar(a.Status) {
			angsuranTerbayar++
			totalAngsuranTerbayar += a.JumlahAngsuran
		}
	}

	sisaAngsuran := p.JangkaWaktu - angsuranTerbayar
	if sisaAngsuran < 0 {
		sisaAngsuran = 0
	}

	persentaseGabungan := 0.0
	if totalKewajiban > 0 {
		persentaseGabungan = totalAngsuranTerbayar / totalKewajiban * 100
	}
	totalTerbayar := totalAngsuranTerbayar
	if totalTerbayar < 0 {
		totalTerbayar = 0
	}
	if totalTerbayar > totalKewajiban {
		totalTerbayar = totalKewajiban
	}
	sisaPokok := totalKewajiban - totalTerbayar
	if sisaPokok < 0 {
		sisaPokok = 0
	}
	if sisaPokok > totalKewajiban {
		sisaPokok = totalKewajiban
	}

	if statusGabungan == "proses" {
		angsuranTerbayar = 0
		sisaAngsuran = totalJangkaWaktu
		persentaseGabungan = 0
		totalTerbayar = 0
		sisaPokok = totalKewajiban
	}

	bisaAjukanLagi := (persentaseGabungan >= 50) && (statusGabungan != "proses")

	if persentaseGabungan >= 100 {
		return &resumePinjamanInfo{
			IDPinjaman:                  p.IDPinjaman,
			Status:                      "lunas",
			TglPinjaman:                 tglPinjamanGabungan,
			JumlahPinjaman:              totalPinjaman,
			JangkaWaktu:                 totalJangkaWaktu,
			AngsuranTerbayar:            angsuranTerbayar,
			SisaAngsuran:                0,
			TotalTerbayar:               totalTerbayar,
			SisaPokok:                   0,
			AngsuranPerBulan:            angsuranPerBulan,
			PersentaseTerbayar:          100,
			BisaAjukanLagi:              true,
			Bunga:                       bungaNominal,
			MetodePencairan:             metodePencairanGabungan,
			MetodeAngsuran:              metodeAngsuranGabungan,
			TotalPinjamanDenganSisaLama: totalKewajiban,
			NomorResume:                 1,
		}
	}

	return &resumePinjamanInfo{
		IDPinjaman:                  p.IDPinjaman,
		Status:                      statusGabungan,
		TglPinjaman:                 tglPinjamanGabungan,
		JumlahPinjaman:              totalPinjaman,
		JangkaWaktu:                 totalJangkaWaktu,
		AngsuranTerbayar:            angsuranTerbayar,
		SisaAngsuran:                sisaAngsuran,
		TotalTerbayar:               totalTerbayar,
		SisaPokok:                   sisaPokok,
		AngsuranPerBulan:            angsuranPerBulan,
		PersentaseTerbayar:          persentaseGabungan,
		BisaAjukanLagi:              bisaAjukanLagi,
		Bunga:                       bungaNominal,
		MetodePencairan:             metodePencairanGabungan,
		MetodeAngsuran:              metodeAngsuranGabungan,
		TotalPinjamanDenganSisaLama: totalKewajiban,
		NomorResume:                 1,
	}
}

func getLaporanSimpananColumns() (map[string]string, []laporanSimpananColumn) {
	defaultLabels := map[string]string{
		"simpanan_pokok":     "Pokok",
		"simpanan_wajib":     "Wajib",
		"simpanan_hari_raya": "Simpanan Hari Raya",
		"simpanan_sukarela":  "Sukarela",
	}
	labelByKey := map[string]string{
		"simpanan_pokok":     defaultLabels["simpanan_pokok"],
		"simpanan_wajib":     defaultLabels["simpanan_wajib"],
		"simpanan_hari_raya": defaultLabels["simpanan_hari_raya"],
		"simpanan_sukarela":  defaultLabels["simpanan_sukarela"],
	}

	baseKeys := map[string]bool{
		"simpanan_pokok":     true,
		"simpanan_wajib":     true,
		"simpanan_hari_raya": true,
		"simpanan_sukarela":  true,
	}

	customCols := []laporanSimpananColumn{}
	added := map[string]bool{}

	halamanSimpanan, errHalaman := repository.GetHalamanBySlug("simpanan")
	if errHalaman != nil || strings.TrimSpace(halamanSimpanan.Konten) == "" {
		return labelByKey, customCols
	}

	var kontenData map[string]interface{}
	if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &kontenData); err != nil {
		return labelByKey, customCols
	}

	rawRows, ok := kontenData["formulir_simpanan"].([]interface{})
	if !ok {
		return labelByKey, customCols
	}

	for _, rawRow := range rawRows {
		rowMap, ok := rawRow.(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := rowMap["key"].(string)
		key = strings.TrimSpace(key)
		if key == "" || key == "total_simpanan" || key == "bukti" || added[key] {
			continue
		}
		added[key] = true

		label, _ := rowMap["label"].(string)
		label = strings.TrimSpace(label)
		if label == "" {
			prettyKey := strings.ReplaceAll(strings.TrimPrefix(key, "simpanan_"), "_", " ")
			label = strings.Title(prettyKey)
		}

		if baseKeys[key] {
			labelByKey[key] = label
			continue
		}

		jenis := strings.TrimPrefix(key, "simpanan_")
		customCols = append(customCols, laporanSimpananColumn{
			Key:          key,
			Label:        label,
			BulananField: "custom_bulanan_" + jenis,
			TotalField:   "custom_total_" + jenis,
			Jenis:        jenis,
		})
	}

	return labelByKey, customCols
}

func hydrateCustomSimpananValuesToLaporanDetail(laporanDetail []map[string]interface{}, customCols []laporanSimpananColumn, bulan, tahun int) {
	if len(laporanDetail) == 0 || len(customCols) == 0 {
		return
	}

	jenisSet := map[string]bool{}
	jenisList := []string{}
	for _, col := range customCols {
		if col.Jenis == "" || jenisSet[col.Jenis] {
			continue
		}
		jenisSet[col.Jenis] = true
		jenisList = append(jenisList, col.Jenis)
	}
	if len(jenisList) == 0 {
		return
	}

	db := config.GetDB()
	query := `
		SELECT d.id_anggota, s.jenis_simpanan,
		       COALESCE(SUM(CASE
		           WHEN ($1 = 0 OR EXTRACT(MONTH FROM d.tgl_transaksi) = $1)
		            AND EXTRACT(YEAR FROM d.tgl_transaksi) = $2
		           THEN d.jumlah_simpanan ELSE 0 END), 0) AS bulanan,
		       COALESCE(SUM(d.jumlah_simpanan), 0) AS total
		FROM detail d
		JOIN simpanan s ON d.id_simpanan = s.id_simpanan
		WHERE COALESCE(d.status, 'pending') IN ('confirmed', 'diterima', 'lunas')
		  AND s.jenis_simpanan = ANY($3)
		GROUP BY d.id_anggota, s.jenis_simpanan
	`
	rows, err := db.Query(query, bulan, tahun, pq.Array(jenisList))
	if err != nil {
		for i := range laporanDetail {
			for _, col := range customCols {
				laporanDetail[i][col.BulananField] = 0.0
				laporanDetail[i][col.TotalField] = 0.0
			}
		}
		return
	}
	defer rows.Close()

	type nilai struct {
		Bulanan float64
		Total   float64
	}
	byAnggotaJenis := map[string]map[string]nilai{}
	for rows.Next() {
		var idAnggota, jenis string
		var bulanan, total float64
		if err := rows.Scan(&idAnggota, &jenis, &bulanan, &total); err != nil {
			continue
		}
		if _, ok := byAnggotaJenis[idAnggota]; !ok {
			byAnggotaJenis[idAnggota] = map[string]nilai{}
		}
		byAnggotaJenis[idAnggota][jenis] = nilai{Bulanan: bulanan, Total: total}
	}

	for i := range laporanDetail {
		idAnggota, _ := laporanDetail[i]["id_anggota"].(string)
		for _, col := range customCols {
			v := nilai{}
			if byJenis, ok := byAnggotaJenis[idAnggota]; ok {
				if found, ok := byJenis[col.Jenis]; ok {
					v = found
				}
			}
			laporanDetail[i][col.BulananField] = v.Bulanan
			laporanDetail[i][col.TotalField] = v.Total
		}
	}
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
		jumlahAngsuranTerbayar := 0
		for _, a := range angsurans {
			if isAngsuranTerbayar(a.Status) {
				totalAngsuranTerbayar += a.JumlahAngsuran
				jumlahAngsuranTerbayar++
			}
		}

		sisaPinjaman := p.JumlahPinjaman - totalAngsuranTerbayar
		if sisaPinjaman < 0 {
			sisaPinjaman = 0
		}

		// Angsuran ke dihitung dari jumlah angsuran yang sudah benar-benar terbayar,
		// agar konsisten dengan perhitungan persentase pelunasan.
		angsuranKe := jumlahAngsuranTerbayar + 1
		if p.JangkaWaktu > 0 && angsuranKe > p.JangkaWaktu {
			angsuranKe = p.JangkaWaktu
		}
		if angsuranKe < 1 {
			angsuranKe = 1
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
	return prioritas, nil
}

func getTotalAngsuranAktif(pinjamans []models.Pinjaman) (float64, float64, error) {
	infos, err := getPinjamanAngsuranInfos(pinjamans)
	if err != nil {
		return 0, 0, err
	}

	var totalPinjaman float64
	var totalSisaPinjaman float64
	for _, info := range infos {
		if info.SisaPinjaman <= 0 {
			continue
		}
		totalPinjaman += info.Pinjaman.JumlahPinjaman
		totalSisaPinjaman += info.SisaPinjaman
	}
	return totalPinjaman, totalSisaPinjaman, nil
}

func getRingkasanPinjamanAktifByAnggotaID(idAnggota string) (float64, float64, error) {
	db := config.GetDB()

	var totalPinjamanAktif float64
	queryTotal := `
		SELECT COALESCE(SUM(jumlah_pinjaman), 0)
		FROM pinjaman
		WHERE id_anggota = $1 AND status = 'aktif'
	`
	if err := db.QueryRow(queryTotal, idAnggota).Scan(&totalPinjamanAktif); err != nil {
		return 0, 0, err
	}

	var totalAngsuranTerkonfirmasi float64
	queryAngsuran := `
		SELECT COALESCE(SUM(a.jumlah_angsuran), 0)
		FROM angsuran a
		JOIN pinjaman p ON a.id_pinjaman = p.id_pinjaman
		WHERE p.id_anggota = $1
		  AND p.status = 'aktif'
		  AND COALESCE(LOWER(a.status), '') IN ('confirmed', 'lunas', 'diterima')
	`
	if err := db.QueryRow(queryAngsuran, idAnggota).Scan(&totalAngsuranTerkonfirmasi); err != nil {
		return 0, 0, err
	}

	totalSisa := totalPinjamanAktif - totalAngsuranTerkonfirmasi
	if totalSisa < 0 {
		totalSisa = 0
	}

	return totalPinjamanAktif, totalSisa, nil
}

// BARU — tambahkan query DB di bagian akhir sebelum return
// Gabungkan resume pinjaman baru (proses/aktif) dan resume lama yang belum lunas (aktif, sisa pokok > 0)
// modeGabungan: true = tampilkan semua pinjaman (termasuk lunas/dibatalkan), false = hanya aktif/proses
func getResumePinjamanInfo(userID string, modeGabungan bool) []resumePinjamanInfo {
	var pinjamans []models.Pinjaman
	var err error
	if modeGabungan {
		pinjamans, err = repository.GetRiwayatPinjamanByAnggotaID(userID, "")
	} else {
		pinjamans, err = repository.GetRiwayatPinjamanByAnggotaID(userID, "")
	}
	if err != nil || len(pinjamans) == 0 {
		return nil
	}
	var result []resumePinjamanInfo
	for i := range pinjamans {
		p := &pinjamans[i]
		angsurans, _ := repository.GetAngsuranByPinjamanID(p.IDPinjaman)
		totalAngsuranTerbayar := 0.0
		jumlahAngsuranTerbayar := 0
		for _, a := range angsurans {
			if isAngsuranTerbayar(a.Status) {
				totalAngsuranTerbayar += a.JumlahAngsuran
				jumlahAngsuranTerbayar++
			}
		}
		sisaPinjaman := p.JumlahPinjaman - totalAngsuranTerbayar
		if sisaPinjaman < 0 {
			sisaPinjaman = 0
		}
		// Filter: hanya tampilkan pinjaman aktif/proses yang belum lunas (sisa pokok > 0)
		status := strings.ToLower(p.Status)
		if !(status == "proses" || (status == "aktif" && sisaPinjaman > 0)) {
			continue
		}
		info := resumePinjamanInfo{
			IDPinjaman:         p.IDPinjaman,
			Status:             p.Status,
			TglPinjaman:        p.TglPinjaman,
			JumlahPinjaman:     p.JumlahPinjaman,
			JangkaWaktu:        p.JangkaWaktu,
			AngsuranTerbayar:   jumlahAngsuranTerbayar,
			SisaAngsuran:       p.JangkaWaktu - jumlahAngsuranTerbayar,
			TotalTerbayar:      0,
			SisaPokok:          sisaPinjaman,
			PersentaseTerbayar: 0,
			BisaAjukanLagi:     false,
			Bunga:              p.Bunga,
		}
		if info.SisaAngsuran < 0 {
			info.SisaAngsuran = 0
		}
		// perkiraanAngsuranBulan tidak diperlukan lagi
		info.TotalTerbayar = totalAngsuranTerbayar
		if p.JumlahPinjaman > 0 {
			info.PersentaseTerbayar = float64(p.JumlahPinjaman-sisaPinjaman) / float64(p.JumlahPinjaman) * 100
			if info.PersentaseTerbayar < 0 {
				info.PersentaseTerbayar = 0
			}
			if info.PersentaseTerbayar > 100 {
				info.PersentaseTerbayar = 100
			}
		}
		info.BisaAjukanLagi = info.PersentaseTerbayar >= 50
		db := config.GetDB()
		db.QueryRow(
			`SELECT COALESCE(metode_pencairan, ''), COALESCE(metode_angsuran, '') 
			FROM pinjaman WHERE id_pinjaman = $1`,
			p.IDPinjaman,
		).Scan(&info.MetodePencairan, &info.MetodeAngsuran)
		result = append(result, info)
	}
	return result
}

// func getResumePinjamanInfo(userID string) resumePinjamanInfo {
// 	info := resumePinjamanInfo{}

// 	pinjamans, err := repository.GetPinjamanAktifByAnggotaID(userID)
// 	if err != nil || len(pinjamans) == 0 {
// 		return info
// 	}

// 	// Gunakan logika yang sama dengan halaman angsuran
// 	pinjamanInfo, err := getPinjamanPrioritasAngsuran(pinjamans)
// 	if err != nil || pinjamanInfo == nil || pinjamanInfo.SisaPinjaman <= 0 {
// 		return info
// 	}

// 	p := pinjamanInfo.Pinjaman
// 	info.HasActiveLoan = true
// 	info.IDPinjaman = p.IDPinjaman
// 	info.Status = p.Status
// 	info.TglPinjaman = p.TglPinjaman
// 	info.JumlahPinjaman = p.JumlahPinjaman
// 	info.JangkaWaktu = p.JangkaWaktu
// 	info.SisaPokok = pinjamanInfo.SisaPinjaman
// 	info.AngsuranTerbayar = pinjamanInfo.AngsuranKe - 1
// 	if info.AngsuranTerbayar < 0 {
// 		info.AngsuranTerbayar = 0
// 	}
// 	info.SisaAngsuran = p.JangkaWaktu - info.AngsuranTerbayar
// 	if info.SisaAngsuran < 0 {
// 		info.SisaAngsuran = 0
// 	}
// 	perkiraanAngsuranBulan := 0.0
// 	if p.JangkaWaktu > 0 {
// 		pokokPerBulan := p.JumlahPinjaman / float64(p.JangkaWaktu)
// 		jasaPerBulan := (p.Bunga / 100 * p.JumlahPinjaman) / float64(p.JangkaWaktu)
// 		perkiraanAngsuranBulan = pokokPerBulan + jasaPerBulan
// 	}
// 	info.TotalTerbayar = float64(info.AngsuranTerbayar) * perkiraanAngsuranBulan
// 	info.PersentaseTerbayar = 0.0
// 	if p.JumlahPinjaman > 0 {
// 		info.PersentaseTerbayar = float64(p.JumlahPinjaman-pinjamanInfo.SisaPinjaman) / float64(p.JumlahPinjaman) * 100
// 		if info.PersentaseTerbayar < 0 {
// 			info.PersentaseTerbayar = 0
// 		}
// 		if info.PersentaseTerbayar > 100 {
// 			info.PersentaseTerbayar = 100
// 		}
// 	}
// 	info.BisaAjukanLagi = info.PersentaseTerbayar >= 50
// 	return info
// }

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

	latestBackground := "/static/images/placeholder.png"
	var latestBackgroundTime int64
	if errLogo == nil {
		for _, file := range dirFiles {
			name := file.Name()
			if strings.HasPrefix(name, "background_") && (strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg")) {
				info, errInfo := file.Info()
				if errInfo == nil {
					modTime := info.ModTime().Unix()
					if modTime > latestBackgroundTime {
						latestBackgroundTime = modTime
						latestBackground = "/static/images/" + name
					}
				}
			}
		}
	}

	// Render halaman dashboard dan kirim data anggota dan konten ke sana
	c.HTML(http.StatusOK, "anggota_dashboard.html", gin.H{
		"Anggota":           anggota,
		"KontenParsed":      kontenParsed,
		"CurrentLogo":       latestLogo,
		"CurrentBackground": latestBackground,
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

	// Hitung Total Simpanan (tidak termasuk Simpanan Pokok karena dibayar saat pendaftaran)
	totalSimpanan := simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

	profilSimpananRows := buildProfilSimpananRows(simpananByJenis)

	// Tampilkan sisa pinjaman aktif yang sudah dikurangi angsuran terbayar.
	// Pengajuan yang masih status "proses" belum boleh muncul di ringkasan ini.
	_, totalPinjaman, err := getRingkasanPinjamanAktifByAnggotaID(userID)
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

	// Render halaman profil dan kirim data anggota, saldo, dan logo ke sana
	c.HTML(http.StatusOK, "anggota_profil.html", gin.H{
		"Anggota":            anggota,
		"TotalSimpanan":      totalSimpanan,
		"TotalPinjaman":      totalPinjaman,
		"ProfilSimpananRows": profilSimpananRows,
		"SimpananPokok":      simpananByJenis["pokok"],
		"SimpananWajib":      simpananByJenis["wajib"],
		"SimpananSukarela":   simpananByJenis["sukarela"],
		"SimpananHariRaya":   simpananByJenis["hari_raya"],
		"SimpananUmrohHaji":  simpananByJenis["umroh_haji"],
		"SimpananQurban":     simpananByJenis["qurban"],
		"CurrentLogo":        latestLogo,
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

	// Saat halaman pesan dibuka, anggap pesan sudah dilihat oleh anggota.
	_ = repository.MarkAllPesanAsReadByAnggotaID(userID)

	// Ambil daftar pesan untuk anggota
	pesans, err := repository.GetPesanByAnggotaID(userID)
	if err != nil {
		// Jika gagal ambil pesan, tetap tampilkan halaman dengan pesan kosong
		pesans = []models.Pesan{}
	}

	latestPesanID, _, errSummary := repository.GetPesanNotifSummaryByAnggotaID(userID)
	if errSummary != nil {
		latestPesanID = 0
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
		"Title":         "Pesan Saya",
		"Anggota":       anggota,
		"Pesans":        pesans,
		"LatestPesanID": latestPesanID,
		"CurrentLogo":   latestLogo,
	})
}

// AnggotaPesanNotifikasi mengembalikan status notifikasi pesan anggota (JSON).
func AnggotaPesanNotifikasi(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	latestPesanID, unreadCount, err := repository.GetPesanNotifSummaryByAnggotaID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil notifikasi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"latest_id":    latestPesanID,
		"unread_count": unreadCount,
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

	// Convert string to float64
	simpananWajib, err := strconv.ParseFloat(simpananWajibStr, 64)
	if err != nil {
		log.Printf("[ERROR] AnggotaAjukanKeluarKoperasi parse simpanan_wajib gagal: %v", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format simpanan wajib tidak valid")
		return
	}

	simpananLainnya, err := strconv.ParseFloat(simpananLainnyaStr, 64)
	if err != nil {
		log.Printf("[ERROR] AnggotaAjukanKeluarKoperasi parse simpanan_lainnya gagal: %v", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format simpanan lainnya tidak valid")
		return
	}

	biayaAdmin, err := strconv.ParseFloat(biayaAdminStr, 64)
	if err != nil {
		log.Printf("[ERROR] AnggotaAjukanKeluarKoperasi parse biaya_admin gagal: %v", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Format biaya admin tidak valid")
		return
	}

	// Validasi alasan keluar tidak kosong
	if strings.TrimSpace(alasanKeluar) == "" {
		log.Printf("[ERROR] AnggotaAjukanKeluarKoperasi alasan_keluar kosong")
		c.Redirect(http.StatusFound, "/anggota/profil?error=Alasan keluar harus diisi")
		return
	}

	// Update status anggota menjadi 'pending_keluar' dan simpan data pengajuan
	db := config.GetDB()

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
		log.Printf("[ERROR] AnggotaAjukanKeluarKoperasi update status keluar gagal: %v", err)
		c.Redirect(http.StatusFound, "/anggota/profil?error=Gagal mengajukan keluar dari koperasi")
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

	templateData := getAjukanPinjamanTemplateData(userID, anggota)
	templateData["CurrentLogo"] = latestLogo
	templateData["Judul"] = "Ajukan Pinjaman"
	// Tambahkan histori pinjaman ke template
	riwayatPinjaman, _ := repository.GetRiwayatPinjamanByAnggotaID(userID, "")
	templateData["RiwayatPinjaman"] = riwayatPinjaman
	c.HTML(http.StatusOK, "anggota_ajukan_pinjaman.html", templateData)
}

// getAjukanPinjamanTemplateData adalah helper function untuk mendapatkan data template yang konsisten
func getAjukanPinjamanTemplateData(userID string, anggota models.Anggota) gin.H {
	// Hitung total simpanan (excluding pokok, same as Profil)
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
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
	totalSimpanan := simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

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

	resumeGabungan := getResumePinjamanGabungan(userID, true)
	var resumeGabunganSlice []resumePinjamanInfo
	if resumeGabungan != nil {
		resumeGabunganSlice = append(resumeGabunganSlice, *resumeGabungan)
	} else {
		resumeGabunganSlice = []resumePinjamanInfo{}
	}
	return gin.H{
		"Judul":          "Ajukan Pinjaman",
		"Anggota":        anggota,
		"TotalSimpanan":  totalSimpanan,
		"LimitPinjaman":  limitPinjaman,
		"JenisAnggota":   jenisAnggota,
		"Bunga":          bungaTerkini,
		"GajiBulanan":    anggota.GajiBulanan,
		"ResumeGabungan": resumeGabunganSlice,
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
	// Ambil field penting dari form
	metodePencairanStr := c.PostForm("metode_pencairan")
	metodeAngsuranStr := c.PostForm("metode_angsuran")
	formDebug := make(map[string][]string)
	for k, v := range c.Request.Form {
		formDebug[k] = v
	}
	// Validasi wajib pilih metode pencairan dan angsuran
	if metodePencairanStr == "" {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Metode pencairan wajib dipilih."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}
	if metodeAngsuranStr == "" {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Metode angsuran wajib dipilih."
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

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

	// PATCH: Validasi pengajuan pinjaman baru menggunakan resumeGabungan
	resumeGabungan := getResumePinjamanGabungan(userID, true)
	if resumeGabungan != nil && !resumeGabungan.BisaAjukanLagi {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		errMsg := "Anda belum memenuhi syarat untuk mengajukan pinjaman baru."
		if resumeGabungan.PersentaseTerbayar < 50 {
			errMsg = fmt.Sprintf("Anda baru melunasi %.2f%% pinjaman. Minimal harus 50%% untuk mengajukan pinjaman baru.", resumeGabungan.PersentaseTerbayar)
		}
		templateData["Error"] = errMsg
		c.HTML(http.StatusBadRequest, "anggota_ajukan_pinjaman.html", templateData)
		return
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
	// pinjaman.MetodePencairan = c.PostForm("metode_pencairan")
	// BARU
	pinjaman.MetodePencairan = c.PostForm("metode_pencairan")
	pinjaman.MetodeAngsuran = c.PostForm("metode_angsuran") // ← tambahan
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

	_, err = repository.CreatePinjamanReturningID(pinjaman)
	if err != nil {
		templateData := getAjukanPinjamanTemplateData(userID, anggota)
		templateData["Error"] = "Gagal mengajukan pinjaman. Silakan coba lagi."
		c.HTML(http.StatusInternalServerError, "anggota_ajukan_pinjaman.html", templateData)
		return
	}

	// Jangan buat jadwal angsuran otomatis saat pengajuan pinjaman masih berstatus 'proses'.
	// Jadwal angsuran baru akan dibuat setelah pinjaman dikonfirmasi dan status diubah menjadi 'aktif'.

	// Kirim notifikasi WA ke bendahara jika ada pengajuan pinjaman baru
	bendahara, err := repository.GetBendahara()
	if err == nil {
		anggota, _ := repository.GetAnggotaByID(userID)
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		nominal := fmt.Sprintf("%.2f", pinjaman.JumlahPinjaman)
		if errWA := sendBendaharaWhatsAppNotification(bendahara.NoTelepon, anggota.NamaAnggota, "Pinjaman", nominal, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi bendahara (pinjaman): %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF] bendahara tidak ditemukan untuk notifikasi pinjaman: %v", err)
	}

	// Berhasil, redirect ke halaman pengajuan pinjaman agar Resume Pinjaman update otomatis
	c.Redirect(http.StatusFound, "/anggota/ajukan-pinjaman")
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

	// Ambil konfigurasi simpanan wajib
	config, _ := repository.GetKonfigurasiSimpananWajib()
	nominalSimpananWajib := 0.0
	if val, ok := config["PersentasePotong"].(float64); ok {
		nominalSimpananWajib = val
	}
	potonganBulanIni, _ := repository.GetPotonganBulanIniAllAnggota()
	sisaGajiPotong := float64(anggota.GajiBulanan) - potonganBulanIni[userID]

	// Ambil data resume gabungan (bungkus ke slice jika tidak nil)
	resumeGabungan := getResumePinjamanGabungan(userID, true)
	var resumeGabunganSlice []resumePinjamanInfo
	if resumeGabungan != nil {
		resumeGabunganSlice = append(resumeGabunganSlice, *resumeGabungan)
	} else {
		resumeGabunganSlice = []resumePinjamanInfo{}
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

	// Ambil nomor rekening koperasi dari repository
	nomorRekening, _ := repository.GetNomorRekening("simpanan")

	log.Printf("DEBUG AnggotaSimpanan: userID=%s UnitKerja=%s GajiBulanan=%d", userID, anggota.UnitKerja, anggota.GajiBulanan)
	c.HTML(http.StatusOK, "anggota_simpanan_fixed.html", gin.H{
		"DebugUnitKerja":       anggota.UnitKerja,
		"Judul":                "Simpanan",
		"Anggota":              anggota,
		"Now":                  time.Now(),
		"NominalSimpananWajib": nominalSimpananWajib,
		"SisaGajiPotong":       sisaGajiPotong,
		"ResumeGabungan":       resumeGabunganSlice,
		"CurrentLogo":          latestLogo,
		"NomorRekening":        nomorRekening,
	})
}

// AnggotaSimpananPost memproses pengajuan simpanan
func AnggotaSimpananPost(c *gin.Context) {
	session := sessions.Default(c)
	userID, ok := session.Get("user_id").(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Session tidak valid. Silakan login ulang."})
		return
	}

	anggota, err := repository.GetAnggotaByID(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Data anggota tidak ditemukan."})
		return
	}

	configSimpananWajib, _ := repository.GetKonfigurasiSimpananWajib()
	nominalSimpananWajib := 0.0
	if val, ok := configSimpananWajib["PersentasePotong"].(float64); ok {
		nominalSimpananWajib = val
	}
	potonganBulanIni, _ := repository.GetPotonganBulanIniAllAnggota()
	sisaGajiPotong := float64(anggota.GajiBulanan) - potonganBulanIni[userID]

	// ==================== BULLETPROOF MULTIPART PARSING FIX ====================
	log.Printf("[SIMPANAN-POST] user=%s ContentType='%s'", userID, c.ContentType())

	// 1. CRITICAL: ParseForm FIRST (populates r.Form map)
	if err := c.Request.ParseForm(); err != nil {
		log.Printf("[SIMPANAN-POST ERROR] ParseForm FAIL: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Form parse error"})
		return
	}

	// 2. ParseMultipartForm with 128MB limit (safe for multiple files)
	if err := c.Request.ParseMultipartForm(128 << 20); err != nil {
		log.Printf("[SIMPANAN-POST ERROR] ParseMultipartForm FAIL: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Multipart parse error: " + err.Error()})
		return
	}

	// 3. BULLETPROOF: DIRECT r.Form.Get() - bypasses c.PostForm() race conditions
	metodePembayaran := strings.TrimSpace(c.Request.Form.Get("metode_pembayaran"))

	// 4. DEBUG: Log ALL form keys received
	formKeys := []string{}
	for key := range c.Request.Form {
		formKeys = append(formKeys, key)
	}
	log.Printf("[SIMPANAN-POST FORM KEYS] user=%s keys=%v | metode='%s'", userID, formKeys, metodePembayaran)

	log.Printf("[SIMPANAN-POST BULLETPROOF] user=%s metode='%s' (len=%d)", userID, metodePembayaran, len(metodePembayaran))

	// 5. VALIDATION: Whitelist + non-empty
	validMethods := map[string]bool{"transfer_bank": true, "potong_gaji": true, "tunai": true}
	validMethodList := []string{"transfer_bank", "potong_gaji", "tunai"}
	if metodePembayaran == "" {
		log.Printf("[SIMPANAN-POST REJECT] metode_pembayaran kosong | all_keys=%v", formKeys)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"field":         "metode_pembayaran",
			"message":       "❌ Metode pembayaran wajib dipilih. Pilih salah satu: transfer_bank, potong_gaji, atau tunai.",
			"debug_keys":    formKeys,
			"valid_methods": validMethodList,
		})
		return
	}
	if !validMethods[metodePembayaran] {
		log.Printf("[SIMPANAN-POST REJECT] invalid method='%s' | all_keys=%v", metodePembayaran, formKeys)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":       false,
			"field":         "metode_pembayaran",
			"message":       fmt.Sprintf("❌ Metode pembayaran '%s' tidak valid. Pilih salah satu: transfer_bank, potong_gaji, atau tunai.", metodePembayaran),
			"debug_keys":    formKeys,
			"valid_methods": validMethodList,
		})
		return
	}
	log.Printf("[SIMPANAN-POST] ✓ METHOD VALIDATED: '%s'", metodePembayaran)

	if metodePembayaran == "potong_gaji" && sisaGajiPotong <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"field":   "metode_pembayaran",
			"message": "âŒ Metode potong gaji tidak dapat dipakai karena data gaji Anda belum ada atau bernilai 0.",
		})
		return
	}

	// Parse fields SAFELY - handle empty/missing
	wajibStr := strings.TrimSpace(c.PostForm("simpanan_wajib"))
	sukarelaStr := strings.TrimSpace(c.PostForm("simpanan_sukarela"))
	hariRayaStr := strings.TrimSpace(c.PostForm("simpanan_hari_raya"))
	umrohHajiStr := strings.TrimSpace(c.PostForm("simpanan_umroh_haji"))
	qurbanStr := strings.TrimSpace(c.PostForm("simpanan_qurban"))

	var wajib, sukarela, hariRaya, umrohHaji, qurban float64 = 0, 0, 0, 0, 0
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

	log.Printf("[SIMPANAN-POST] user=%s ✓ METHOD OK parsed: wajib=%.0f sukarela=%.0f hariRaya=%.0f umroh=%.0f qurban=%.0f total=%.0f metode='%s'",
		userID, wajib, sukarela, hariRaya, umrohHaji, qurban, wajib+sukarela+hariRaya+umrohHaji+qurban, metodePembayaran)

	if wajib <= 0 && sukarela <= 0 && hariRaya <= 0 && umrohHaji <= 0 && qurban <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "❌ Isi minimal 1 jenis simpanan > Rp0", "debug": fmt.Sprintf("amounts=[%.0f,%.0f,%.0f,%.0f,%.0f]", wajib, sukarela, hariRaya, umrohHaji, qurban)})
		return
	}

	if nominalSimpananWajib > 0 && wajib > 0 && (wajib < nominalSimpananWajib || math.Mod(wajib, nominalSimpananWajib) != 0) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"field":   "simpanan_wajib",
			"message": fmt.Sprintf("Nominal Simpanan Wajib harus Rp%.0f atau kelipatannya, misalnya Rp%.0f.", nominalSimpananWajib, nominalSimpananWajib*2),
		})
		return
	}

	// BUKTI for TRANSFER_BANK - FIXED
	if metodePembayaran == "transfer_bank" {
		file, err := c.FormFile("bukti")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "field": "bukti", "message": "❌ Transfer Bank → Upload bukti (.jpg/.png/.pdf)!"})
			return
		}
		if file.Size == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "field": "bukti", "message": "❌ File bukti kosong! Pilih file yang valid."})
			return
		}
		log.Printf("[SIMPANAN-POST] user=%s bukti OK: %s (size=%d)", userID, file.Filename, file.Size)
	}

	log.Printf("[SIMPANAN-FIX] user=%s(373) metode='%s' amounts=[%.0f,%.0f,%.0f,%.0f,%.0f]", userID, metodePembayaran, wajib, sukarela, hariRaya, umrohHaji, qurban)

	// Set tanggal pengajuan otomatis (fallback to today)
	tanggalPengajuan := time.Now()
	if t := c.PostForm("tanggal_pengajuan"); t != "" {
		log.Printf("[SIMPANAN-POST] tanggal_pengajuan='%s'", t)
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			now := time.Now()
			tanggalPengajuan = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
		}
	} else {
		log.Printf("[SIMPANAN-POST] No tanggal_pengajuan - using now")
	}

	var total float64 = wajib + sukarela + hariRaya + umrohHaji + qurban

	if metodePembayaran == "potong_gaji" && total >= sisaGajiPotong {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"field":   "metode_pembayaran",
			"message": "Metode potong gaji tidak dapat dipakai jika total simpanan menghabiskan sisa gaji Anda. Kurangi nominal atau pilih metode lain.",
		})
		return
	}

	// Handle file upload: hanya wajib untuk Transfer Bank
	var filename string
	if metodePembayaran == "transfer_bank" {
		file, err := c.FormFile("bukti")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Bukti pembayaran wajib diupload untuk metode Transfer Bank.",
			})
			return
		}
		if file.Size == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Bukti pembayaran tidak boleh kosong untuk metode Transfer Bank.",
			})
			return
		}

		// Save the uploaded file
		filename = time.Now().Format("20060102150405") + "_" + file.Filename
		dst := "./static/uploads/" + filename
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Gagal menyimpan file bukti pembayaran.",
			})
			return
		}
	} else {
		// Untuk potong gaji / tunai, bukti opsional
		file, err := c.FormFile("bukti")
		if err == nil {
			filename = time.Now().Format("20060102150405") + "_" + file.Filename
			dst := "./static/uploads/" + filename
			_ = c.SaveUploadedFile(file, dst)
		}
	}

	// Buat entri untuk setiap jenis simpanan yang > 0
	// IDSimpanan mapping: pokok(1), wajib(2), sukarela(3), hari_raya(4), umroh_haji(5), qurban(6)
	var errs []error

	// DEBUG: Log sebelum create simpanan
	log.Printf("[SIMPANAN-CREATE] user=%s memulai penyimpanan simpanan | wajib=%.0f sukarela=%.0f hariRaya=%.0f umroh=%.0f qurban=%.0f metode=%s",
		userID, wajib, sukarela, hariRaya, umrohHaji, qurban, metodePembayaran)

	if wajib > 0 {
		d := models.Detail{
			IDAnggota:        userID,
			IDSimpanan:       2,
			IDPengelola:      1,
			TglTransaksi:     tanggalPengajuan,
			JumlahSimpanan:   wajib,
			TotalSimpanan:    total,
			BuktiPembayaran:  filename,
			MetodePembayaran: metodePembayaran,
		}
		log.Printf("[SIMPANAN-CREATE] inserting simpanan wajib: id_anggota=%s id_simpanan=2 jumlah=%.0f status=%s",
			d.IDAnggota, d.JumlahSimpanan, d.Status)
		if e := repository.CreateSimpanan(d); e != nil {
			log.Printf("[SIMPANAN-CREATE] ERROR wajib: %v", e)
			errs = append(errs, e)
		} else {
			log.Printf("[SIMPANAN-CREATE] SUCCESS wajib disimpan untuk %s", userID)
		}
	}
	if sukarela > 0 {
		d := models.Detail{
			IDAnggota:        userID,
			IDSimpanan:       3,
			IDPengelola:      1,
			TglTransaksi:     tanggalPengajuan,
			JumlahSimpanan:   sukarela,
			TotalSimpanan:    total,
			BuktiPembayaran:  filename,
			MetodePembayaran: metodePembayaran,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if hariRaya > 0 {
		d := models.Detail{
			IDAnggota:        userID,
			IDSimpanan:       4,
			IDPengelola:      1,
			TglTransaksi:     tanggalPengajuan,
			JumlahSimpanan:   hariRaya,
			TotalSimpanan:    total,
			BuktiPembayaran:  filename,
			MetodePembayaran: metodePembayaran,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if umrohHaji > 0 {
		d := models.Detail{
			IDAnggota:        userID,
			IDSimpanan:       5,
			IDPengelola:      1,
			TglTransaksi:     tanggalPengajuan,
			JumlahSimpanan:   umrohHaji,
			TotalSimpanan:    total,
			BuktiPembayaran:  filename,
			MetodePembayaran: metodePembayaran,
		}
		if e := repository.CreateSimpanan(d); e != nil {
			errs = append(errs, e)
		}
	}
	if qurban > 0 {
		d := models.Detail{
			IDAnggota:        userID,
			IDSimpanan:       6,
			IDPengelola:      1,
			TglTransaksi:     tanggalPengajuan,
			JumlahSimpanan:   qurban,
			TotalSimpanan:    total,
			BuktiPembayaran:  filename,
			MetodePembayaran: metodePembayaran,
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

	// Kirim notifikasi WA ke bendahara
	bendahara, err := repository.GetBendahara()
	if err == nil {
		anggota, _ := repository.GetAnggotaByID(userID)
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		totalStr := fmt.Sprintf("%.2f", total)
		if errWA := sendBendaharaWhatsAppNotification(bendahara.NoTelepon, anggota.NamaAnggota, "Simpanan", totalStr, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi bendahara (simpanan): %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF] bendahara tidak ditemukan untuk notifikasi simpanan: %v", err)
	}
	session.Set("flash_success", fmt.Sprintf("Simpanan Rp%.0f berhasil diajukan via %s dan sudah masuk ke riwayat.", total, metodePembayaran))
	if err := session.Save(); err != nil {
		log.Printf("[SIMPANAN-POST] gagal menyimpan flash session: %v", err)
	}

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

	// Ambil data gabungan pinjaman aktif
	resumeGabungan := getResumePinjamanGabungan(userID, false)
	var jumlahPinjaman, sisaPinjaman, totalTerbayar, bunga float64
	var angsuranKe, sisaAngsuran, jangkaWaktu int
	var persentasePelunasan float64
	if resumeGabungan != nil {
		// ...statusPinjaman sudah tidak digunakan
		// Untuk semua status, tampilkan nilai dari resumeGabungan
		jumlahPinjaman = resumeGabungan.JumlahPinjaman
		sisaPinjaman = resumeGabungan.SisaPokok
		totalTerbayar = resumeGabungan.TotalTerbayar
		bunga = resumeGabungan.Bunga
		angsuranKe = resumeGabungan.AngsuranTerbayar + 1
		sisaAngsuran = resumeGabungan.SisaAngsuran
		jangkaWaktu = resumeGabungan.JangkaWaktu
		persentasePelunasan = resumeGabungan.PersentaseTerbayar
		if persentasePelunasan < 0 {
			persentasePelunasan = 0
		}
	} else {
		jumlahPinjaman = 0
		sisaPinjaman = 0
		totalTerbayar = 0
		bunga = 0
		angsuranKe = 0
		sisaAngsuran = 0
		jangkaWaktu = 0
		persentasePelunasan = 0
	}

	// Perkiraan angsuran per bulan (jika ada jangka waktu)
	perkiraanAngsuranBulan := 0.0
	if jangkaWaktu > 0 {
		pokokPerBulan := jumlahPinjaman / float64(jangkaWaktu)
		jasaPerBulan := bunga / float64(jangkaWaktu)
		perkiraanAngsuranBulan = pokokPerBulan + jasaPerBulan
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

	metodeAngsuranDebug := ""
	if resumeGabungan != nil {
		metodeAngsuranDebug = resumeGabungan.MetodeAngsuran
	}
	c.HTML(http.StatusOK, "anggota_angsuran.html", gin.H{
		"Judul":                  "Angsuran",
		"Anggota":                anggota,
		"JumlahPinjaman":         jumlahPinjaman,
		"SisaPinjaman":           sisaPinjaman,
		"AngsuranKe":             angsuranKe,
		"SisaAngsuran":           sisaAngsuran,
		"TotalTerbayar":          totalTerbayar,
		"Bunga":                  bunga,
		"JangkaWaktu":            jangkaWaktu,
		"NomorRekening":          nomorRekening,
		"CurrentLogo":            latestLogo,
		"PersentasePelunasan":    persentasePelunasan,
		"PerkiraanAngsuranBulan": perkiraanAngsuranBulan,
		"MetodeAngsuran":         metodeAngsuranDebug,
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
		var perkiraanAngsuranBulan float64
		var bunga float64
		var metodeAngsuran string

		totalPinjamanAktif, totalSisaAktif, errTotal := getRingkasanPinjamanAktifByAnggotaID(userID)
		pinjamanInfo, err := getPinjamanPrioritasAngsuran(pinjamans)
		if errTotal == nil {
			jumlahPinjaman = totalPinjamanAktif
			sisaPinjaman = totalSisaAktif
		}
		if err == nil && pinjamanInfo != nil && sisaPinjaman > 0 {
			angsuranKe = pinjamanInfo.AngsuranKe
			bunga = pinjamanInfo.Pinjaman.Bunga
			metodeAngsuran = pinjamanInfo.Pinjaman.MetodeAngsuran
			if pinjamanInfo.Pinjaman.JangkaWaktu > 0 {
				pokokPerBulan := pinjamanInfo.Pinjaman.JumlahPinjaman / float64(pinjamanInfo.Pinjaman.JangkaWaktu)
				jasaPerBulan := (pinjamanInfo.Pinjaman.Bunga / 100 * pinjamanInfo.Pinjaman.JumlahPinjaman) / float64(pinjamanInfo.Pinjaman.JangkaWaktu)
				perkiraanAngsuranBulan = pokokPerBulan + jasaPerBulan
			}
		}

		c.HTML(status, "anggota_angsuran.html", gin.H{
			"Judul":                  "Angsuran",
			"Anggota":                anggota,
			"Error":                  msg,
			"JumlahPinjaman":         jumlahPinjaman,
			"SisaPinjaman":           sisaPinjaman,
			"AngsuranKe":             angsuranKe,
			"Bunga":                  bunga,
			"Pinjamans":              pinjamans,
			"TotalPinjaman":          jumlahPinjaman,
			"PerkiraanAngsuranBulan": perkiraanAngsuranBulan,
			"MetodeAngsuran":         metodeAngsuran,
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

	// Ambil ID pinjaman dari form (jika ada) atau gunakan pinjaman aktif pertama
	idPinjamanStr := c.PostForm("id_pinjaman")
	var idPinjaman int
	var pinjamansAktif []models.Pinjaman
	if idPinjamanStr != "" {
		if parsedID, err := strconv.Atoi(idPinjamanStr); err == nil {
			idPinjaman = parsedID
		} else {
			// fallback: cari pinjaman aktif
			pinjamansAktif, err = repository.GetPinjamanAktifByAnggotaID(userID)
			pinjamanInfo, errInfo := getPinjamanPrioritasAngsuran(pinjamansAktif)
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
		pinjamansAktif, err = repository.GetPinjamanAktifByAnggotaID(userID)
		pinjamanInfo, errInfo := getPinjamanPrioritasAngsuran(pinjamansAktif)
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

	if pinjamansAktif == nil {
		pinjamansAktif, err = repository.GetPinjamanAktifByAnggotaID(userID)
		if err != nil {
			renderWithTotals(http.StatusInternalServerError, "Gagal mengambil data pinjaman aktif.")
			return
		}
	}

	infosAngsuran, err := getPinjamanAngsuranInfos(pinjamansAktif)
	if err != nil {
		renderWithTotals(http.StatusInternalServerError, "Gagal menghitung sisa pinjaman.")
		return
	}

	sisaPinjamanSebelum := 0.0
	for _, info := range infosAngsuran {
		if info.Pinjaman.IDPinjaman == idPinjaman {
			sisaPinjamanSebelum = info.SisaPinjaman
			break
		}
	}
	if sisaPinjamanSebelum <= 0 {
		renderWithTotals(http.StatusBadRequest, "Tidak ada sisa pinjaman aktif yang dapat dibayar.")
		return
	}
	if jumlahAngsuran > sisaPinjamanSebelum {
		renderWithTotals(http.StatusBadRequest, fmt.Sprintf("Jumlah angsuran tidak boleh melebihi sisa pinjaman Rp %.0f.", sisaPinjamanSebelum))
		return
	}

	sisaSetelah := sisaPinjamanSebelum - jumlahAngsuran
	if sisaSetelah < 0 {
		sisaSetelah = 0
	}

	// Handle file upload: only required for transfer method, after amount validation passes.
	var filename string
	if strings.ToLower(metodePembayaran) == "transfer" {
		file, err := c.FormFile("bukti")
		if err != nil {
			renderWithTotals(http.StatusBadRequest, "Bukti pembayaran wajib diupload.")
			return
		}

		filename = time.Now().Format("20060102150405") + "_" + file.Filename
		dst := "./static/uploads/" + filename
		if err := c.SaveUploadedFile(file, dst); err != nil {
			renderWithTotals(http.StatusInternalServerError, "Gagal menyimpan file bukti pembayaran.")
			return
		}
	}

	// Buat angsuran baru
	angsuran := models.Angsuran{
		IDPinjaman:     idPinjaman,
		IDPengelola:    sql.NullInt64{Int64: 1, Valid: true}, // Default pengelola
		TglBayar:       tanggalPembayaran,
		JumlahAngsuran: jumlahAngsuran, // Nominal pembayaran angsuran
		SisaPinjaman:   sisaSetelah,    // Sisa hutang setelah pembayaran
		BuktiAngsuran:  filename,       // Simpan nama file sebagai string
		Status:         "",             // Status akan diset ke pending oleh repository
		NamaAnggota:    anggota.NamaAnggota,
		MetodeAngsuran: metodePembayaran, // Pastikan metode angsuran terisi
	}

	// Simpan ke database
	err = repository.CreateAngsuran(angsuran)
	if err != nil {
		fmt.Printf("CreateAngsuran error: %v\nAngsuran: %+v\n", err, angsuran)
		renderWithTotals(http.StatusInternalServerError, "Gagal menyimpan angsuran. Silakan coba lagi.")
		return
	}

	// Setelah angsuran berhasil disimpan, cek apakah pinjaman sudah lunas
	pinjaman, err := repository.GetPinjamanByID(idPinjaman)
	if err == nil {
		// Hitung total angsuran terbayar
		angsurans, _ := repository.GetAngsuranByPinjamanID(idPinjaman)
		totalAngsuranTerbayar := 0.0
		for _, a := range angsurans {
			if isAngsuranTerbayar(a.Status) {
				totalAngsuranTerbayar += a.SisaPinjaman
			}
		}
		sisaPinjaman := pinjaman.JumlahPinjaman - totalAngsuranTerbayar
		if sisaPinjaman < 0 {
			sisaPinjaman = 0
		}
		if sisaPinjaman == 0 && strings.ToLower(pinjaman.Status) != "lunas" {
			if err := repository.UpdatePinjamanStatus(idPinjaman, "lunas"); err != nil {
				fmt.Printf("Gagal update status pinjaman ke lunas: %v\n", err)
				// Anda bisa menambahkan notifikasi ke user/admin di sini jika perlu
			}
		}
	}

	// Kirim notifikasi WA ke bendahara
	bendahara, err := repository.GetBendahara()
	if err == nil {
		anggota, _ := repository.GetAnggotaByID(userID)
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		jumlahStr := fmt.Sprintf("%.2f", jumlahAngsuran)
		if errWA := sendBendaharaWhatsAppNotification(bendahara.NoTelepon, anggota.NamaAnggota, "Angsuran", jumlahStr, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi bendahara (angsuran): %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF] bendahara tidak ditemukan untuk notifikasi angsuran: %v", err)
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

	// Hitung Total Simpanan (excluding pokok, SAME as Profil)
	simpananByJenis, err := repository.GetDetailSimpananByJenis(userID)
	if err != nil {
		simpananByJenis = make(map[string]float64)
	}
	totalSimpanan := simpananByJenis["wajib"] +
		simpananByJenis["sukarela"] + simpananByJenis["hari_raya"] +
		simpananByJenis["umroh_haji"] + simpananByJenis["qurban"]

	// Ambil detail simpanan per jenis (already loaded above)

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
		log.Printf("[ERROR] AnggotaPengambilanSimpananAjax simpan pengajuan gagal: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pengajuan"})
		return
	}

	// Kirim notifikasi WA ke bendahara jika ada pengajuan penarikan simpanan baru
	bendahara, err := repository.GetBendahara()
	if err == nil {
		anggota, _ := repository.GetAnggotaByID(userID)
		appBaseURL := resolveAppBaseURL(c, config.GetDB())
		nominal := fmt.Sprintf("%.2f", jumlah)
		if errWA := sendBendaharaWhatsAppNotification(bendahara.NoTelepon, anggota.NamaAnggota, "Penarikan Simpanan", nominal, appBaseURL); errWA != nil {
			log.Printf("[WA NOTIF] gagal kirim notifikasi bendahara (penarikan): %v", errWA)
		}
	} else {
		log.Printf("[WA NOTIF] bendahara tidak ditemukan untuk notifikasi penarikan: %v", err)
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

	// Bentuk slice UnifiedTransaction agar cocok dengan template
	type UnifiedTransaction struct {
		ID          int
		Date        time.Time
		Time        string
		Type        string
		Description string
		Amount      string
		Status      string
	}

	var allTransactions []UnifiedTransaction

	riwayatSimpanan, err := repository.GetRiwayatSimpananByAnggotaID(userID, "")
	if err != nil {
		log.Printf("[WARN] gagal mengambil riwayat simpanan anggota %s: %v", userID, err)
		riwayatSimpanan = []models.Detail{}
	}

	riwayatPinjaman, err := repository.GetRiwayatPinjamanByAnggotaID(userID, "")
	if err != nil {
		log.Printf("[WARN] gagal mengambil riwayat pinjaman anggota %s: %v", userID, err)
		riwayatPinjaman = []models.Pinjaman{}
	}

	riwayatAngsuran, err := repository.GetRiwayatAngsuranByAnggotaID(userID, "")
	if err != nil {
		log.Printf("[WARN] gagal mengambil riwayat angsuran anggota %s: %v", userID, err)
		riwayatAngsuran = []models.Angsuran{}
	}

	riwayatPengambilan, err := repository.GetRiwayatPengambilanSimpananByAnggotaID(userID, "")
	if err != nil {
		log.Printf("[WARN] gagal mengambil riwayat pengambilan simpanan anggota %s: %v", userID, err)
		riwayatPengambilan = []models.PengambilanSimpanan{}
	}

	for _, s := range riwayatSimpanan {
		jenis := "Simpanan"
		if strings.TrimSpace(s.Simpanan.JenisSimpanan) != "" {
			jenis = "Simpanan " + s.Simpanan.JenisSimpanan
		}
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", s.JumlahSimpanan)), " ", "")
		timeStr := s.TglTransaksi.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		status := "Dalam Proses"
		switch s.Status {
		case "pending", "confirmed":
			status = "Proses (Menunggu ACC Ketua)"
		case "diterima":
			status = "Diterima"
		case "rejected":
			status = "Ditolak"
		case "lunas":
			status = "Lunas"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          s.IDDetail,
			Date:        s.TglTransaksi,
			Time:        timeStr,
			Type:        jenis,
			Description: jenis,
			Amount:      amount,
			Status:      status,
		})
	}

	for _, p := range riwayatPinjaman {
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", p.JumlahPinjaman)), " ", "")
		timeStr := p.TglPinjaman.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		status := "Dalam Proses"
		switch p.Status {
		case "aktif":
			status = "Aktif"
		case "diterima":
			status = "Diterima"
		case "gagal", "rejected":
			status = "Ditolak"
		case "lunas":
			status = "Lunas"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          p.IDPinjaman,
			Date:        p.TglPinjaman,
			Time:        timeStr,
			Type:        "Pinjaman",
			Description: "Pinjaman",
			Amount:      amount,
			Status:      status,
		})
	}

	for _, a := range riwayatAngsuran {
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", a.JumlahAngsuran)), " ", "")
		timeStr := a.TglBayar.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		status := "Dalam Proses"
		switch a.Status {
		case "pending", "confirmed":
			status = "Proses (Menunggu ACC Ketua)"
		case "diterima", "valid":
			status = "Diterima"
		case "rejected", "invalid":
			status = "Ditolak"
		case "lunas":
			status = "Lunas"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          a.IDAngsuran,
			Date:        a.TglBayar,
			Time:        timeStr,
			Type:        "Angsuran",
			Description: "Angsuran",
			Amount:      amount,
			Status:      status,
		})
	}

	for _, ps := range riwayatPengambilan {
		jenis := "Penarikan Simpanan"
		if strings.TrimSpace(ps.JenisSimpanan) != "" {
			jenis = "Penarikan " + ps.JenisSimpanan
		}
		amount := "Rp " + strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%.0f", ps.Jumlah)), " ", "")
		timeStr := ps.TglPengajuan.Format("15:04:05")
		if timeStr == "00:00:00" {
			timeStr = "-"
		}
		status := "Dalam Proses"
		switch ps.Status {
		case "approved":
			status = "Diterima"
		case "rejected":
			status = "Ditolak"
		}
		allTransactions = append(allTransactions, UnifiedTransaction{
			ID:          ps.IDPengambilan,
			Date:        ps.TglPengajuan,
			Time:        timeStr,
			Type:        jenis,
			Description: jenis,
			Amount:      amount,
			Status:      status,
		})
	}

	sort.Slice(allTransactions, func(i, j int) bool {
		ti := allTransactions[i].Date.UnixNano()
		tj := allTransactions[j].Date.UnixNano()
		if ti == tj {
			return allTransactions[i].ID > allTransactions[j].ID
		}
		return ti > tj
	})

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

	flashSuccess := ""
	if flashVal := session.Get("flash_success"); flashVal != nil {
		if msg, ok := flashVal.(string); ok {
			flashSuccess = msg
		}
		session.Delete("flash_success")
		_ = session.Save()
	}

	c.HTML(http.StatusOK, "anggota_riwayat.html", gin.H{
		"Judul":         "Riwayat Transaksi",
		"Anggota":       anggota,
		"Riwayat":       allTransactions,
		"CurrentLogo":   latestLogo,
		"flash_success": flashSuccess,
	})
}

// AnggotaSimpananJSON mengembalikan data jenis_simpanan dalam format JSON
func AnggotaSimpananJSON(c *gin.Context) {
	if c.GetHeader("Accept") == "application/json" {
		halamanSimpanan, err := repository.GetHalamanBySlug("simpanan")
		if err != nil || len(halamanSimpanan.Konten) == 0 {
			c.JSON(200, gin.H{"konten": gin.H{"jenis_simpanan": []interface{}{}}})
			return
		}
		var konten map[string]interface{}
		if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &konten); err != nil {
			c.JSON(200, gin.H{"konten": gin.H{"jenis_simpanan": []interface{}{}}})
			return
		}
		c.JSON(200, gin.H{"konten": konten})
		return
	}
	// fallback ke handler lama jika bukan JSON
	AnggotaSimpanan(c)
}

// ApiMetodeAngsuran mengembalikan daftar metode angsuran yang tersedia
func ApiMetodeAngsuran(c *gin.Context) {
	// Jika ingin dari database, bisa query di sini. Untuk sekarang, hardcoded dulu.
	data := []map[string]string{
		{"key": "transfer_bank", "nama": "Transfer Bank"},
		{"key": "potong_gaji", "nama": "Potong Gaji"},
		{"key": "tunai", "nama": "Tunai"},
	}
	c.JSON(200, gin.H{"metode_angsuran": data})
}
