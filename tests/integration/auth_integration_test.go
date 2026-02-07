package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite defines the test suite for auth integration tests
type AuthIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
	cfg    *config.Config
}

// SetupSuite runs once before all tests
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Set test environment variables
	os.Setenv("GO_ENV", "test")
	os.Setenv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/kendalls_nails_test?sslmode=disable")
	os.Setenv("AUTH0_DOMAIN", "test.auth0.com")
	os.Setenv("AUTH0_AUDIENCE", "https://api.test.com")
	os.Setenv("PORT", "8080")
	// Mock AWS S3 credentials for testing
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_S3_BUCKET", "test-bucket")
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	// Load configuration
	cfg, err := config.Load()
	suite.NoError(err)
	suite.cfg = cfg
}

// SetupTest runs before each test
func (suite *AuthIntegrationTestSuite) SetupTest() {
	// Create a new router for each test
	suite.router = gin.New()

	// Add test routes
	v1 := suite.router.Group("/api/v1")
	{
		// Public endpoint
		v1.GET("/public", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Public endpoint",
			})
		})

		// Protected endpoint
		v1.GET("/protected", middleware.EnsureValidToken(suite.cfg), func(c *gin.Context) {
			userID, _ := middleware.GetUserID(c)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Protected endpoint",
				"user_id": userID,
			})
		})
	}
}

// TestPublicEndpoint tests that public endpoints work without authentication
func (suite *AuthIntegrationTestSuite) TestPublicEndpoint() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public", nil)

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), "Public endpoint", response["message"])
}

// TestProtectedEndpointWithoutToken tests that protected endpoints reject requests without tokens
func (suite *AuthIntegrationTestSuite) TestProtectedEndpointWithoutToken() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)

	suite.router.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))
}

// TestProtectedEndpointWithInvalidToken tests that protected endpoints reject invalid tokens
func (suite *AuthIntegrationTestSuite) TestProtectedEndpointWithInvalidToken() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")

	suite.router.ServeHTTP(w, req)

	// Should return 401 Unauthorized
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))
}

// TestProtectedEndpointWithMalformedAuthHeader tests various malformed auth headers
func (suite *AuthIntegrationTestSuite) TestProtectedEndpointWithMalformedAuthHeader() {
	testCases := []struct {
		name   string
		header string
	}{
		{"Missing Bearer prefix", "token-without-bearer"},
		{"Wrong prefix", "Basic token"},
		{"Empty token", "Bearer "},
		{"Only Bearer", "Bearer"},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
			req.Header.Set("Authorization", tc.header)

			suite.router.ServeHTTP(w, req)

			// Should return 401 Unauthorized
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestProtectedEndpointResponseFormat tests the error response format
func (suite *AuthIntegrationTestSuite) TestProtectedEndpointResponseFormat() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)

	suite.router.ServeHTTP(w, req)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	// Check response format
	assert.Contains(suite.T(), response, "success")
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response, "error")

	errorObj := response["error"].(map[string]interface{})
	assert.Contains(suite.T(), errorObj, "code")
	assert.Contains(suite.T(), errorObj, "message")
}

// TestRunSuite runs the test suite
func TestAuthIntegrationTestSuite(t *testing.T) {
	// Skip if running in CI without proper Auth0 setup
	if os.Getenv("SKIP_AUTH_TESTS") == "true" {
		t.Skip("Skipping auth integration tests")
	}

	suite.Run(t, new(AuthIntegrationTestSuite))
}
