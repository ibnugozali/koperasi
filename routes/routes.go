package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"koperasi-simpan-pinjam/controllers"
	"koperasi-simpan-pinjam/middleware"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	// Endpoint Chrome DevTools JSON (tanpa CORS header)
	router.GET("/.well-known/appspecific/com.chrome.devtools.json", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Set trusted proxies to avoid security warning
	router.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	// 1. Setup session middleware
	store := cookie.NewStore([]byte("kuncirahasia-anda-yang-aman"))
	router.Use(sessions.Sessions("koperasisession", store))

	// 2. Middleware untuk disable cache pada static files
	router.Use(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 7 && c.Request.URL.Path[:7] == "/static" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	// 3. Muat file statis dan template HTML
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/images/logo_b930646c-0b96-43a6-a168-7b982a01ba15.png")

	// PERBAIKAN: Gunakan baris ini untuk memuat semua file .html dari folder templates dan semua subfoldernya.
	router.SetFuncMap(template.FuncMap{
		"add": func(nums ...interface{}) float64 {
			sum := 0.0
			for _, num := range nums {
				switch v := num.(type) {
				case int:
					sum += float64(v)
				case int64:
					sum += float64(v)
				case float64:
					sum += v
				case float32:
					sum += float64(v)
				}
			}
			return sum
		},
		"div": func(a, b interface{}) float64 {
			var fa, fb float64
			switch v := a.(type) {
			case int:
				fa = float64(v)
			case int64:
				fa = float64(v)
			case float64:
				fa = v
			case float32:
				fa = float64(v)
			}
			switch v := b.(type) {
			case int:
				fb = float64(v)
			case int64:
				fb = float64(v)
			case float64:
				fb = v
			case float32:
				fb = float64(v)
			}
			if fb == 0 {
				return 0
			}
			return fa / fb
		},
		"mul": func(a, b, c float64) float64 {
			return a * b * c
		},
		"formatRupiah": func(n float64) string {
			s := fmt.Sprintf("%.0f", n)
			run := []rune(s)
			var out []rune
			count := 0
			for i := len(run) - 1; i >= 0; i-- {
				if count > 0 && count%3 == 0 {
					out = append([]rune{'.'}, out...)
				}
				out = append([]rune{run[i]}, out...)
				count++
			}
			return string(out)
		},
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
		"toJson": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(a)
		},
		"now":       func() time.Time { return time.Now() },
		"hasPrefix": func(s, prefix string) bool { return len(s) >= len(prefix) && s[:len(prefix)] == prefix },
		"iterate": func(count int) []int {
			var items []int
			for i := 0; i < count; i++ {
				items = append(items, i)
			}
			return items
		},
		"title": func(s string) string { return strings.Title(s) },
	})

	// Load templates from subdirectories and root templates folder
	templ := template.New("").Funcs(template.FuncMap{
		"add": func(nums ...interface{}) float64 {
			sum := 0.0
			for _, num := range nums {
				switch v := num.(type) {
				case int:
					sum += float64(v)
				case int64:
					sum += float64(v)
				case float64:
					sum += v
				case float32:
					sum += float64(v)
				}
			}
			return sum
		},
		"div": func(a, b interface{}) float64 {
			var fa, fb float64
			switch v := a.(type) {
			case int:
				fa = float64(v)
			case int64:
				fa = float64(v)
			case float64:
				fa = v
			case float32:
				fa = float64(v)
			}
			switch v := b.(type) {
			case int:
				fb = float64(v)
			case int64:
				fb = float64(v)
			case float64:
				fb = v
			case float32:
				fb = float64(v)
			}
			if fb == 0 {
				return 0
			}
			return fa / fb
		},
		"mul": func(a, b, c float64) float64 {
			return a * b * c
		},
		"formatRupiah": func(n float64) string {
			s := fmt.Sprintf("%.0f", n)
			run := []rune(s)
			var out []rune
			count := 0
			for i := len(run) - 1; i >= 0; i-- {
				if count > 0 && count%3 == 0 {
					out = append([]rune{'.'}, out...)
				}
				out = append([]rune{run[i]}, out...)
				count++
			}
			return string(out)
		},
		"json": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
		"toJson": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"now": func() time.Time {
			return time.Now()
		},
		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},
		"iterate": func(count int) []int {
			var items []int
			for i := 1; i <= count; i++ {
				items = append(items, i)
			}
			return items
		},
	})

	// Load templates from each subfolder explicitly
	templ = template.Must(templ.ParseGlob("templates/admin/*.html"))
	templ = template.Must(templ.ParseGlob("templates/anggota/*.html"))
	templ = template.Must(templ.ParseGlob("templates/bendahara/*.html"))
	templ = template.Must(templ.ParseGlob("templates/ketua/*.html"))
	templ = template.Must(templ.ParseGlob("templates/layouts/*.html"))
	templ = template.Must(templ.ParseGlob("templates/utama/*.html"))

	// Also parse templates in root templates folder
	templ = template.Must(templ.ParseGlob("templates/*.html"))

	router.SetHTMLTemplate(templ)

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
	router.GET("/api/jenis-simpanan", controllers.PublicJenisSimpananJSON)
	router.GET("/api/jenis-angsuran", controllers.ApiJenisAngsuran)
	router.GET("/api/metode-angsuran", controllers.ApiMetodeAngsuran)
	// --- Rute Anggota (Dilindungi Middleware) ---
	anggotaRoutes := router.Group("/anggota")
	anggotaRoutes.Use(middleware.AuthRequired())
	{
		anggotaRoutes.GET("/dashboard", controllers.AnggotaDashboard)
		anggotaRoutes.GET("/profil", controllers.AnggotaProfil)
		anggotaRoutes.GET("/pesan", controllers.AnggotaPesan)
		anggotaRoutes.GET("/pesan/notifikasi", controllers.AnggotaPesanNotifikasi)
		anggotaRoutes.GET("/ganti-password", controllers.GantiPassword)
		anggotaRoutes.POST("/ganti-password", controllers.GantiPasswordPost)
		anggotaRoutes.POST("/keluar", controllers.KeluarKoperasi)
		anggotaRoutes.GET("/ajukan-pinjaman", controllers.AjukanPinjaman)
		anggotaRoutes.POST("/ajukan-pinjaman", controllers.AjukanPinjamanPost)
		anggotaRoutes.GET("/simpanan", controllers.AnggotaSimpananJSON)
		anggotaRoutes.POST("/simpanan", controllers.AnggotaSimpananPost)
		anggotaRoutes.GET("/ajukan-pengambilan-simpanan", controllers.AjukanPengambilanSimpanan)
		anggotaRoutes.POST("/ajukan-pengambilan-simpanan", controllers.AjukanPengambilanSimpananPost)
		anggotaRoutes.GET("/angsuran", controllers.AnggotaAngsuran)
		anggotaRoutes.POST("/angsuran", controllers.AnggotaAngsuranPost)
		anggotaRoutes.GET("/sejarah", controllers.AnggotaSejarah)
		anggotaRoutes.GET("/visi-misi", controllers.AnggotaVisiMisi)
		anggotaRoutes.GET("/struktur", controllers.AnggotaStruktur)
		anggotaRoutes.GET("/riwayat", controllers.AnggotaRiwayatPage)
	}

	// --- Rute Admin (Dilindungi Middleware) ---
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AuthRequired(), middleware.AdminOnly())
	adminRoutes.Use(middleware.AdminLogoMiddleware())
	{
		adminRoutes.GET("/dashboard", controllers.AdminDashboard)
		adminRoutes.GET("/import-referensi", controllers.AdminImportReferensiPage)
		adminRoutes.POST("/import-referensi", controllers.AdminImportReferensiPendaftaran)
		adminRoutes.GET("/anggota", controllers.AdminDataAnggota)
		adminRoutes.GET("/anggota/tambah", controllers.AdminTambahAnggotaForm)
		adminRoutes.POST("/anggota/tambah", controllers.AdminTambahAnggotaPost)
		adminRoutes.POST("/anggota/tambah/import", controllers.AdminImportAnggotaExcel)
		adminRoutes.GET("/anggota/:id", controllers.AdminViewAnggota)
		adminRoutes.GET("/halaman/edit/:slug", controllers.ShowEditHalamanForm)
		adminRoutes.POST("/halaman/update/:slug", controllers.UpdateHalaman)
		adminRoutes.POST("/upload", controllers.UploadFile)
		adminRoutes.POST("/upload/struktur", controllers.UploadStruktur)
		adminRoutes.POST("/upload-kop", controllers.AdminUploadKop)
		adminRoutes.POST("/upload-signature", controllers.AdminUploadSignature)
		adminRoutes.GET("/tanda-tangan", controllers.AdminEditTandaTangan)
		adminRoutes.POST("/tanda-tangan/nama", controllers.AdminUpdateSignatureNames)
		adminRoutes.GET("/transaksi", controllers.AdminTransaksi)
		adminRoutes.POST("/transaksi/simpanan", controllers.CatatSimpanan)
		adminRoutes.POST("/transaksi/pinjaman", controllers.CatatPinjaman)
		adminRoutes.GET("/riwayat", controllers.AdminRiwayat)
		adminRoutes.GET("/riwayat-login", controllers.AdminKeamananLogin)
		adminRoutes.POST("/login-history/delete/:id", controllers.DeleteLoginHistory)
		adminRoutes.POST("/login-history/delete-all", controllers.DeleteAllLoginHistory)
		adminRoutes.GET("/laporan", controllers.AdminLaporan)
		adminRoutes.GET("/laporan/download", controllers.AdminDownloadLaporan)
		adminRoutes.GET("/laporan/get-neraca", controllers.AdminGetNeraca)
		adminRoutes.GET("/tentang", controllers.AdminTentang)
		adminRoutes.GET("/pengaturan", controllers.AdminPengaturan)
		adminRoutes.POST("/pengaturan/wa-notif", controllers.AdminSaveWAGatewayConfig)
		adminRoutes.POST("/update-user", controllers.UpdateUser)
		adminRoutes.POST("/update-anggota", controllers.UpdateAnggota)
		adminRoutes.GET("/keamanan/login", controllers.AdminKeamananLogin)
		adminRoutes.GET("/edit-logo", controllers.AdminLogo)
		adminRoutes.POST("/upload-logo", controllers.UploadLogo)
		adminRoutes.GET("/edit-background", controllers.AdminBackground)
		adminRoutes.POST("/upload-background", controllers.UploadBackground)
		// adminRoutes.GET("/keamanan/simpanan", controllers.AdminKeamananSimpanan)
		// adminRoutes.GET("/keamanan/pinjaman", controllers.AdminKeamananPinjaman)
		// adminRoutes.GET("/keamanan/pembayaran", controllers.AdminKeamananPembayaran)
		// adminRoutes.GET("/keamanan/dashboard", controllers.AdminKeamananDashboard)
		// adminRoutes.GET("/keamanan/riwayat", controllers.AdminKeamananRiwayat)
		// adminRoutes.GET("/keamanan/organisasi", controllers.AdminKeamananOrganisasi)
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
		bendaharaRoutes.POST("/reject/:id", controllers.BendaharaRejectMembership)
		bendaharaRoutes.GET("/konfirmasi-transaksi", controllers.BendaharaKonfirmasiTransaksi)
		bendaharaRoutes.GET("/lihat-detail-simpanan/:id", controllers.BendaharaLihatDetailSimpanan)
		bendaharaRoutes.GET("/view-detail-simpanan/:id", controllers.BendaharaViewDetailSimpanan)
		bendaharaRoutes.GET("/lihat-persyaratan-pinjaman/:id", controllers.BendaharaLihatPersyaratanPinjaman)
		bendaharaRoutes.GET("/view-detail-pinjaman/:id", controllers.BendaharaViewDetailPinjaman)
		bendaharaRoutes.GET("/detail-angsuran/:id", controllers.BendaharaDetailAngsuran)
		bendaharaRoutes.GET("/view-detail-angsuran/:id", controllers.BendaharaViewDetailAngsuran)
		bendaharaRoutes.GET("/anggota-angsuran/:id", controllers.BendaharaLihatDetailAngsuran)
		bendaharaRoutes.GET("/detail-ajukan-pengambilan/:id", controllers.BendaharaDetailAjukanPengambilan)
		bendaharaRoutes.POST("/konfirmasi-transaksi/:type/:id", controllers.BendaharaKonfirmasiTransaksiPost)
		bendaharaRoutes.GET("/konfirmasi-transaksi/download-template-potong-gaji", controllers.BendaharaDownloadTemplatePotongGajiExcel)
		bendaharaRoutes.POST("/konfirmasi-transaksi/import-potong-gaji", controllers.BendaharaImportPotongGajiExcel)
		bendaharaRoutes.GET("/halaman/edit/:slug", controllers.BendaharaShowEditHalamanForm)
		bendaharaRoutes.POST("/halaman/update/:slug", controllers.BendaharaUpdateHalaman)
		bendaharaRoutes.POST("/upload", controllers.BendaharaUploadFile)
		bendaharaRoutes.GET("/transaksi", controllers.BendaharaTransaksi)
		bendaharaRoutes.POST("/transaksi/simpanan", controllers.BendaharaCatatSimpanan)
		bendaharaRoutes.POST("/transaksi/pinjaman", controllers.BendaharaCatatPinjaman)
		bendaharaRoutes.POST("/transaksi/angsuran", controllers.BendaharaCatatAngsuran)
		bendaharaRoutes.GET("/riwayat", controllers.BendaharaRiwayat)
		bendaharaRoutes.GET("/transaksi-anggota", controllers.BendaharaTransaksiDataAnggota)
		bendaharaRoutes.GET("/tentang", controllers.BendaharaTentang)
		bendaharaRoutes.GET("/pengaturan", controllers.BendaharaPengaturan)
		bendaharaRoutes.GET("/anggota", controllers.BendaharaListAllAnggota)
		bendaharaRoutes.GET("/anggota/keluar", controllers.BendaharaListAnggotaKeluar)
		bendaharaRoutes.GET("/anggota/keluar/view/:id", controllers.BendaharaViewAnggotaKeluar)
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

		// Route for login history
		bendaharaRoutes.GET("/login-history", controllers.BendaharaLoginHistory)
		bendaharaRoutes.POST("/login-history/delete/:id", controllers.BendaharaDeleteLoginHistory)
		bendaharaRoutes.POST("/login-history/delete-all", controllers.BendaharaDeleteAllLoginHistory)

		// Route for import anggota from XLSX
		bendaharaRoutes.GET("/import-anggota", controllers.BendaharaImportAnggotaPage)
		bendaharaRoutes.POST("/import-anggota/preview", controllers.BendaharaPreviewImportAnggota)
		bendaharaRoutes.POST("/import-anggota", controllers.BendaharaImportAnggota)
		bendaharaRoutes.PUT("/import-anggota/update", controllers.BendaharaUpdateImportData)
		bendaharaRoutes.DELETE("/import-anggota/clear", controllers.BendaharaClearImportHistory)

		// Route for setting simpanan wajib otomatis
		bendaharaRoutes.GET("/setting-simpanan-wajib", controllers.BendaharaSettingSimpananWajib)
		bendaharaRoutes.POST("/setting-simpanan-wajib", controllers.BendaharaSaveSettingSimpananWajib)
		bendaharaRoutes.POST("/proses-simpanan-wajib", controllers.BendaharaProsesSimpananWajib)
		bendaharaRoutes.GET("/cek-pemotongan-otomatis", controllers.BendaharaCekDanProsesPemotonganOtomatis)

		// Routes for approving/rejecting bukti transfer gaji (moved from ketua to bendahara)
		bendaharaRoutes.POST("/konfirmasi-transaksi/bukti-transfer/approve/:id", controllers.BendaharaApproveBuktiTransferGaji)
		bendaharaRoutes.POST("/konfirmasi-transaksi/bukti-transfer/reject/:id", controllers.BendaharaRejectBuktiTransferGaji)

		// Route for pesan
		bendaharaRoutes.GET("/pesan", controllers.BendaharaPesan)
		bendaharaRoutes.POST("/pesan", controllers.BendaharaPesan)
	}

	// --- Rute Ketua (Dilindungi Middleware) ---
	ketuaRoutes := router.Group("/ketua")
	ketuaRoutes.Use(middleware.AuthRequired(), middleware.KetuaOnly())
	{
		ketuaRoutes.GET("/dashboard", controllers.KetuaDashboard)
		ketuaRoutes.GET("/ketua-dashboard", controllers.KetuaDashboard)
		ketuaRoutes.GET("/konfirmasi", controllers.KetuaKonfirmasiAnggota)
		ketuaRoutes.POST("/confirm/:id", controllers.KetuaConfirmMembership)
		ketuaRoutes.POST("/reject/:id", controllers.KetuaRejectMembership)
		ketuaRoutes.POST("/approve-keluar/:id", controllers.KetuaApproveAnggotaKeluar)
		ketuaRoutes.POST("/reject-keluar/:id", controllers.KetuaRejectAnggotaKeluar)
		ketuaRoutes.GET("/konfirmasi-transaksi", controllers.KetuaKonfirmasiTransaksi)
		ketuaRoutes.GET("/anggota", controllers.KetuaDataAnggota)
		ketuaRoutes.GET("/ketua-data-anggota", controllers.KetuaDataAnggota)
		ketuaRoutes.GET("/anggota/:id", controllers.KetuaViewAnggota)
		ketuaRoutes.GET("/anggota/keluar", controllers.KetuaListAnggotaKeluar)
		ketuaRoutes.GET("/anggota/keluar/view/:id", controllers.KetuaViewAnggotaKeluar)
		ketuaRoutes.GET("/ketua-riwayat-login", controllers.KetuaRiwayat)
		ketuaRoutes.GET("/riwayat", controllers.KetuaRiwayat)
		ketuaRoutes.GET("/ketua-laporan", controllers.KetuaLaporan)
		ketuaRoutes.GET("/laporan", controllers.KetuaLaporan)
		ketuaRoutes.GET("/laporan/download", controllers.KetuaDownloadLaporan)
		ketuaRoutes.POST("/laporan/save-neraca", controllers.KetuaSaveNeraca)
		ketuaRoutes.GET("/laporan/get-neraca", controllers.KetuaGetNeraca)
		ketuaRoutes.GET("/ketua-pengaturan", controllers.KetuaPengaturan)
		ketuaRoutes.GET("/lihat-detail-simpanan/:id", controllers.KetuaLihatDetailSimpanan)
		ketuaRoutes.GET("/lihat-persyaratan-pinjaman/:id", controllers.KetuaLihatPersyaratanPinjaman)
		ketuaRoutes.GET("/detail-angsuran/:id", controllers.KetuaDetailAngsuran)
		ketuaRoutes.GET("/detail-ajukan-pengambilan/:id", controllers.KetuaDetailAjukanPengambilan)
		ketuaRoutes.POST("/konfirmasi-transaksi/:type/:id", controllers.KetuaKonfirmasiTransaksiPost)
		ketuaRoutes.GET("/upload-bukti-transfer-gaji", controllers.KetuaUploadBuktiTransferGaji)
		ketuaRoutes.POST("/upload-bukti-transfer-gaji", controllers.KetuaUploadBuktiTransferGajiPost)
		ketuaRoutes.POST("/update-profile", controllers.UpdateKetuaProfile)
	}
	return router
}
