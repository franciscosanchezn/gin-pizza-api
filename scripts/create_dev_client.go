package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type OAuthClient struct {
	ID          string `gorm:"primaryKey"`
	Secret      string `gorm:"not null"`
	Name        string `gorm:"not null"`
	Domain      string
	UserID      uint
	Scopes      string `gorm:"not null"`
	GrantTypes  string
	RedirectURI string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}

func main() {
	db, err := gorm.Open(sqlite.Open("test.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	clientID := "dev-client"
	clientSecret := "dev-secret-123"

	// Check if client already exists
	var existing OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		fmt.Println("Development client already exists!")
		fmt.Printf("Client ID: %s\n", clientID)
		fmt.Printf("Client Secret: %s\n", clientSecret)
		return
	}

	// Create new client
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash secret:", err)
	}

	client := OAuthClient{
		ID:          clientID,
		Secret:      string(hash),
		Name:        "Development Client",
		Domain:      "http://localhost",
		UserID:      1, // Default user ID for dev client
		Scopes:      "read write",
		GrantTypes:  "client_credentials",
		RedirectURI: "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(&client).Error; err != nil {
		log.Fatal("Failed to create client:", err)
	}

	fmt.Println("✓ Development OAuth client created!")
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("Client Secret: %s\n", clientSecret)
	fmt.Println("\nUse these credentials for testing:")
	fmt.Printf("curl -X POST http://localhost:8080/api/v1/oauth/token \\\n")
	fmt.Printf("  -d 'grant_type=client_credentials' \\\n")
	fmt.Printf("  -d 'client_id=%s' \\\n", clientID)
	fmt.Printf("  -d 'client_secret=%s'\n", clientSecret)
}
