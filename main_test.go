package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestHealthCheck is a unit test for the healthCheck handler function
func TestHealthCheck(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test context and response recorder
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Call the handler
	healthCheck(c)

	// Assert the status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")

	// Parse the response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response should be valid JSON")

	// Assert the response structure
	assert.Equal(t, true, response["success"], "Expected success to be true")
	assert.Equal(t, "Custom Nails API is running", response["message"], "Expected correct message")
}

// TestHealthCheckResponseFormat tests the exact JSON format
func TestHealthCheckResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	healthCheck(c)

	// Verify JSON content type
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response has exactly 2 fields
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Len(t, response, 2, "Response should have exactly 2 fields")
	assert.Contains(t, response, "success")
	assert.Contains(t, response, "message")
}
