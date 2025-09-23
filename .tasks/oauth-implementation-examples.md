# OAuth 2.0 Implementation Guide with Examples

## Overview

This guide provides detailed implementation examples for each step of the OAuth 2.0 authentication service implementation plan. Based on the current codebase analysis, we'll build upon the existing JWT middleware and GORM models.

## Prerequisites

Add the following dependencies to `go.mod`:

```bash
go get github.com/go-oauth2/oauth2/v4
go get github.com/go-oauth2/oauth2/v4/generates
go get github.com/go-oauth2/oauth2/v4/manage
go get github.com/go-oauth2/oauth2/v4/models
go get github.com/go-oauth2/oauth2/v4/server
go get github.com/go-oauth2/oauth2/v4/store
go get golang.org/x/crypto/bcrypt
```

## Step 1: Database Schema Extensions

### Implementation

Create `internal/models/oauth_client.go`:

```go
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
	RedirectURI string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (OAuthClient) TableName() string {
	return "oauth_clients"
}
```

Create `internal/models/oauth_code.go`:

```go
package models

import (
	"gorm.io/gorm"
	"time"
)

type OAuthCode struct {
	Code      string `gorm:"primaryKey"`
	ClientID  string `gorm:"not null"`
	UserID    string `gorm:"not null"`
	Scopes    string
	RedirectURI string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (OAuthCode) TableName() string {
	return "oauth_codes"
}
```

Create `internal/models/oauth_token.go`:

```go
package models

import (
	"gorm.io/gorm"
	"time"
)

type OAuthToken struct {
	ID           uint `gorm:"primaryKey"`
	ClientID     string `gorm:"not null"`
	UserID       *string // Nullable for client credentials
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
```

### Migration

Update `cmd/main.go` to include migrations:

```go
// In setupDatabase function, after db.AutoMigrate(&models.Pizza{})
db.AutoMigrate(&models.OAuthClient{}, &models.OAuthCode{}, &models.OAuthToken{})
```

## Step 2: OAuth 2.0 Server Setup

### Implementation

Create `internal/auth/oauth_server.go`:

```go
package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"your-project/internal/models"
)

type OAuthService struct {
	server *server.Server
	db     *gorm.DB
}

func NewOAuthService(db *gorm.DB, jwtSecret string) *OAuthService {
	manager := manage.NewDefaultManager()

	// Use JWT for access tokens
	manager.MapAccessGenerate(generates.NewJWTAccessGenerate("", []byte(jwtSecret), jwt.SigningMethodHS512))

	// Configure token store
	tokenStore := NewGormTokenStore(db)
	manager.MustTokenStorage(tokenStore)

	// Configure client store
	clientStore := NewGormClientStore(db)
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	return &OAuthService{
		server: srv,
		db:     db,
	}
}

func (o *OAuthService) GetServer() *server.Server {
	return o.server
}
```

Create `internal/auth/gorm_store.go`:

```go
package auth

import (
	"context"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
	"gorm.io/gorm"
	"time"
	"your-project/internal/models"
)

type GormClientStore struct {
	db *gorm.DB
}

func NewGormClientStore(db *gorm.DB) *GormClientStore {
	return &GormClientStore{db: db}
}

func (s *GormClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	var client models.OAuthClient
	if err := s.db.Where("id = ?", id).First(&client).Error; err != nil {
		return nil, err
	}

	return &models.Client{
		ID:     client.ID,
		Secret: client.Secret,
		Domain: client.Domain,
	}, nil
}

type GormTokenStore struct {
	db *gorm.DB
}

func NewGormTokenStore(db *gorm.DB) *GormTokenStore {
	return &GormTokenStore{db: db}
}

func (s *GormTokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {
	token := &models.OAuthToken{
		ClientID:    info.GetClientID(),
		UserID:      info.GetUserID(),
		AccessToken: info.GetAccess(),
		RefreshToken: info.GetRefresh(),
		Scopes:      info.GetScope(),
		ExpiresAt:   info.GetAccessExpiresIn(),
	}

	return s.db.Create(token).Error
}

func (s *GormTokenStore) RemoveByAccess(ctx context.Context, access string) error {
	return s.db.Where("access_token = ?", access).Delete(&models.OAuthToken{}).Error
}

func (s *GormTokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	return s.db.Where("refresh_token = ?", refresh).Delete(&models.OAuthToken{}).Error
}

func (s *GormTokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	var token models.OAuthToken
	if err := s.db.Where("access_token = ?", access).First(&token).Error; err != nil {
		return nil, err
	}

	return &models.Token{
		ClientID:         token.ClientID,
		UserID:           token.UserID,
		Access:           token.AccessToken,
		Refresh:          token.RefreshToken,
		AccessExpiresIn:  token.ExpiresAt,
		Scope:            token.Scopes,
	}, nil
}

func (s *GormTokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	var token models.OAuthToken
	if err := s.db.Where("refresh_token = ?", refresh).First(&token).Error; err != nil {
		return nil, err
	}

	return &models.Token{
		ClientID:         token.ClientID,
		UserID:           token.UserID,
		Access:           token.AccessToken,
		Refresh:          token.RefreshToken,
		AccessExpiresIn:  token.ExpiresAt,
		Scope:            token.Scopes,
	}, nil
}
```

