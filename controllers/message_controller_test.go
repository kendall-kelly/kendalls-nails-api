package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMessageTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(&models.User{}, &models.Order{}, &models.Message{}); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

func TestSendMessage(t *testing.T) {
	// Setup
	db := setupMessageTestDB(t)
	config.SetDB(db)

	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer123",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create technician
	technician := models.User{
		Auth0ID: "auth0|tech123",
		Name:    "Tech User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	db.Create(&technician)

	// Create another customer for testing unauthorized access
	otherCustomer := models.User{
		Auth0ID: "auth0|othercustomer",
		Name:    "Other Customer",
		Email:   "other@example.com",
		Role:    "customer",
	}
	db.Create(&otherCustomer)

	// Create another technician for testing unauthorized access
	otherTech := models.User{
		Auth0ID: "auth0|othertech",
		Name:    "Other Tech",
		Email:   "othertech@example.com",
		Role:    "technician",
	}
	db.Create(&otherTech)

	// Create order assigned to first technician
	techID := technician.ID
	order := models.Order{
		Description:  "Test order",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &techID,
	}
	db.Create(&order)

	// Create unassigned order
	unassignedOrder := models.Order{
		Description:  "Unassigned order",
		Quantity:     1,
		Status:       "submitted",
		CustomerID:   customer.ID,
		TechnicianID: nil,
	}
	db.Create(&unassignedOrder)

	tests := []struct {
		name           string
		auth0ID        string
		role           string
		orderID        string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:    "Customer sends message on their own order",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			orderID: "1",
			requestBody: map[string]interface{}{
				"text": "Can you make the glitter more subtle?",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(t, "Can you make the glitter more subtle?", data["text"])
				assert.Equal(t, float64(1), data["order_id"])
				assert.Equal(t, float64(customer.ID), data["sender_id"])

				// Verify sender relationship is loaded
				sender := data["sender"].(map[string]interface{})
				assert.Equal(t, customer.Email, sender["email"])
				assert.Equal(t, customer.Name, sender["name"])
			},
		},
		{
			name:    "Technician sends message on assigned order",
			auth0ID: technician.Auth0ID,
			role:    "technician",
			orderID: "1",
			requestBody: map[string]interface{}{
				"text": "Sure, I can reduce the glitter amount.",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].(map[string]interface{})
				assert.Equal(t, "Sure, I can reduce the glitter amount.", data["text"])
				assert.Equal(t, float64(technician.ID), data["sender_id"])
			},
		},
		{
			name:    "Customer cannot message on another customer's order",
			auth0ID: otherCustomer.Auth0ID,
			role:    "customer",
			orderID: "1",
			requestBody: map[string]interface{}{
				"text": "This should fail",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:    "Technician cannot message on unassigned order",
			auth0ID: technician.Auth0ID,
			role:    "technician",
			orderID: "2",
			requestBody: map[string]interface{}{
				"text": "This should fail",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:    "Technician cannot message on order assigned to another technician",
			auth0ID: otherTech.Auth0ID,
			role:    "technician",
			orderID: "1",
			requestBody: map[string]interface{}{
				"text": "This should fail",
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:           "Fail with missing text",
			auth0ID:        customer.Auth0ID,
			role:           "customer",
			orderID:        "1",
			requestBody:    map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:    "Fail with invalid order ID",
			auth0ID: customer.Auth0ID,
			role:    "customer",
			orderID: "999",
			requestBody: map[string]interface{}{
				"text": "This should fail",
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "ORDER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			router := setupTestRouter()
			router.POST("/orders/:id/messages",
				mockAuthMiddleware(tt.auth0ID, tt.role, "mock-token"),
				SendMessage,
			)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orders/%s/messages", tt.orderID), bytes.NewBuffer(body))
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

			if tt.expectedError != "" {
				// Check error response
				assert.False(t, response["success"].(bool))
				errorData := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorData["code"])
			} else if tt.checkResponse != nil {
				// Check success response
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestListMessages(t *testing.T) {
	// Setup
	db := setupMessageTestDB(t)
	config.SetDB(db)

	// Create customer
	customer := models.User{
		Auth0ID: "auth0|customer123",
		Name:    "Customer User",
		Email:   "customer@example.com",
		Role:    "customer",
	}
	db.Create(&customer)

	// Create technician
	technician := models.User{
		Auth0ID: "auth0|tech123",
		Name:    "Tech User",
		Email:   "tech@example.com",
		Role:    "technician",
	}
	db.Create(&technician)

	// Create another customer for testing unauthorized access
	otherCustomer := models.User{
		Auth0ID: "auth0|othercustomer",
		Name:    "Other Customer",
		Email:   "other@example.com",
		Role:    "customer",
	}
	db.Create(&otherCustomer)

	// Create another technician for testing unauthorized access
	otherTech := models.User{
		Auth0ID: "auth0|othertech",
		Name:    "Other Tech",
		Email:   "othertech@example.com",
		Role:    "technician",
	}
	db.Create(&otherTech)

	// Create order assigned to first technician
	techID := technician.ID
	order := models.Order{
		Description:  "Test order",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &techID,
	}
	db.Create(&order)

	// Create messages for the order
	msg1 := models.Message{
		OrderID:  order.ID,
		SenderID: customer.ID,
		Text:     "First message from customer",
	}
	db.Create(&msg1)

	msg2 := models.Message{
		OrderID:  order.ID,
		SenderID: technician.ID,
		Text:     "Reply from technician",
	}
	db.Create(&msg2)

	msg3 := models.Message{
		OrderID:  order.ID,
		SenderID: customer.ID,
		Text:     "Second message from customer",
	}
	db.Create(&msg3)

	// Create order with no messages
	emptyOrder := models.Order{
		Description:  "Order with no messages",
		Quantity:     1,
		Status:       "accepted",
		CustomerID:   customer.ID,
		TechnicianID: &techID,
	}
	db.Create(&emptyOrder)

	tests := []struct {
		name           string
		auth0ID        string
		role           string
		orderID        string
		expectedStatus int
		expectedError  string
		checkResponse  func(t *testing.T, response map[string]interface{})
	}{
		{
			name:           "Customer lists messages on their own order",
			auth0ID:        customer.Auth0ID,
			role:           "customer",
			orderID:        "1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].([]interface{})
				assert.Len(t, data, 3)

				// Check messages are in chronological order
				msg1 := data[0].(map[string]interface{})
				assert.Equal(t, "First message from customer", msg1["text"])
				assert.Equal(t, float64(customer.ID), msg1["sender_id"])

				msg2 := data[1].(map[string]interface{})
				assert.Equal(t, "Reply from technician", msg2["text"])
				assert.Equal(t, float64(technician.ID), msg2["sender_id"])

				msg3 := data[2].(map[string]interface{})
				assert.Equal(t, "Second message from customer", msg3["text"])
				assert.Equal(t, float64(customer.ID), msg3["sender_id"])

				// Verify sender relationships are loaded
				sender1 := msg1["sender"].(map[string]interface{})
				assert.Equal(t, customer.Email, sender1["email"])

				sender2 := msg2["sender"].(map[string]interface{})
				assert.Equal(t, technician.Email, sender2["email"])
			},
		},
		{
			name:           "Technician lists messages on assigned order",
			auth0ID:        technician.Auth0ID,
			role:           "technician",
			orderID:        "1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].([]interface{})
				assert.Len(t, data, 3)
			},
		},
		{
			name:           "Returns empty array when no messages exist",
			auth0ID:        customer.Auth0ID,
			role:           "customer",
			orderID:        "2",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response map[string]interface{}) {
				assert.True(t, response["success"].(bool))
				data := response["data"].([]interface{})
				assert.Len(t, data, 0)
			},
		},
		{
			name:           "Customer cannot list messages on another customer's order",
			auth0ID:        otherCustomer.Auth0ID,
			role:           "customer",
			orderID:        "1",
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:           "Technician cannot list messages on order assigned to another technician",
			auth0ID:        otherTech.Auth0ID,
			role:           "technician",
			orderID:        "1",
			expectedStatus: http.StatusForbidden,
			expectedError:  "FORBIDDEN",
		},
		{
			name:           "Fail with invalid order ID",
			auth0ID:        customer.Auth0ID,
			role:           "customer",
			orderID:        "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "ORDER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router
			router := setupTestRouter()
			router.GET("/orders/:id/messages",
				mockAuthMiddleware(tt.auth0ID, tt.role, "mock-token"),
				ListMessages,
			)

			// Create request
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orders/%s/messages", tt.orderID), nil)

			// Execute request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedError != "" {
				// Check error response
				assert.False(t, response["success"].(bool))
				errorData := response["error"].(map[string]interface{})
				assert.Equal(t, tt.expectedError, errorData["code"])
			} else if tt.checkResponse != nil {
				// Check success response
				tt.checkResponse(t, response)
			}
		})
	}
}
