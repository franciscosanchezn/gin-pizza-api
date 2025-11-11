package models

import (
	"time"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	Name      string
	Role      string `gorm:"default:'admin'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