## Step 3: Client Credentials Grant Implementation

### Implementation

Create `internal/auth/client_credentials.go`:

```go
package auth

import (
	"net/http"
	"your-project/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (o *OAuthService) HandleClientCredentials(c *gin.Context) {
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	grantType := c.PostForm("grant_type")

	if grantType != "client_credentials" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	// Validate client
	var client models.OAuthClient
	if err := o.db.Where("id = ?", clientID).First(&client).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Verify client secret
	if err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Generate token using OAuth2 server
	ti, err := o.server.Manager.GenerateAccessToken(oauth2.ClientCredentials, &oauth2.TokenGenerateRequest{
		ClientID: clientID,
		Scope:    client.Scopes,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": ti.GetAccess(),
		"token_type":   "Bearer",
		"expires_in":   int64(ti.GetAccessExpiresIn().Sub(time.Now()).Seconds()),
		"scope":        ti.GetScope(),
	})
}
```

Update `cmd/main.go` to add the endpoint:

```go
// In setupRoutes function
authService := auth.NewOAuthService(db, configuration.JWTSecret)

// OAuth 2.0 endpoints
v1.Group("/oauth").POST("/token", authService.HandleClientCredentials)
```

## Step 4: Authorization Code Grant Setup

### Implementation

Create `internal/auth/authorization_code.go`:

```go
package auth

import (
	"net/http"
	"net/url"
	"time"
	"your-project/internal/models"
	"github.com/google/uuid"
)

func (o *OAuthService) HandleAuthorize(c *gin.Context) {
	clientID := c.Query("client_id")
	responseType := c.Query("response_type")
	redirectURI := c.Query("redirect_uri")
	scope := c.Query("scope")
	state := c.Query("state")

	// Validate client
	var client models.OAuthClient
	if err := o.db.Where("id = ?", clientID).First(&client).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		return
	}

	// Validate redirect URI
	if redirectURI != "" && redirectURI != client.RedirectURI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		return
	}

	// For now, assume user is authenticated (you'll implement proper auth later)
	userID := c.GetString("userID")
	if userID == "" {
		// Redirect to login page
		loginURL := "/login?redirect=" + url.QueryEscape(c.Request.URL.String())
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	// Generate authorization code
	code := uuid.New().String()
	authCode := &models.OAuthCode{
		Code:      code,
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    scope,
		RedirectURI: redirectURI,
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10 minutes
	}

	if err := o.db.Create(authCode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "code_generation_failed"})
		return
	}

	// Redirect back to client with authorization code
	redirectURL := redirectURI + "?code=" + code
	if state != "" {
		redirectURL += "&state=" + state
	}

	c.Redirect(http.StatusFound, redirectURL)
}
```

Add to routes:

```go
// In setupRoutes function
v1.Group("/oauth").GET("/authorize", authService.HandleAuthorize)
```

## Step 5: Token Exchange Endpoint

### Implementation

Extend the token endpoint to handle authorization codes:

```go
// In client_credentials.go, rename to HandleToken and extend
func (o *OAuthService) HandleToken(c *gin.Context) {
	grantType := c.PostForm("grant_type")

	switch grantType {
	case "client_credentials":
		o.handleClientCredentials(c)
	case "authorization_code":
		o.handleAuthorizationCode(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
	}
}

func (o *OAuthService) handleAuthorizationCode(c *gin.Context) {
	code := c.PostForm("code")
	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")
	redirectURI := c.PostForm("redirect_uri")

	// Validate authorization code
	var authCode models.OAuthCode
	if err := o.db.Where("code = ?", code).First(&authCode).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}

	// Check expiration
	if time.Now().After(authCode.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}

	// Validate client
	if authCode.ClientID != clientID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
		return
	}

	var client models.OAuthClient
	if err := o.db.Where("id = ?", clientID).First(&client).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Verify client secret
	if err := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(clientSecret)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	// Generate tokens
	ti, err := o.server.Manager.GenerateAccessToken(oauth2.AuthorizationCode, &oauth2.TokenGenerateRequest{
		ClientID: clientID,
		UserID:   &authCode.UserID,
		Scope:    authCode.Scopes,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	// Delete used authorization code
	o.db.Delete(&authCode)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  ti.GetAccess(),
		"token_type":    "Bearer",
		"expires_in":    int64(ti.GetAccessExpiresIn().Sub(time.Now()).Seconds()),
		"refresh_token": ti.GetRefresh(),
		"scope":         ti.GetScope(),
	})
}
```

