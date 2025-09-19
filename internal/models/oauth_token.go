package models

import (
	"time"
)

type OAuthToken struct {
	ID           uint `gorm:"primaryKey"`
	ClientID     string `gorm:"not null"`
	UserID       *string // Nullable for client credentials *string by default will be null, string will be ""
	AccessToken  string `gorm:"uniqueIndex;not null"`
	RefreshToken *string
	Scopes       string
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (OAuthToken) TableName() string {
	return "oauth_tokens"
}