package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/controllers"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/kendall-kelly/kendalls-nails-api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// FileUploadIntegrationTestSuite defines the integration test suite for file upload
type FileUploadIntegrationTestSuite struct {
	suite.Suite
	db        *gorm.DB
	router    *gin.Engine
	uploadDir string
}

// SetupSuite runs once before all tests
func (suite *FileUploadIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.NoError(err)
	suite.db = db

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Order{})
	suite.NoError(err)

	config.SetDB(db)

	// Create temporary upload directory
	suite.uploadDir = suite.T().TempDir()

	// Override the global upload directory for testing
	utils.UploadDir = suite.uploadDir

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
}

// createRouter creates a test router
func (suite *FileUploadIntegrationTestSuite) createRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.CreateOrder)
		v1.GET("/orders/:id", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.GetOrder)
		v1.GET("/uploads/:filename", controllers.GetUploadedImage)
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

// createMultipartRequest creates a multipart form request with file upload
func (suite *FileUploadIntegrationTestSuite) createMultipartRequest(filename string, fileContent []byte, description string, quantity string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	if filename != "" && fileContent != nil {
		part, err := writer.CreateFormFile("image", filename)
		if err != nil {
			return nil, err
		}
		part.Write(fileContent)
	}

	// Add form fields
	writer.WriteField("description", description)
	writer.WriteField("quantity", quantity)

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/api/v1/orders", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// TestCreateOrder_WithValidPNGFile tests creating an order with a valid PNG file
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_WithValidPNGFile() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create multipart request with PNG file
	fileContent := []byte("fake PNG file content")
	req, err := suite.createMultipartRequest("design.png", fileContent, "Nail design with image", "2")
	suite.NoError(err)

	// Make request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Nail design with image", orderData["description"])
	assert.Equal(suite.T(), float64(2), orderData["quantity"])
	assert.NotNil(suite.T(), orderData["image_path"])
	assert.NotEmpty(suite.T(), orderData["image_path"])

	// Verify file was saved
	imagePath := orderData["image_path"].(string)
	fullPath := filepath.Join(suite.uploadDir, imagePath)
	assert.FileExists(suite.T(), fullPath)

	// Verify database record
	var order models.Order
	suite.db.First(&order, uint(orderData["id"].(float64)))
	assert.NotNil(suite.T(), order.ImagePath)
	assert.Equal(suite.T(), imagePath, *order.ImagePath)
}

// TestCreateOrder_WithoutFile tests creating an order without a file (should still work)
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_WithoutFile() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create multipart request without file
	req, err := suite.createMultipartRequest("", nil, "Nail design without image", "1")
	suite.NoError(err)

	// Make request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Nail design without image", orderData["description"])
	assert.Nil(suite.T(), orderData["image_path"])

	// Verify database record
	var order models.Order
	suite.db.First(&order, uint(orderData["id"].(float64)))
	assert.Nil(suite.T(), order.ImagePath)
}

// TestCreateOrder_InvalidFileFormat tests rejection of non-PNG files
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_InvalidFileFormat() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create multipart request with JPG file (not allowed)
	fileContent := []byte("fake JPG file content")
	req, err := suite.createMultipartRequest("design.jpg", fileContent, "Nail design", "2")
	suite.NoError(err)

	// Make request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response - should fail
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.False(suite.T(), response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "INVALID_FILE_FORMAT", errorData["code"])
	assert.Contains(suite.T(), errorData["message"], "Only .png files are allowed")

	// Verify no order was created
	var count int64
	suite.db.Model(&models.Order{}).Count(&count)
	assert.Equal(suite.T(), int64(0), count)
}

// TestCreateOrder_FileTooLarge tests rejection of files exceeding size limit
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_FileTooLarge() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create multipart request with file exceeding 10MB
	fileContent := make([]byte, 11*1024*1024) // 11MB
	req, err := suite.createMultipartRequest("large.png", fileContent, "Nail design", "2")
	suite.NoError(err)

	// Make request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response - should fail
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.False(suite.T(), response["success"].(bool))
	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FILE_TOO_LARGE", errorData["code"])
	assert.Contains(suite.T(), errorData["message"], "File size exceeds")

	// Verify no order was created
	var count int64
	suite.db.Model(&models.Order{}).Count(&count)
	assert.Equal(suite.T(), int64(0), count)
}

// TestCreateOrder_JSONRequest_BackwardCompatibility tests that JSON requests still work
func (suite *FileUploadIntegrationTestSuite) TestCreateOrder_JSONRequest_BackwardCompatibility() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create JSON request (no file)
	reqBody := map[string]interface{}{
		"description": "JSON order",
		"quantity":    3,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/orders", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Make request
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	assert.True(suite.T(), response["success"].(bool))
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "JSON order", orderData["description"])
	assert.Equal(suite.T(), float64(3), orderData["quantity"])
	assert.Nil(suite.T(), orderData["image_path"])
}

// TestServeUploadedFile tests that uploaded files can be retrieved
func (suite *FileUploadIntegrationTestSuite) TestServeUploadedFile() {
	// Create a test file in the upload directory
	testContent := []byte("test image content")
	testFilename := "test123.png"
	testPath := filepath.Join(suite.uploadDir, testFilename)

	err := os.WriteFile(testPath, testContent, 0644)
	suite.NoError(err)

	// Request the file
	req := httptest.NewRequest("GET", "/api/v1/uploads/"+testFilename, nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify content
	body, err := io.ReadAll(w.Body)
	suite.NoError(err)
	assert.Equal(suite.T(), testContent, body)
}

// TestFileUploadIntegrationSuite runs the test suite
func TestFileUploadIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FileUploadIntegrationTestSuite))
}
