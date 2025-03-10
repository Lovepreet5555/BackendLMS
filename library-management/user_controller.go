// 🔍 Search Books by Title, Author, Publisher
package controllers

import (
	"library-management/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchBooks allows users to search for books in their registered libraries
func SearchBooks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		var userLibraries []uint
		if err := db.Table("user_libraries").Where("user_id = ?", userID).Pluck("library_id", &userLibraries).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user libraries"})
			return
		}

		if len(userLibraries) == 0 {
			c.JSON(http.StatusOK, gin.H{"books": []gin.H{}})
			return
		}

		title := c.Query("title")
		author := c.Query("author")
		publisher := c.Query("publisher")

		var books []models.Book
		query := db.Where("library_id IN (?)", userLibraries)

		if title != "" {
			query = query.Where("title ILIKE ?", "%"+title+"%")
		}
		if author != "" {
			query = query.Where("authors ILIKE ?", "%"+author+"%")
		}
		if publisher != "" {
			query = query.Where("publisher ILIKE ?", "%"+publisher+"%")
		}

		if err := query.Select("isbn, title, authors, publisher, available_copies, library_id").Find(&books).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error searching books"})
			return
		}

		response := make([]gin.H, 0, len(books))
		for _, book := range books {
			authors := book.Authors
			if authors == "" {
				authors = "Unknown"
			}

			bookData := gin.H{
				"isbn":             book.ISBN,
				"title":            book.Title,
				"author":           authors,
				"publisher":        book.Publisher,
				"available_copies": book.AvailableCopies,
				"library_id":       book.LibraryID,
			}

			if book.AvailableCopies == 0 {
				var nextAvailableDate time.Time
				var issue models.IssueRegistry

				if err := db.Where("isbn = ? AND return_date IS NULL", book.ISBN).
					Order("expected_return_date ASC").
					First(&issue).Error; err == nil {
					nextAvailableDate = time.Unix(issue.ExpectedReturnDate, 0)
					bookData["next_available_date"] = nextAvailableDate.Format("2006-01-02 15:04:05")
				} else {
					bookData["next_available_date"] = "Unknown"
				}
			}

			response = append(response, bookData)
		}

		c.JSON(http.StatusOK, gin.H{"books": response})
	}
}

// RequestIssue allows users to request books from admins
func RequestIssue(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input struct {
			BookID    string `json:"isbn" binding:"required"`
			LibraryID uint   `json:"libraryid" binding:"required"`
		}

		// Bind the input JSON
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Retrieve userID from context
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		// Check if the book exists in the specified library
		var book models.Book
		if err := db.Where("isbn = ? AND library_id = ?", input.BookID, input.LibraryID).First(&book).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found in the specified library"})
			return
		}

		// Check if the book is available for issue
		if book.AvailableCopies == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Book not available for issue"})
			return
		}

		// Check if the user is registered in the library
		var userLibrary models.UserLibrary
		if err := db.Where("user_id = ? AND library_id = ?", userID, input.LibraryID).First(&userLibrary).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only request books from libraries you are registered in"})
			return
		}

		// Check if the user already has a pending request for this book in this library
		var existingRequest models.RequestEvent
		if err := db.Where("reader_id = ? AND book_id = ? AND library_id = ? AND approval_date IS NULL", userID, input.BookID, input.LibraryID).First(&existingRequest).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "You already have a pending request for this book in this library"})
			return
		}

		// Create the issue request
		requestDate := time.Now()
		request := models.RequestEvent{
			BookID:       input.BookID,
			LibraryID:    input.LibraryID,
			ReaderID:     userID.(uint),
			RequestDate:  requestDate.Unix(),
			ApprovalDate: nil,
			ApproverID:   nil,
			RequestType:  "issue",
		}

		// Save the request to the database
		if err := db.Create(&request).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create issue request"})
			return
		}

		// Return success response
		c.JSON(http.StatusCreated, gin.H{"message": "Issue request submitted", "request": request})
	}
}
