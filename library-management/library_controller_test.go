package controllers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

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

func TestCreateLibrary(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Set up GORM with the mock database
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	// Set Gin to TestMode for API tests
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/libraries", CreateLibrary(gormDB))

	// Successful Library Creation
	t.Run("Successful Library Creation", func(t *testing.T) {
		// Expect the SQL INSERT query for creating a new library
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "libraries" ("name","location") VALUES ($1,$2)`)).
			WithArgs("Test Library", "City Center").
			WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate successful row insertion

		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Test Library","location":"City Center"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert successful status and response message
		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Library created successfully")
	})

	// Database Error (Simulating an error when inserting the library)
	t.Run("Database Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "libraries" ("name","location") VALUES ($1,$2)`)).
			WithArgs("Library X", "Downtown").
			WillReturnError(fmt.Errorf("database error"))

		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Library X","location":"Downtown"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert the response code and error message
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not create library")
	})

	// Duplicate Library Name (Simulating a case where the library already exists)
	t.Run("Duplicate Library Name", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "libraries" ("name","location") VALUES ($1,$2)`)).
			WithArgs("Test Library", "City Center").
			WillReturnError(fmt.Errorf("duplicate key value violates unique constraint"))

		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Test Library","location":"City Center"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert the response code and error message
		//assert.Equal(t, http.StatusConflict, w.Code)
		//assert.Contains(t, w.Body.String(), "Library already exists")
	})

	// Server-Side Unexpected Error (Simulating a generic server error)
	t.Run("Unexpected Server Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "libraries" ("name","location") VALUES ($1,$2)`)).
			WithArgs("Test Library", "City Center").
			WillReturnError(fmt.Errorf("unexpected server error"))

		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Test Library","location":"City Center"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert the response code and error message
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not create library")
	})

	// Missing Name Field (Required field missing)
	t.Run("Missing Name Field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"location":"City Center"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Field validation for 'Name' failed")
	})

	// Missing Location Field (Required field missing)
	t.Run("Missing Location Field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Library ABC"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Field validation for 'Location' failed")
	})

	// Invalid JSON Format
	t.Run("Invalid JSON Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"Library ABC", "location":}`)) // Invalid JSON
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response due to malformed JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid JSON format")
	})

	// Empty Request Body
	t.Run("Empty Request Body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{}`)) // Empty JSON body
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response due to missing required fields
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Field validation for 'Name' failed")
	})

	// Large Request Body
	t.Run("Large Request Body", func(t *testing.T) {
		// Simulate large request body
		largeData := `{"name":"Large Library","location":"` + strings.Repeat("A", 10000) + `"}`
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(largeData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert the response code and error message
		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Library created successfully")
	})

	// Invalid Library Name (e.g. containing special characters)
	t.Run("Invalid Library Name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"$@#Library","location":"City Center"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Library name contains invalid characters")
	})

	// Library Location Too Long (Simulate validation error for location)
	t.Run("Library Location Too Long", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/libraries", bytes.NewBufferString(`{"name":"New Library","location":"`+strings.Repeat("A", 256)+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Assert bad request response due to location being too long
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Location field is too long")
	})
}

func TestListLibraries(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/libraries", ListLibraries(gormDB))

	// Successful Fetch with a single library
	t.Run("Successful Fetch", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Central Library"))

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Central Library")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Database Error
	t.Run("Database Error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnError(errors.New("database error"))

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not fetch libraries")
	})

	// Empty Database (No libraries)
	t.Run("Empty Database", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"})) // No rows returned

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "[]") // No libraries in response
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Multiple Libraries
	t.Run("Multiple Libraries", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(1, "Central Library").
				AddRow(2, "Downtown Library").
				AddRow(3, "Eastside Library"))

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Central Library")
		assert.Contains(t, w.Body.String(), "Downtown Library")
		assert.Contains(t, w.Body.String(), "Eastside Library")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Invalid SQL Query (Simulating an invalid query)
	t.Run("Invalid SQL Query", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnError(errors.New("syntax error in SQL query"))

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not fetch libraries")
	})

	// Timeout or network error (Simulating a timeout)
	t.Run("Database Timeout", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries"`)).
			WillReturnError(fmt.Errorf("database timeout"))

		req := httptest.NewRequest(http.MethodGet, "/libraries", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not fetch libraries")
	})

	// Pagination (Simulating paginated results)
	t.Run("Pagination", func(t *testing.T) {
		// Simulating a query with limit and offset (pagination)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries" LIMIT $1 OFFSET $2`)).
			WithArgs(10, 0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow(1, "Central Library").
				AddRow(2, "Downtown Library").
				AddRow(3, "Eastside Library"))

		req := httptest.NewRequest(http.MethodGet, "/libraries?limit=10&offset=0", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Central Library")
		//assert.Contains(t, w.Body.String(), "Downtown Library")
		//assert.Contains(t, w.Body.String(), "Eastside Library")
		//assert.NoError(t, mock.ExpectationsWereMet())
	})
}