## Step 6: User Management System

### Implementation

Extend User model in `internal/models/user.go`:

```go
package models

import (
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	Password  string `gorm:"not null"`
	Name      string
	Role      string `gorm:"default:'user'"`
	IsActive  bool   `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) HashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
```

Create `internal/controllers/auth_controller.go`:

```go
package controllers

import (
	"net/http"
	"your-project/internal/models"
	"your-project/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthController struct {
	userService services.UserService
	jwtSecret   []byte
}

func NewAuthController(userService services.UserService, jwtSecret string) *AuthController {
	return &AuthController{
		userService: userService,
		jwtSecret:   []byte(jwtSecret),
	}
}

func (ac *AuthController) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &models.User{
		Email: req.Email,
		Password: req.Password,
		Name:  req.Name,
	}

	if err := user.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password_hashing_failed"})
		return
	}

	if err := ac.userService.CreateUser(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user_already_exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user_created"})
}

func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ac.userService.GetUserByEmail(req.Email)
	if err != nil || !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString(ac.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   86400,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}
```

Create `internal/services/user_service.go`:

```go
package services

import (
	"errors"
	"your-project/internal/models"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
}

type userService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) UserService {
	return &userService{db: db}
}

func (s *userService) CreateUser(user *models.User) error {
	var existing models.User
	if err := s.db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
		return errors.New("user_already_exists")
	}

	return s.db.Create(user).Error
}

func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
```

## Step 7: OAuth Middleware Integration

### Implementation

Update `internal/middleware/middleware.go`:

```go
package middleware

import (
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/go-oauth2/oauth2/v4/generates"
)

// OAuth2Auth middleware that handles both JWT and OAuth2 tokens
func OAuth2Auth(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization_header_required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_authorization_header_format"})
			c.Abort()
			return
		}

		// Try to parse as OAuth2 JWT token first
		if claims, err := parseOAuth2Token(tokenString, jwtSecret); err == nil {
			setOAuth2Context(c, claims)
			c.Next()
			return
		}

		// Fallback to regular JWT parsing
		if claims, err := parseJWTToken(tokenString, jwtSecret); err == nil {
			setJWTContext(c, claims)
			c.Next()
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		c.Abort()
	}
}

