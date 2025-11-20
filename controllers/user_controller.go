package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
	"github.com/kendall-kelly/kendalls-nails-api/services"
)

// UpdateUserRequest represents the request body for updating a user profile
type UpdateUserRequest struct {
	Name  string `json:"name" binding:"omitempty"`
	Email string `json:"email" binding:"omitempty,email"`
}

// CreateUser handles POST /api/v1/users - creates a new user from Auth0 userinfo
// This endpoint requires authentication and fetches user data from Auth0's /userinfo endpoint
func CreateUser(c *gin.Context) {
	// Get the Auth0 user ID from the validated JWT
	auth0ID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Could not extract user ID from token",
			},
		})
		return
	}

	// Get the access token to call Auth0's /userinfo endpoint
	accessToken, err := middleware.GetAccessToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_TOKEN",
				"message": "Access token not found",
			},
		})
		return
	}

	// Fetch user info from Auth0
	cfg := config.GetConfig()
	auth0Service := services.NewAuth0Service(cfg)
	userInfo, err := auth0Service.GetUserInfo(accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "AUTH0_ERROR",
				"message": "Failed to fetch user information from Auth0",
			},
		})
		return
	}

	// Validate that required fields are present
	if userInfo.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_EMAIL",
				"message": "Email not provided by Auth0",
			},
		})
		return
	}

	if userInfo.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "MISSING_NAME",
				"message": "Name not provided by Auth0",
			},
		})
		return
	}

	// Get role from custom claims (if present)
	claims, err := middleware.GetClaims(c)
	role := "customer" // default role
	if err == nil {
		if customClaims, ok := claims.CustomClaims.(*middleware.CustomClaims); ok && customClaims.Role != "" {
			role = customClaims.Role
		}
	}

	// Create user in database using data from Auth0
	user := models.User{
		Auth0ID: auth0ID,
		Name:    userInfo.Name,
		Email:   userInfo.Email,
		Role:    role,
	}

	db := config.GetDB()
	if err := db.Create(&user).Error; err != nil {
		// Check for duplicate Auth0ID or email (works with both PostgreSQL and SQLite)
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "duplicate") ||
		   strings.Contains(errMsg, "unique constraint") ||
		   strings.Contains(errMsg, "unique") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "USER_EXISTS",
					"message": "A user with this Auth0 ID or email already exists",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to create user",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

// GetMyProfile handles GET /api/v1/users/me - gets current user's profile
func GetMyProfile(c *gin.Context) {
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

	// Find user by Auth0ID
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// UpdateMyProfile handles PUT /api/v1/users/me - updates current user's profile
func UpdateMyProfile(c *gin.Context) {
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

	// Parse request body
	var req UpdateUserRequest
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

	// Find user by Auth0ID
	db := config.GetDB()
	var user models.User
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "USER_NOT_FOUND",
				"message": "User profile not found",
			},
		})
		return
	}

	// Update fields if provided
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	// If no fields to update, return current user
	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    user,
		})
		return
	}

	// Update user in database
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		// Check for duplicate email (works with both PostgreSQL and SQLite)
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "duplicate") ||
		   strings.Contains(errMsg, "unique constraint") ||
		   strings.Contains(errMsg, "unique") {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "EMAIL_EXISTS",
					"message": "A user with this email already exists",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to update user profile",
			},
		})
		return
	}

	// Fetch updated user to return
	if err := db.Where("auth0_id = ?", auth0ID).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to fetch updated profile",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}
