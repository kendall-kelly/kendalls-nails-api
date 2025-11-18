package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupRouter creates and configures the router for integration testing
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthCheck)
	}

	return router
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
