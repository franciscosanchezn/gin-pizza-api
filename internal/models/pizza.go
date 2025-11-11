package models

import (
	"time"

	"gorm.io/gorm"
)

// Pizza represents a pizza with its properties
type Pizza struct {
	ID          int            `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description"`
	Ingredients []string       `json:"ingredients" gorm:"serializer:json"`
	Price       float64        `json:"price"`
	CreatedBy   uint           `json:"created_by" gorm:"not null;index"`
	Creator     *User          `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
