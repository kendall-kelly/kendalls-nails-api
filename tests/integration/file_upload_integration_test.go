package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/controllers"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/kendall-kelly/kendalls-nails-api/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FileUploadIntegrationTestSuite defines the integration test suite for file upload
type FileUploadIntegrationTestSuite struct {
	suite.Suite
	db         *gorm.DB
	router     *gin.Engine
	uploadDir  string
	mockImage  *services.MockImageService
}

// SetupSuite runs once before all tests
func (suite *FileUploadIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup mock image service for testing
	suite.mockImage = services.NewMockImageService()
	suite.mockImage.SetAsMockForTesting()

	// Mock AWS S3 credentials for testing (required for config validation)
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_S3_BUCKET", "test-bucket")
	os.Setenv("AWS_ACCESS_KEY_ID", "test-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret")

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.NoError(err)
	suite.db = db

	err = db.AutoMigrate(&models.User{}, &models.Order{})
	suite.NoError(err)

	config.SetDB(db)

	// Setup router
	suite.router = suite.createRouter()
}

// TearDownSuite runs once after all tests
func (suite *FileUploadIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// SetupTest runs before each test
func (suite *FileUploadIntegrationTestSuite) SetupTest() {
	// Clean up database before each test
	suite.db.Exec("DELETE FROM orders")
	suite.db.Exec("DELETE FROM users")

	// Clear mock image storage storage
	suite.mockImage.Clear()
}

// createRouter creates a test router
func (suite *FileUploadIntegrationTestSuite) createRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.CreateOrder)
		v1.GET("/orders/:id", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.GetOrder)
	}

	return router
}

// mockAuthMiddleware simulates authentication for testing
func (suite *FileUploadIntegrationTestSuite) mockAuthMiddleware(auth0ID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", auth0ID)
		c.Set("access_token", "mock-token")

		customClaims := &middleware.CustomClaims{
			Role: role,
		}
		c.Set("custom_claims", customClaims)

		c.Next()
	}
}

// TestCreateOrder_WithValidPNGFile tests creating an order with a valid PNG file
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_WithValidPNGFile() {
	// Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Jane Customer",
		Email:   "jane@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create multipart form with image
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	imageContent := []byte("fake PNG content")
	part, err := writer.CreateFormFile("image", "design.png")
	suite.NoError(err)
	_, err = part.Write(imageContent)
	suite.NoError(err)

	// Add description
	writer.WriteField("description", "Beautiful nail design")
	writer.WriteField("quantity", "2")

	err = writer.Close()
	suite.NoError(err)

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Beautiful nail design", orderData["description"])
	assert.Equal(suite.T(), float64(2), orderData["quantity"])
	assert.NotNil(suite.T(), orderData["image_s3_key"])

	// Verify file was uploaded to mock image storage
	s3Key := orderData["image_s3_key"].(string)
	assert.True(suite.T(), suite.mockImage.ImageExists(s3Key), "File should exist in mock image storage")

	// Verify file content in mock image storage
	uploadedFiles := suite.mockImage.GetUploadedImages()
	savedContent, exists := uploadedFiles[s3Key]
	assert.True(suite.T(), exists, "File content should be stored in mock image storage")
	assert.Equal(suite.T(), imageContent, savedContent)
}

// TestCreateOrder_WithoutFile tests creating an order without a file
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_WithoutFile() {
	// Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "John Customer",
		Email:   "john@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create multipart form WITHOUT image
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("description", "Simple design without image")
	writer.WriteField("quantity", "1")

	err = writer.Close()
	suite.NoError(err)

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Simple design without image", orderData["description"])
	assert.Equal(suite.T(), float64(1), orderData["quantity"])
	assert.Nil(suite.T(), orderData["image_s3_key"])
}

// TestCreateOrder_InvalidFileFormat tests creating an order with invalid file format
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_InvalidFileFormat() {
	// Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "test@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create multipart form with JPEG file (not allowed)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JPEG file
	part, err := writer.CreateFormFile("image", "design.jpg")
	suite.NoError(err)
	_, err = part.Write([]byte("fake JPEG content"))
	suite.NoError(err)

	writer.WriteField("description", "Design with invalid format")
	writer.WriteField("quantity", "2")

	err = writer.Close()
	suite.NoError(err)

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.False(suite.T(), response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "INVALID_FILE_FORMAT", errorData["code"])
}

// TestCreateOrder_FileTooLarge tests creating an order with a file that's too large
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_FileTooLarge() {
	// Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "test@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create multipart form with large file (11MB)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add large PNG file
	largeContent := make([]byte, 11*1024*1024) // 11MB
	part, err := writer.CreateFormFile("image", "large.png")
	suite.NoError(err)
	_, err = part.Write(largeContent)
	suite.NoError(err)

	writer.WriteField("description", "Design with large file")
	writer.WriteField("quantity", "2")

	err = writer.Close()
	suite.NoError(err)

	// Make request
	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.False(suite.T(), response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FILE_TOO_LARGE", errorData["code"])
}

// TestCreateOrder_JSONRequest_BackwardCompatibility tests JSON requests still work
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_JSONRequest_BackwardCompatibility() {
	// Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "JSON Customer",
		Email:   "json@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create JSON request (no image)
	body := bytes.NewBufferString(`{"description": "JSON order", "quantity": 3}`)
	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "JSON order", orderData["description"])
	assert.Equal(suite.T(), float64(3), orderData["quantity"])
	assert.Nil(suite.T(), orderData["image_s3_key"])
}

// TestFileUploadIntegrationSuite runs the test suite
func TestFileUploadIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FileUploadIntegrationTestSuite))
}
