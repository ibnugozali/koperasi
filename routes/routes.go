package routes

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/controllers"
	"koperasi-simpan-pinjam/middleware"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Set trusted proxies to avoid security warning
	router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// 1. Setup session middleware
	store := cookie.NewStore([]byte("kuncirahasia-anda-yang-aman"))
	router.Use(sessions.Sessions("koperasisession", store))

	// 2. Muat file statis dan template HTML
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/images/logo.png")

	// PERBAIKAN: Gunakan SATU baris ini untuk memuat semua file .html
	// dari folder templates dan semua subfoldernya.
	router.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
		"now": func() time.Time { return time.Now() },
	})
	router.LoadHTMLGlob("templates/**/*.html")

	// 3. Definisikan rute dan middleware
	// --- Rute Publik (tidak perlu login) ---
	router.GET("/", controllers.ShowLoginPage)
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.Login)
	router.GET("/register", controllers.ShowRegisterPage)
	router.POST("/register", controllers.Register)
	router.GET("/logout", controllers.Logout)
	router.POST("/logout", controllers.Logout)
	router.GET("/tentang/:slug", controllers.ShowTentang)
	router.GET("/pelayanan/:slug", controllers.ShowHalaman)
	router.GET("/riwayat", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/anggota/riwayat")
	})
	router.GET("/riwayat/riwayat", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/anggota/riwayat")
	})
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
		anggotaRoutes.GET("/ajukan-pinjaman", controllers.AjukanPinjaman)
		anggotaRoutes.POST("/ajukan-pinjaman", controllers.AjukanPinjamanPost)
		anggotaRoutes.GET("/simpanan", controllers.AnggotaSimpanan)
		anggotaRoutes.POST("/simpanan", controllers.AnggotaSimpananPost)
		anggotaRoutes.GET("/angsuran", controllers.AnggotaAngsuran)
		anggotaRoutes.POST("/angsuran", controllers.AnggotaAngsuranPost)
		anggotaRoutes.GET("/sejarah", controllers.AnggotaSejarah)
		anggotaRoutes.GET("/visi-misi", controllers.AnggotaVisiMisi)
		anggotaRoutes.GET("/struktur", controllers.AnggotaStruktur)
		anggotaRoutes.GET("/riwayat", controllers.ShowRiwayatPage)
	}

	// --- Rute Admin (Dilindungi Middleware) ---
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AuthRequired(), middleware.AdminOnly())
	{
		adminRoutes.GET("/dashboard", controllers.AdminDashboard)
		adminRoutes.GET("/konfirmasi", controllers.AdminKonfirmasi)
		adminRoutes.POST("/confirm/:id", controllers.ConfirmMembership)
		adminRoutes.POST("/reject/:id", controllers.RejectMembership)
		adminRoutes.GET("/view-registration/:id", controllers.AdminAnggotaRegister)
		adminRoutes.GET("/halaman/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/halaman/update/:slug", controllers.UpdateHalaman)
		adminRoutes.POST("/upload", controllers.UploadFile)
		adminRoutes.POST("/upload/struktur", controllers.UploadStruktur)
		adminRoutes.GET("/transaksi", controllers.AdminTransaksi)
		adminRoutes.POST("/transaksi/simpanan", controllers.CatatSimpanan)
		adminRoutes.POST("/transaksi/pinjaman", controllers.CatatPinjaman)
		adminRoutes.GET("/riwayat", controllers.AdminRiwayat)
		adminRoutes.GET("/laporan", controllers.AdminLaporan)
		adminRoutes.GET("/tentang", controllers.AdminTentang)
		adminRoutes.GET("/pengaturan", controllers.AdminPengaturan)
		adminRoutes.GET("/keamanan/login", controllers.AdminKeamananLogin)
		adminRoutes.GET("/login-history", controllers.AdminLoginHistory)
		adminRoutes.DELETE("/login-history/:id", controllers.DeleteLoginHistory)
		adminRoutes.GET("/edit-logo", controllers.AdminLogo)
		adminRoutes.POST("/upload-logo", controllers.UploadLogo)
		adminRoutes.GET("/keamanan/simpanan", controllers.AdminKeamananSimpanan)
		adminRoutes.GET("/keamanan/pinjaman", controllers.AdminKeamananPinjaman)
		adminRoutes.GET("/keamanan/pembayaran", controllers.AdminKeamananPembayaran)
		adminRoutes.GET("/keamanan/dashboard", controllers.AdminKeamananDashboard)
		adminRoutes.GET("/keamanan/riwayat", controllers.AdminKeamananRiwayat)
		adminRoutes.GET("/keamanan/organisasi", controllers.AdminKeamananOrganisasi)
		adminRoutes.GET("/anggota", controllers.ListAllAnggota)
		adminRoutes.GET("/anggota/:id", controllers.ViewAnggota)
		adminRoutes.GET("/anggota/edit/:id", controllers.EditAnggota)
		adminRoutes.POST("/anggota/update/:id", controllers.UpdateAnggota)
		adminRoutes.POST("/anggota/delete/:id", controllers.DeleteAnggota)
		adminRoutes.POST("/update-anggota-password/:id", controllers.UpdateAnggotaPassword)
		adminRoutes.GET("/pesan", controllers.AdminPesan)
		adminRoutes.POST("/update-profile", controllers.UpdateAdminProfile)
	}

	// --- Rute Bendahara (Dilindungi Middleware) ---
	bendaharaRoutes := router.Group("/bendahara")
	bendaharaRoutes.Use(middleware.AuthRequired(), middleware.BendaharaOnly(), middleware.BendaharaLogoMiddleware())
	{
		bendaharaRoutes.GET("/dashboard", controllers.BendaharaDashboard)
		bendaharaRoutes.GET("/konfirmasi", controllers.BendaharaKonfirmasi)
		bendaharaRoutes.POST("/confirm/:id", controllers.BendaharaConfirmMembership)
		bendaharaRoutes.GET("/konfirmasi-transaksi", controllers.BendaharaKonfirmasiTransaksi)
		bendaharaRoutes.POST("/konfirmasi-transaksi/:type/:id", controllers.BendaharaKonfirmasiTransaksiPost)
		bendaharaRoutes.GET("/halaman", controllers.BendaharaListHalaman)
		bendaharaRoutes.POST("/halaman/update/:slug", controllers.BendaharaUpdateHalaman)
		bendaharaRoutes.POST("/upload", controllers.BendaharaUploadFile)
		bendaharaRoutes.GET("/transaksi", controllers.BendaharaTransaksi)
		bendaharaRoutes.POST("/transaksi/simpanan", controllers.BendaharaCatatSimpanan)
		bendaharaRoutes.POST("/transaksi/pinjaman", controllers.BendaharaCatatPinjaman)
		bendaharaRoutes.GET("/riwayat", controllers.BendaharaRiwayat)
		bendaharaRoutes.GET("/laporan", controllers.BendaharaLaporan)
		bendaharaRoutes.GET("/tentang", controllers.BendaharaTentang)
		bendaharaRoutes.GET("/pengaturan", controllers.BendaharaPengaturan)
		bendaharaRoutes.GET("/anggota", controllers.BendaharaListAllAnggota)
		bendaharaRoutes.GET("/anggota/:id", controllers.BendaharaViewAnggota)
		bendaharaRoutes.GET("/anggota/edit/:id", controllers.BendaharaEditAnggota)
		bendaharaRoutes.POST("/anggota/update/:id", controllers.BendaharaUpdateAnggota)
		bendaharaRoutes.POST("/anggota/delete/:id", controllers.BendaharaDeleteAnggota)
		bendaharaRoutes.POST("/update-profile", controllers.UpdateBendaharaProfile)
		bendaharaRoutes.GET("/edit-rekening-register", controllers.BendaharaEditRekeningRegister)
		bendaharaRoutes.POST("/edit-rekening-register", controllers.BendaharaUpdateRekeningRegister)

		// Routes for editing bunga (interest)
		bendaharaRoutes.GET("/edit-bunga", controllers.BendaharaEditBunga)
		bendaharaRoutes.POST("/edit-bunga", controllers.BendaharaUpdateBunga)
	}

	return router
}
