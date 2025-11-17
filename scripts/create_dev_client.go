package main

import (
	"flag"
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

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	Name      string
	Role      string `gorm:"default:'admin'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func main() {
	// Parse command line flags
	role := flag.String("role", "admin", "User role (admin or user)")
	flag.Parse()

	db, err := gorm.Open(sqlite.Open("test.sqlite"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Determine client credentials based on role
	var clientID, clientSecret string
	if *role == "user" {
		clientID = "user-client"
		clientSecret = "user-secret-123"
	} else {
		clientID = "dev-client"
		clientSecret = "dev-secret-123"
	}

	// Check if client already exists
	var existing OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		fmt.Printf("Development client already exists for role '%s'!\n", *role)
		fmt.Printf("Client ID: %s\n", clientID)
		fmt.Printf("Client Secret: %s\n", clientSecret)
		return
	}

	// Get or create user with specified role
	userID := getUserIDForRole(db, *role)
	if userID == 0 {
		log.Fatal("Failed to get user ID for role:", *role)
	}

	// Create new client
	hash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash secret:", err)
	}

	client := OAuthClient{
		ID:          clientID,
		Secret:      string(hash),
		Name:        fmt.Sprintf("Development %s Client", *role),
		Domain:      "http://localhost",
		UserID:      userID,
		Scopes:      "read write",
		GrantTypes:  "client_credentials",
		RedirectURI: "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := db.Create(&client).Error; err != nil {
		log.Fatal("Failed to create client:", err)
	}

	fmt.Printf("✓ Development OAuth client created for role '%s'!\n", *role)
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("Client Secret: %s\n", clientSecret)
	fmt.Printf("User ID: %d\n", userID)
	fmt.Println("\nUse these credentials for testing:")
	fmt.Printf("curl -X POST http://localhost:8080/api/v1/oauth/token \\\n")
	fmt.Printf("  -d 'grant_type=client_credentials' \\\n")
	fmt.Printf("  -d 'client_id=%s' \\\n", clientID)
	fmt.Printf("  -d 'client_secret=%s'\n", clientSecret)
}

// getUserIDForRole gets or creates a user with the specified role
func getUserIDForRole(db *gorm.DB, role string) uint {
	var user User
	email := fmt.Sprintf("%s@pizza.com", role)
	
	// Try to find existing user
	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		fmt.Printf("Found existing user: %s (ID: %d, Role: %s)\n", user.Email, user.ID, user.Role)
		return user.ID
	}

	// Create new user
	user = User{
		Email: email,
		Name:  fmt.Sprintf("%s User", role),
		Role:  role,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := db.Create(&user).Error; err != nil {
		log.Printf("Failed to create user: %v", err)
		return 0
	}

	fmt.Printf("Created new user: %s (ID: %d, Role: %s)\n", user.Email, user.ID, user.Role)
	return user.ID
}
