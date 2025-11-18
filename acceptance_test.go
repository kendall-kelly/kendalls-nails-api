package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStartup is an acceptance test that verifies the server can start
// This test uses the actual setupRouter function to ensure the full application works
func TestServerStartup(t *testing.T) {
	router := setupRouter()
	assert.NotNil(t, router, "Router should be initialized")
}

// TestAPIHealthEndpointAcceptance is an end-to-end acceptance test
// It simulates a real HTTP request to verify the API works as expected
func TestAPIHealthEndpointAcceptance(t *testing.T) {
	router := setupRouter()

	// Start a test server
	server := &http.Server{
		Addr:    ":0", // Random available port
		Handler: router,
	}

	// Create a request as a real client would
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	assert.NoError(t, err, "Should be able to create request")

	// Use the router's ServeHTTP to simulate the request
	recorder := &testResponseWriter{header: make(http.Header)}
	router.ServeHTTP(recorder, req)

	// Verify the response matches acceptance criteria
	assert.Equal(t, http.StatusOK, recorder.statusCode, "Health endpoint should return 200 OK")

	// Parse response
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(recorder.body, &response)
	assert.NoError(t, err, "Response should be valid JSON")

	// Verify acceptance criteria from IMPLEMENTATION_PLAN.md
	assert.True(t, response.Success, "Success field should be true")
	assert.Equal(t, "Custom Nails API is running", response.Message, "Message should match specification")

	server.Close()
}

// TestHealthEndpointAvailability tests that the health endpoint is available immediately
func TestHealthEndpointAvailability(t *testing.T) {
	router := setupRouter()

	// Make multiple requests to ensure consistency
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		recorder := &testResponseWriter{header: make(http.Header)}
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.statusCode,
			fmt.Sprintf("Request %d should succeed", i+1))

		// Verify consistent response
		var response map[string]interface{}
		json.Unmarshal(recorder.body, &response)
		assert.Equal(t, true, response["success"],
			fmt.Sprintf("Request %d should have success=true", i+1))
	}
}

// TestHealthEndpointResponseTime tests that the endpoint responds quickly
func TestHealthEndpointResponseTime(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	recorder := &testResponseWriter{header: make(http.Header)}

	start := time.Now()
	router.ServeHTTP(recorder, req)
	duration := time.Since(start)

	// Health check should be very fast (under 100ms)
	assert.Less(t, duration, 100*time.Millisecond,
		"Health endpoint should respond in less than 100ms")
}

// testResponseWriter is a helper for acceptance testing
type testResponseWriter struct {
	header     http.Header
	body       []byte
	statusCode int
}

func (w *testResponseWriter) Header() http.Header {
	return w.header
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// TestDatabaseStatusEndpointAcceptance is an end-to-end acceptance test for database status
// Verifies the acceptance criteria from Iteration 2
func TestDatabaseStatusEndpointAcceptance(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database acceptance tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupRouter()

	// Create a request as a real client would
	req, err := http.NewRequest("GET", "/api/v1/database/status", nil)
	require.NoError(t, err, "Should be able to create request")

	// Use the router's ServeHTTP to simulate the request
	recorder := &testResponseWriter{header: make(http.Header)}
	router.ServeHTTP(recorder, req)

	// Verify acceptance criteria from IMPLEMENTATION_PLAN.md Iteration 2:
	// Response: {"success": true, "message": "Database connected", "tables": ["users"]}
	assert.Equal(t, http.StatusOK, recorder.statusCode, "Database status endpoint should return 200 OK")

	// Parse response
	var response struct {
		Success bool     `json:"success"`
		Message string   `json:"message"`
		Tables  []string `json:"tables"`
	}
	err = json.Unmarshal(recorder.body, &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify acceptance criteria
	assert.True(t, response.Success, "Success field should be true")
	assert.Equal(t, "Database connected", response.Message, "Message should match specification")
	assert.Contains(t, response.Tables, "users", "Tables array should contain 'users'")
}

// TestDatabaseStatusEndpointAvailability tests that the database endpoint is available
func TestDatabaseStatusEndpointAvailability(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database acceptance tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupRouter()

	// Make multiple requests to ensure consistency
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", "/api/v1/database/status", nil)
		recorder := &testResponseWriter{header: make(http.Header)}
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.statusCode,
			fmt.Sprintf("Request %d should succeed", i+1))

		// Verify consistent response
		var response map[string]any
		json.Unmarshal(recorder.body, &response)
		assert.Equal(t, true, response["success"],
			fmt.Sprintf("Request %d should have success=true", i+1))
		assert.Equal(t, "Database connected", response["message"],
			fmt.Sprintf("Request %d should have correct message", i+1))
	}
}

// TestDatabaseStatusEndpointResponseTime tests that the endpoint responds quickly
func TestDatabaseStatusEndpointResponseTime(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database acceptance tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupRouter()

	req, _ := http.NewRequest("GET", "/api/v1/database/status", nil)
	recorder := &testResponseWriter{header: make(http.Header)}

	start := time.Now()
	router.ServeHTTP(recorder, req)
	duration := time.Since(start)

	// Database status check should be reasonably fast (under 500ms)
	assert.Less(t, duration, 500*time.Millisecond,
		"Database status endpoint should respond in less than 500ms")
}

// TestFullAPIAcceptanceCriteria validates all Iteration 2 acceptance criteria
func TestFullAPIAcceptanceCriteria(t *testing.T) {
	// Skip if database is not available
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database acceptance tests")
	}

	setupTestDB(t)
	defer cleanupTestDB(t)

	router := setupRouter()

	// Test 1: Health endpoint still works
	t.Run("Health endpoint available", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/health", nil)
		recorder := &testResponseWriter{header: make(http.Header)}
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.statusCode)
		var response map[string]any
		json.Unmarshal(recorder.body, &response)
		assert.Equal(t, true, response["success"])
	})

	// Test 2: Database status endpoint works
	t.Run("Database status endpoint available", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/database/status", nil)
		recorder := &testResponseWriter{header: make(http.Header)}
		router.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.statusCode)
		var response map[string]any
		json.Unmarshal(recorder.body, &response)
		assert.Equal(t, true, response["success"])
		assert.Equal(t, "Database connected", response["message"])
	})

	// Test 3: Users table exists (verified via database status)
	t.Run("Users table created", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/database/status", nil)
		recorder := &testResponseWriter{header: make(http.Header)}
		router.ServeHTTP(recorder, req)

		var response struct {
			Success bool     `json:"success"`
			Tables  []string `json:"tables"`
		}
		json.Unmarshal(recorder.body, &response)
		assert.Contains(t, response.Tables, "users", "Users table should exist")
	})
}
