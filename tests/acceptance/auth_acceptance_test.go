package acceptance

import (
	"encoding/json"
	"io"
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

// AuthAcceptanceTestSuite defines the acceptance test suite for authentication
type AuthAcceptanceTestSuite struct {
	suite.Suite
	server *httptest.Server
	cfg    *config.Config
}

// SetupSuite runs once before all tests
func (suite *AuthAcceptanceTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Set test environment
	os.Setenv("GO_ENV", "test")
	os.Setenv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/kendalls_nails_test?sslmode=disable")
	os.Setenv("AUTH0_DOMAIN", "test.auth0.com")
	os.Setenv("AUTH0_AUDIENCE", "https://api.test.com")
	os.Setenv("PORT", "8080")

	cfg, err := config.Load()
	suite.NoError(err)
	suite.cfg = cfg

	// Create test server
	router := suite.createRouter()
	suite.server = httptest.NewServer(router)
}

// TearDownSuite runs once after all tests
func (suite *AuthAcceptanceTestSuite) TearDownSuite() {
	suite.server.Close()
}

// createRouter creates the test router with all routes
func (suite *AuthAcceptanceTestSuite) createRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		// Health check endpoint (public)
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Custom Nails API is running",
			})
		})

		// Protected endpoint (requires auth)
		v1.GET("/protected", middleware.EnsureValidToken(suite.cfg), func(c *gin.Context) {
			userID, err := middleware.GetUserID(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Could not extract user information",
					},
				})
				return
			}

			claims, err := middleware.GetClaims(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "UNAUTHORIZED",
						"message": "Could not retrieve claims",
					},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "You have accessed a protected endpoint",
				"data": gin.H{
					"user_id": userID,
					"issuer":  claims.RegisteredClaims.Issuer,
					"subject": claims.RegisteredClaims.Subject,
				},
			})
		})
	}

	return router
}

// makeRequest is a helper function to make HTTP requests
func (suite *AuthAcceptanceTestSuite) makeRequest(method, path string, authHeader string) *http.Response {
	req, err := http.NewRequest(method, suite.server.URL+path, nil)
	suite.NoError(err)

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)

	return resp
}

// TestHealthEndpoint tests the public health endpoint
func (suite *AuthAcceptanceTestSuite) TestHealthEndpoint() {
	resp := suite.makeRequest("GET", "/api/v1/health", "")
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(suite.T(), err)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(suite.T(), err)

	assert.True(suite.T(), response["success"].(bool))
	assert.Equal(suite.T(), "Custom Nails API is running", response["message"])
}

// TestProtectedEndpointWorkflow tests the complete authentication workflow
func (suite *AuthAcceptanceTestSuite) TestProtectedEndpointWorkflow() {
	// Step 1: Try to access protected endpoint without auth - should fail
	suite.T().Run("Without Authentication", func(t *testing.T) {
		resp := suite.makeRequest("GET", "/api/v1/protected", "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response, "error")
	})

	// Step 2: Try with invalid token - should fail
	suite.T().Run("With Invalid Token", func(t *testing.T) {
		resp := suite.makeRequest("GET", "/api/v1/protected", "Bearer invalid-token")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		assert.NoError(t, err)

		assert.False(t, response["success"].(bool))
	})

	// Step 3: Try with malformed header - should fail
	suite.T().Run("With Malformed Header", func(t *testing.T) {
		resp := suite.makeRequest("GET", "/api/v1/protected", "InvalidFormat token")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestErrorResponseFormat validates consistent error response format
func (suite *AuthAcceptanceTestSuite) TestErrorResponseFormat() {
	resp := suite.makeRequest("GET", "/api/v1/protected", "")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(suite.T(), err)

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	assert.NoError(suite.T(), err)

	// Validate error format matches API spec
	assert.Contains(suite.T(), response, "success")
	assert.False(suite.T(), response["success"].(bool))
	assert.Contains(suite.T(), response, "error")

	errorObj := response["error"].(map[string]interface{})
	assert.Contains(suite.T(), errorObj, "code")
	assert.Contains(suite.T(), errorObj, "message")

	// Verify error code and message are strings
	assert.IsType(suite.T(), "", errorObj["code"])
	assert.IsType(suite.T(), "", errorObj["message"])
	assert.NotEmpty(suite.T(), errorObj["code"])
	assert.NotEmpty(suite.T(), errorObj["message"])
}

// TestContentTypeHeaders validates that responses have correct content type
func (suite *AuthAcceptanceTestSuite) TestContentTypeHeaders() {
	testCases := []struct {
		name     string
		endpoint string
		auth     string
	}{
		{"Health endpoint", "/api/v1/health", ""},
		{"Protected endpoint without auth", "/api/v1/protected", ""},
		{"Protected endpoint with invalid auth", "/api/v1/protected", "Bearer invalid"},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			resp := suite.makeRequest("GET", tc.endpoint, tc.auth)
			defer resp.Body.Close()

			contentType := resp.Header.Get("Content-Type")
			assert.Contains(t, contentType, "application/json")
		})
	}
}

// TestRunSuite runs the acceptance test suite
func TestAuthAcceptanceTestSuite(t *testing.T) {
	// Skip if running in CI without proper setup
	if os.Getenv("SKIP_AUTH_TESTS") == "true" {
		t.Skip("Skipping auth acceptance tests")
	}

	suite.Run(t, new(AuthAcceptanceTestSuite))
}
