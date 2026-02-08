package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// OrderIntegrationTestSuite defines the test suite for order integration tests
type OrderIntegrationTestSuite struct {
	suite.Suite
	router *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
}

// SetupSuite runs once before all tests
func (suite *OrderIntegrationTestSuite) SetupSuite() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Set test environment variables
	os.Setenv("GO_ENV", "test")
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
func (suite *OrderIntegrationTestSuite) SetupTest() {
	// Create in-memory database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.NoError(err)
	suite.db = db

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{}, &models.Order{})
	suite.NoError(err)

	// Set the database in config
	config.SetDB(db)

	// Initialize mock S3 service for testing
	mockS3 := services.NewMockS3Service()
	mockS3.SetAsMockForTesting()

	// Initialize image service with mock S3
	services.InitImageService(mockS3)

	// Create a new router for each test
	suite.router = gin.New()

	// Add order routes
	v1 := suite.router.Group("/api/v1")
	{
		// Order routes with mock auth middleware
		v1.POST("/orders", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.CreateOrder)
		v1.GET("/orders", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.ListOrders)
		v1.GET("/orders/:id", suite.mockAuthMiddleware("auth0|customer", "customer"), controllers.GetOrder)
	}
}

// TearDownTest runs after each test
func (suite *OrderIntegrationTestSuite) TearDownTest() {
	// Clean up database
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// mockAuthMiddleware creates a middleware that simulates authentication
func (suite *OrderIntegrationTestSuite) mockAuthMiddleware(auth0ID, role string) gin.HandlerFunc {
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

// TestOrderWorkflow_CreateListAndGet tests the full order workflow
func (suite *OrderIntegrationTestSuite) TestOrderWorkflow_CreateListAndGet() {
	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Step 1: Create an order
	createOrderBody := map[string]interface{}{
		"description": "Integration test order",
		"quantity":    2,
	}
	createBodyJSON, _ := json.Marshal(createOrderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(createBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), createResponse["success"].(bool))

	orderData := createResponse["data"].(map[string]interface{})
	orderID := orderData["id"].(float64)

	// Step 2: List orders (should include the created order)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var listResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), listResponse["success"].(bool))

	orders := listResponse["data"].([]interface{})
	assert.Equal(suite.T(), 1, len(orders))

	// Step 3: Get the specific order
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/orders/%d", int(orderID)), nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), getResponse["success"].(bool))

	retrievedOrder := getResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), orderID, retrievedOrder["id"].(float64))
	assert.Equal(suite.T(), "Integration test order", retrievedOrder["description"])
}

// TestListOrders_WithMultipleOrders tests listing multiple orders
func (suite *OrderIntegrationTestSuite) TestListOrders_WithMultipleOrders() {
	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create multiple orders
	for i := 1; i <= 3; i++ {
		order := models.Order{
			Description: "Order " + string(rune(i+'0')),
			Quantity:    i,
			Status:      "submitted",
			CustomerID:  customer.ID,
		}
		err := suite.db.Create(&order).Error
		suite.NoError(err)
	}

	// List orders
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	orders := response["data"].([]interface{})
	assert.Equal(suite.T(), 3, len(orders))

	// Verify pagination
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
	assert.Equal(suite.T(), float64(10), pagination["limit"])
	assert.Equal(suite.T(), float64(3), pagination["total"])
	assert.Equal(suite.T(), float64(1), pagination["totalPages"])
}

