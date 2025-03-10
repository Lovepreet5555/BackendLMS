package controllers

import (
	"bytes"
	"errors"
	"library-management/models"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestLogin(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Open a GORM database using the mock connection
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	assert.NoError(t, err)

	// Initialize Gin router
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/login", Login(gormDB))

	// Mock user data
	mockUser := models.User{
		ID:       1,
		Email:    "testuser@example.com",
		Password: "password123",
		Role:     "admin",
	}

	// Define test cases
	tests := []struct {
		name           string
		input          string
		mockQuery      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:  "Valid credentials",
			input: `{"email": "testuser@example.com", "password": "password123"}`,
			mockQuery: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND deleted_at IS NULL) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(mockUser.Email, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).
						AddRow(mockUser.ID, mockUser.Email, mockUser.Password, mockUser.Role))
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:  "Invalid credentials - wrong password",
			input: `{"email": "testuser@example.com", "password": "wrongpassword"}`,
			mockQuery: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND deleted_at IS NULL) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(mockUser.Email, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).
						AddRow(mockUser.ID, mockUser.Email, "password123", mockUser.Role))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name:  "User not found",
			input: `{"email": "nonexistent@example.com", "password": "password123"}`,
			mockQuery: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND deleted_at IS NULL) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
					WithArgs("nonexistent@example.com", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid credentials",
		},
		{
			name:           "Missing fields",
			input:          `{}`,
			mockQuery:      func() {}, // No DB call expected
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Email' failed",
		},
		{
			name:  "Database error",
			input: `{"email": "testuser@example.com", "password": "password123"}`,
			mockQuery: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND deleted_at IS NULL) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(mockUser.Email, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Database error",
		},
		// New Edge Cases
		{
			name:           "Missing email field",
			input:          `{"password": "password123"}`,
			mockQuery:      func() {}, // No DB call expected
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Email' failed",
		},
		{
			name:           "Missing password field",
			input:          `{"email": "testuser@example.com"}`,
			mockQuery:      func() {}, // No DB call expected
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Password' failed",
		},
		{
			name:           "Empty password",
			input:          `{"email": "testuser@example.com", "password": ""}`,
			mockQuery:      func() {}, // No DB call expected
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Password' failed",
		},

		{
			name:  "SQL Query Timeout/Delay",
			input: `{"email": "testuser@example.com", "password": "password123"}`,
			mockQuery: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND deleted_at IS NULL) AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
					WithArgs(mockUser.Email, 1).
					WillDelayFor(2 * time.Second). // Simulating a delay
					WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password", "role"}).
						AddRow(mockUser.ID, mockUser.Email, mockUser.Password, mockUser.Role))
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	// Execute test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mock expectations
			tt.mockQuery()

			// Create a request with JSON input
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tt.input))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Assert response status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Assert error message if expected
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}

	// Ensure all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
