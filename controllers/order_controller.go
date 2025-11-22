package controllers

import (
	"net/http"
	"strconv"

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

// ListOrders handles GET /api/v1/orders - lists orders with role-based filtering
// Customers see only their orders
// Technicians see orders assigned to them + unassigned orders
func ListOrders(c *gin.Context) {
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

	// Parse pagination parameters
	page := 1
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	offset := (page - 1) * limit

	// Build query based on user role
	query := db.Model(&models.Order{})

	switch user.Role {
	case "customer":
		// Customers see only their own orders
		query = query.Where("customer_id = ?", user.ID)
	case "technician":
		// Technicians see orders assigned to them + unassigned orders
		query = query.Where("technician_id = ? OR technician_id IS NULL", user.ID)
	}

	// Get total count for pagination info
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to count orders",
			},
		})
		return
	}

	// Fetch orders with pagination
	var orders []models.Order
	if err := query.Preload("Customer").Preload("Technician").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to fetch orders",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orders,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetOrder handles GET /api/v1/orders/:id - gets a single order with authorization
func GetOrder(c *gin.Context) {
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

	// Get order ID from URL parameter
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
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
	if err := db.Preload("Customer").Preload("Technician").First(&order, orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ORDER_NOT_FOUND",
				"message": "Order not found",
			},
		})
		return
	}

	// Authorization check: Can user access this order?
	canAccess := false
	switch user.Role {
	case "customer":
		// Customers can only access their own orders
		canAccess = order.CustomerID == user.ID
	case "technician":
		// Technicians can access orders assigned to them or unassigned orders
		canAccess = order.TechnicianID == nil || (order.TechnicianID != nil && *order.TechnicianID == user.ID)
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You do not have permission to access this order",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// ReviewOrderRequest represents the request body for reviewing an order
type ReviewOrderRequest struct {
	Action   string   `json:"action" binding:"required,oneof=accept reject"`
	Price    *float64 `json:"price"`
	Feedback *string  `json:"feedback"`
}

// ReviewOrder handles PUT /api/v1/orders/:id/review - accepts or rejects an order (technicians only)
func ReviewOrder(c *gin.Context) {
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

	// Check if user is a technician (only technicians can review orders)
	if user.Role != "technician" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Only technicians can review orders",
			},
		})
		return
	}

	// Get order ID from URL parameter
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
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
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ORDER_NOT_FOUND",
				"message": "Order not found",
			},
		})
		return
	}

	// Check if order has already been reviewed
	if order.Status != "submitted" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_STATE",
				"message": "Order has already been reviewed",
			},
		})
		return
	}

	// Parse request body
	var req ReviewOrderRequest
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

	// Validate action-specific requirements
	switch req.Action {
	case "accept":
		if req.Price == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Price is required when accepting an order",
				},
			})
			return
		}
		if *req.Price <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Price must be greater than zero",
				},
			})
			return
		}
	case "reject":
		if req.Feedback == nil || *req.Feedback == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Feedback is required when rejecting an order",
				},
			})
			return
		}
	}

	// Update the order based on the action
	if req.Action == "accept" {
		order.Status = "accepted"
		order.Price = req.Price
		order.TechnicianID = &user.ID
	} else {
		order.Status = "rejected"
		order.Feedback = req.Feedback
		order.TechnicianID = &user.ID
	}

	// Save the changes
	if err := db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to update order",
			},
		})
		return
	}

	// Load relationships for complete response
	if err := db.Preload("Customer").Preload("Technician").First(&order, order.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}