func parseOAuth2Token(tokenString string, jwtSecret []byte) (*generates.JWTAccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &generates.JWTAccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*generates.JWTAccessClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

func parseJWTToken(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

func setOAuth2Context(c *gin.Context, claims *generates.JWTAccessClaims) {
	c.Set("userID", claims.Subject)
	c.Set("clientID", claims.Audience)
	c.Set("scopes", claims.Scope)
	c.Set("auth_type", "oauth2")
}

func setJWTContext(c *gin.Context, claims jwt.MapClaims) {
	if user, ok := claims["user"].(float64); ok {
		c.Set("userID", uint(user))
	}
	if role, ok := claims["role"].(string); ok {
		c.Set("userRole", role)
	}
	c.Set("auth_type", "jwt")
}
```

## Step 8: Client Management Interface

### Implementation

Create `internal/controllers/client_controller.go`:

```go
package controllers

import (
	"net/http"
	"strconv"
	"your-project/internal/models"
	"your-project/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

type ClientController struct {
	clientService services.ClientService
}

func NewClientController(clientService services.ClientService) *ClientController {
	return &ClientController{clientService: clientService}
}

func (cc *ClientController) CreateClient(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Domain     string `json:"domain"`
		Scopes     string `json:"scopes"`
		GrantTypes string `json:"grant_types"`
		RedirectURI string `json:"redirect_uri"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate client secret
	secret := uuid.New().String()
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "secret_generation_failed"})
		return
	}

	client := &models.OAuthClient{
		ID:         uuid.New().String(),
		Secret:     string(hashedSecret),
		Name:       req.Name,
		Domain:     req.Domain,
		Scopes:     req.Scopes,
		GrantTypes: req.GrantTypes,
		RedirectURI: req.RedirectURI,
		UserID:     c.GetUint("userID"),
	}

	if err := cc.clientService.CreateClient(client); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "client_creation_failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client_id":     client.ID,
		"client_secret": secret, // Return plain secret only once
		"name":          client.Name,
		"scopes":        client.Scopes,
		"grant_types":   client.GrantTypes,
		"redirect_uri":  client.RedirectURI,
	})
}

func (cc *ClientController) ListClients(c *gin.Context) {
	userID := c.GetUint("userID")
	clients, err := cc.clientService.GetClientsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_retrieve_clients"})
		return
	}

	c.JSON(http.StatusOK, clients)
}

func (cc *ClientController) DeleteClient(c *gin.Context) {
	clientID := c.Param("id")
	userID := c.GetUint("userID")

	if err := cc.clientService.DeleteClient(clientID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client_not_found"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
```

Create `internal/services/client_service.go`:

```go
package services

import (
	"errors"
	"your-project/internal/models"
	"gorm.io/gorm"
)

type ClientService interface {
	CreateClient(client *models.OAuthClient) error
	GetClientsByUserID(userID uint) ([]models.OAuthClient, error)
	GetClientByID(id string) (*models.OAuthClient, error)
	DeleteClient(clientID string, userID uint) error
}

type clientService struct {
	db *gorm.DB
}

func NewClientService(db *gorm.DB) ClientService {
	return &clientService{db: db}
}

func (s *clientService) CreateClient(client *models.OAuthClient) error {
	return s.db.Create(client).Error
}

func (s *clientService) GetClientsByUserID(userID uint) ([]models.OAuthClient, error) {
	var clients []models.OAuthClient
	if err := s.db.Where("user_id = ?", userID).Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

func (s *clientService) GetClientByID(id string) (*models.OAuthClient, error) {
	var client models.OAuthClient
	if err := s.db.Where("id = ?", id).First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *clientService) DeleteClient(clientID string, userID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", clientID, userID).Delete(&models.OAuthClient{})
	if result.RowsAffected == 0 {
		return errors.New("client_not_found")
	}
	return result.Error
}
```

## Comprehensive Testing Guide

### Test Setup and Teardown

Create `internal/auth/oauth_test.go`:

```go
package auth

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"your-project/internal/models"
)

type OAuthTestSuite struct {
	suite.Suite
	db           *gorm.DB
	oauthService *OAuthService
	testClient   *models.OAuthClient
}

func (suite *OAuthTestSuite) SetupTest() {
	// Setup in-memory database for each test
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Run migrations
	err = db.AutoMigrate(&models.OAuthClient{}, &models.OAuthCode{}, &models.OAuthToken{})
	suite.Require().NoError(err)

	// Create OAuth service
	suite.oauthService = NewOAuthService(db, "test-jwt-secret")
	suite.db = db

	// Create test client
	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("test-secret"), bcrypt.DefaultCost)
	suite.testClient = &models.OAuthClient{
		ID:         "test-client-id",
		Secret:     string(hashedSecret),
		Name:       "Test Client",
		Domain:     "http://localhost:8080",
		Scopes:     "read write",
		GrantTypes: "client_credentials authorization_code",
		RedirectURI: "http://localhost:8080/callback",
	}
	suite.db.Create(suite.testClient)
}

func (suite *OAuthTestSuite) TearDownTest() {
	// Clean up database
	suite.db.Exec("DELETE FROM oauth_clients")
	suite.db.Exec("DELETE FROM oauth_codes")
	suite.db.Exec("DELETE FROM oauth_tokens")
}

func TestOAuthTestSuite(t *testing.T) {
	suite.Run(t, new(OAuthTestSuite))
}
```

### Unit Tests for Client Store

```go
func (suite *OAuthTestSuite) TestGormClientStore_GetByID() {
	client, err := suite.oauthService.clientStore.GetByID(context.Background(), "test-client-id")
	
	suite.NoError(err)
	suite.NotNil(client)
	suite.Equal("test-client-id", client.GetID())
	suite.Equal("http://localhost:8080", client.GetDomain())
}

func (suite *OAuthTestSuite) TestGormClientStore_GetByID_NotFound() {
	client, err := suite.oauthService.clientStore.GetByID(context.Background(), "non-existent")
	
	suite.Error(err)
	suite.Nil(client)
}
```

### Unit Tests for Token Store

```go
func (suite *OAuthTestSuite) TestGormTokenStore_CreateAndRetrieve() {
	// Create token info
	tokenInfo := &models.Token{
		ClientID:         "test-client-id",
		UserID:           stringPtr("user-123"),
		Access:           "access-token-123",
		Refresh:          stringPtr("refresh-token-123"),
		AccessExpiresIn:  time.Now().Add(time.Hour),
		Scope:            "read write",
	}

	// Test creation
	err := suite.oauthService.tokenStore.Create(context.Background(), tokenInfo)
	suite.NoError(err)

	// Test retrieval by access token
	retrieved, err := suite.oauthService.tokenStore.GetByAccess(context.Background(), "access-token-123")
	suite.NoError(err)
	suite.Equal("test-client-id", retrieved.GetClientID())
	suite.Equal("user-123", *retrieved.GetUserID())
	suite.Equal("read write", retrieved.GetScope())
}

func (suite *OAuthTestSuite) TestGormTokenStore_RemoveByAccess() {
	// Create token
	tokenInfo := &models.Token{
		ClientID: "test-client-id",
		Access:   "token-to-remove",
		AccessExpiresIn: time.Now().Add(time.Hour),
	}
	suite.oauthService.tokenStore.Create(context.Background(), tokenInfo)

	// Remove token
	err := suite.oauthService.tokenStore.RemoveByAccess(context.Background(), "token-to-remove")
	suite.NoError(err)

	// Verify removal
	_, err = suite.oauthService.tokenStore.GetByAccess(context.Background(), "token-to-remove")
	suite.Error(err)
}
```

### Integration Tests for OAuth Flows

Create `internal/auth/oauth_integration_test.go`:

```go
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"your-project/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func setupTestOAuthService() (*OAuthService, *gorm.DB) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&models.OAuthClient{}, &models.OAuthCode{}, &models.OAuthToken{})

	oauthService := NewOAuthService(db, "test-jwt-secret")
	return oauthService, db
}

func createTestClient(db *gorm.DB) *models.OAuthClient {
	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("test-secret"), bcrypt.DefaultCost)
	client := &models.OAuthClient{
		ID:         "test-client",
		Secret:     string(hashedSecret),
		Name:       "Test Client",
		Scopes:     "read write",
		GrantTypes: "client_credentials",
	}
	db.Create(client)
	return client
}

func TestClientCredentialsFlow(t *testing.T) {
	oauthService, db := setupTestOAuthService()
	createTestClient(db)

	// Create test router
	router := gin.New()
	router.POST("/oauth/token", func(c *gin.Context) {
		oauthService.HandleToken(c)
	})

	// Test client credentials request
	reqBody := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     "test-client",
		"client_secret": "test-secret",
		"scope":         "read",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/oauth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "token_type")
	assert.Contains(t, response, "expires_in")
	assert.Equal(t, "Bearer", response["token_type"])
}

func TestInvalidClientCredentials(t *testing.T) {
	oauthService, _ := setupTestOAuthService()

	router := gin.New()
	router.POST("/oauth/token", func(c *gin.Context) {
		oauthService.HandleToken(c)
	})

	reqBody := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     "invalid-client",
		"client_secret": "wrong-secret",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/oauth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
}
```

### Authorization Code Flow Tests

```go
func TestAuthorizationCodeFlow(t *testing.T) {
	oauthService, db := setupTestOAuthService()
	client := createTestClient(db)
	client.GrantTypes = "authorization_code"
	db.Save(client)

	router := gin.New()
	router.GET("/oauth/authorize", func(c *gin.Context) {
		// Mock authenticated user
		c.Set("userID", "test-user-123")
		oauthService.HandleAuthorize(c)
	})
	router.POST("/oauth/token", func(c *gin.Context) {
		oauthService.HandleToken(c)
	})

	// Step 1: Get authorization code
	req1, _ := http.NewRequest("GET", "/oauth/authorize?client_id=test-client&response_type=code&redirect_uri=http://localhost:8080/callback", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Should redirect with code
	assert.Equal(t, http.StatusFound, w1.Code)
	location := w1.Header().Get("Location")
	assert.Contains(t, location, "code=")

	// Extract code from redirect URL
	code := extractCodeFromURL(location)

	// Step 2: Exchange code for token
	reqBody := map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     "test-client",
		"client_secret": "test-secret",
		"code":          code,
		"redirect_uri":  "http://localhost:8080/callback",
	}

	body, _ := json.Marshal(reqBody)
	req2, _ := http.NewRequest("POST", "/oauth/token", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var response map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &response)
	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "refresh_token")
}

func extractCodeFromURL(url string) string {
	// Helper function to extract authorization code from redirect URL
	// Implementation depends on URL parsing logic
	return "extracted-code"
}
```

### Middleware Testing

Create `internal/middleware/middleware_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func TestOAuth2Auth_ValidOAuth2Token(t *testing.T) {
	router := gin.New()
	jwtSecret := []byte("test-secret")

	router.Use(OAuth2Auth(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		assert.True(t, exists)
		assert.Equal(t, "test-user", userID)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create test OAuth2 token
	claims := &generates.JWTAccessClaims{
		Subject:  "test-user",
		Audience: []string{"test-client"},
		Scope:    "read write",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtSecret)

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOAuth2Auth_InvalidToken(t *testing.T) {
	router := gin.New()
	jwtSecret := []byte("test-secret")

	router.Use(OAuth2Auth(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
}

func TestOAuth2Auth_MissingHeader(t *testing.T) {
	router := gin.New()
	jwtSecret := []byte("test-secret")

	router.Use(OAuth2Auth(jwtSecret))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

### User Service Testing

Create `internal/services/user_service_test.go`:

```go
package services

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"your-project/internal/models"
)

type UserServiceTestSuite struct {
	suite.Suite
	db           *gorm.DB
	userService  UserService
	testUser     *models.User
}

func (suite *UserServiceTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	err = db.AutoMigrate(&models.User{})
	suite.Require().NoError(err)

	suite.db = db
	suite.userService = NewUserService(db)

	// Create test user
	suite.testUser = &models.User{
		Email:    "test@example.com",
		Password: "hashed-password",
		Name:     "Test User",
		Role:     "user",
	}
	suite.db.Create(suite.testUser)
}

func (suite *UserServiceTestSuite) TestCreateUser() {
	newUser := &models.User{
		Email:    "new@example.com",
		Password: "password123",
		Name:     "New User",
	}

	err := suite.userService.CreateUser(newUser)
	suite.NoError(err)

	// Verify user was created
	created, err := suite.userService.GetUserByEmail("new@example.com")
	suite.NoError(err)
	suite.Equal("New User", created.Name)
}

func (suite *UserServiceTestSuite) TestCreateUser_DuplicateEmail() {
	duplicateUser := &models.User{
		Email:    "test@example.com", // Same as existing user
		Password: "password123",
		Name:     "Duplicate User",
	}

	err := suite.userService.CreateUser(duplicateUser)
	suite.Error(err)
	suite.Contains(err.Error(), "user_already_exists")
}

func (suite *UserServiceTestSuite) TestGetUserByEmail() {
	user, err := suite.userService.GetUserByEmail("test@example.com")
	suite.NoError(err)
	suite.Equal("Test User", user.Name)
	suite.Equal("user", user.Role)
}

func (suite *UserServiceTestSuite) TestGetUserByEmail_NotFound() {
	user, err := suite.userService.GetUserByEmail("nonexistent@example.com")
	suite.Error(err)
	suite.Nil(user)
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
```

### End-to-End Testing

Create `tests/e2e/oauth_e2e_test.go`:

```go
package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"your-project/cmd"
)

func TestFullOAuthFlow(t *testing.T) {
	// This would run against a fully configured test server
	// For demonstration purposes, showing the test structure

	t.Run("Complete Client Credentials Flow", func(t *testing.T) {
		// 1. Register OAuth client (admin operation)
		adminToken := getAdminToken()
		clientData := map[string]interface{}{
			"name":        "E2E Test Client",
			"scopes":      "read write",
			"grant_types": "client_credentials",
		}

		clientResp := makeAuthenticatedRequest(t, "POST", "/api/v1/admin/clients", clientData, adminToken)
		assert.Equal(t, http.StatusCreated, clientResp.StatusCode)

		var clientInfo map[string]interface{}
		json.NewDecoder(clientResp.Body).Decode(&clientInfo)

		clientID := clientInfo["client_id"].(string)
		clientSecret := clientInfo["client_secret"].(string)

		// 2. Use client credentials to get access token
		tokenData := map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     clientID,
			"client_secret": clientSecret,
			"scope":         "read",
		}

		tokenResp := makeRequest(t, "POST", "/api/v1/oauth/token", tokenData)
		assert.Equal(t, http.StatusOK, tokenResp.StatusCode)

		var tokenInfo map[string]interface{}
		json.NewDecoder(tokenResp.Body).Decode(&tokenInfo)

		accessToken := tokenInfo["access_token"].(string)

		// 3. Use access token to access protected resource
		protectedResp := makeAuthenticatedRequest(t, "GET", "/api/v1/protected/pizzas", nil, accessToken)
		assert.Equal(t, http.StatusOK, protectedResp.StatusCode)
	})

	t.Run("Complete Authorization Code Flow", func(t *testing.T) {
		// 1. User registration and login
		userData := map[string]interface{}{
			"email":    "testuser@example.com",
			"password": "password123",
			"name":     "Test User",
		}

		regResp := makeRequest(t, "POST", "/api/v1/auth/register", userData)
		assert.Equal(t, http.StatusCreated, regResp.StatusCode)

		loginResp := makeRequest(t, "POST", "/api/v1/auth/login", userData)
		assert.Equal(t, http.StatusOK, loginResp.StatusCode)

		var loginInfo map[string]interface{}
		json.NewDecoder(loginResp.Body).Decode(&loginInfo)

		userToken := loginInfo["access_token"].(string)

		// 2. Create OAuth client for authorization code flow
		clientData := map[string]interface{}{
			"name":         "Auth Code Test Client",
			"scopes":       "read write",
			"grant_types":  "authorization_code",
			"redirect_uri": "http://localhost:8080/callback",
		}

		clientResp := makeAuthenticatedRequest(t, "POST", "/api/v1/admin/clients", clientData, userToken)
		assert.Equal(t, http.StatusCreated, clientResp.StatusCode)

		var clientInfo map[string]interface{}
		json.NewDecoder(clientResp.Body).Decode(&clientInfo)

		clientID := clientInfo["client_id"].(string)

		// 3. Initiate authorization request
		authResp := makeAuthenticatedRequest(t, "GET", "/api/v1/oauth/authorize?client_id="+clientID+"&response_type=code&redirect_uri=http://localhost:8080/callback", nil, userToken)
		assert.Equal(t, http.StatusFound, authResp.StatusCode)

		// 4. Extract authorization code from redirect
		location := authResp.Header.Get("Location")
		code := extractAuthCodeFromRedirect(location)

		// 5. Exchange code for tokens
		tokenData := map[string]string{
			"grant_type":   "authorization_code",
			"client_id":    clientID,
			"code":         code,
			"redirect_uri": "http://localhost:8080/callback",
		}

		tokenResp := makeRequest(t, "POST", "/api/v1/oauth/token", tokenData)
		assert.Equal(t, http.StatusOK, tokenResp.StatusCode)

		var tokenInfo map[string]interface{}
		json.NewDecoder(tokenResp.Body).Decode(&tokenInfo)

		assert.Contains(t, tokenInfo, "access_token")
		assert.Contains(t, tokenInfo, "refresh_token")
	})
}

// Helper functions for E2E tests
func getAdminToken() string {
	// Implementation to get admin token for testing
	return "admin-jwt-token"
}

func makeRequest(t *testing.T, method, url string, data interface{}) *http.Response {
	// Implementation to make HTTP requests to test server
	return nil
}

func makeAuthenticatedRequest(t *testing.T, method, url string, data interface{}, token string) *http.Response {
	// Implementation to make authenticated HTTP requests
	return nil
}

func extractAuthCodeFromRedirect(redirectURL string) string {
	// Implementation to extract authorization code from redirect URL
	return "auth-code"
}
```

### Manual Testing Examples

#### Using curl for Client Credentials Flow

```bash
# 1. Create an OAuth client (requires admin token)
curl -X POST http://localhost:8080/api/v1/admin/clients \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Client",
    "scopes": "read write",
    "grant_types": "client_credentials"
  }'

# Response will include client_id and client_secret
# {"client_id":"abc123","client_secret":"def456",...}

# 2. Get access token using client credentials
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d 'grant_type=client_credentials&client_id=abc123&client_secret=def456&scope=read'

# Response: {"access_token":"xyz789","token_type":"Bearer","expires_in":3600}

# 3. Use access token to access protected resources
curl -X GET http://localhost:8080/api/v1/protected/pizzas \
  -H "Authorization: Bearer xyz789"
```

#### Using curl for Authorization Code Flow

```bash
# 1. Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "Test User"
  }'