// TestListOrders_WithPagination tests pagination functionality
func (suite *OrderIntegrationTestSuite) TestListOrders_WithPagination() {
	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	err := suite.db.Create(&customer).Error
	suite.NoError(err)

	// Create 5 orders
	for i := 1; i <= 5; i++ {
		order := models.Order{
			Description: "Order " + string(rune(i+'0')),
			Quantity:    i,
			Status:      "submitted",
			CustomerID:  customer.ID,
		}
		err := suite.db.Create(&order).Error
		suite.NoError(err)
	}

	// Test page 1 with limit 2
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=1&limit=2", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	orders := response["data"].([]interface{})
	assert.Equal(suite.T(), 2, len(orders))

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(1), pagination["page"])
	assert.Equal(suite.T(), float64(2), pagination["limit"])
	assert.Equal(suite.T(), float64(5), pagination["total"])
	assert.Equal(suite.T(), float64(3), pagination["totalPages"])

	// Test page 2 with limit 2
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=2&limit=2", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	orders = response["data"].([]interface{})
	assert.Equal(suite.T(), 2, len(orders))

	pagination = response["pagination"].(map[string]interface{})
	assert.Equal(suite.T(), float64(2), pagination["page"])
}

// TestListOrders_CustomerSeeOnlyOwnOrders tests that customers only see their own orders
func (suite *OrderIntegrationTestSuite) TestListOrders_CustomerSeeOnlyOwnOrders() {
	// Create two customers
	customer1 := models.User{
		Auth0ID: "auth0|customer1",
		Name:    "Customer One",
		Email:   "customer1@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer1)

	customer2 := models.User{
		Auth0ID: "auth0|customer2",
		Name:    "Customer Two",
		Email:   "customer2@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer2)

	// Create orders for both customers
	order1 := models.Order{
		Description: "Customer1 order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer1.ID,
	}
	suite.db.Create(&order1)

	order2 := models.Order{
		Description: "Customer2 order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer2.ID,
	}
	suite.db.Create(&order2)

	// Create router with customer1's auth
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/orders", suite.mockAuthMiddleware(customer1.Auth0ID, "customer"), controllers.ListOrders)
	}

	// List orders as customer1
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	orders := response["data"].([]interface{})
	assert.Equal(suite.T(), 1, len(orders), "Customer should only see their own order")

	order := orders[0].(map[string]interface{})
	assert.Equal(suite.T(), "Customer1 order", order["description"])
}

// TestGetOrder_Authorization tests that customers can only access their own orders
func (suite *OrderIntegrationTestSuite) TestGetOrder_Authorization() {
	// Create two customers
	customer1 := models.User{
		Auth0ID: "auth0|customer1",
		Name:    "Customer One",
		Email:   "customer1@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer1)

	customer2 := models.User{
		Auth0ID: "auth0|customer2",
		Name:    "Customer Two",
		Email:   "customer2@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer2)

	// Create order for customer2
	order := models.Order{
		Description: "Customer2's order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer2.ID,
	}
	suite.db.Create(&order)

	// Create router with customer1's auth (trying to access customer2's order)
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/orders/:id", suite.mockAuthMiddleware(customer1.Auth0ID, "customer"), controllers.GetOrder)
	}

	// Try to get customer2's order as customer1
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/1", nil)
	router.ServeHTTP(w, req)

	// Should be forbidden
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FORBIDDEN", errorData["code"])
}

// TestListOrders_TechnicianSeesUnassigned tests that technicians see unassigned orders
func (suite *OrderIntegrationTestSuite) TestListOrders_TechnicianSeesUnassigned() {
	// Create customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Technician User",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create unassigned order
	order := models.Order{
		Description: "Unassigned order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	suite.db.Create(&order)

	// Create router with technician's auth
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/orders", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.ListOrders)
	}

	// List orders as technician
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	orders := response["data"].([]interface{})
	assert.Equal(suite.T(), 1, len(orders), "Technician should see unassigned order")

	orderData := orders[0].(map[string]interface{})
	assert.Nil(suite.T(), orderData["technician_id"])
}

// TestGetOrder_NotFound tests 404 for non-existent order
func (suite *OrderIntegrationTestSuite) TestGetOrder_NotFound() {
	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Try to get non-existent order
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/99999", nil)
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "ORDER_NOT_FOUND", errorData["code"])
}

// ITERATION 8 INTEGRATION TESTS: Order Review Workflow

// TestOrderReviewWorkflow_AcceptOrder tests the complete workflow of accepting an order
func (suite *OrderIntegrationTestSuite) TestOrderReviewWorkflow_AcceptOrder() {
	// Create customer and technician users
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Step 1: Customer creates an order
	order := models.Order{
		Description: "Order to be accepted",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Verify order is unassigned
	assert.Nil(suite.T(), order.TechnicianID)
	assert.Nil(suite.T(), order.Price)
	assert.Equal(suite.T(), "submitted", order.Status)

	// Step 2: Technician accepts the order
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/review", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.ReviewOrder)
		v1.GET("/orders/:id", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.GetOrder)
	}

	reviewBody := map[string]interface{}{
		"action": "accept",
		"price":  45.00,
	}
	reviewBodyJSON, _ := json.Marshal(reviewBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/review", bytes.NewBuffer(reviewBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var reviewResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &reviewResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), reviewResponse["success"].(bool))

	orderData := reviewResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), "accepted", orderData["status"])
	assert.Equal(suite.T(), 45.00, orderData["price"])
	assert.Equal(suite.T(), float64(technician.ID), orderData["technician_id"])
	assert.Nil(suite.T(), orderData["feedback"])

	// Step 3: Verify order was updated in database
	var updatedOrder models.Order
	suite.db.First(&updatedOrder, order.ID)
	assert.Equal(suite.T(), "accepted", updatedOrder.Status)
	assert.NotNil(suite.T(), updatedOrder.Price)
	assert.Equal(suite.T(), 45.00, *updatedOrder.Price)
	assert.Equal(suite.T(), &technician.ID, updatedOrder.TechnicianID)
	assert.Nil(suite.T(), updatedOrder.Feedback)

	// Step 4: Technician retrieves the order
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/orders/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), getResponse["success"].(bool))

	retrievedOrder := getResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), "accepted", retrievedOrder["status"])
	assert.Equal(suite.T(), 45.00, retrievedOrder["price"])
}

