package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/kendall-kelly/kendalls-nails-api/services"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate the User model
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// setupMockAuth0Server creates a mock HTTP server that simulates Auth0's /userinfo endpoint
func setupMockAuth0Server(userInfoMap map[string]*services.Auth0UserInfo) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/userinfo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || len(authHeader) < 7 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := authHeader[7:] // Remove "Bearer " prefix

		// Look up user info by token
		userInfo, exists := userInfoMap[token]
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	}))
}

// mockAuthMiddleware simulates the Auth0 JWT middleware for testing
// It sets up the context exactly as the real EnsureValidToken middleware does
func mockAuthMiddleware(auth0ID, role, accessToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set the user_id (Auth0 ID from 'sub' claim)
		c.Set("user_id", auth0ID)

		// Set the access token for calling /userinfo
		c.Set("access_token", accessToken)

		// Create custom claims matching the real structure (only role, no email/name)
		customClaims := &middleware.CustomClaims{
			Role: role,
		}

		// Create a proper ValidatedClaims structure
		// This matches what the real JWT middleware creates
		mockClaims := &validator.ValidatedClaims{
			CustomClaims: customClaims,
		}

		// Store in context the same way the real middleware does
		c.Set("validated_claims", mockClaims)

		c.Next()
	}
}

func TestCreateUser(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)

	tests := []struct {
		name           string
		auth0ID        string
		email          string
		userName       string
		role           string
		accessToken    string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Create customer user successfully",
			auth0ID:        "auth0|123456",
			email:          "john@example.com",
			userName:       "John Doe",
			role:           "customer",
			accessToken:    "token-123456",
			expectedStatus: http.StatusCreated,
			expectedCode:   "",
		},
		{
			name:           "Create technician user successfully",
			auth0ID:        "auth0|tech789",
			email:          "tech@example.com",
			userName:       "Tech User",
			role:           "technician",
			accessToken:    "token-tech789",
			expectedStatus: http.StatusCreated,
			expectedCode:   "",
		},
		{
			name:           "Create user with default role when role is empty",
			auth0ID:        "auth0|norole",
			email:          "norole@example.com",
			userName:       "No Role User",
			role:           "",
			accessToken:    "token-norole",
			expectedStatus: http.StatusCreated,
			expectedCode:   "",
		},
		{
			name:           "Fail with missing email",
			auth0ID:        "auth0|noemail",
			email:          "",
			userName:       "No Email User",
			role:           "customer",
			accessToken:    "token-noemail",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_EMAIL",
		},
		{
			name:           "Fail with missing name",
			auth0ID:        "auth0|noname",
			email:          "noname@example.com",
			userName:       "",
			role:           "customer",
			accessToken:    "token-noname",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_NAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear database before each test
			db.Exec("DELETE FROM users")

			// Setup mock Auth0 server
			userInfoMap := map[string]*services.Auth0UserInfo{
				tt.accessToken: {
					Sub:   tt.auth0ID,
					Email: tt.email,
					Name:  tt.userName,
				},
			}
			mockServer := setupMockAuth0Server(userInfoMap)
			defer mockServer.Close()

			// Configure test to use mock Auth0 domain
			// The mock server URL is "http://127.0.0.1:port" but Auth0Service expects just the domain
			// and will prepend "https://". For testing, we'll extract just the host:port part
			testConfig := &config.Config{
				Auth0Domain: mockServer.URL, // Pass full URL for testing (http://...)
			}

			// Store the config temporarily for the test
			originalConfig := config.GetConfig()
			defer func() {
				// Restore original config after test
				config.SetConfig(originalConfig)
			}()
			config.SetConfig(testConfig)

			// Setup route with mock auth middleware
			router := setupTestRouter()
			router.POST("/users", mockAuthMiddleware(tt.auth0ID, tt.role, tt.accessToken), CreateUser)

			req := httptest.NewRequest(http.MethodPost, "/users", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response body: %s", w.Body.String())

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedStatus == http.StatusCreated {
				assert.True(t, response["success"].(bool))
				assert.NotNil(t, response["data"])
				data := response["data"].(map[string]interface{})
				assert.Equal(t, tt.email, data["email"])
				assert.Equal(t, tt.userName, data["name"])
				assert.Equal(t, tt.auth0ID, data["auth0_id"])
				if tt.role != "" {
					assert.Equal(t, tt.role, data["role"])
				} else {
					assert.Equal(t, "customer", data["role"])
				}
			} else {
				assert.False(t, response["success"].(bool))
				errorData := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedCode, errorData["code"])
			}
		})
	}
}

