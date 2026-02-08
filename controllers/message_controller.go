package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
)

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	Text string `json:"text" binding:"required"`
}

// SendMessage handles POST /api/v1/orders/:id/messages - sends a message on an order
func SendMessage(c *gin.Context) {
	// Extract Auth0 user ID from JWT token
	auth0ID, err := middleware.GetUserID(c)
	if err != nil {
		c.PureJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Could not extract user information",
			},
		})
		return
	}

	// Find the user in the database
	db := config.GetDB()
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		c.PureJSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User profile not found. Please create a profile first.",
			},
		})
		return
	}

	// Get order ID from URL parameter
	orderID := c.Param("id")
	if orderID == "" {
		c.PureJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Order ID is required",
			},
		})
		return
	}

	// Fetch the order
	var order models.Order
	if err := db.First(&order, orderID).Error; err != nil {
		c.PureJSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ORDER_NOT_FOUND",
				"message": "Order not found",
			},
		})
		return
	}

	// Authorization check: Can user message on this order?
	// Customers can only message on their own orders
	// Technicians can only message on orders assigned to them
	canMessage := false
	switch user.Role {
	case "customer":
		canMessage = order.CustomerID == user.ID
	case "technician":
		canMessage = order.TechnicianID != nil && *order.TechnicianID == user.ID
	}

	if !canMessage {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You do not have permission to message on this order",
			},
		})
		return
	}

	// Parse request body
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.PureJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Create the message
	message := models.Message{
		OrderID:  order.ID,
		SenderID: user.ID,
		Text:     req.Text,
	}

	if err := db.Create(&message).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to create message",
			},
		})
		return
	}

	// Load the sender relationship to return complete data
	if err := db.Preload("Sender").First(&message, message.ID).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load message details",
			},
		})
		return
	}

	c.PureJSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    message,
	})
}

// ListMessages handles GET /api/v1/orders/:id/messages - lists messages for an order
func ListMessages(c *gin.Context) {
	// Extract Auth0 user ID from JWT token
	auth0ID, err := middleware.GetUserID(c)
	if err != nil {
		c.PureJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Could not extract user information",
			},
		})
		return
	}

	// Find the user in the database
	db := config.GetDB()
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		c.PureJSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User profile not found. Please create a profile first.",
			},
		})
		return
	}

	// Get order ID from URL parameter
	orderID := c.Param("id")
	if orderID == "" {
		c.PureJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Order ID is required",
			},
		})
		return
	}

	// Fetch the order
	var order models.Order
	if err := db.First(&order, orderID).Error; err != nil {
		c.PureJSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ORDER_NOT_FOUND",
				"message": "Order not found",
			},
		})
		return
	}

	// Authorization check: Can user view messages on this order?
	// Customers can view messages on their own orders
	// Technicians can view messages on orders assigned to them
	canView := false
	switch user.Role {
	case "customer":
		canView = order.CustomerID == user.ID
	case "technician":
		canView = order.TechnicianID != nil && *order.TechnicianID == user.ID
	}

	if !canView {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You do not have permission to view messages on this order",
			},
		})
		return
	}

	// Fetch messages for this order
	var messages []models.Message
	if err := db.Where("order_id = ?", order.ID).
		Preload("Sender").
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to fetch messages",
			},
		})
		return
	}

	c.PureJSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
	})
}