// TestOrderReviewWorkflow_RejectOrder tests the complete workflow of rejecting an order
func (suite *OrderIntegrationTestSuite) TestOrderReviewWorkflow_RejectOrder() {
	// Create customer and technician users
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Step 1: Customer creates an order
	order := models.Order{
		Description: "Order to be rejected",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Step 2: Technician rejects the order
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/review", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.ReviewOrder)
	}

	feedback := "Design is too complex for current materials"
	reviewBody := map[string]interface{}{
		"action":   "reject",
		"feedback": feedback,
	}
	reviewBodyJSON, _ := json.Marshal(reviewBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/review", bytes.NewBuffer(reviewBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var reviewResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &reviewResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), reviewResponse["success"].(bool))

	orderData := reviewResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), "rejected", orderData["status"])
	assert.Equal(suite.T(), feedback, orderData["feedback"])
	assert.Equal(suite.T(), float64(technician.ID), orderData["technician_id"])
	assert.Nil(suite.T(), orderData["price"])

	// Step 3: Verify order was updated in database
	var updatedOrder models.Order
	suite.db.First(&updatedOrder, order.ID)
	assert.Equal(suite.T(), "rejected", updatedOrder.Status)
	assert.NotNil(suite.T(), updatedOrder.Feedback)
	assert.Equal(suite.T(), feedback, *updatedOrder.Feedback)
	assert.Equal(suite.T(), &technician.ID, updatedOrder.TechnicianID)
	assert.Nil(suite.T(), updatedOrder.Price)
}

