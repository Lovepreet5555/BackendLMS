package controllers

import (
	"bytes"
	"context"
	"fmt"

	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAddBook(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/books", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("userRole", "admin")
		AddBook(gormDB)(c)
	})

	// Test Case 1: Successful book addition
	t.Run("Successful book addition", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE isbn = $1 AND library_id = $2 AND "books"."deleted_at" IS NULL`)).
			WithArgs("123456789", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "books" (isbn, title, total_copies, available_copies, library_id) VALUES ($1, $2, $3, $4, $5)`)).
			WithArgs("123456789", "Test Book", 3, 3, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book","library_id":1,"total_copies":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusCreated, w.Code)
		//assert.Contains(t, w.Body.String(), "Book added successfully")
		//assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Test Case 2: Unauthorized user
	t.Run("Unauthorized request", func(t *testing.T) {
		// Simulate an unauthorized request (e.g., missing or invalid credentials)
		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book","library_id":1,"total_copies":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusUnauthorized, w.Code)
		//assert.Contains(t, w.Body.String(), "Unauthorized request")
	})

	// Test Case 3: Bad Request (invalid JSON)
	t.Run("Bad Request (invalid JSON)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{ invalid json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "error")
	})

	// Test Case 4: Library not found (invalid library ID)
	t.Run("Library not found", func(t *testing.T) {
		// Simulate a situation where the library doesn't exist in the database
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 9999, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book","library_id":9999,"total_copies":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusNotFound, w.Code)
		//assert.Contains(t, w.Body.String(), "Library not found")
	})

	// Test Case 5: Duplicate Book (Already exists)
	t.Run("Duplicate book", func(t *testing.T) {
		// Simulate that the book already exists in the library
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE isbn = $1 AND library_id = $2 AND "books"."deleted_at" IS NULL`)).
			WithArgs("123456789", 1).
			WillReturnRows(sqlmock.NewRows([]string{"isbn", "title", "total_copies", "available_copies", "library_id"}).
				AddRow("123456789", "Test Book", 3, 3, 1))

		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book","library_id":1,"total_copies":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusConflict, w.Code)
		//assert.Contains(t, w.Body.String(), "Book already exists")
	})

	// Test Case 6: Internal server error (database failure)
	t.Run("Internal server error", func(t *testing.T) {
		// Simulate a database failure when trying to insert the book
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE isbn = $1 AND library_id = $2 AND "books"."deleted_at" IS NULL`)).
			WithArgs("123456789", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		// Simulate database failure during book insertion
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "books" (isbn, title, total_copies, available_copies, library_id) VALUES ($1, $2, $3, $4, $5)`)).
			WithArgs("123456789", "Test Book", 3, 3, 1).
			WillReturnError(fmt.Errorf("database error"))

		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book","library_id":1,"total_copies":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusInternalServerError, w.Code)
		//assert.Contains(t, w.Body.String(), "Failed to add book")
	})

	// Test Case 7: Missing required fields
	t.Run("Missing required fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"isbn":"123456789","title":"Test Book"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//	assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Missing required fields")
	})

}

func TestUpdateBook(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.PUT("/books/:isbn", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("userRole", "admin")
		UpdateBook(gormDB)(c)
	})

	// Ensure correct request body
	payload := `{"library_id":1, "title":"Updated Title","authors":"Updated Author","publisher":"Updated Publisher","version":"2nd Edition","total_copies":5}`
    

	
	t.Run("Unauthorized User", func(t *testing.T) {
		// Simulate non-admin user
		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		ctx := req.Context()
		ctx = context.WithValue(ctx, "userRole", "user") // Non-admin role
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusUnauthorized, w.Code)
		//assert.Contains(t, w.Body.String(), "Unauthorized request")
	})

	t.Run("Library Not Found", func(t *testing.T) {
		// Simulate invalid library association for the user
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, library_id FROM "user_libraries" WHERE user_id = $1 AND library_id = $2`)).
			WithArgs(1, 1).
			WillReturnError(fmt.Errorf("library not found"))

		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusForbidden, w.Code)
		// assert.Contains(t, w.Body.String(), "Library not found")
	})

	t.Run("Book Not Found", func(t *testing.T) {
		// Simulate book not found in the library
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, library_id FROM "user_libraries" WHERE user_id = $1 AND library_id = $2`)).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT isbn, title, authors, publisher, version, total_copies, available_copies FROM "books" WHERE isbn = $1 AND library_id = $2`)).
			WithArgs("123456789", 1).
			WillReturnError(fmt.Errorf("book not found"))

		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusNotFound, w.Code)
		// assert.Contains(t, w.Body.String(), "Book not found")
	})

	t.Run("Invalid Input (JSON Binding Error)", func(t *testing.T) {
		// Simulate invalid JSON input (missing required fields)
		invalidPayload := `{"library_id":1, "title":"Updated Title","authors":"Updated Author","publisher":"Updated Publisher"}` // Missing "total_copies" and "version"

		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(invalidPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Bad request")
	})

	t.Run("Failed Update (Database Error)", func(t *testing.T) {
		// Simulate database error during the update
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, library_id FROM "user_libraries" WHERE user_id = $1 AND library_id = $2`)).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT isbn, title, authors, publisher, version, total_copies, available_copies FROM "books" WHERE isbn = $1 AND library_id = $2`)).
			WithArgs("123456789", 1).
			WillReturnRows(sqlmock.NewRows([]string{"isbn", "title", "authors", "publisher", "version", "total_copies", "available_copies"}).
				AddRow("123456789", "Test Book", "Test Author", "Test Publisher", "1st Edition", 5, 5))

		// Simulate a database failure on the update
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "books" SET title = $1, authors = $2, publisher = $3, version = $4, total_copies = $5, available_copies = $6 WHERE isbn = $7 AND library_id = $8`)).
			WithArgs("Updated Title", "Updated Author", "Updated Publisher", "2nd Edition", 5, 5, "123456789", 1).
			WillReturnError(fmt.Errorf("failed to update"))

		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusInternalServerError, w.Code)
		//assert.Contains(t, w.Body.String(), "Failed to update book")
	})

	t.Run("Valid Book Update", func(t *testing.T) {
		// Simulate valid update
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, library_id FROM "user_libraries" WHERE user_id = $1 AND library_id = $2`)).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT isbn, title, authors, publisher, version, total_copies, available_copies FROM "books" WHERE isbn = $1 AND library_id = $2`)).
			WithArgs("123456789", 1).
			WillReturnRows(sqlmock.NewRows([]string{"isbn", "title", "authors", "publisher", "version", "total_copies", "available_copies"}).
				AddRow("123456789", "Test Book", "Test Author", "Test Publisher", "1st Edition", 5, 5))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "books" SET title = $1, authors = $2, publisher = $3, version = $4, total_copies = $5, available_copies = $6 WHERE isbn = $7 AND library_id = $8`)).
			WithArgs("Updated Title", "Updated Author", "Updated Publisher", "2nd Edition", 5, 5, "123456789", 1).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row updated

		req := httptest.NewRequest(http.MethodPut, "/books/123456789", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Book updated successfully")
	})
}

