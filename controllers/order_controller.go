package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/kendall-kelly/kendalls-nails-api/services"
	"github.com/kendall-kelly/kendalls-nails-api/utils"
)

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	Description string `json:"description" binding:"required"`
	Quantity    int    `json:"quantity" binding:"required,gt=0"`
}

// populateOrderImageURL generates presigned URLs for images
func populateOrderImageURL(order *models.Order) {
	if order.ImageS3Key == nil || *order.ImageS3Key == "" {
		return
	}

	imageService := services.GetImageService()
	if url, err := imageService.GetImageURL(*order.ImageS3Key); err == nil {
		order.ImageURL = &url
	}
}

// populateOrdersImageURLs populates image URLs for a slice of orders
func populateOrdersImageURLs(orders []models.Order) {
	for i := range orders {
		populateOrderImageURL(&orders[i])
	}
}

// CreateOrder handles POST /api/v1/orders - creates a new order (customers only)
func CreateOrder(c *gin.Context) {
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

	// Check if user is a customer (only customers can create orders)
	if user.Role != "customer" {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Only customers can create orders",
			},
		})
		return
	}

	// Check content type to determine if this is multipart form data or JSON
	contentType := c.ContentType()
	var description string
	var quantity int
	var imagePath *string

	if contentType == "application/json" {
		// Parse JSON request (legacy support, no file upload)
		var req CreateOrderRequest
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
		description = req.Description
		quantity = req.Quantity
	} else {
		// Parse multipart form data (with potential file upload)
		description = c.PostForm("description")
		quantityStr := c.PostForm("quantity")

		// Validate required fields
		if description == "" {
			c.PureJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Description is required",
				},
			})
			return
		}

		if quantityStr == "" {
			c.PureJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Quantity is required",
				},
			})
			return
		}

		// Parse quantity
		parsedQuantity, err := strconv.Atoi(quantityStr)
		if err != nil || parsedQuantity <= 0 {
			c.PureJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Quantity must be a positive integer",
				},
			})
			return
		}
		quantity = parsedQuantity

		// Handle file upload if present
		fileHeader, err := c.FormFile("image")
		if err == nil {
			// File was provided, upload it using image service
			imageService := services.GetImageService()
			imageKey, uploadErr := imageService.UploadImage(fileHeader)
			if uploadErr != nil {
				// Check if it's a validation error
				if fileErr, ok := uploadErr.(*utils.FileUploadError); ok {
					c.PureJSON(http.StatusBadRequest, gin.H{
						"success": false,
						"error": gin.H{
							"code":    fileErr.Code,
							"message": fileErr.Message,
						},
					})
					return
				}
				// Generic upload error
				c.PureJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "IMAGE_UPLOAD_ERROR",
						"message": "Failed to upload image",
					},
				})
				return
			}
			imagePath = &imageKey
		}
		// If err != nil, no file was provided, which is okay (image is optional)
	}

	// Create the order
	order := models.Order{
		Description: description,
		Quantity:    quantity,
		Status:      "submitted",
		CustomerID:  user.ID,
		ImageS3Key:  imagePath, // Store S3 key if image was uploaded
	}

	if err := db.Create(&order).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
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
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	// Generate presigned URL for image if using S3
	populateOrderImageURL(&order)

	c.PureJSON(http.StatusCreated, gin.H{
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
		c.PureJSON(http.StatusInternalServerError, gin.H{
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
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to fetch orders",
			},
		})
		return
	}

	// Generate image URLs for all orders
	populateOrdersImageURLs(orders)

	c.PureJSON(http.StatusOK, gin.H{
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
	if err := db.Preload("Customer").Preload("Technician").First(&order, orderID).Error; err != nil {
		c.PureJSON(http.StatusNotFound, gin.H{
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
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You do not have permission to access this order",
			},
		})
		return
	}

	// Generate image URL
	populateOrderImageURL(&order)

	c.PureJSON(http.StatusOK, gin.H{
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

	// Check if user is a technician (only technicians can review orders)
	if user.Role != "technician" {
		c.PureJSON(http.StatusForbidden, gin.H{
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

	// Check if order has already been reviewed
	if order.Status != "submitted" {
		c.PureJSON(http.StatusUnprocessableEntity, gin.H{
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

	// Validate action-specific requirements
	switch req.Action {
	case "accept":
		if req.Price == nil {
			c.PureJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "VALIDATION_ERROR",
					"message": "Price is required when accepting an order",
				},
			})
			return
		}
		if *req.Price <= 0 {
			c.PureJSON(http.StatusBadRequest, gin.H{
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
			c.PureJSON(http.StatusBadRequest, gin.H{
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
		c.PureJSON(http.StatusInternalServerError, gin.H{
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
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	// Generate image URL
	populateOrderImageURL(&order)

	c.PureJSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// UpdateOrderStatusRequest represents the request body for updating order status
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=in_production shipped delivered"`
}

// UpdateOrderStatus handles PUT /api/v1/orders/:id/status - updates order status (technicians only)
func UpdateOrderStatus(c *gin.Context) {
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

	// Check if user is a technician (only technicians can update order status)
	if user.Role != "technician" {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Only technicians can update order status",
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

	// Check if order is assigned to this technician
	if order.TechnicianID == nil || *order.TechnicianID != user.ID {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "You can only update status of orders assigned to you",
			},
		})
		return
	}

	// Parse request body
	var req UpdateOrderStatusRequest
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

	// Define valid status transitions
	validTransitions := map[string][]string{
		"accepted":      {"in_production"},
		"in_production": {"shipped"},
		"shipped":       {"delivered"},
		"delivered":     {}, // Terminal state
	}

	// Check if the current status allows the requested transition
	allowedStatuses, exists := validTransitions[order.Status]
	if !exists {
		c.PureJSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_STATE",
				"message": "Cannot update status from current order state",
			},
		})
		return
	}

	// Check if the requested status is in the list of allowed transitions
	isValid := false
	for _, allowed := range allowedStatuses {
		if allowed == req.Status {
			isValid = true
			break
		}
	}

	if !isValid {
		c.PureJSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_TRANSITION",
				"message": "Invalid status transition",
				"details": gin.H{
					"current_status":   order.Status,
					"requested_status": req.Status,
					"allowed_statuses": allowedStatuses,
				},
			},
		})
		return
	}

	// Update the order status
	order.Status = req.Status

	// Save the changes
	if err := db.Save(&order).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to update order status",
			},
		})
		return
	}

	// Load relationships for complete response
	if err := db.Preload("Customer").Preload("Technician").First(&order, order.ID).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	// Generate image URL
	populateOrderImageURL(&order)

	c.PureJSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}

// AssignOrder handles PUT /api/v1/orders/:id/assign - assigns an order to the current technician
func AssignOrder(c *gin.Context) {
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

	// Check if user is a technician (only technicians can assign orders)
	if user.Role != "technician" {
		c.PureJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "FORBIDDEN",
				"message": "Only technicians can assign orders",
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

	// Check if order is already assigned to another technician
	if order.TechnicianID != nil && *order.TechnicianID != user.ID {
		c.PureJSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ALREADY_ASSIGNED",
				"message": "Order is already assigned to another technician",
			},
		})
		return
	}

	// Check if order is already assigned to this technician
	if order.TechnicianID != nil && *order.TechnicianID == user.ID {
		c.PureJSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ALREADY_ASSIGNED",
				"message": "Order is already assigned to you",
			},
		})
		return
	}

	// Assign the order to the current technician
	order.TechnicianID = &user.ID

	// Save the changes
	if err := db.Save(&order).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to assign order",
			},
		})
		return
	}

	// Load relationships for complete response
	if err := db.Preload("Customer").Preload("Technician").First(&order, order.ID).Error; err != nil {
		c.PureJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to load order details",
			},
		})
		return
	}

	// Generate image URL
	populateOrderImageURL(&order)

	c.PureJSON(http.StatusOK, gin.H{
		"success": true,
		"data":    order,
	})
}