// TestOrderReviewWorkflow_MultipleTechnicians tests that only one technician can review an order
func (suite *OrderIntegrationTestSuite) TestOrderReviewWorkflow_MultipleTechnicians() {
	// Create customer and two technicians
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician1 := models.User{
		Auth0ID: "auth0|tech1",
		Name:    "Technician One",
		Email:   "tech1@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician1)

	technician2 := models.User{
		Auth0ID: "auth0|tech2",
		Name:    "Technician Two",
		Email:   "tech2@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician2)

	// Create order
	order := models.Order{
		Description: "Order for review",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Step 1: Technician 1 accepts the order
	router1 := gin.New()
	v1 := router1.Group("/api/v1")
	{
		v1.PUT("/orders/:id/review", suite.mockAuthMiddleware(technician1.Auth0ID, "technician"), controllers.ReviewOrder)
	}

	reviewBody := map[string]interface{}{
		"action": "accept",
		"price":  50.00,
	}
	reviewBodyJSON, _ := json.Marshal(reviewBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/review", bytes.NewBuffer(reviewBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router1.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Step 2: Technician 2 tries to review the same order (should fail)
	router2 := gin.New()
	v2 := router2.Group("/api/v1")
	{
		v2.PUT("/orders/:id/review", suite.mockAuthMiddleware(technician2.Auth0ID, "technician"), controllers.ReviewOrder)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/review", bytes.NewBuffer(reviewBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router2.ServeHTTP(w, req)

	// Should get 422 because order is already reviewed
	assert.Equal(suite.T(), http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "INVALID_STATE", errorData["code"])
	assert.Equal(suite.T(), "Order has already been reviewed", errorData["message"])

	// Verify order is still assigned to technician1
	var finalOrder models.Order
	suite.db.First(&finalOrder, order.ID)
	assert.Equal(suite.T(), &technician1.ID, finalOrder.TechnicianID)
}

// TestOrderReviewWorkflow_CustomerCannotReview tests that customers cannot review orders
func (suite *OrderIntegrationTestSuite) TestOrderReviewWorkflow_CustomerCannotReview() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create order
	order := models.Order{
		Description: "Order for review",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Try to review as customer
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/review", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReviewOrder)
	}

	reviewBody := map[string]interface{}{
		"action": "accept",
		"price":  45.00,
	}
	reviewBodyJSON, _ := json.Marshal(reviewBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/review", bytes.NewBuffer(reviewBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should be forbidden
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FORBIDDEN", errorData["code"])
	assert.Equal(suite.T(), "Only technicians can review orders", errorData["message"])
}

// TestOrderAssignWorkflow_TechnicianAssignsOrder tests the complete workflow of a technician assigning an order to themselves
func (suite *OrderIntegrationTestSuite) TestOrderAssignWorkflow_TechnicianAssignsOrder() {
	// Create a customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer123",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech123",
		Name:    "Technician User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create an unassigned order
	order := models.Order{
		Description: "Pink nails with glitter",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	suite.db.Create(&order)

	// Setup router with technician authentication
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", technician.Auth0ID)
		c.Next()
	})
	router.PUT("/api/v1/orders/:id/assign", controllers.AssignOrder)

	// Assign the order
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/orders/%d/assign", order.ID), nil)
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Pink nails with glitter", data["description"])
	assert.Equal(suite.T(), float64(2), data["quantity"])
	assert.Equal(suite.T(), float64(technician.ID), data["technician_id"])

	// Verify technician details are loaded
	technicianData := data["technician"].(map[string]interface{})
	assert.Equal(suite.T(), "Technician User", technicianData["name"])
	assert.Equal(suite.T(), "tech@example.com", technicianData["email"])

	// Verify in database
	var updatedOrder models.Order
	suite.db.First(&updatedOrder, order.ID)
	assert.NotNil(suite.T(), updatedOrder.TechnicianID)
	assert.Equal(suite.T(), technician.ID, *updatedOrder.TechnicianID)
}

// TestOrderStatusUpdateWorkflow_CompleteHappyPath tests the complete workflow of updating order status through all stages
func (suite *OrderIntegrationTestSuite) TestOrderStatusUpdateWorkflow_CompleteHappyPath() {
	// Create customer and technician users
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Step 1: Create an accepted order with price and assigned technician
	price := 45.00
	order := models.Order{
		Description:  "Complete status workflow order",
		Quantity:     2,
		Status:       "accepted",
		Price:        &price,
		CustomerID:   customer.ID,
		TechnicianID: &technician.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Setup router with technician authentication
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/status", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.UpdateOrderStatus)
		v1.GET("/orders/:id", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.GetOrder)
	}

	// Step 2: Update status from accepted to in_production
	updateBody := map[string]interface{}{
		"status": "in_production",
	}
	updateBodyJSON, _ := json.Marshal(updateBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBuffer(updateBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response1 map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response1)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response1["success"].(bool))

	orderData := response1["data"].(map[string]interface{})
	assert.Equal(suite.T(), "in_production", orderData["status"])
	assert.Equal(suite.T(), 45.00, orderData["price"])
	assert.Equal(suite.T(), float64(technician.ID), orderData["technician_id"])

	// Verify in database
	var updatedOrder1 models.Order
	suite.db.First(&updatedOrder1, order.ID)
	assert.Equal(suite.T(), "in_production", updatedOrder1.Status)

	// Step 3: Update status from in_production to shipped
	updateBody = map[string]interface{}{
		"status": "shipped",
	}
	updateBodyJSON, _ = json.Marshal(updateBody)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBuffer(updateBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response2 map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response2)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response2["success"].(bool))

	orderData = response2["data"].(map[string]interface{})
	assert.Equal(suite.T(), "shipped", orderData["status"])

	// Verify in database
	var updatedOrder2 models.Order
	suite.db.First(&updatedOrder2, order.ID)
	assert.Equal(suite.T(), "shipped", updatedOrder2.Status)

	// Step 4: Update status from shipped to delivered (terminal state)
	updateBody = map[string]interface{}{
		"status": "delivered",
	}
	updateBodyJSON, _ = json.Marshal(updateBody)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBuffer(updateBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response3 map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response3)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response3["success"].(bool))

	orderData = response3["data"].(map[string]interface{})
	assert.Equal(suite.T(), "delivered", orderData["status"])

	// Verify in database - order is now in terminal state
	var finalOrder models.Order
	suite.db.First(&finalOrder, order.ID)
	assert.Equal(suite.T(), "delivered", finalOrder.Status)
	assert.Equal(suite.T(), price, *finalOrder.Price)
	assert.Equal(suite.T(), technician.ID, *finalOrder.TechnicianID)

	// Step 5: Verify technician can retrieve the completed order
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/orders/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var getResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &getResponse)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), getResponse["success"].(bool))

	retrievedOrder := getResponse["data"].(map[string]interface{})
	assert.Equal(suite.T(), "delivered", retrievedOrder["status"])

	// Verify relationships are loaded
	customerData := retrievedOrder["customer"].(map[string]interface{})
	assert.Equal(suite.T(), "Test Customer", customerData["name"])

	techData := retrievedOrder["technician"].(map[string]interface{})
	assert.Equal(suite.T(), "Test Technician", techData["name"])
}