func TestRemoveBook(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.Default()

	r.DELETE("/books/:isbn", func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("userRole", "admin")
		RemoveBook(gormDB)(c)
	})

	t.Run("Unauthorized User", func(t *testing.T) {
		// Test for unauthorized user (non-admin)
		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{"libraryid":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Simulate user role as non-admin
		ctx := req.Context()
		ctx = context.WithValue(ctx, "userRole", "user") // Non-admin user
		req = req.WithContext(ctx)

		r.ServeHTTP(w, req)

		//	assert.Equal(t, http.StatusUnauthorized, w.Code)
		//	assert.Contains(t, w.Body.String(), "Unauthorized request")
	})

	t.Run("Book Not Found", func(t *testing.T) {
		// Simulate book not found scenario
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).
				AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE (isbn = $1 AND library_id = $2) 
            AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $3`)).
			WithArgs("123456789", 1, 1).
			WillReturnError(fmt.Errorf("record not found"))

		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{"libraryid":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Book not found")
	})

	t.Run("Library ID Missing", func(t *testing.T) {
		// Simulate missing library ID
		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Library ID is required")
	})

	t.Run("Database Error (Failed Deletion)", func(t *testing.T) {
		// Simulate database error during deletion
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).
				AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE (isbn = $1 AND library_id = $2) 
            AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $3`)).
			WithArgs("123456789", 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"isbn", "total_copies", "available_copies"}).
				AddRow("123456789", 1, 1))

		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "books" WHERE "books"."isbn" = $1 AND "books"."library_id" = $2`)).
			WithArgs("123456789", 1).
			WillReturnError(fmt.Errorf("failed to delete book"))

		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{"libraryid":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to remove book")
	})

	t.Run("Valid Book Removal", func(t *testing.T) {
		// Simulate valid book removal
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "library_id"}).
				AddRow(1, 1, 1))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "books" WHERE (isbn = $1 AND library_id = $2) 
            AND "books"."deleted_at" IS NULL ORDER BY "books"."id" LIMIT $3`)).
			WithArgs("123456789", 1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"isbn", "total_copies", "available_copies"}).
				AddRow("123456789", 1, 1))

		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "books" WHERE "books"."isbn" = $1 AND "books"."library_id" = $2`)).
			WithArgs("123456789", 1).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{"libraryid":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Book removed from inventory")
	})

	t.Run("Library Admin Not Found", func(t *testing.T) {
		// Simulate user not assigned as an admin in the library
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "user_libraries" WHERE user_id = $1 AND library_id = $2 ORDER BY "user_libraries"."user_id" LIMIT $3`)).
			WithArgs(1, 1, 1).
			WillReturnError(fmt.Errorf("record not found"))

		req := httptest.NewRequest(http.MethodDelete, "/books/123456789", bytes.NewBufferString(`{"libraryid":1}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "You are not assigned as an admin for this library")
	})
}
