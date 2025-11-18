package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRouter creates and configures the router for integration testing
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthCheck)
		v1.GET("/database/status", databaseStatus)
	}

	return router
}

// setupTestDB initializes database for integration testing
func setupTestDB(t *testing.T) {
	// Use dedicated test database (separate from development database)
	testDatabaseURL := os.Getenv("TEST_DATABASE_URL")
	if testDatabaseURL == "" {
		// Default to test database on same PostgreSQL server
		testDatabaseURL = "postgresql://postgres:postgres@postgres:5432/kendalls_nails_test?sslmode=disable"
	}

	// Save original DATABASE_URL and temporarily override with test database
	originalURL := os.Getenv("DATABASE_URL")
	os.Setenv("DATABASE_URL", testDatabaseURL)

	// Reset DB connection to force reconnection to test database
	config.DB = nil

	// Connect to test database
	err := config.ConnectDatabase()
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	db := config.GetDB()
	err = db.AutoMigrate(&models.User{})
	require.NoError(t, err, "Failed to migrate test database")

	// Restore original DATABASE_URL (though tests will continue using test DB)
	if originalURL != "" {
		os.Setenv("DATABASE_URL", originalURL)
	} else {
		os.Unsetenv("DATABASE_URL")
	}
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T) {
	db := config.GetDB()
	if db != nil {
		// Clean up test data
		db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	}
}

// TestHealthEndpointIntegration tests the /api/v1/health endpoint with full routing
func TestHealthEndpointIntegration(t *testing.T) {
	router := setupRouter()

	// Create a test request
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200 OK")

	// Parse and verify response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Custom Nails API is running", response["message"])
}

// TestHealthEndpointMethod tests that only GET method is allowed
func TestHealthEndpointMethod(t *testing.T) {
	router := setupRouter()

	// Test POST method (should fail)
	req, _ := http.NewRequest("POST", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "POST should not be allowed")

	// Test PUT method (should fail)
	req, _ = http.NewRequest("PUT", "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "PUT should not be allowed")

	// Test DELETE method (should fail)
	req, _ = http.NewRequest("DELETE", "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "DELETE should not be allowed")
}

// TestAPIV1Prefix tests that the endpoint requires /api/v1 prefix
func TestAPIV1Prefix(t *testing.T) {
	router := setupRouter()

	// Test without /api/v1 prefix (should fail)
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code, "Endpoint should require /api/v1 prefix")

	// Test with correct prefix (should succeed)
	req, _ = http.NewRequest("GET", "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Endpoint should work with /api/v1 prefix")
}

// TestHealthEndpointHeaders tests that proper headers are set
func TestHealthEndpointHeaders(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify Content-Type header
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

// TestDatabaseConnection tests database connectivity
func TestDatabaseConnection(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	db := config.GetDB()
	assert.NotNil(t, db, "Database connection should not be nil")

	// Test that we can ping the database
	sqlDB, err := db.DB()
	require.NoError(t, err, "Should get underlying SQL database")

	err = sqlDB.Ping()
	assert.NoError(t, err, "Should be able to ping database")

	// Verify we're connected to the TEST database (not development database)
	var currentDB string
	err = db.Raw("SELECT current_database()").Scan(&currentDB).Error
	require.NoError(t, err, "Should be able to query current database")
	assert.Equal(t, "kendalls_nails_test", currentDB, "Should be connected to test database, not development database")
}

// TestDatabaseMigration tests that User table is created correctly
func TestDatabaseMigration(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	db := config.GetDB()

	// Check that users table exists
	var tableExists bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users')").Scan(&tableExists).Error
	require.NoError(t, err, "Should be able to query for table existence")
	assert.True(t, tableExists, "Users table should exist after migration")

	// Check that table has expected columns
	type ColumnInfo struct {
		ColumnName string
		DataType   string
	}
	var columns []ColumnInfo
	err = db.Raw("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'users' ORDER BY ordinal_position").Scan(&columns).Error
	require.NoError(t, err, "Should be able to query for columns")

	expectedColumns := map[string]string{
		"id":         "bigint",
		"email":      "text",
		"role":       "text",
		"created_at": "timestamp with time zone",
		"updated_at": "timestamp with time zone",
		"deleted_at": "timestamp with time zone",
	}

	assert.Equal(t, len(expectedColumns), len(columns), "Should have correct number of columns")

	for _, col := range columns {
		expectedType, exists := expectedColumns[col.ColumnName]
		assert.True(t, exists, "Column %s should be expected", col.ColumnName)
		assert.Equal(t, expectedType, col.DataType, "Column %s should have correct data type", col.ColumnName)
	}
}

// TestDatabaseStatusEndpoint tests the /api/v1/database/status endpoint
func TestDatabaseStatusEndpoint(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/database/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status 200 OK")

	// Parse and verify response
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be valid JSON")
	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Database connected", response["message"])

	// Check that tables list includes "users"
	tables, ok := response["tables"].([]any)
	assert.True(t, ok, "Tables should be an array")
	assert.Contains(t, tables, "users", "Tables should contain 'users'")
}

// TestCreateUserInDatabase tests creating a user record in the database
func TestCreateUserInDatabase(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	db := config.GetDB()

	// Create a test user
	user := models.User{
		Email: "integration-test@example.com",
		Role:  "customer",
	}

	result := db.Create(&user)
	assert.NoError(t, result.Error, "Should create user without error")
	assert.NotZero(t, user.ID, "User ID should be set after creation")
	assert.NotZero(t, user.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, user.UpdatedAt, "UpdatedAt should be set")

	// Verify user was created
	var foundUser models.User
	err := db.Where("email = ?", "integration-test@example.com").First(&foundUser).Error
	assert.NoError(t, err, "Should find created user")
	assert.Equal(t, "integration-test@example.com", foundUser.Email)
	assert.Equal(t, "customer", foundUser.Role)
}

// TestUserEmailUniqueness tests that email field is unique
func TestUserEmailUniqueness(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	db := config.GetDB()

	// Create first user
	user1 := models.User{
		Email: "unique-test@example.com",
		Role:  "customer",
	}
	result := db.Create(&user1)
	assert.NoError(t, result.Error, "First user should be created successfully")

	// Try to create second user with same email
	user2 := models.User{
		Email: "unique-test@example.com",
		Role:  "technician",
	}
	result = db.Create(&user2)
	assert.Error(t, result.Error, "Should not allow duplicate email")
}