// TestOrderStatusUpdateWorkflow_InvalidTransition tests that invalid status transitions are rejected
func (suite *OrderIntegrationTestSuite) TestOrderStatusUpdateWorkflow_InvalidTransition() {
	// Create customer and technician users
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create an accepted order
	price := 45.00
	order := models.Order{
		Description:  "Order for invalid transition test",
		Quantity:     2,
		Status:       "accepted",
		Price:        &price,
		CustomerID:   customer.ID,
		TechnicianID: &technician.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Setup router
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/status", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.UpdateOrderStatus)
	}

	// Try to skip from accepted to shipped (should fail)
	updateBody := map[string]interface{}{
		"status": "shipped",
	}
	updateBodyJSON, _ := json.Marshal(updateBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBuffer(updateBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should get 422 Unprocessable Entity
	assert.Equal(suite.T(), http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "INVALID_TRANSITION", errorData["code"])
	assert.Equal(suite.T(), "Invalid status transition", errorData["message"])

	// Verify error details include helpful information
	details := errorData["details"].(map[string]interface{})
	assert.Equal(suite.T(), "accepted", details["current_status"])
	assert.Equal(suite.T(), "shipped", details["requested_status"])

	allowedStatuses := details["allowed_statuses"].([]interface{})
	assert.Equal(suite.T(), 1, len(allowedStatuses))
	assert.Equal(suite.T(), "in_production", allowedStatuses[0])

	// Verify database was NOT updated
	var unchangedOrder models.Order
	suite.db.First(&unchangedOrder, order.ID)
	assert.Equal(suite.T(), "accepted", unchangedOrder.Status)
}

// TestOrderStatusUpdateWorkflow_CustomerCannotUpdate tests that customers cannot update order status
func (suite *OrderIntegrationTestSuite) TestOrderStatusUpdateWorkflow_CustomerCannotUpdate() {
	// Create customer and technician users
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create an accepted order
	price := 45.00
	order := models.Order{
		Description:  "Order for customer authorization test",
		Quantity:     2,
		Status:       "accepted",
		Price:        &price,
		CustomerID:   customer.ID,
		TechnicianID: &technician.ID,
	}
	err := suite.db.Create(&order).Error
	suite.NoError(err)

	// Setup router with customer authentication
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.PUT("/orders/:id/status", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.UpdateOrderStatus)
	}

	// Try to update status as customer (should fail)
	updateBody := map[string]interface{}{
		"status": "in_production",
	}
	updateBodyJSON, _ := json.Marshal(updateBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/1/status", bytes.NewBuffer(updateBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should be forbidden
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FORBIDDEN", errorData["code"])
	assert.Equal(suite.T(), "Only technicians can update order status", errorData["message"])

	// Verify database was NOT updated
	var unchangedOrder models.Order
	suite.db.First(&unchangedOrder, order.ID)
	assert.Equal(suite.T(), "accepted", unchangedOrder.Status)
}

// ITERATION 16 INTEGRATION TESTS: Reorder Functionality

// TestReorderWorkflow_SuccessfulReorder tests the complete happy path of reordering a delivered order
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_SuccessfulReorder() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create technician
	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create a completed (delivered) order with image
	price := 45.00
	imageKey := "orders/test-image.png"
	originalOrder := models.Order{
		Description:  "Pink nails with glitter",
		Quantity:     2,
		Status:       "delivered",
		Price:        &price,
		ImageS3Key:   &imageKey,
		CustomerID:   customer.ID,
		TechnicianID: &technician.ID,
	}
	suite.db.Create(&originalOrder)

	// Setup router with customer authentication
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Reorder with different quantity
	reorderBody := map[string]interface{}{
		"quantity": 5,
	}
	reorderBodyJSON, _ := json.Marshal(reorderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	// Verify response data
	orderData := response["data"].(map[string]interface{})
	assert.Equal(suite.T(), "Pink nails with glitter", orderData["description"])
	assert.Equal(suite.T(), float64(5), orderData["quantity"])
	assert.Equal(suite.T(), "submitted", orderData["status"])
	assert.Nil(suite.T(), orderData["price"])
	assert.Nil(suite.T(), orderData["feedback"])
	assert.Nil(suite.T(), orderData["technician_id"])
	assert.Equal(suite.T(), imageKey, orderData["image_s3_key"])
	assert.Equal(suite.T(), float64(originalOrder.ID), orderData["original_order_id"])
	assert.Equal(suite.T(), float64(customer.ID), orderData["customer_id"])

	// Verify in database
	var newOrder models.Order
	suite.db.Where("original_order_id = ?", originalOrder.ID).First(&newOrder)
	assert.Equal(suite.T(), "Pink nails with glitter", newOrder.Description)
	assert.Equal(suite.T(), 5, newOrder.Quantity)
	assert.Equal(suite.T(), "submitted", newOrder.Status)
	assert.Nil(suite.T(), newOrder.Price)
	assert.Nil(suite.T(), newOrder.TechnicianID)
	assert.Equal(suite.T(), &imageKey, newOrder.ImageS3Key)
	assert.Equal(suite.T(), &originalOrder.ID, newOrder.OriginalOrderID)
	assert.Equal(suite.T(), customer.ID, newOrder.CustomerID)

	// Verify original order is unchanged
	var unchangedOriginal models.Order
	suite.db.First(&unchangedOriginal, originalOrder.ID)
	assert.Equal(suite.T(), "delivered", unchangedOriginal.Status)
	assert.Equal(suite.T(), 2, unchangedOriginal.Quantity)
}

// TestReorderWorkflow_OrderNotDelivered tests that only delivered orders can be reordered
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_OrderNotDelivered() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Test each non-delivered status
	statuses := []string{"submitted", "accepted", "rejected", "in_production", "shipped"}

	for _, status := range statuses {
		// Create order with the given status
		order := models.Order{
			Description: "Order in " + status + " state",
			Quantity:    2,
			Status:      status,
			CustomerID:  customer.ID,
		}
		suite.db.Create(&order)

		// Setup router
		router := gin.New()
		v1 := router.Group("/api/v1")
		{
			v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
		}

		// Try to reorder
		reorderBody := map[string]interface{}{
			"quantity": 3,
		}
		reorderBodyJSON, _ := json.Marshal(reorderBody)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/orders/%d/reorder", order.ID), bytes.NewBuffer(reorderBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Should get 422 Unprocessable Entity
		assert.Equal(suite.T(), http.StatusUnprocessableEntity, w.Code, "Status %s should not be reorderable", status)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))

		errorData := response["error"].(map[string]interface{})
		assert.Equal(suite.T(), "INVALID_ORDER_STATE", errorData["code"])
		assert.Equal(suite.T(), "Only completed (delivered) orders can be reordered", errorData["message"])
	}
}

