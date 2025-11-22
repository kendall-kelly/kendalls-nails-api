package config

import (
	"fmt"
	"os"
	"testing"
)

// TestMain runs before all tests in the config package
// It ensures GO_ENV is set to "test" to prevent accidental data loss
func TestMain(m *testing.M) {
	env := os.Getenv("GO_ENV")
	if env != "test" {
		fmt.Fprintf(os.Stderr, "\n"+
			"╔════════════════════════════════════════════════════════════════╗\n"+
			"║                    SAFETY CHECK FAILED                         ║\n"+
			"║                                                                ║\n"+
			"║  Tests must run with GO_ENV=test to prevent data loss!        ║\n"+
			"║                                                                ║\n"+
			"║  Current GO_ENV: %-45s ║\n"+
			"║                                                                ║\n"+
			"║  To run tests safely:                                          ║\n"+
			"║    make test                                                   ║\n"+
			"║    GO_ENV=test go test ./...                                   ║\n"+
			"╚════════════════════════════════════════════════════════════════╝\n\n",
			fmt.Sprintf("%q", env))
		os.Exit(1)
	}

	// Run tests
	os.Exit(m.Run())
}
