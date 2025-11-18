package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserTableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName(), "Table name should be 'users'")
}

func TestUserStructFields(t *testing.T) {
	user := User{
		Email: "test@example.com",
		Role:  "customer",
	}

	assert.Equal(t, "test@example.com", user.Email, "Email should be set correctly")
	assert.Equal(t, "customer", user.Role, "Role should be set correctly")
}

func TestUserDefaultValues(t *testing.T) {
	// Test that a new user can be created
	user := User{
		Email: "new@example.com",
	}

	assert.Equal(t, "new@example.com", user.Email, "Email should be set")
	assert.Equal(t, "", user.Role, "Role should be empty string by default in Go struct")
}

func TestUserRoleValues(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{"customer role", "customer"},
		{"technician role", "technician"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{
				Email: "test@example.com",
				Role:  tt.role,
			}
			assert.Equal(t, tt.role, user.Role, "Role should be set correctly")
		})
	}
}
