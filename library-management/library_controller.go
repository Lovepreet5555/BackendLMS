package controllers

import (
	"library-management/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateLibrary handles creating a new library
func CreateLibrary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.Library

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Create(&input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create library"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Library created successfully", "library": input})
	}
}

// ListLibraries fetches all libraries
func ListLibraries(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var libraries []models.Library

		if err := db.Find(&libraries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch libraries"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"libraries": libraries})
	}
}
