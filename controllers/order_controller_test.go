package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOrderTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate the User and Order models
	if err := db.AutoMigrate(&models.User{}, &models.Order{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestCreateOrder(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer123",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create a technician user for testing RBAC
	technician := models.User{
		Auth0ID: "auth0|tech123",
		Name:    "Tech User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	db.Create(&technician)

	// Test cases
	tests := []struct {
		name           string
		auth0ID        string
		role           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:    "Successfully create order as customer",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
				"quantity":    2,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(t, "Pink nails with glitter", data["description"])
				assert.Equal(t, float64(2), data["quantity"])
				assert.Equal(t, "submitted", data["status"])
				assert.Equal(t, float64(customer.ID), data["customer_id"])
				assert.Nil(t, data["price"])
				assert.Nil(t, data["technician_id"])

				// Verify customer relationship is loaded
				customerData := data["customer"].(map[string]interface{})
				assert.Equal(t, customer.Email, customerData["email"])
			},
		},
		{
			name:    "Fail to create order as technician",
			auth0ID: technician.Auth0ID,
			role:    "technician",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
				"quantity":    2,
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:    "Fail with missing description",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			requestBody: map[string]interface{}{
				"quantity": 2,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:    "Fail with missing quantity",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:    "Fail with zero quantity",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
				"quantity":    0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:    "Fail with negative quantity",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
				"quantity":    -1,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:    "Fail with user not found",
			auth0ID: "auth0|nonexistent",
			role:    "customer",
			requestBody: map[string]interface{}{
				"description": "Pink nails with glitter",
				"quantity":    2,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "USER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			router := setupTestRouter()
			router.POST("/orders",
				mockAuthMiddleware(tt.auth0ID, tt.role, "mock-token"),
				CreateOrder,
			)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check for expected error
			if tt.expectedError != "" {
				assert.False(t, response["success"].(bool))
				errorData := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorData["code"])
			}

			// Run custom response checks if provided
			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestCreateOrder_MultipleOrders(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer user
	customer := models.User{
		Auth0ID: "auth0|customer123",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Setup router
	router := setupTestRouter()
	router.POST("/orders",
		mockAuthMiddleware(customer.Auth0ID, "customer", "mock-token"),
		CreateOrder,
	)

	// Create multiple orders
	orders := []map[string]interface{}{
		{"description": "Pink nails with glitter", "quantity": 2},
		{"description": "Blue nails with stripes", "quantity": 1},
		{"description": "Rainbow nails", "quantity": 3},
	}

	for i, orderData := range orders {
		body, _ := json.Marshal(orderData)
		req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, orderData["description"], data["description"])
		assert.Equal(t, float64(i+1), data["id"]) // IDs should increment
	}

	// Verify all orders are in database
	var count int64
	db.Model(&models.Order{}).Count(&count)
	assert.Equal(t, int64(3), count)
}

// ITERATION 7 TESTS: ListOrders and GetOrder

func TestListOrders_AsCustomer(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create two customers
	customer1 := models.User{
		Auth0ID: "auth0|customer1",
		Name:    "Customer One",
		Email:   "customer1@example.com",
		Role:    "customer",
	}
	db.Create(&customer1)

	customer2 := models.User{
		Auth0ID: "auth0|customer2",
		Name:    "Customer Two",
		Email:   "customer2@example.com",
		Role:    "customer",
	}
	db.Create(&customer2)

	// Create orders for customer1
	order1 := models.Order{
		Description: "Order 1 for customer1",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer1.ID,
	}
	db.Create(&order1)

	order2 := models.Order{
		Description: "Order 2 for customer1",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer1.ID,
	}
	db.Create(&order2)

	// Create order for customer2
	order3 := models.Order{
		Description: "Order for customer2",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer2.ID,
	}
	db.Create(&order3)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders",
		mockAuthMiddleware(customer1.Auth0ID, "customer", "mock-token"),
		ListOrders,
	)

	// Make request as customer1
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	// Check data
	data := response["data"].([]interface{})
	assert.Equal(t, 2, len(data), "Customer should only see their own 2 orders")

	// Verify all returned orders belong to customer1
	for _, orderInterface := range data {
		order := orderInterface.(map[string]interface{})
		assert.Equal(t, float64(customer1.ID), order["customer_id"])
	}

	// Check pagination
	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(10), pagination["limit"])
	assert.Equal(t, float64(2), pagination["total"])
	assert.Equal(t, float64(1), pagination["totalPages"])
}

func TestListOrders_AsTechnician(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	technician1 := models.User{
		Auth0ID: "auth0|tech1",
		Name:    "Technician One",
		Email:   "tech1@example.com",
		Role:    "technician",
	}
	db.Create(&technician1)

	technician2 := models.User{
		Auth0ID: "auth0|tech2",
		Name:    "Technician Two",
		Email:   "tech2@example.com",
		Role:    "technician",
	}
	db.Create(&technician2)

	// Create unassigned order
	unassignedOrder := models.Order{
		Description: "Unassigned order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	db.Create(&unassignedOrder)

	// Create order assigned to technician1
	assignedToTech1 := models.Order{
		Description:  "Assigned to tech1",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &technician1.ID,
	}
	db.Create(&assignedToTech1)

	// Create order assigned to technician2
	assignedToTech2 := models.Order{
		Description:  "Assigned to tech2",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &technician2.ID,
	}
	db.Create(&assignedToTech2)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders",
		mockAuthMiddleware(technician1.Auth0ID, "technician", "mock-token"),
		ListOrders,
	)

	// Make request as technician1
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	// Check data - should see unassigned order + order assigned to tech1
	data := response["data"].([]interface{})
	assert.Equal(t, 2, len(data), "Technician should see unassigned orders and their assigned orders")

	// Verify orders are correct
	foundUnassigned := false
	foundAssignedToTech1 := false
	foundAssignedToTech2 := false

	for _, orderInterface := range data {
		order := orderInterface.(map[string]interface{})
		techID := order["technician_id"]

		if techID == nil {
			foundUnassigned = true
		} else if float64(technician1.ID) == techID.(float64) {
			foundAssignedToTech1 = true
		} else if float64(technician2.ID) == techID.(float64) {
			foundAssignedToTech2 = true
		}
	}

	assert.True(t, foundUnassigned, "Should see unassigned order")
	assert.True(t, foundAssignedToTech1, "Should see order assigned to self")
	assert.False(t, foundAssignedToTech2, "Should NOT see order assigned to other technician")
}

func TestListOrders_Pagination(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create 5 orders
	for i := 1; i <= 5; i++ {
		order := models.Order{
			Description: "Order " + string(rune(i)),
			Quantity:    i,
			Status:      "submitted",
			CustomerID:  customer.ID,
		}
		db.Create(&order)
	}

	tests := []struct {
		name              string
		queryParams       string
		expectedPage      float64
		expectedLimit     float64
		expectedDataCount int
		expectedTotal     float64
		expectedPages     float64
	}{
		{
			name:              "Default pagination",
			queryParams:       "",
			expectedPage:      1,
			expectedLimit:     10,
			expectedDataCount: 5,
			expectedTotal:     5,
			expectedPages:     1,
		},
		{
			name:              "Page 1 with limit 2",
			queryParams:       "?page=1&limit=2",
			expectedPage:      1,
			expectedLimit:     2,
			expectedDataCount: 2,
			expectedTotal:     5,
			expectedPages:     3,
		},
		{
			name:              "Page 2 with limit 2",
			queryParams:       "?page=2&limit=2",
			expectedPage:      2,
			expectedLimit:     2,
			expectedDataCount: 2,
			expectedTotal:     5,
			expectedPages:     3,
		},
		{
			name:              "Page 3 with limit 2",
			queryParams:       "?page=3&limit=2",
			expectedPage:      3,
			expectedLimit:     2,
			expectedDataCount: 1,
			expectedTotal:     5,
			expectedPages:     3,
		},
		{
			name:              "Custom limit of 3",
			queryParams:       "?limit=3",
			expectedPage:      1,
			expectedLimit:     3,
			expectedDataCount: 3,
			expectedTotal:     5,
			expectedPages:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			router := setupTestRouter()
			router.GET("/orders",
				mockAuthMiddleware(customer.Auth0ID, "customer", "mock-token"),
				ListOrders,
			)

			// Make request
			req, _ := http.NewRequest(http.MethodGet, "/orders"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check pagination
			pagination := response["pagination"].(map[string]interface{})
			assert.Equal(t, tt.expectedPage, pagination["page"])
			assert.Equal(t, tt.expectedLimit, pagination["limit"])
			assert.Equal(t, tt.expectedTotal, pagination["total"])
			assert.Equal(t, tt.expectedPages, pagination["totalPages"])

			// Check data count
			data := response["data"].([]interface{})
			assert.Equal(t, tt.expectedDataCount, len(data))
		})
	}
}

func TestListOrders_Sorting(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create orders (they'll be created with incrementing created_at)
	order1 := models.Order{Description: "First order", Quantity: 1, Status: "submitted", CustomerID: customer.ID}
	db.Create(&order1)

	order2 := models.Order{Description: "Second order", Quantity: 1, Status: "submitted", CustomerID: customer.ID}
	db.Create(&order2)

	order3 := models.Order{Description: "Third order", Quantity: 1, Status: "submitted", CustomerID: customer.ID}
	db.Create(&order3)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders",
		mockAuthMiddleware(customer.Auth0ID, "customer", "mock-token"),
		ListOrders,
	)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check that orders are sorted DESC by created_at (most recent first)
	data := response["data"].([]interface{})
	assert.Equal(t, 3, len(data))

	// The most recently created order should be first
	firstOrder := data[0].(map[string]interface{})
	assert.Equal(t, "Third order", firstOrder["description"])

	lastOrder := data[2].(map[string]interface{})
	assert.Equal(t, "First order", lastOrder["description"])
}

func TestListOrders_WithoutAuth(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Setup router without auth middleware
	router := setupTestRouter()
	router.GET("/orders", ListOrders)

	// Make request without auth
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - should fail to get user_id from context
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

func TestGetOrder_AsCustomer_OwnOrder(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create an order
	order := models.Order{
		Description: "Test order",
		Quantity:    2,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	db.Create(&order)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(customer.Auth0ID, "customer", "mock-token"),
		GetOrder,
	)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	// Check data
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(order.ID), data["id"])
	assert.Equal(t, order.Description, data["description"])
	assert.Equal(t, float64(order.Quantity), data["quantity"])
	assert.Equal(t, order.Status, data["status"])
	assert.Equal(t, float64(customer.ID), data["customer_id"])

	// Verify customer relationship is preloaded
	customerData := data["customer"].(map[string]interface{})
	assert.Equal(t, customer.Email, customerData["email"])
}

func TestGetOrder_AsCustomer_OtherCustomerOrder(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create two customers
	customer1 := models.User{
		Auth0ID: "auth0|customer1",
		Name:    "Customer One",
		Email:   "customer1@example.com",
		Role:    "customer",
	}
	db.Create(&customer1)

	customer2 := models.User{
		Auth0ID: "auth0|customer2",
		Name:    "Customer Two",
		Email:   "customer2@example.com",
		Role:    "customer",
	}
	db.Create(&customer2)

	// Create order for customer2
	order := models.Order{
		Description: "Customer2's order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer2.ID,
	}
	db.Create(&order)

	// Setup router with customer1's auth
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(customer1.Auth0ID, "customer", "mock-token"),
		GetOrder,
	)

	// Make request as customer1 trying to access customer2's order
	req, _ := http.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - should be forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errorData["code"])
	assert.Equal(t, "You do not have permission to access this order", errorData["message"])
}

func TestGetOrder_AsTechnician_UnassignedOrder(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Technician User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	db.Create(&technician)

	// Create unassigned order
	order := models.Order{
		Description: "Unassigned order",
		Quantity:    1,
		Status:      "submitted",
		CustomerID:  customer.ID,
	}
	db.Create(&order)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(technician.Auth0ID, "technician", "mock-token"),
		GetOrder,
	)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - technician should be able to access unassigned order
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(order.ID), data["id"])
	assert.Nil(t, data["technician_id"])
}

