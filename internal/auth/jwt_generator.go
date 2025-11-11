package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// CustomJWTAccessGenerate generates JWT access tokens with custom claims including UserID and Role
type CustomJWTAccessGenerate struct {
	SignedKey    []byte
	SignedMethod jwt.SigningMethod
	DB           *gorm.DB // Database connection to fetch user information
}

// NewCustomJWTAccessGenerate creates a new custom JWT access token generator
func NewCustomJWTAccessGenerate(key []byte, method jwt.SigningMethod, db *gorm.DB) *CustomJWTAccessGenerate {
	return &CustomJWTAccessGenerate{
		SignedKey:    key,
		SignedMethod: method,
		DB:           db,
	}
}

// Token generates a JWT access token with custom claims
// This method is called by the OAuth2 library to generate access tokens
func (g *CustomJWTAccessGenerate) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	// Create base claims with standard fields
	claims := jwt.MapClaims{
		"aud": data.Client.GetID(),
		"exp": data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix(),
	}

	// Extract UserID from OAuth2 flow
	// For client_credentials flow, GenerateBasic.UserID is empty, so we get it from Client.GetUserID()
	// For other flows (authorization_code, password), it comes from GenerateBasic.UserID
	userID := data.UserID
	if userID == "" {
		userID = data.Client.GetUserID()
	}

	if userID == "" {
		return "", "", fmt.Errorf("cannot generate token: no user ID available")
	}

	claims["uid"] = userID

	// Fetch user role from database and include in token
	// This ensures the role is always accurate and prevents privilege escalation
	role, err := g.getUserRole(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch user role: %w", err)
	}
	claims["role"] = role

	// Add scope if present
	if data.TokenInfo.GetScope() != "" {
		claims["scope"] = data.TokenInfo.GetScope()
	}

	// Generate the access token
	token := jwt.NewWithClaims(g.SignedMethod, claims)
	access, err := token.SignedString(g.SignedKey)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token if requested
	refresh := ""
	if isGenRefresh {
		refreshClaims := jwt.MapClaims{
			"id":  data.TokenInfo.GetAccess(),
			"exp": data.TokenInfo.GetRefreshCreateAt().Add(data.TokenInfo.GetRefreshExpiresIn()).Unix(),
		}
		t := jwt.NewWithClaims(g.SignedMethod, refreshClaims)
		refresh, err = t.SignedString(g.SignedKey)
		if err != nil {
			return "", "", err
		}
	}

	return access, refresh, nil
}

// getUserRole fetches the user's role from the database
func (g *CustomJWTAccessGenerate) getUserRole(userIDStr string) (string, error) {
	// Parse userID string to uint
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return "", fmt.Errorf("invalid user ID format: %w", err)
	}

	// Fetch user from database
	var user models.User
	if err := g.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("user with ID %d not found", userID)
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// Validate role is set
	if user.Role == "" {
		// Default to 'user' role if not set (defensive programming)
		return "user", nil
	}

	return user.Role, nil
}
