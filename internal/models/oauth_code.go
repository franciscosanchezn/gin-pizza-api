package models

import (
	"time"
)

type OAuthCode struct {
	Code                string `gorm:"primaryKey"`
	ClientID            string `gorm:"not null"`
	UserID              string `gorm:"not null"`
	Scopes              string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time `gorm:"not null"`
	CreatedAt           time.Time
}

func (OAuthCode) TableName() string {
	return "oauth_codes"
}
