package controllers

import (
	"fmt"
	"library-management/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterOwnerNew allows an existing owner to create a new owner
func RegisterOwnerNew(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.User

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if input.Role != "owner" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role, must be 'owner'"})
			return
		}

		if err := db.Create(&input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create owner"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "New owner registered successfully", "owner": input})
	}
}

// RegisterAdmin allows an owner to create an admin
func RegisterAdmin(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name       string `json:"name" binding:"required"`
			Email      string `json:"email" binding:"required,email"`
			Password   string `json:"password" binding:"required"`
			Contact    string `json:"contact"`
			LibraryIDs []uint `json:"library_ids" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		creatorID := c.GetUint("userID")
		var creator models.User
		if err := db.First(&creator, creatorID).Error; err != nil || creator.Role != "owner" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only an owner can create an admin"})
			return
		}

		admin := models.User{
			Name:     input.Name,
			Email:    input.Email,
			Password: input.Password,
			Contact:  input.Contact,
			Role:     "admin",
		}

		if err := db.Create(&admin).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create admin"})
			return
		}

		for _, libID := range input.LibraryIDs {
			var library models.Library
			if err := db.First(&library, libID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Library ID %d not found", libID)})
				return
			}

			adminLibrary := models.UserLibrary{
				UserID:    admin.ID,
				LibraryID: libID,
			}
			if err := db.Create(&adminLibrary).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate admin with library"})
				return
			}
		}

		// Preload libraries to include them in the response
		var adminWithLibraries models.User
		if err := db.Preload("Library").First(&adminWithLibraries, admin.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load libraries"})
			return
		}

		// Return the response with the libraries included
		c.JSON(http.StatusCreated, gin.H{
			"message": "Admin registered successfully",
			"admin": gin.H{
				"ID":      adminWithLibraries.ID,
				"Name":    adminWithLibraries.Name,
				"Email":   adminWithLibraries.Email,
				"Role":    adminWithLibraries.Role,
				"Contact": adminWithLibraries.Contact,
				"Library": adminWithLibraries.Library, // This will include associated libraries
			},
		})
	}
}

// RegisterUser allows an admin to create a user
func RegisterUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			Name       string `json:"name" binding:"required"`
			Email      string `json:"email" binding:"required,email"`
			Password   string `json:"password" binding:"required,min=8"`
			Contact    string `json:"contact"`
			LibraryIDs []uint `json:"library_ids" binding:"required"`
		}

		// Check for validation errors in the request body
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
			return
		}

		// Admin authentication check
		adminID := c.GetUint("userID")
		var admin models.User
		if err := db.First(&admin, adminID).Error; err != nil || admin.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create users"})
			return
		}

		// Fetch admin's accessible libraries
		var adminLibraries []uint
		if err := db.Table("user_libraries").Where("user_id = ?", adminID).Pluck("library_id", &adminLibraries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not verify admin libraries"})
			return
		}

		// Check if the libraries provided in the input are accessible by the admin
		for _, libID := range input.LibraryIDs {
			found := false
			for _, adminLibID := range adminLibraries {
				if libID == adminLibID {
					found = true
					break
				}
			}
			if !found {
				c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("You can only add users to libraries you manage (Library ID: %d)", libID)})
				return
			}
		}

		// Check for duplicate email
		var existingUser models.User
		if err := db.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
			return
		}

		// Create new user
		user := models.User{
			Name:     input.Name,
			Email:    input.Email,
			Password: input.Password,
			Contact:  input.Contact,
			Role:     "user", // Default role as "user"
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not register user"})
			return
		}

		// Associate user with libraries
		for _, libID := range input.LibraryIDs {
			userLibrary := models.UserLibrary{
				UserID:    user.ID,
				LibraryID: libID,
			}
			if err := db.Create(&userLibrary).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate user with library"})
				return
			}
		}

		// Preload libraries for the response
		var userWithLibraries models.User
		if err := db.Preload("Library").First(&userWithLibraries, user.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load libraries"})
			return
		}

		// Return the response with the libraries included
		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user": gin.H{
				"ID":      userWithLibraries.ID,
				"Name":    userWithLibraries.Name,
				"Email":   userWithLibraries.Email,
				"Role":    userWithLibraries.Role,
				"Contact": userWithLibraries.Contact,
				"Library": userWithLibraries.Library, // This will include associated libraries
			},
		})
	}
}
