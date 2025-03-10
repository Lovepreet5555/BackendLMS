package controllers

import (
	"library-management/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AddBook adds a book or increments copies - Only Admin
func AddBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.Book

		// Extract user ID and role from JWT
		userID, exists := c.Get("userID")
		userRole, roleExists := c.Get("userRole")
		if !exists || !roleExists || userRole != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		// Bind JSON input
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON input"})
			return
		}

		// Ensure user is an admin of the library
		var admin models.UserLibrary
		if err := db.Where("user_id = ? AND library_id = ?", userID, input.LibraryID).First(&admin).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only add books to libraries you manage"})
			return
		}

		// Ensure book has valid copies
		if input.TotalCopies <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Number of copies must be greater than zero"})
			return
		}

		// Check if book already exists in the library
		var existingBook models.Book
		if err := db.Where("isbn = ? AND library_id = ?", input.ISBN, input.LibraryID).First(&existingBook).Error; err == nil {
			// Book already exists, update the total copies
			existingBook.TotalCopies += input.TotalCopies
			existingBook.AvailableCopies += input.TotalCopies

			if err := db.Save(&existingBook).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book copies"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Book copies updated successfully", "book": existingBook})
			return
		}

		// New book → Insert into DB
		input.AvailableCopies = input.TotalCopies
		if err := db.Create(&input).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not add book"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Book added successfully", "book": input})
	}
}

// UpdateBook updates book details - Only Admin
func UpdateBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		isbn := c.Param("isbn")
		var input models.Book

		userID, exists := c.Get("userID")
		userRole, roleExists := c.Get("userRole")
		if !exists || !roleExists || userRole != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if input.LibraryID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Library ID is required"})
			return
		}

		var admin models.UserLibrary
		if err := db.Where("user_id = ? AND library_id = ?", userID, input.LibraryID).First(&admin).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned as an admin for this library"})
			return
		}

		var book models.Book
		if err := db.Where("isbn = ? AND library_id = ?", isbn, input.LibraryID).First(&book).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found in the specified library"})
			return
		}

		issuedCopies := book.TotalCopies - book.AvailableCopies
		if input.TotalCopies < issuedCopies {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Total copies cannot be less than issued copies"})
			return
		}

		book.Title = input.Title
		book.Authors = input.Authors
		book.Publisher = input.Publisher
		book.Version = input.Version
		book.TotalCopies = input.TotalCopies
		book.AvailableCopies = input.TotalCopies - issuedCopies

		if err := db.Save(&book).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Book updated successfully", "book": book})
	}
}

// RemoveBook removes a book - Only Admin
func RemoveBook(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		isbn := c.Param("isbn")
		var input struct {
			LibraryID uint `json:"libraryid"`
		}

		userID, exists := c.Get("userID")
		userRole, roleExists := c.Get("userRole")
		if !exists || !roleExists || userRole != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var admin models.UserLibrary
		if err := db.Where("user_id = ? AND library_id = ?", userID, input.LibraryID).First(&admin).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not assigned as an admin for this library"})
			return
		}

		var book models.Book
		if err := db.Where("isbn = ? AND library_id = ?", isbn, input.LibraryID).First(&book).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found in the specified library"})
			return
		}

		if book.TotalCopies > 1 {
			book.TotalCopies--
			book.AvailableCopies--
			if err := db.Save(&book).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrement book copies"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Book copies decremented", "book": book})
		} else {
			if err := db.Delete(&book).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove book"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Book removed from inventory"})
		}
	}
}
