package routes

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/controllers"
	"koperasi-simpan-pinjam/middleware"

)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// 1. Setup session middleware
	store := cookie.NewStore([]byte("kuncirahasia-anda-yang-aman"))
	router.Use(sessions.Sessions("koperasisession", store))

	// 2. Muat file statis dan template HTML
	router.Static("/static", "./static")

	// PERBAIKAN: Gunakan SATU baris ini untuk memuat semua file .html
	// dari folder templates dan semua subfoldernya.
	router.LoadHTMLGlob("templates/**/*.html")

	// 3. Definisikan rute dan middleware
	// --- Rute Publik (tidak perlu login) ---
	router.GET("/", controllers.ShowLoginPage)
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.Login)
	router.GET("/register", controllers.ShowRegisterPage)
	router.POST("/register", controllers.Register)
	router.POST("/logout", controllers.Logout)
	router.GET("/tentang/:slug", controllers.ShowHalaman)
	router.GET("/pelayanan/:slug", controllers.ShowHalaman)
	router.GET("/riwayat/:slug", controllers.ShowRiwayatPage)
	router.GET("/hubungi-kami", controllers.ShowHubungiKami)
	

	// --- Rute Anggota (Dilindungi Middleware) ---
	anggotaRoutes := router.Group("/anggota")
	anggotaRoutes.Use(middleware.AuthRequired())
	{
		anggotaRoutes.GET("/dashboard", controllers.AnggotaDashboard)
		anggotaRoutes.GET("/profil", controllers.AnggotaProfil)
		anggotaRoutes.GET("/pesan", controllers.AnggotaPesan)
		anggotaRoutes.GET("/ganti-password", controllers.GantiPassword)
	}

	// --- Rute Admin (Dilindungi Middleware) ---
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AuthRequired(), middleware.AdminOnly())
	{
		adminRoutes.GET("/dashboard", controllers.AdminDashboard)
		adminRoutes.POST("/confirm/:id", controllers.ConfirmMembership)
		adminRoutes.GET("/halaman", controllers.ListHalaman)
		adminRoutes.GET("/halaman/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/halaman/update/:slug", controllers.UpdateHalaman)
		//adminRoutes.GET("/pelayanan/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/upload", controllers.UploadFile)
	}

	return router
}
