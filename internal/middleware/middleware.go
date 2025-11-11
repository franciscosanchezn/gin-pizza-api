package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// OAuth2Auth middleware that handles OAuth2 JWT access tokens
// This middleware validates JWT tokens and extracts user information from claims
// following RFC 6749 (OAuth2) and RFC 7519 (JWT) specifications
func OAuth2Auth(jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// RFC 6750: Extract Bearer token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondWithOAuth2Error(c, http.StatusUnauthorized, "authorization_required",
				"Missing Authorization header. A valid Bearer token is required.")
			return
		}

		// Validate Bearer scheme format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			respondWithOAuth2Error(c, http.StatusUnauthorized, "invalid_request",
				"Authorization header must use Bearer scheme. Format: 'Bearer <token>'")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			respondWithOAuth2Error(c, http.StatusUnauthorized, "invalid_token",
				"Bearer token is empty")
			return
		}

		// Parse and validate the JWT token
		claims, err := parseAndValidateJWT(tokenString, jwtSecret)
		if err != nil {
			respondWithOAuth2Error(c, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}

		// Extract and validate required claims, setting context
		if err := extractAndSetClaims(c, claims); err != nil {
			respondWithOAuth2Error(c, http.StatusUnauthorized, "invalid_token", err.Error())
			return
		}

		c.Next()
	}
}

// respondWithOAuth2Error responds with RFC 6750 compliant error format
func respondWithOAuth2Error(c *gin.Context, status int, errorCode, description string) {
	c.JSON(status, gin.H{
		"error":             errorCode,
		"error_description": description,
	})
	c.Abort()
}

// parseJWTToken validates and parses a JWT token using HMAC signing method
// Returns the claims if valid, error otherwise
func parseJWTToken(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
	// Parse with validation
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method to prevent algorithm confusion attacks
		// This protects against attacks where an attacker changes the algorithm header
		// See: https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v. Expected HMAC", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	// Extract and validate claims
	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims format")
	}

	return claims, nil
}

// parseAndValidateJWT parses the JWT and performs strict validation
func parseAndValidateJWT(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
	claims, err := parseJWTToken(tokenString, jwtSecret)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Validate token expiration (exp claim)
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("invalid exp claim: %w", err)
	}
	if exp != nil && exp.Before(now) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate not before (nbf claim) if present
	nbf, err := claims.GetNotBefore()
	if err != nil {
		return nil, fmt.Errorf("invalid nbf claim: %w", err)
	}
	if nbf != nil && nbf.After(now) {
		return nil, fmt.Errorf("token not yet valid")
	}

	// Validate issued at (iat claim) - prevents using tokens issued in the future
	iat, err := claims.GetIssuedAt()
	if err != nil {
		return nil, fmt.Errorf("invalid iat claim: %w", err)
	}
	if iat != nil && iat.After(now) {
		return nil, fmt.Errorf("token issued in the future")
	}

	return claims, nil
}

// extractAndSetClaims extracts user information from JWT claims and sets it in the Gin context
// This function follows strict validation rules to prevent security issues
func extractAndSetClaims(c *gin.Context, claims jwt.MapClaims) error {
	// Extract UserID - this is REQUIRED for all tokens
	// We support the "uid" claim (used by our OAuth2 implementation)
	userID, err := extractUserID(claims)
	if err != nil {
		return err
	}

	// Validate that userID is valid (non-zero)
	if userID == 0 {
		return fmt.Errorf("invalid user identifier: cannot be zero")
	}

	c.Set("userID", userID)

	// Extract and validate audience claim (aud) - helps prevent token misuse
	if aud, ok := claims["aud"].(string); ok && aud != "" {
		c.Set("clientID", aud)
	} else if audArray, ok := claims["aud"].([]interface{}); ok && len(audArray) > 0 {
		if firstAud, ok := audArray[0].(string); ok && firstAud != "" {
			c.Set("clientID", firstAud)
		}
	}

	// Extract role claim - STRICTLY required, no defaults
	role, err := extractRole(claims)
	if err != nil {
		return err
	}
	c.Set("userRole", role)

	// Extract optional scope claim
	if scope, ok := claims["scope"].(string); ok && scope != "" {
		c.Set("scopes", scope)
	}

	// Store token type for debugging/logging
	if clientID, _ := c.Get("clientID"); clientID != nil {
		c.Set("auth_type", "oauth2")
	} else {
		c.Set("auth_type", "jwt")
	}

	return nil
}

// extractUserID extracts and validates the user ID from JWT claims
// Supports the "uid" claim as the primary source (used by our OAuth2 implementation)
func extractUserID(claims jwt.MapClaims) (uint, error) {
	// Try "uid" claim first (OAuth2 client credentials flow)
	if uid, ok := claims["uid"].(string); ok && uid != "" {
		parsedID, err := strconv.ParseUint(uid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid uid claim format: must be a numeric string, got: %s", uid)
		}
		return uint(parsedID), nil
	}

	// Try "uid" as float64 (JSON numbers are parsed as float64)
	if uid, ok := claims["uid"].(float64); ok {
		if uid <= 0 {
			return 0, fmt.Errorf("invalid uid claim: must be positive, got: %f", uid)
		}
		return uint(uid), nil
	}

	// If no uid found, reject the token
	return 0, fmt.Errorf("token missing required 'uid' claim. This token is not valid for this API")
}

// extractRole extracts and validates the role from JWT claims
// All tokens must have an explicit role claim - no defaults are provided
func extractRole(claims jwt.MapClaims) (string, error) {
	role, ok := claims["role"].(string)
	if !ok || role == "" {
		return "", fmt.Errorf("token missing required 'role' claim. Tokens must explicitly specify user roles")
	}

	// Validate role against allowed values
	allowedRoles := map[string]bool{
		"admin": true,
		"user":  true,
	}

	if !allowedRoles[role] {
		return "", fmt.Errorf("invalid role '%s'. Allowed roles: admin, user", role)
	}

	return role, nil
}
