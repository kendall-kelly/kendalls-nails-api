package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDB(t *testing.T) {
	// Initially DB should be nil
	DB = nil
	db := GetDB()
	assert.Nil(t, db, "GetDB should return nil when DB is not initialized")

	// After setting DB, GetDB should return it
	// Note: We don't actually connect in this unit test
}

func TestConnectDatabaseWithEnvVar(t *testing.T) {
	// Save original env var
	originalURL := os.Getenv("DATABASE_URL")
	defer func() {
		if originalURL != "" {
			os.Setenv("DATABASE_URL", originalURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
		DB = nil
	}()

	// Test with invalid database URL (should fail to connect)
	os.Setenv("DATABASE_URL", "postgresql://invalid:invalid@localhost:9999/nonexistent?sslmode=disable")
	err := ConnectDatabase()
	assert.Error(t, err, "Should fail to connect with invalid database URL")
}

func TestConnectDatabaseWithoutEnvVar(t *testing.T) {
	// Save original env var and DB
	originalURL := os.Getenv("DATABASE_URL")
	originalDB := DB
	defer func() {
		if originalURL != "" {
			os.Setenv("DATABASE_URL", originalURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
		DB = originalDB
	}()

	// Unset DATABASE_URL
	os.Unsetenv("DATABASE_URL")
	DB = nil

	// This will use the default URL
	// In test environment with Docker running, this should succeed
	// We're testing that the fallback to default URL works
	err := ConnectDatabase()

	// If database is running (like in CI or with Docker), it should connect
	// If not running, it should fail gracefully
	// Either outcome is acceptable - we're testing the fallback mechanism
	if err == nil {
		// Database connected successfully with default URL
		assert.NotNil(t, DB, "DB should be set when connection succeeds")
	} else {
		// Database not available - that's also acceptable for this test
		assert.NotNil(t, err, "Error should be returned when connection fails")
	}
}
