package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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

// FileUploadAcceptanceTestSuite defines the acceptance test suite for file upload feature
type FileUploadAcceptanceTestSuite struct {
	suite.Suite
	server    *httptest.Server
	db        *gorm.DB
	mockImage *services.MockImageService
}

// SetupSuite runs once before all tests
func (suite *FileUploadAcceptanceTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup mock image service for testing
	suite.mockImage = services.NewMockImageService()
	suite.mockImage.SetAsMockForTesting()

	// Setup database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.NoError(err)
	suite.db = db

	err = db.AutoMigrate(&models.User{}, &models.Order{})
	suite.NoError(err)

	config.SetDB(db)

	// Create test server
	router := suite.createRouter()
	suite.server = httptest.NewServer(router)
}

// TearDownSuite runs once after all tests
func (suite *FileUploadAcceptanceTestSuite) TearDownSuite() {
	suite.server.Close()
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// SetupTest runs before each test
func (suite *FileUploadAcceptanceTestSuite) SetupTest() {
	// Clean up database before each test
	suite.db.Exec("DELETE FROM orders")
	suite.db.Exec("DELETE FROM users")

	// Clear mock image storage storage
	suite.mockImage.Clear()
}

// createRouter creates the full application router for acceptance testing
func (suite *FileUploadAcceptanceTestSuite) createRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.CreateOrder)
		v1.GET("/orders/:id", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.GetOrder)
	}

	return router
}

// mockAuthMiddleware simulates authentication for acceptance testing
func (suite *FileUploadAcceptanceTestSuite) mockAuthMiddleware(auth0ID, role string) gin.HandlerFunc {
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
func (suite *FileUploadAcceptanceTestSuite) createMultipartRequest(url, filename string, fileContent []byte, description string, quantity string) (*http.Request, error) {
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

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// TestCompleteFileUploadWorkflow_Acceptance tests the complete end-to-end workflow
// This is the happy path: customer uploads image, creates order, retrieves order with image, accesses image
func (suite *FileUploadAcceptanceTestSuite) TestCompleteFileUploadWorkflow_Acceptance() {
	// Step 1: Setup - Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Jane Designer",
		Email:   "jane@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Step 2: Customer creates an order with a PNG image
	imageContent := []byte("This is a fake PNG image content for testing purposes")
	req, err := suite.createMultipartRequest(
		suite.server.URL+"/api/v1/orders",
		"my-nail-design.png",
		imageContent,
		"Beautiful pink nails with glitter and stars",
		"2",
	)
	suite.NoError(err)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Step 3: Verify order creation was successful
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	suite.NoError(err)

	assert.True(suite.T(), createResponse["success"].(bool))
	orderData := createResponse["data"].(map[string]interface{})

	// Verify order details
	orderID := int(orderData["id"].(float64))
	assert.Equal(suite.T(), "Beautiful pink nails with glitter and stars", orderData["description"])
	assert.Equal(suite.T(), float64(2), orderData["quantity"])
	assert.Equal(suite.T(), "submitted", orderData["status"])

	// Verify image was uploaded
	assert.NotNil(suite.T(), orderData["image_s3_key"])
	s3Key := orderData["image_s3_key"].(string)
	assert.NotEmpty(suite.T(), s3Key)
	assert.Contains(suite.T(), s3Key, ".png")

	// Step 4: Verify the file was actually uploaded to mock image storage
	assert.True(suite.T(), suite.mockImage.ImageExists(s3Key), "File should exist in mock image storage")

	// Verify file content in mock image storage
	uploadedFiles := suite.mockImage.GetUploadedImages()
	savedContent, exists := uploadedFiles[s3Key]
	assert.True(suite.T(), exists, "File content should be stored in mock image storage")
	assert.Equal(suite.T(), imageContent, savedContent)

	// Step 5: Customer retrieves the order to verify it includes the image path
	getReq, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/orders/%d", suite.server.URL, orderID), nil)
	suite.NoError(err)

	getResp, err := http.DefaultClient.Do(getReq)
	suite.NoError(err)
	defer getResp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, getResp.StatusCode)

	var getResponse map[string]interface{}
	err = json.NewDecoder(getResp.Body).Decode(&getResponse)
	suite.NoError(err)

	assert.True(suite.T(), getResponse["success"].(bool))
	retrievedOrder := getResponse["data"].(map[string]interface{})

	// Verify the S3 key is included
	assert.Equal(suite.T(), s3Key, retrievedOrder["image_s3_key"].(string))

	// Step 6: Verify presigned URL is included (when using S3)
	// The image_url field should be populated with a presigned URL
	assert.NotNil(suite.T(), retrievedOrder["image_url"])
	imageURL := retrievedOrder["image_url"].(string)
	assert.NotEmpty(suite.T(), imageURL)
	assert.Contains(suite.T(), imageURL, s3Key)

	// Step 7: Verify in the database
	var dbOrder models.Order
	err = suite.db.Preload("Customer").First(&dbOrder, orderID).Error
	suite.NoError(err)

	assert.Equal(suite.T(), "Beautiful pink nails with glitter and stars", dbOrder.Description)
	assert.Equal(suite.T(), 2, dbOrder.Quantity)
	assert.Equal(suite.T(), "submitted", dbOrder.Status)
	assert.NotNil(suite.T(), dbOrder.ImageS3Key)
	assert.Equal(suite.T(), s3Key, *dbOrder.ImageS3Key)
	assert.Equal(suite.T(), customer.ID, dbOrder.CustomerID)
	assert.Equal(suite.T(), "Jane Designer", dbOrder.Customer.Name)
}

// TestCreateOrderWithoutImage_Acceptance tests that orders can still be created without images
func (suite *FileUploadAcceptanceTestSuite) TestCreateOrderWithoutImage_Acceptance() {
	// Step 1: Setup - Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "John Customer",
		Email:   "john@example.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Step 2: Customer creates an order WITHOUT an image (using multipart form)
	req, err := suite.createMultipartRequest(
		suite.server.URL+"/api/v1/orders",
		"", // no filename
		nil, // no file content
		"Simple nail design without image",
		"1",
	)
	suite.NoError(err)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	// Step 3: Verify order creation was successful
	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var createResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResponse)
	suite.NoError(err)

	assert.True(suite.T(), createResponse["success"].(bool))
	orderData := createResponse["data"].(map[string]interface{})

	// Verify order details
	assert.Equal(suite.T(), "Simple nail design without image", orderData["description"])
	assert.Equal(suite.T(), float64(1), orderData["quantity"])
	assert.Nil(suite.T(), orderData["image_s3_key"]) // No image

	// Step 4: Verify in the database
	orderID := uint(orderData["id"].(float64))
	var dbOrder models.Order
	err = suite.db.First(&dbOrder, orderID).Error
	suite.NoError(err)

	assert.Nil(suite.T(), dbOrder.ImageS3Key)
}

