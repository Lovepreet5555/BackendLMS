// 🔍 Manage Issue Requests
package controllers

import (
	"library-management/models"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// ListIssueRequests retrieves all issue requests for admin's libraries
func ListIssueRequests(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		var adminLibraryIDs []uint
		if err := db.Table("user_libraries").Where("user_id = ?", adminID).Pluck("library_id", &adminLibraryIDs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch admin libraries"})
			return
		}

		if len(adminLibraryIDs) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin is not associated with any library"})
			return
		}

		var requests []models.RequestEvent
		if err := db.Joins("JOIN books ON request_events.book_id = books.isbn").
			Where("books.library_id IN (?)", adminLibraryIDs).
			Find(&requests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch issue requests"})
			return
		}

		formattedRequests := make([]gin.H, len(requests))
		for i, request := range requests {
			formattedRequests[i] = gin.H{
				"id":            request.ID,
				"book_id":       request.BookID,
				"user_id":       request.ReaderID,
				"request_type":  request.RequestType,
				"request_date":  formatUnixTime(&request.RequestDate),
				"approval_date": formatUnixTime(request.ApprovalDate),
				"approver_id":   request.ApproverID,
			}
		}
		c.JSON(http.StatusOK, gin.H{"requests": formattedRequests})
	}
}

// ApproveIssue allows an admin to approve a book issue request
func ApproveIssue(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Param("id")
		var request models.RequestEvent

		if err := db.First(&request, requestID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue request not found"})
			return
		}

		adminID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		var book models.Book
		if err := db.Where("isbn = ?", request.BookID).First(&book).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
			return
		}

		var count int64
		if err := db.Table("user_libraries").Where("user_id = ? AND library_id = ?", adminID, book.LibraryID).Count(&count).Error; err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only approve requests for books in your assigned library"})
			return
		}

		if request.ApprovalDate != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request is already approved"})
			return
		}

		if book.AvailableCopies == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No available copies to issue"})
			return
		}

		now := time.Now().Unix()
		request.ApprovalDate = &now
		request.ApproverID = new(uint)
		*request.ApproverID = adminID.(uint)

		if err := db.Save(&request).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not approve request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Issue request approved"})
	}
}

// DisapproveIssue allows an admin to reject an issue request
func DisapproveIssue(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Param("id")
		var request models.RequestEvent

		if err := db.First(&request, requestID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Issue request not found"})
			return
		}

		if err := db.Delete(&request).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not disapprove request"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Issue request disapproved successfully"})
	}
}

func IssueBookToUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		isbn := c.Param("isbn")

		adminID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
			return
		}

		var input struct {
			UserID    uint `json:"user_id"`
			LibraryID uint `json:"library_id"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		var book models.Book
		if err := db.Where("isbn = ? AND library_id = ?", isbn, input.LibraryID).First(&book).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found in this library"})
			return
		}

		if book.AvailableCopies == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No available copies to issue"})
			return
		}

		book.AvailableCopies--
		db.Save(&book)

		issueDate := time.Now()
		expectedReturnDate := issueDate.AddDate(0, 0, 14)

		issueRecord := models.IssueRegistry{
			ISBN:               isbn,
			ReaderID:           input.UserID,
			IssueApproverID:    adminID.(uint),
			IssueStatus:        "issued",
			IssueDate:          issueDate.Unix(),
			ExpectedReturnDate: expectedReturnDate.Unix(),
			ReturnDate:         0,
			ReturnApproverID:   0,
		}

		if err := db.Create(&issueRecord).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not issue book"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Book issued successfully"})
	}
}

func formatUnixTime(timestamp *int64) string {
	if timestamp == nil || *timestamp == 0 {
		return "N/A"
	}
	return time.Unix(*timestamp, 0).Format("2006-01-02 15:04:05")
}