package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
)

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	Description string `json:"description" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,gt=0"`
}

// CreateOrder handles POST /api/v1/orders - creates a new order (customers only)
func CreateOrder(c *gin.Context) {
	// Extract Auth0 user ID from JWT token
	auth0ID, err := middleware.GetUserID(c)
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

	// Find the user in the database
	db := config.GetDB()
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User profile not found. Please create a profile first.",
			},
		})
		return
	}

	// Check if user is a customer (only customers can create orders)
	if user.Role != "customer" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Only customers can create orders",
			},
		})
		return
	}

	// Parse request body
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request data",
				"details": err.Error(),
			},
		})
		return
	}

	// Create the order
	order := models.Order{
		Description: req.Description,
		Quantity:    req.Quantity,
		Status:      "submitted",
		CustomerID:  user.ID,
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to create order",
			},
		})
		return
	}

	// Load the customer relationship to return complete data
	if err := db.Preload("Customer").First(&order, order.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    order,
	})
}