func TestGetOrder_AsTechnician_AssignedToSelf(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create customer and technician
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	technician := models.User{
		Auth0ID: "auth0|tech",
		Name:    "Technician User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	db.Create(&technician)

	// Create order assigned to technician
	order := models.Order{
		Description:  "Assigned order",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &technician.ID,
	}
	db.Create(&order)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(technician.Auth0ID, "technician", "mock-token"),
		GetOrder,
	)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(technician.ID), data["technician_id"])

	// Verify technician relationship is preloaded
	techData := data["technician"].(map[string]interface{})
	assert.Equal(t, technician.Email, techData["email"])
}

func TestGetOrder_AsTechnician_AssignedToOther(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create customer and two technicians
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	technician1 := models.User{
		Auth0ID: "auth0|tech1",
		Name:    "Technician One",
		Email:   "tech1@example.com",
		Role:    "technician",
	}
	db.Create(&technician1)

	technician2 := models.User{
		Auth0ID: "auth0|tech2",
		Name:    "Technician Two",
		Email:   "tech2@example.com",
		Role:    "technician",
	}
	db.Create(&technician2)

	// Create order assigned to technician2
	order := models.Order{
		Description:  "Assigned to tech2",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &technician2.ID,
	}
	db.Create(&order)

	// Setup router with technician1's auth
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(technician1.Auth0ID, "technician", "mock-token"),
		GetOrder,
	)

	// Make request as technician1 trying to access technician2's order
	req, _ := http.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert - should be forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "FORBIDDEN", errorData["code"])
}

func TestGetOrder_NotFound(t *testing.T) {
	// Setup
	db := setupOrderTestDB(t)
	config.SetDB(db)

	// Create a customer
	customer := models.User{
		Auth0ID: "auth0|customer",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Setup router
	router := setupTestRouter()
	router.GET("/orders/:id",
		mockAuthMiddleware(customer.Auth0ID, "customer", "mock-token"),
		GetOrder,
	)

	// Make request for non-existent order
	req, _ := http.NewRequest(http.MethodGet, "/orders/99999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))

	errorData := response["error"].(map[string]interface{})
	assert.Equal(t, "ORDER_NOT_FOUND", errorData["code"])
	assert.Equal(t, "Order not found", errorData["message"])
}