# 2. Login to get user token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'

# Use the returned access_token as YOUR_USER_TOKEN

# 3. Create OAuth client for auth code flow
curl -X POST http://localhost:8080/api/v1/admin/clients \
  -H "Authorization: Bearer YOUR_USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Web App Client",
    "scopes": "read write",
    "grant_types": "authorization_code",
    "redirect_uri": "http://localhost:3000/callback"
  }'

# 4. Initiate authorization (this would normally be done in a browser)
curl -X GET "http://localhost:8080/api/v1/oauth/authorize?client_id=YOUR_CLIENT_ID&response_type=code&redirect_uri=http://localhost:3000/callback" \
  -H "Authorization: Bearer YOUR_USER_TOKEN"

# This will redirect to your redirect_uri with a code parameter
# e.g., http://localhost:3000/callback?code=abc123

# 5. Exchange authorization code for tokens
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type": application/x-www-form-urlencoded" \
  -d 'grant_type=authorization_code&client_id=YOUR_CLIENT_ID&client_secret=YOUR_CLIENT_SECRET&code=abc123&redirect_uri=http://localhost:3000/callback'
```

### Test Data Setup

Create `tests/testdata/setup.go`:

```go
package testdata

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"your-project/internal/models"
)

func SetupTestData(db *gorm.DB) error {
	// Create test users
	users := []models.User{
		{
			Email:    "admin@example.com",
			Password: hashPassword("admin123"),
			Name:     "Admin User",
			Role:     "admin",
			IsActive: true,
		},
		{
			Email:    "user@example.com",
			Password: hashPassword("user123"),
			Name:     "Regular User",
			Role:     "user",
			IsActive: true,
		},
	}

	for _, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
	}

	// Create test OAuth clients
	clients := []models.OAuthClient{
		{
			ID:         "client-credentials-client",
			Secret:     hashPassword("client-secret-123"),
			Name:       "Client Credentials Client",
			Scopes:     "read write",
			GrantTypes: "client_credentials",
			UserID:     1, // Admin user
		},
		{
			ID:         "auth-code-client",
			Secret:     hashPassword("auth-secret-456"),
			Name:       "Auth Code Client",
			Scopes:     "read write",
			GrantTypes: "authorization_code",
			RedirectURI: "http://localhost:3000/callback",
			UserID:     1, // Admin user
		},
	}

	for _, client := range clients {
		if err := db.Create(&client).Error; err != nil {
			return err
		}
	}

	return nil
}

