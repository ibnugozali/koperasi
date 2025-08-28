package handlers

import (
	"koperasi/internal/config"
	"koperasi/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

var db = config.ConnectDB()

func GetAnggota(c *gin.Context) {
	role := c.GetString("role")
	if role != "administrator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	var anggota []models.Anggota
	db.Find(&anggota)
	c.JSON(http.StatusOK, anggota)
}

func CreateAnggota(c *gin.Context) {
	role := c.GetString("role")
	if role != "administrator" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	var anggota models.Anggota
	if err := c.BindJSON(&anggota); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.Create(&anggota)
	c.JSON(http.StatusOK, anggota)
}

// Tambahkan Update, Delete serupa dengan check role