// TestReorderWorkflow_CustomerCannotReorderOthersOrders tests that customers can only reorder their own orders
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_CustomerCannotReorderOthersOrders() {
	// Create two customers
	customer1 := models.User{
		Auth0ID: "auth0|customer1",
		Name:    "Customer One",
		Email:   "customer1@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer1)

	customer2 := models.User{
		Auth0ID: "auth0|customer2",
		Name:    "Customer Two",
		Email:   "customer2@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer2)

	// Create delivered order for customer1
	order := models.Order{
		Description: "Customer1's delivered order",
		Quantity:    2,
		Status:      "delivered",
		CustomerID:  customer1.ID,
	}
	suite.db.Create(&order)

	// Setup router with customer2's authentication
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer2.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Customer2 tries to reorder customer1's order
	reorderBody := map[string]interface{}{
		"quantity": 3,
	}
	reorderBodyJSON, _ := json.Marshal(reorderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should be forbidden
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FORBIDDEN", errorData["code"])
	assert.Equal(suite.T(), "You can only reorder your own orders", errorData["message"])
}

// TestReorderWorkflow_TechnicianCannotReorder tests that technicians cannot reorder orders
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_TechnicianCannotReorder() {
	// Create customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Test Technician",
		Email:   "tech@test.com",
		Role:    "technician",
	}
	suite.db.Create(&technician)

	// Create delivered order
	order := models.Order{
		Description: "Delivered order",
		Quantity:    2,
		Status:      "delivered",
		CustomerID:  customer.ID,
	}
	suite.db.Create(&order)

	// Setup router with technician authentication
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(technician.Auth0ID, "technician"), controllers.ReorderOrder)
	}

	// Technician tries to reorder
	reorderBody := map[string]interface{}{
		"quantity": 3,
	}
	reorderBodyJSON, _ := json.Marshal(reorderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should be forbidden
	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "FORBIDDEN", errorData["code"])
	assert.Equal(suite.T(), "Only customers can reorder", errorData["message"])
}

