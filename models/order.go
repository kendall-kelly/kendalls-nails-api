package models

import (
	"time"

	"gorm.io/gorm"
)

// Order represents a custom nail order in the system
type Order struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Description  string         `gorm:"not null" json:"description"`
	Quantity     int            `gorm:"not null;check:quantity > 0" json:"quantity"`
	Status       string         `gorm:"not null;default:'submitted'" json:"status"` // submitted, accepted, rejected, in_production, shipped, delivered
	Price        *float64       `json:"price"`                                       // nullable, set when order is accepted
	Feedback     *string        `json:"feedback"`                                    // nullable, set when order is rejected
	CustomerID   uint           `gorm:"not null;index" json:"customer_id"`           // foreign key to users table
	Customer     User           `gorm:"foreignKey:CustomerID" json:"customer"`
	TechnicianID *uint          `gorm:"index" json:"technician_id"` // nullable, assigned when order is reviewed
	Technician   *User          `gorm:"foreignKey:TechnicianID" json:"technician,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the Order model
func (Order) TableName() string {
	return "orders"
}
