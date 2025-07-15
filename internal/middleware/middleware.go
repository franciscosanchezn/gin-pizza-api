package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
)

// JWTSecret holds the JWT secret key
var JWTSecret []byte

// SetJWTSecret sets the JWT secret from configuration
func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

// JWTAuth is a middleware that checks for a valid JWT token in the request header.
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format. Use 'Bearer <token>'"})
			c.Abort()
			return
		}

		// 2. Validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 3. Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// 4. Set user context
		if user, ok := claims["user"].(string); ok {
			c.Set("userID", user)
		}
		if role, ok := claims["role"].(string); ok {
			c.Set("userRole", role)
		}

		c.Next()
	}
}