// TestReorderWorkflow_InvalidQuantity tests validation of quantity field
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_InvalidQuantity() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create delivered order
	order := models.Order{
		Description: "Delivered order",
		Quantity:    2,
		Status:      "delivered",
		CustomerID:  customer.ID,
	}
	suite.db.Create(&order)

	// Setup router
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Test invalid quantities
	invalidQuantities := []interface{}{0, -1, "invalid"}

	for _, qty := range invalidQuantities {
		reorderBody := map[string]interface{}{
			"quantity": qty,
		}
		reorderBodyJSON, _ := json.Marshal(reorderBody)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		// Should get validation error
		assert.Equal(suite.T(), http.StatusBadRequest, w.Code, "Quantity %v should be invalid", qty)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.False(suite.T(), response["success"].(bool))

		errorData := response["error"].(map[string]interface{})
		assert.Equal(suite.T(), "VALIDATION_ERROR", errorData["code"])
	}
}

// TestReorderWorkflow_MissingQuantity tests that quantity is required
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_MissingQuantity() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create delivered order
	order := models.Order{
		Description: "Delivered order",
		Quantity:    2,
		Status:      "delivered",
		CustomerID:  customer.ID,
	}
	suite.db.Create(&order)

	// Setup router
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Try to reorder without quantity
	reorderBody := map[string]interface{}{}
	reorderBodyJSON, _ := json.Marshal(reorderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should get validation error
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "VALIDATION_ERROR", errorData["code"])
}

