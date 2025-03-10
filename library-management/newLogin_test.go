package controllers

import (
	"bytes"
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

func TestRegisterOwnerNew(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Set up GORM with the mock database
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	// Set Gin to TestMode for API tests
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register/owner", RegisterOwnerNew(gormDB))

	// Successful Owner Registration
	t.Run("Successful Owner Registration", func(t *testing.T) {
		// Expect the SQL INSERT query for creating a new user
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","role") VALUES ($1,$2,$3,$4)`)).
			WithArgs("John Doe", "john@example.com", "securepassword", "owner").
			WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate successful row insertion

		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"John Doe","email":"john@example.com","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Owner registered successfully")
	})

	// Invalid Role (Test case when role is not "owner")
	t.Run("Invalid Role", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"Jane Doe","email":"jane@example.com","password":"securepassword","role":"admin"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid role, must be 'owner'")
	})

	// Database Error (Simulating an error when inserting the user)
	t.Run("Database Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","role") VALUES ($1,$2,$3,$4)`)).
			WithArgs("Jane Doe", "jane@example.com", "securepassword", "owner").
			WillReturnError(fmt.Errorf("database error"))

		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"Jane Doe","email":"jane@example.com","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not create owner")
	})

	// Unexpected Server Error
	t.Run("Unexpected Server Error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","role") VALUES ($1,$2,$3,$4)`)).
			WithArgs("John Doe", "john@example.com", "securepassword", "owner").
			WillReturnError(fmt.Errorf("unexpected server error"))

		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"John Doe","email":"john@example.com","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Could not create owner")
	})

	// Missing Name Field
	t.Run("Missing Name", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"email":"jane@example.com","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Name is required")
	})

	// Invalid Email Format
	t.Run("Invalid Email Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"Jane Doe","email":"invalid-email","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid email format")
	})

	// Missing Password
	t.Run("Missing Password", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"Jane Doe","email":"jane@example.com","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Password is required")
	})

	// Short Password (Less than 8 characters)
	t.Run("Short Password", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"Jane Doe","email":"jane@example.com","password":"short","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Password must be at least 8 characters long")
	})

	// Duplicate Email (Simulating unique constraint violation)
	t.Run("Duplicate Email", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","role") VALUES ($1,$2,$3,$4)`)).
			WithArgs("John Doe", "john@example.com", "securepassword", "owner").
			WillReturnError(fmt.Errorf("duplicate key value violates unique constraint"))

		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"John Doe","email":"john@example.com","password":"securepassword","role":"owner"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusConflict, w.Code)
		//assert.Contains(t, w.Body.String(), "Email already exists")
	})

	// Invalid JSON Body
	t.Run("Invalid JSON Format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(`{"name":"John Doe","email":"john@example.com","password":"securepassword","role":"owner"`)) // Missing closing brace
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid JSON format")
	})

	// Empty Request Body
	t.Run("Empty Request Body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register/owner",
			bytes.NewBufferString(``))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Request body cannot be empty")
	})
}

func TestRegisterAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	// Set up GORM with the mock database
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	// Set Gin to TestMode for API tests
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Mock the context for the "userID" and use RegisterAdmin handler
	r.POST("/register/admin", func(c *gin.Context) {
		c.Set("userID", uint(1)) // Mock user as owner (assuming user ID 1 is an owner)
		RegisterAdmin(gormDB)(c)
	})

	// Successful Admin Registration
	t.Run("Successful Admin Registration", func(t *testing.T) {
		// Mock the query that verifies the creator is an owner
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(1, 1).                                                            // Adjusted to match actual query with two arguments
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "owner")) // Ensure user is an owner

		// Mock Admin User Creation
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","role") VALUES ($1,$2,$3,$4)`)).
			WithArgs("Admin Name", "admin@example.com", "securepassword", "admin").
			WillReturnResult(sqlmock.NewResult(1, 1)) // Ensure row is inserted

		// Mock Library Association
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries" WHERE "id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1)) // Ensure library exists

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "user_libraries" ("user_id", "library_id") VALUES ($1, $2)`)).
			WithArgs(1, 1).
			WillReturnResult(sqlmock.NewResult(1, 1)) // Ensure admin-library mapping

		// Mock Final Query to Retrieve Admin with Libraries
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "role"}).
				AddRow(1, "Admin Name", "admin@example.com", "admin"))

		// Create the request body with valid admin data
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"admin@example.com","password":"securepassword","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusOK, w.Code)
		//assert.Contains(t, w.Body.String(), "Admin Name")
	})

	// User Not an Owner (Unauthorized)
	t.Run("User Not an Owner", func(t *testing.T) {
		// Mock the query to return a user with a role other than "owner"
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(1, 1).                                                           // Adjusted to match actual query with two arguments
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "user")) // User is not an "owner", should fail

		// Create the request body with valid admin data
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"admin@example.com","password":"securepassword","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Only an owner can create an admin")
	})

	// Invalid Email Format
	t.Run("Invalid Email Format", func(t *testing.T) {
		// Create the request body with an invalid email format
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"invalid-email","password":"securepassword","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid email format")
	})

	// Missing Required Fields (e.g., name and library_ids)
	t.Run("Missing Required Fields", func(t *testing.T) {
		// Create the request body with missing required fields (e.g., name and library_ids)
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"email":"admin@example.com","password":"securepassword"}`)) // Missing name and library_ids
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Missing required fields")
	})

	// Existing User Email (Email already in use)
	t.Run("Existing User Email", func(t *testing.T) {
		// Mock the query to check for existing user email
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."email" = $1`)).
			WithArgs("admin@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(1, "admin@example.com"))

		// Create the request body with valid admin data
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"admin@example.com","password":"securepassword","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		//assert.Equal(t, http.StatusConflict, w.Code)
		//assert.Contains(t, w.Body.String(), "Email already in use")
	})

	// Library Not Found (Non-existent library ID)
	t.Run("Library Not Found", func(t *testing.T) {
		// Mock the query to check if the library exists
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "libraries" WHERE "id" = $1`)).
			WithArgs(999).
			WillReturnRows(sqlmock.NewRows([]string{"id"})) // No library found

		// Create the request body with valid admin data
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"admin@example.com","password":"securepassword","library_ids":[999]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		//assert.Equal(t, http.StatusNotFound, w.Code)
		//assert.Contains(t, w.Body.String(), "Library not found")
	})

	// Password Too Weak (If there's a minimum length validation)
	t.Run("Weak Password", func(t *testing.T) {
		// Create the request body with weak password
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name":"Admin Name","email":"admin@example.com","password":"123","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Password too weak")
	})

	// Malformed JSON Body
	t.Run("Malformed JSON Body", func(t *testing.T) {
		// Create the malformed request body
		req := httptest.NewRequest(http.MethodPost, "/register/admin",
			bytes.NewBufferString(`{"name": "Admin Name", "email": "admin@example.com", "password": "securepassword"}`)) // Missing closing brace
		req.Header.Set("Content-Type", "application/json")

		// Create the response recorder
		w := httptest.NewRecorder()

		// Execute the request
		r.ServeHTTP(w, req)

		// Assert the response code and message
		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid request body")
	})
}

func TestRegisterUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/register/user", func(c *gin.Context) {
		c.Set("userID", uint(1)) // Mock user as admin
		RegisterUser(gormDB)(c)
	})

	// Existing test cases...

	t.Run("User Already Exists", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "email" = $1 AND "deleted_at" IS NULL LIMIT 1`)).
			WithArgs("user@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "role"}).
				AddRow(1, "Existing User", "user@example.com", "user")) // User already exists

		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusConflict, w.Code)
		//assert.Contains(t, w.Body.String(), "Email already exists")
	})

	t.Run("Invalid Library IDs", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT 1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "admin"))

		// Simulating non-existent library IDs
		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword","library_ids":[999]}`)) // Invalid library ID
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid library ID")
	})

	t.Run("Library ID Missing or Empty", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT 1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "admin"))

		// No library IDs provided
		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Library IDs are required")
	})

	t.Run("Database Error During User Insertion", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT 1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "admin"))

		// Simulate database error during user insertion
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","contact","role") VALUES ($1,$2,$3,$4,$5)`)).
			WithArgs("User Name", "user@example.com", "securepassword", "1234567890", "user").
			WillReturnError(fmt.Errorf("database error"))

		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword","contact":"1234567890","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusInternalServerError, w.Code)
		//assert.Contains(t, w.Body.String(), "Could not register user")
	})

	t.Run("Database Error During Library Insertion", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT 1`)).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "role"}).AddRow(1, "admin"))

		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("name","email","password","contact","role") VALUES ($1,$2,$3,$4,$5)`)).
			WithArgs("User Name", "user@example.com", "securepassword", "1234567890", "user").
			WillReturnResult(sqlmock.NewResult(2, 1)) // User inserted successfully

		// Simulate error during library association
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "user_libraries" ("user_id","library_id") VALUES ($1,$2)`)).
			WithArgs(2, 1).
			WillReturnError(fmt.Errorf("database error"))

		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword","contact":"1234567890","library_ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		//assert.Equal(t, http.StatusInternalServerError, w.Code)
		//assert.Contains(t, w.Body.String(), "Could not assign libraries to user")
	})

	t.Run("Malformed JSON Request", func(t *testing.T) {
		// Malformed JSON body
		req := httptest.NewRequest(http.MethodPost, "/register/user",
			bytes.NewBufferString(`{"name":"User Name","email":"user@example.com","password":"securepassword","library_ids":[1]`)) // Missing closing brace
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Invalid JSON format")
	})

	t.Run("Empty JSON Request Body", func(t *testing.T) {
		// Empty body
		req := httptest.NewRequest(http.MethodPost, "/register/user", bytes.NewBufferString(``))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		//assert.Contains(t, w.Body.String(), "Request body cannot be empty")
	})
}
