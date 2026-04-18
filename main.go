package main

import (
	"koperasi-simpan-pinjam/config"
	"koperasi-simpan-pinjam/routes"
)

func main() {
	// Inisialisasi koneksi database saat aplikasi dimulai
	config.InitDB()

	// Menjalankan router
	router := routes.SetupRouter()
	router.Run(":8081")
}