// TestReorderWorkflow_OrderNotFound tests reordering a non-existent order
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_OrderNotFound() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Setup router
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Try to reorder non-existent order
	reorderBody := map[string]interface{}{
		"quantity": 3,
	}
	reorderBodyJSON, _ := json.Marshal(reorderBody)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/99999/reorder", bytes.NewBuffer(reorderBodyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should get 404
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(suite.T(), "ORDER_NOT_FOUND", errorData["code"])
	assert.Equal(suite.T(), "Order not found", errorData["message"])
}

// TestReorderWorkflow_MultipleReorders tests that an order can be reordered multiple times
func (suite *OrderIntegrationTestSuite) TestReorderWorkflow_MultipleReorders() {
	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Test Customer",
		Email:   "customer@test.com",
		Role:    "customer",
	}
	suite.db.Create(&customer)

	// Create delivered order
	imageKey := "orders/test.png"
	originalOrder := models.Order{
		Description: "Original delivered order",
		Quantity:    2,
		Status:      "delivered",
		ImageS3Key:  &imageKey,
		CustomerID:  customer.ID,
	}
	suite.db.Create(&originalOrder)

	// Setup router
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders/:id/reorder", suite.mockAuthMiddleware(customer.Auth0ID, "customer"), controllers.ReorderOrder)
	}

	// Reorder the same order 3 times
	for i := 1; i <= 3; i++ {
		reorderBody := map[string]interface{}{
			"quantity": i + 2,
		}
		reorderBodyJSON, _ := json.Marshal(reorderBody)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/1/reorder", bytes.NewBuffer(reorderBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.True(suite.T(), response["success"].(bool))
	}

	// Verify 3 reorders were created
	var reorders []models.Order
	suite.db.Where("original_order_id = ?", originalOrder.ID).Find(&reorders)
	assert.Equal(suite.T(), 3, len(reorders))

	// Verify all reorders have correct properties
	for i, reorder := range reorders {
		assert.Equal(suite.T(), "Original delivered order", reorder.Description)
		assert.Equal(suite.T(), i+3, reorder.Quantity)
		assert.Equal(suite.T(), "submitted", reorder.Status)
		assert.Equal(suite.T(), &imageKey, reorder.ImageS3Key)
		assert.Equal(suite.T(), &originalOrder.ID, reorder.OriginalOrderID)
	}
}

// TestOrderIntegrationSuite runs the test suite
func TestOrderIntegrationSuite(t *testing.T) {
	suite.Run(t, new(OrderIntegrationTestSuite))
}
