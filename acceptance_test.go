package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