// TestFileUploadValidation_Acceptance tests end-to-end validation errors
func (suite *FileUploadAcceptanceTestSuite) TestFileUploadValidation_Acceptance() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "test@example.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Test 1: Try to upload a JPEG file (should fail)
	jpegContent := []byte("fake jpeg content")
	req, err := suite.createMultipartRequest(
		suite.server.URL+"/api/v1/orders",
		"design.jpeg",
		jpegContent,
		"Design with invalid format",
		"2",
	)
	suite.NoError(err)

	resp, err := http.DefaultClient.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	var errorResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	suite.NoError(err)

	assert.False(suite.T(), errorResponse["success"].(bool))
	errorData := errorResponse["error"].(map[string]interface{})
	assert.Equal(suite.T(), "INVALID_FILE_FORMAT", errorData["code"])
	assert.Contains(suite.T(), errorData["message"], "Only .png files are allowed")

	// Verify no order was created
	var count int64
	suite.db.Model(&models.Order{}).Count(&count)
	assert.Equal(suite.T(), int64(0), count)
}

// TestMultipleOrdersWithImages_Acceptance tests creating multiple orders with different images
func (suite *FileUploadAcceptanceTestSuite) TestMultipleOrdersWithImages_Acceptance() {
	// Setup: Create customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Multi Order Customer",
		Email:   "multi@example.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create first order with image
	image1Content := []byte("First design image content")
	req1, err := suite.createMultipartRequest(
		suite.server.URL+"/api/v1/orders",
		"design1.png",
		image1Content,
		"First nail design",
		"1",
	)
	suite.NoError(err)

	resp1, err := http.DefaultClient.Do(req1)
	suite.NoError(err)
	defer resp1.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp1.StatusCode)

	var response1 map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&response1)
	order1Data := response1["data"].(map[string]interface{})
	s3Key1 := order1Data["image_s3_key"].(string)

	// Create second order with different image
	image2Content := []byte("Second design image content - different content")
	req2, err := suite.createMultipartRequest(
		suite.server.URL+"/api/v1/orders",
		"design2.png",
		image2Content,
		"Second nail design",
		"3",
	)
	suite.NoError(err)

	resp2, err := http.DefaultClient.Do(req2)
	suite.NoError(err)
	defer resp2.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp2.StatusCode)

	var response2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response2)
	order2Data := response2["data"].(map[string]interface{})
	s3Key2 := order2Data["image_s3_key"].(string)

	// Verify images have different S3 keys
	assert.NotEqual(suite.T(), s3Key1, s3Key2)

	// Verify both files exist in mock image storage
	assert.True(suite.T(), suite.mockImage.ImageExists(s3Key1), "First file should exist in mock image storage")
	assert.True(suite.T(), suite.mockImage.ImageExists(s3Key2), "Second file should exist in mock image storage")

	// Verify both files have different content in mock image storage
	uploadedFiles := suite.mockImage.GetUploadedImages()
	content1 := uploadedFiles[s3Key1]
	content2 := uploadedFiles[s3Key2]
	assert.NotEqual(suite.T(), content1, content2)
	assert.Equal(suite.T(), image1Content, content1)
	assert.Equal(suite.T(), image2Content, content2)

	// Verify both orders exist in database
	var orderCount int64
	suite.db.Model(&models.Order{}).Count(&orderCount)
	assert.Equal(suite.T(), int64(2), orderCount)
}

// TestFileUploadAcceptanceSuite runs the test suite
func TestFileUploadAcceptanceSuite(t *testing.T) {
	suite.Run(t, new(FileUploadAcceptanceTestSuite))
}