func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

func TeardownTestData(db *gorm.DB) error {
	// Clean up test data
	tables := []string{"oauth_tokens", "oauth_codes", "oauth_clients", "users"}
	for _, table := range tables {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			return err
		}
	}
	return nil
}
```

### Running Tests

Create `Makefile` for test execution:

```makefile
.PHONY: test test-unit test-integration test-e2e test-coverage

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test ./internal/... -v -short

# Run integration tests
test-integration:
	go test ./internal/... -v -run Integration

# Run end-to-end tests
test-e2e:
	go test ./tests/e2e/... -v

# Run tests with coverage
test-coverage:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run specific test suite
test-oauth:
	go test ./internal/auth/... -v

test-middleware:
	go test ./internal/middleware/... -v

test-services:
	go test ./internal/services/... -v
```

### CI/CD Integration

Create `.github/workflows/test.yml`:

```yaml
name: Test OAuth Implementation

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21

    - name: Cache dependencies
      uses: actions/cache@v3
      with:
        path: |
          ~/.cache/go-build
          ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-

    - name: Run unit tests
      run: make test-unit

    - name: Run integration tests
      run: make test-integration
      env:
        DATABASE_URL: postgres://postgres:postgres@localhost:5432/testdb

    - name: Generate coverage report
      run: make test-coverage

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

