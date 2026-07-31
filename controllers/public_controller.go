package controllers

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/repository"
)

// PublicJenisSimpananJSON mengembalikan data jenis_simpanan tanpa autentikasi
func PublicJenisSimpananJSON(c *gin.Context) {
	// Prioritas 1: Ambil dari tabel simpanan di database agar selalu lengkap dan up-to-date
	db := config.GetDB()
	rows, err := db.Query("SELECT id_simpanan, jenis_simpanan FROM simpanan ORDER BY id_simpanan")
	if err == nil {
		defer rows.Close()
		var jenisList []map[string]string
		for rows.Next() {
			var id int
			var jenis string
			if err := rows.Scan(&id, &jenis); err == nil {
				// Buat nama readable dari key, contoh: "hari_raya" -> "Hari Raya"
				nama := strings.ReplaceAll(jenis, "_", " ")
				nama = strings.Title(nama)
				jenisList = append(jenisList, map[string]string{
					"key":  jenis,
					"nama": nama,
				})
			}
		}
		if rows.Err() == nil && len(jenisList) > 0 {
			c.JSON(200, gin.H{"jenis_simpanan": jenisList})
			return
		}
	}

	// Fallback: Ambil dari halaman simpanan (JSON konten)
	halamanSimpanan, err := repository.GetHalamanBySlug("simpanan")
	if err != nil || len(halamanSimpanan.Konten) == 0 {
		c.JSON(200, gin.H{"jenis_simpanan": []interface{}{}})
		return
	}
	var konten map[string]interface{}
	if err := json.Unmarshal([]byte(halamanSimpanan.Konten), &konten); err != nil {
		c.JSON(200, gin.H{"jenis_simpanan": []interface{}{}})
		return
	}
	if jenis, ok := konten["jenis_simpanan"]; ok {
		c.JSON(200, gin.H{"jenis_simpanan": jenis})
		return
	}
	c.JSON(200, gin.H{"jenis_simpanan": []interface{}{}})
}