func TestCreateUser_DuplicateAuth0ID(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)

	// Create first user
	user := models.User{
		Auth0ID: "auth0|duplicate",
		Name:    "First User",
		Email:   "first@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Setup mock Auth0 server
	accessToken := "token-duplicate"
	userInfoMap := map[string]*services.Auth0UserInfo{
		accessToken: {
			Sub:   "auth0|duplicate",
			Email: "second@example.com",
			Name:  "Second User",
		},
	}
	mockServer := setupMockAuth0Server(userInfoMap)
	defer mockServer.Close()

	// Configure test to use mock Auth0 domain
	testConfig := &config.Config{
		Auth0Domain: mockServer.URL,
	}
	originalConfig := config.GetConfig()
	defer func() {
		config.SetConfig(originalConfig)
	}()
	config.SetConfig(testConfig)

	// Try to create user with duplicate Auth0ID
	router := setupTestRouter()
	router.POST("/users", mockAuthMiddleware("auth0|duplicate", "customer", accessToken), CreateUser)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_EXISTS", errorData["code"])
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)

	// Create first user
	user := models.User{
		Auth0ID: "auth0|first",
		Name:    "First User",
		Email:   "duplicate@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Setup mock Auth0 server
	accessToken := "token-second"
	userInfoMap := map[string]*services.Auth0UserInfo{
		accessToken: {
			Sub:   "auth0|second",
			Email: "duplicate@example.com",
			Name:  "Second User",
		},
	}
	mockServer := setupMockAuth0Server(userInfoMap)
	defer mockServer.Close()

	// Configure test to use mock Auth0 domain
	testConfig := &config.Config{
		Auth0Domain: mockServer.URL,
	}
	originalConfig := config.GetConfig()
	defer func() {
		config.SetConfig(originalConfig)
	}()
	config.SetConfig(testConfig)

	// Try to create user with duplicate email
	router := setupTestRouter()
	router.POST("/users", mockAuthMiddleware("auth0|second", "customer", accessToken), CreateUser)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_EXISTS", errorData["code"])
}

func TestGetMyProfile_Success(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.GET("/users/me", func(c *gin.Context) {
		// Mock middleware setting user_id
		c.Set("user_id", "auth0|testuser")
		GetMyProfile(c)
	})

	// Create a user in the database
	user := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Test User",
		Email:   "test@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "Test User", data["name"])
	assert.Equal(t, "customer", data["role"])
}

func TestGetMyProfile_UserNotFound(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.GET("/users/me", func(c *gin.Context) {
		// Mock middleware setting user_id for non-existent user
		c.Set("user_id", "auth0|nonexistent")
		GetMyProfile(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
}

func TestUpdateMyProfile_Success(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		// Mock middleware setting user_id
		c.Set("user_id", "auth0|testuser")
		UpdateMyProfile(c)
	})

	// Create a user in the database
	user := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Old Name",
		Email:   "old@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Update user
	payload := UpdateUserRequest{
		Name:  "New Name",
		Email: "new@example.com",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "new@example.com", data["email"])
	assert.Equal(t, "New Name", data["name"])
}

func TestUpdateMyProfile_PartialUpdate(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		// Mock middleware setting user_id
		c.Set("user_id", "auth0|testuser")
		UpdateMyProfile(c)
	})

	// Create a user in the database
	user := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Original Name",
		Email:   "original@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Update only name
	payload := UpdateUserRequest{
		Name: "Updated Name",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "original@example.com", data["email"]) // Email unchanged
	assert.Equal(t, "Updated Name", data["name"])          // Name updated
}

func TestUpdateMyProfile_UserNotFound(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		// Mock middleware setting user_id for non-existent user
		c.Set("user_id", "auth0|nonexistent")
		UpdateMyProfile(c)
	})

	payload := UpdateUserRequest{
		Name: "New Name",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "USER_NOT_FOUND", errorData["code"])
}

func TestUpdateMyProfile_InvalidEmail(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		c.Set("user_id", "auth0|testuser")
		UpdateMyProfile(c)
	})

	// Create a user
	user := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Test User",
		Email:   "test@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Try to update with invalid email
	payload := UpdateUserRequest{
		Email: "invalid-email",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errorData["code"])
}

func TestUpdateMyProfile_DuplicateEmail(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		c.Set("user_id", "auth0|testuser")
		UpdateMyProfile(c)
	})

	// Create two users
	user1 := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Test User 1",
		Email:   "user1@example.com",
		Role:    "customer",
	}
	db.Create(&user1)

	user2 := models.User{
		Auth0ID: "auth0|otheruser",
		Name:    "Test User 2",
		Email:   "user2@example.com",
		Role:    "customer",
	}
	db.Create(&user2)

	// Try to update user1's email to user2's email
	payload := UpdateUserRequest{
		Email: "user2@example.com",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.False(t, response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "EMAIL_EXISTS", errorData["code"])
}

func TestUpdateMyProfile_EmptyUpdate(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	config.SetDB(db)
	router := setupTestRouter()

	router.PUT("/users/me", func(c *gin.Context) {
		c.Set("user_id", "auth0|testuser")
		UpdateMyProfile(c)
	})

	// Create a user
	user := models.User{
		Auth0ID: "auth0|testuser",
		Name:    "Test User",
		Email:   "test@example.com",
		Role:    "customer",
	}
	db.Create(&user)

	// Send empty update
	payload := UpdateUserRequest{}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "Test User", data["name"])
}
