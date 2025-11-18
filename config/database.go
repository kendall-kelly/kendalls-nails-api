package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDatabase establishes a connection to the PostgreSQL database
func ConnectDatabase() error {
	// Get database URL from environment variable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Fallback to default local database URL for development
		databaseURL = "postgresql://postgres:postgres@localhost:5432/kendalls_nails?sslmode=disable"
		log.Println("DATABASE_URL not set, using default:", databaseURL)
	}

	// Connect to database
	var err error
	DB, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established successfully")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
