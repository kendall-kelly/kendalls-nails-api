package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kendall-kelly/kendalls-nails-api/config"
	"github.com/kendall-kelly/kendalls-nails-api/models"
)

func main() {
	// Basic logging
	log.Println("Starting Custom Nails API server...")

	// Connect to database
	if err := config.ConnectDatabase(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate database models
	db := config.GetDB()
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	log.Println("Database migration completed successfully")

	// Initialize Gin router
	router := gin.Default()

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", healthCheck)

		// Database status endpoint
		v1.GET("/database/status", databaseStatus)
	}

	// Start server
	port := ":8080"
	log.Printf("Server is running on http://localhost%s", port)
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
