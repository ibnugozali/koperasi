package main

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Inisialisasi koneksi database saat aplikasi dimulai
	config.InitDB()

	// Menjalankan router
	router := routes.SetupRouter()
	router.GET("/.well-known/appspecific/com.chrome.devtools.json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	router.Run(":8081")
}
