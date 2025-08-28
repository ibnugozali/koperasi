package main

import (
	"koperasi/internal/config"
	"koperasi/internal/handlers"
	"koperasi/internal/middleware"
	"koperasi/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.ConnectDB()
	db.AutoMigrate(&models.Anggota{}, &models.Pengelola{}, &models.Pinjaman{}, &models.Angsuran{}, &models.Simpanan{}, &models.Detail{})

	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*.html")
	r.Static("/static", "./web/static")

	// Routes publik
	r.GET("/", func(c *gin.Context) { c.HTML(200, "login.html", nil) })
	r.POST("/login", handlers.Login)
	r.POST("/register", handlers.RegisterAnggota) // Hanya untuk anggota

	// Routes terproteksi
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Anggota routes
		protected.GET("/anggota", handlers.GetAnggota)
		protected.POST("/anggota", handlers.CreateAnggota) // Admin only, tambah check role di handler

		// Pinjaman
		protected.POST("/pinjaman", handlers.CreatePinjaman)
		protected.GET("/pinjaman", handlers.GetPinjaman)

		// Simpanan
		protected.POST("/simpanan", handlers.CreateSimpanan)
		protected.GET("/simpanan", handlers.GetSimpanan)

		// Laporan (bendahara/pembina)
		protected.GET("/laporan", handlers.GetLaporan)

		// Dashboard
		protected.GET("/dashboard", func(c *gin.Context) { c.HTML(200, "dashboard.html", nil) })
	}

	r.Run(":8080")
}
