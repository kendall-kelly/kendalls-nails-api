package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system (customer or technician)
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Auth0ID   string         `gorm:"uniqueIndex;not null" json:"auth0_id"` // Auth0 user ID (from 'sub' claim)
	Name      string         `gorm:"not null" json:"name"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Role      string         `gorm:"not null;default:'customer'" json:"role"` // "customer" or "technician"
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}
