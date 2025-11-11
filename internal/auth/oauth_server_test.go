package auth

import (
	"context"
	"testing"

	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&models.User{}, &models.OAuthClient{})
	require.NoError(t, err)

	return db
}

func TestOAuthServerInitialization(t *testing.T) {
	db := setupTestDB(t)

	// Fix: NewOAuthService requires jwtSecret parameter
	oauthService := NewOAuthService(db, "test-jwt-secret-key-32-characters")
	assert.NotNil(t, oauthService)
	assert.NotNil(t, oauthService.GetServer())
}

func TestJWTTokenGeneration(t *testing.T) {
	db := setupTestDB(t)

	// Fix: Provide JWT secret parameter
	oauthService := NewOAuthService(db, "test-jwt-secret-key-32-characters")
	require.NotNil(t, oauthService)

	// Create a test user first (required for token generation)
	testUser := &models.User{
		Email: "test3@example.com",
		Name:  "Test User 3",
		Role:  "admin",
	}
	err := db.Create(testUser).Error
	require.NoError(t, err)

	// Generate bcrypt hash for the test client secret
	plainSecret := "test_secret"
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(plainSecret), bcrypt.DefaultCost)
	require.NoError(t, err)

	client := &models.OAuthClient{
		ID:         "test_client",
		Secret:     string(hashedSecret), // Store bcrypt hash
		Domain:     "http://localhost",
		Scopes:     "read,write",
		UserID:     testUser.ID,          // Associate with user
		GrantTypes: "client_credentials",
	}
	err = db.Create(client).Error
	require.NoError(t, err)

	// Fix: Use proper OAuth2 token generation request
	ctx := context.Background()
	tokenRequest := &oauth2.TokenGenerateRequest{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		UserID:       "",       // Will be populated from client's UserID
		Scope:        "read,write",
	}

	// Generate access token through the OAuth server
	tokenInfo, err := oauthService.GetServer().Manager.GenerateAccessToken(ctx, oauth2.ClientCredentials, tokenRequest)
	assert.NoError(t, err)
	assert.NotNil(t, tokenInfo)
	assert.NotEmpty(t, tokenInfo.GetAccess())

	// Test that the token is a valid JWT
	accessToken := tokenInfo.GetAccess()
	assert.Contains(t, accessToken, ".")  // JWT has dots
	assert.True(t, len(accessToken) > 50) // JWT tokens are longer
}

func TestClientStoreIntegration(t *testing.T) {
	db := setupTestDB(t)

	// Create test client in database
	client := &models.OAuthClient{
		ID:     "integration_test_client",
		Secret: "integration_test_secret",
		Domain: "http://localhost:8080",
		Scopes: "read,write",
	}
	err := db.Create(client).Error
	require.NoError(t, err)

	// Test client store can retrieve the client
	clientStore := NewGormClientStore(db)
	ctx := context.Background()

	retrievedClient, err := clientStore.GetByID(ctx, "integration_test_client")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedClient)
}
