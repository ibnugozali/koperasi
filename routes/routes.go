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
	router.GET("/logout", controllers.Logout)
	router.GET("/tentang/:slug", controllers.ShowHalaman)
	router.GET("/pelayanan/:slug", controllers.ShowHalaman)
	router.GET("/riwayat", controllers.ShowRiwayatPage)
	router.GET("/hubungi-kami", controllers.ShowHubungiKami)

	// --- Rute Anggota (Dilindungi Middleware) ---
	anggotaRoutes := router.Group("/anggota")
	anggotaRoutes.Use(middleware.AuthRequired())
	{
		anggotaRoutes.GET("/dashboard", controllers.AnggotaDashboard)
		anggotaRoutes.GET("/profil", controllers.AnggotaProfil)
		anggotaRoutes.GET("/pesan", controllers.AnggotaPesan)
		anggotaRoutes.GET("/ganti-password", controllers.GantiPassword)
		anggotaRoutes.POST("/ganti-password", controllers.GantiPasswordPost)
		anggotaRoutes.POST("/keluar", controllers.KeluarKoperasi)
	}

	// --- Rute Admin (Dilindungi Middleware) ---
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AuthRequired(), middleware.AdminOnly())
	{
		adminRoutes.GET("/dashboard", controllers.AdminDashboard)
		adminRoutes.GET("/konfirmasi", controllers.AdminKonfirmasi)
		adminRoutes.POST("/confirm/:id", controllers.ConfirmMembership)
		adminRoutes.GET("/halaman", controllers.ListHalaman)
		adminRoutes.GET("/halaman/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/halaman/update/:slug", controllers.UpdateHalaman)
		//adminRoutes.GET("/pelayanan/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/upload", controllers.UploadFile)
		adminRoutes.GET("/transaksi", controllers.AdminTransaksi)
		adminRoutes.GET("/laporan", controllers.AdminLaporan)
		adminRoutes.GET("/tentang", controllers.AdminTentang)
		adminRoutes.GET("/pengaturan", controllers.AdminPengaturan)
		adminRoutes.GET("/anggota", controllers.ListAllAnggota)
		adminRoutes.GET("/anggota/:id", controllers.ViewAnggota)
		adminRoutes.GET("/anggota/edit/:id", controllers.EditAnggota)
		adminRoutes.POST("/anggota/update/:id", controllers.UpdateAnggota)
		adminRoutes.POST("/anggota/delete/:id", controllers.DeleteAnggota)
		adminRoutes.POST("/update-profile", controllers.UpdateAdminProfile)
	}

	// --- Rute Bendahara (Dilindungi Middleware) ---
	bendaharaRoutes := router.Group("/bendahara")
	bendaharaRoutes.Use(middleware.AuthRequired(), middleware.BendaharaOnly())
	{
		bendaharaRoutes.GET("/dashboard", controllers.BendaharaDashboard)
		bendaharaRoutes.GET("/konfirmasi", controllers.BendaharaKonfirmasi)
		bendaharaRoutes.POST("/confirm/:id", controllers.BendaharaConfirmMembership)
		bendaharaRoutes.GET("/halaman", controllers.BendaharaListHalaman)
		bendaharaRoutes.POST("/halaman/update/:slug", controllers.BendaharaUpdateHalaman)
		bendaharaRoutes.POST("/upload", controllers.BendaharaUploadFile)
		bendaharaRoutes.GET("/transaksi", controllers.BendaharaTransaksi)
		bendaharaRoutes.GET("/laporan", controllers.BendaharaLaporan)
		bendaharaRoutes.GET("/tentang", controllers.BendaharaTentang)
		bendaharaRoutes.GET("/pengaturan", controllers.BendaharaPengaturan)
		bendaharaRoutes.GET("/anggota", controllers.BendaharaListAllAnggota)
		bendaharaRoutes.GET("/anggota/:id", controllers.BendaharaViewAnggota)
		bendaharaRoutes.GET("/anggota/edit/:id", controllers.BendaharaEditAnggota)
		bendaharaRoutes.POST("/anggota/update/:id", controllers.BendaharaUpdateAnggota)
		bendaharaRoutes.POST("/anggota/delete/:id", controllers.BendaharaDeleteAnggota)
		bendaharaRoutes.POST("/update-profile", controllers.UpdateBendaharaProfile)
	}

	// --- Rute Ketua (Dilindungi Middleware) ---
	ketuaRoutes := router.Group("/ketua")
	ketuaRoutes.Use(middleware.AuthRequired(), middleware.KetuaOnly())
	{
		ketuaRoutes.GET("/dashboard", controllers.KetuaDashboard)
		ketuaRoutes.GET("/konfirmasi", controllers.KetuaKonfirmasi)
		ketuaRoutes.POST("/confirm/:id", controllers.KetuaConfirmMembership)
		ketuaRoutes.GET("/halaman", controllers.KetuaListHalaman)
		ketuaRoutes.GET("/halaman/edit/:slug", controllers.KetuaShowEditHalamanForm)
		ketuaRoutes.POST("/halaman/update/:slug", controllers.KetuaUpdateHalaman)
		ketuaRoutes.POST("/upload", controllers.KetuaUploadFile)
		ketuaRoutes.GET("/transaksi", controllers.KetuaTransaksi)
		ketuaRoutes.GET("/laporan", controllers.KetuaLaporan)
		ketuaRoutes.GET("/tentang", controllers.KetuaTentang)
		ketuaRoutes.GET("/pengaturan", controllers.KetuaPengaturan)
		ketuaRoutes.GET("/anggota", controllers.KetuaListAllAnggota)
		ketuaRoutes.GET("/anggota/:id", controllers.KetuaViewAnggota)
		ketuaRoutes.GET("/anggota/edit/:id", controllers.KetuaEditAnggota)
		ketuaRoutes.POST("/anggota/update/:id", controllers.KetuaUpdateAnggota)
		ketuaRoutes.POST("/anggota/delete/:id", controllers.KetuaDeleteAnggota)
		ketuaRoutes.POST("/update-profile", controllers.UpdateKetuaProfile)
	}

	return router
}
