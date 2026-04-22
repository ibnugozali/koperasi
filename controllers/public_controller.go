package controllers

import (
	"encoding/json"
	"koperasi-simpan-pinjam/repository"

	"github.com/gin-gonic/gin"
)

// PublicJenisSimpananJSON mengembalikan data jenis_simpanan tanpa autentikasi
func PublicJenisSimpananJSON(c *gin.Context) {
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
