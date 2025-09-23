package middleware

import (
	"net/http"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

func parseOAuth2Token(tokenString string, jwtSecret []byte) (jwt.MapClaims, error) {
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

func setOAuth2Context(c *gin.Context, claims jwt.MapClaims) {
    // subject -> userID
    if sub, ok := claims["sub"].(string); ok { c.Set("userID", sub) }

    // audience -> clientID (string or array)
    switch aud := claims["aud"].(type) {
    case string:
        c.Set("clientID", aud)
    case []interface{}:
        if len(aud) > 0 {
            if s, ok := aud[0].(string); ok { c.Set("clientID", s) }
        }
    }

    // optional scope if present
    if sc, ok := claims["scope"].(string); ok { c.Set("scopes", sc) }
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