package config

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDatabase establishes a connection to the PostgreSQL database
// It uses the Config struct to get the appropriate database URL
func ConnectDatabase() error {
	// Load configuration
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get the appropriate database URL based on environment
	databaseURL := cfg.GetDatabaseURL()

	// Connect to database
	DB, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("Database connection established successfully (env: %s)", cfg.GoEnv)
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// SetDB sets the database instance (primarily for testing)
func SetDB(db *gorm.DB) {
	DB = db
}
