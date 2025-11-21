package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired adalah middleware untuk memeriksa apakah pengguna sudah login.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user_id") // Cek apakah ada user_id di session

		if user == nil {
			// Jika tidak ada, batalkan request dan arahkan ke halaman login
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		// Jika ada, lanjutkan ke handler berikutnya
		c.Next()
	}
}

// AdminOnly adalah middleware untuk memeriksa apakah pengguna adalah admin.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role")

		if role != "admin" {
			// Jika bukan admin, tolak akses
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			return
		}
		c.Next()
	}
}

// BendaharaOnly adalah middleware untuk memeriksa apakah pengguna adalah bendahara.
func BendaharaOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role")

		if role != "bendahara" {
			// Jika bukan bendahara, tolak akses
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			return
		}
		c.Next()
	}
}

// KetuaOnly adalah middleware untuk memeriksa apakah pengguna adalah ketua.
func KetuaOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		role := session.Get("role")

		if role != "ketua" {
			// Jika bukan ketua, tolak akses
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
			return
		}
		c.Next()
	}
}

// BendaharaLogoMiddleware adalah middleware untuk mengatur path logo terbaru.
func BendaharaLogoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logoPath := "/static/images/placeholder.png"
		// Find the latest logo file based on modification time
		files, err := os.ReadDir("static/images")
		if err == nil {
			var latestLogo string
			var latestTime int64
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "logo_") && (strings.HasSuffix(file.Name(), ".png") || strings.HasSuffix(file.Name(), ".jpg")) {
					info, err := file.Info()
					if err == nil {
						modTime := info.ModTime().Unix()
						if modTime > latestTime {
							latestTime = modTime
							latestLogo = "/static/images/" + file.Name()
						}
					}
				}
			}
			if latestLogo != "" {
				logoPath = latestLogo
			}
		}
		c.Set("LogoPath", logoPath)
		c.Next()
	}
}
