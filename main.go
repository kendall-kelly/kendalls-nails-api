package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/controllers"
	"github.com/kendall-kelly/kendalls-nails-api/middleware"
	"github.com/kendall-kelly/kendalls-nails-api/models"
)

func main() {
	// Basic logging
	log.Println("Starting Custom Nails API server...")

	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	if err := config.ConnectDatabase(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database models
	db := config.GetDB()
	if err := db.AutoMigrate(&models.User{}, &models.Order{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed successfully")

	// Initialize Gin router
	router := gin.Default()

	// Configure CORS middleware
	// Allows Single Page Apps to make API calls from different origins
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.GetCORSOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	log.Printf("CORS configured for origins: %v", cfg.GetCORSOrigins())

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", healthCheck)

		// Database status endpoint
		v1.GET("/database/status", databaseStatus)

		// Protected endpoint - requires valid JWT token
		v1.GET("/protected", middleware.EnsureValidToken(cfg), protectedEndpoint)

		// User management routes
		v1.POST("/users", middleware.EnsureValidToken(cfg), controllers.CreateUser)
		v1.GET("/users/me", middleware.EnsureValidToken(cfg), controllers.GetMyProfile)
		v1.PUT("/users/me", middleware.EnsureValidToken(cfg), controllers.UpdateMyProfile)

		// Order management routes
		v1.POST("/orders", middleware.EnsureValidToken(cfg), controllers.CreateOrder)
		v1.GET("/orders", middleware.EnsureValidToken(cfg), controllers.ListOrders)
		v1.GET("/orders/:id", middleware.EnsureValidToken(cfg), controllers.GetOrder)
		v1.PUT("/orders/:id/assign", middleware.EnsureValidToken(cfg), controllers.AssignOrder)
		v1.PUT("/orders/:id/review", middleware.EnsureValidToken(cfg), controllers.ReviewOrder)
		v1.PUT("/orders/:id/status", middleware.EnsureValidToken(cfg), controllers.UpdateOrderStatus)

		// File upload routes - serve uploaded images
		v1.GET("/uploads/:filename", controllers.GetUploadedImage)
	}

	// Start server
	port := ":" + cfg.Port
	log.Printf("Server is running on http://localhost%s (env: %s)", port, cfg.GoEnv)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// healthCheck handles the health check endpoint
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Custom Nails API is running",
	})
}

// databaseStatus checks database connectivity and returns table information
func databaseStatus(c *gin.Context) {
	db := config.GetDB()

	// Get the underlying SQL database to check connection
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_ERROR",
				"message": "Failed to get database instance",
			},
		})
		return
	}

	// Ping the database to verify connection
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_CONNECTION_ERROR",
				"message": "Database connection failed",
			},
		})
		return
	}

	// Get list of tables
	var tables []string
	if err := db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "DATABASE_QUERY_ERROR",
				"message": "Failed to query tables",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database connected",
		"tables":  tables,
	})
}

// protectedEndpoint is an endpoint that requires valid JWT authentication
func protectedEndpoint(c *gin.Context) {
	// Extract user ID from the authenticated token
	userID, err := middleware.GetUserID(c)
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

	// Get the validated claims
	claims, err := middleware.GetClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Could not retrieve claims",
			},
		})
		return
	}

	// Return success with user information
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "You have accessed a protected endpoint",
		"data": gin.H{
			"user_id": userID,
			"issuer":  claims.RegisteredClaims.Issuer,
			"subject": claims.RegisteredClaims.Subject,
		},
	})
}
