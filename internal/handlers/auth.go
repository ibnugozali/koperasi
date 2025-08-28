package handlers

import (
	"koperasi/internal/config"
	"koperasi/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var db = config.ConnectDB()
var jwtKey = []byte("secret_key")

type Claims struct {
	ID   uint   `json:"id"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"` // Gunakan nama atau email
		Password string `json:"password"`
	}
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek anggota atau pengelola
	var user interface{}
	var role string
	var id uint
	var hashedPass string

	anggota := models.Anggota{}
	if err := db.Where("nama_anggota = ?", input.Username).First(&anggota).Error; err == nil {
		user = anggota
		role = "anggota"
		id = anggota.IDAnggota
		hashedPass = anggota.Password
	} else {
		pengelola := models.Pengelola{}
		if err := db.Where("nama_pengelola = ? OR email = ?", input.Username, input.Username).First(&pengelola).Error; err == nil {
			user = pengelola
			role = pengelola.Jabatan // administrator, bendahara, pembina
			id = pengelola.IDPengelola
			hashedPass = pengelola.Password
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate JWT
	claims := &Claims{ID: id, Role: role}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(jwtKey)

	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func RegisterAnggota(c *gin.Context) {
	var anggota models.Anggota
	if err := c.BindJSON(&anggota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hashed, _ := bcrypt.HashPassword(anggota.Password, bcrypt.DefaultCost)
	anggota.Password = string(hashed)
	if err := db.Create(&anggota).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, anggota)
}