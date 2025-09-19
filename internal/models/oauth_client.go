package models

import (
	"gorm.io/gorm"
	"time"
)

type OAuthClient struct {
	ID          string `gorm:"primaryKey"`
	Secret      string `gorm:"not null"`
	Name        string
	Domain      string
	UserID      uint   // Reference to User model for admin management
	Scopes      string // Space-separated list of allowed scopes
	GrantTypes  string // Space-separated list: "authorization_code client_credentials"
	RedirectURI string  // validation tags can be added as needed
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}