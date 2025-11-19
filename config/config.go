package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	DatabaseURL        string
	Port               string
	GoEnv              string
	Auth0Domain        string
	Auth0Audience      string
	JWTSecret          string
	AWSRegion          string
	AWSS3Bucket        string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	LogLevel           string
}

// Load loads the configuration from environment variables
// It automatically determines which .env file to load based on GO_ENV
func Load() (*Config, error) {
	// Determine which environment file to load
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	// Try to load environment-specific file first
	envFile := fmt.Sprintf(".env.%s", env)
	if err := godotenv.Load(envFile); err != nil {
		// If environment-specific file doesn't exist, try .env
		if err := godotenv.Load(); err != nil {
			// In production (Heroku), environment variables are set directly
			// so it's okay if .env files don't exist
			log.Printf("No .env file found, using system environment variables")
		}
	} else {
		log.Printf("Loaded configuration from %s", envFile)
	}

	config := &Config{
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		Port:               getEnv("PORT", "8080"),
		GoEnv:              getEnv("GO_ENV", "development"),
		Auth0Domain:        getEnv("AUTH0_DOMAIN", ""),
		Auth0Audience:      getEnv("AUTH0_AUDIENCE", ""),
		AWSRegion:          getEnv("AWS_REGION", "us-east-1"),
		AWSS3Bucket:        getEnv("AWS_S3_BUCKET", ""),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that all required configuration values are set
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.GoEnv == "production"
}

// IsTest returns true if the application is running in test mode
func (c *Config) IsTest() bool {
	return c.GoEnv == "test"
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.GoEnv == "development"
}

// GetDatabaseURL returns the database URL
func (c *Config) GetDatabaseURL() string {
	return c.DatabaseURL
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
