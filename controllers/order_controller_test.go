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
