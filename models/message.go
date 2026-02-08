package models

import (
	"time"

	"gorm.io/gorm"
)

// Message represents a message in an order conversation
type Message struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	OrderID   uint           `gorm:"not null;index" json:"order_id"` // foreign key to orders table
	Order     Order          `gorm:"foreignKey:OrderID" json:"-"`    // don't include full order in JSON
	SenderID  uint           `gorm:"not null;index" json:"sender_id"` // foreign key to users table
	Sender    User           `gorm:"foreignKey:SenderID" json:"sender"`
	Text      string         `gorm:"type:text;not null" json:"text"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for the Message model
func (Message) TableName() string {
	return "messages"
}
