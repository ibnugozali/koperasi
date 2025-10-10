package middleware

import (
	"net/http"

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
