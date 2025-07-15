package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// RequireRole is a middleware that checks if the user has the required role.
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context (set by JWTAuth middleware)
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Get role from JWT claims
		role, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "User role not found in token"})
			c.Abort()
			return
		}

		// Check if user has required role
		userRole, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role format"})
			c.Abort()
			return
		}

		if userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Insufficient permissions",
				"required_role": requiredRole,
				"user_role":     userRole,
				"user_id":       userID,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