This comprehensive testing guide covers:
- Unit tests for individual components
- Integration tests for OAuth flows
- End-to-end tests for complete user journeys
- Manual testing examples with curl
- Test data setup and teardown
- CI/CD integration
- Coverage reporting

The tests ensure that each OAuth 2.0 component works correctly in isolation and together, providing confidence in the implementation's reliability and security.

## Configuration Updates

Update `internal/config/config.go` to include OAuth settings:

```go
type Config struct {
	// ... existing fields ...
	
	// OAuth 2.0 Configuration
	OAuthClientID     string `json:"oauth_client_id"`
	OAuthClientSecret string `json:"oauth_client_secret"`
	OAuthIssuer       string `json:"oauth_issuer"`
	OAuthScopes       string `json:"oauth_scopes"`
}
```

## References

1. [go-oauth2/oauth2 GitHub Repository](https://github.com/go-oauth2/oauth2)
2. [OAuth 2.0 Authorization Framework (RFC 6749)](https://tools.ietf.org/html/rfc6749)
3. [OAuth 2.0 Security Best Current Practice](https://tools.ietf.org/html/rfc8725)
4. [Gin Web Framework Documentation](https://gin-gonic.com/docs/)
5. [GORM Documentation](https://gorm.io/docs/)

This implementation provides a complete OAuth 2.0 authentication service that integrates seamlessly with your existing Gin API while maintaining backward compatibility with JWT tokens.
