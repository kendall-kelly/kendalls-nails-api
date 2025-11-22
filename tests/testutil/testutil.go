package testutil

import (
	"fmt"
	"os"
	"testing"
)

// RequireTestEnvironment ensures that tests are running in the test environment.
// This prevents accidental execution of tests against production or development databases.
// It will fail the test immediately if GO_ENV is not set to "test".
func RequireTestEnvironment(t *testing.T) {
	t.Helper()

	env := os.Getenv("GO_ENV")
	if env != "test" {
		t.Fatalf("SAFETY CHECK FAILED: Tests must run with GO_ENV=test to prevent data loss. Current GO_ENV=%q. Set GO_ENV=test before running tests.", env)
	}
}

// RequireTestEnvironmentOrSkip is similar to RequireTestEnvironment but skips the test
// instead of failing it. Use this for optional tests that should only run in test environment.
func RequireTestEnvironmentOrSkip(t *testing.T) {
	t.Helper()

	env := os.Getenv("GO_ENV")
	if env != "test" {
		t.Skipf("Skipping test: GO_ENV must be 'test' (current: %q)", env)
	}
}

// MustSetTestEnvironment sets GO_ENV to test and fails if it cannot be set.
// Use this in TestMain or suite setup functions.
func MustSetTestEnvironment(t *testing.T) {
	t.Helper()

	if err := os.Setenv("GO_ENV", "test"); err != nil {
		t.Fatalf("Failed to set GO_ENV=test: %v", err)
	}

	// Verify it was set
	if os.Getenv("GO_ENV") != "test" {
		t.Fatal("Failed to verify GO_ENV=test")
	}
}

// PrintEnvironmentInfo prints the current test environment configuration.
// Useful for debugging test environment issues.
func PrintEnvironmentInfo() {
	fmt.Printf("Test Environment Info:\n")
	fmt.Printf("  GO_ENV: %s\n", os.Getenv("GO_ENV"))
	fmt.Printf("  DATABASE_URL: %s\n", maskDatabaseURL(os.Getenv("DATABASE_URL")))
	fmt.Printf("  PORT: %s\n", os.Getenv("PORT"))
}

// maskDatabaseURL masks sensitive parts of the database URL for safe printing
func maskDatabaseURL(url string) string {
	if url == "" {
		return "(not set)"
	}
	// Simple masking - just show if it contains "test"
	if len(url) > 20 {
		return url[:20] + "..." + (map[bool]string{true: " [contains 'test']", false: " [WARNING: may not be test DB]"})[containsTest(url)]
	}
	return url
}

func containsTest(s string) bool {
	return len(s) > 0 && (s[len(s)-5:] == "_test" || s[len(s)-4:] == "test")
}
