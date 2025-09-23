package models

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type OAuthClient struct {
	ID          string `gorm:"primaryKey"`
	Secret      string `gorm:"not null"`
	Name        string
	Domain      string
	UserID      uint   // Reference to User model for admin management
	Scopes      string // Space-separated list of allowed scopes
	GrantTypes  string // Space-separated list: "authorization_code client_credentials"
	RedirectURI string // validation tags can be added as needed
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (c *OAuthClient) GetID() string {
	return c.ID
}

func (c *OAuthClient) GetSecret() string {
	return c.Secret
}

func (c *OAuthClient) GetDomain() string {
	return c.Domain
}

func (c *OAuthClient) IsPublic() bool {
	// Assuming clients with empty secret are public
	return c.Secret == ""
}

func (c *OAuthClient) GetUserID() string {
	return fmt.Sprint(c.UserID)
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}

// VerifyPassword implements the ClientPasswordVerifier interface
// This allows the OAuth2 library to verify bcrypt-hashed passwords
func (c *OAuthClient) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(c.Secret), []byte(password))
	return err == nil
}
