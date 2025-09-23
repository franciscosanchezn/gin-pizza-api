package auth

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "golang.org/x/crypto/bcrypt"
    "github.com/franciscosanchezn/gin-pizza-api/internal/models"
)

func TestClientCredentialsFlow(t *testing.T) {
    db := setupTestDB(t)
    
    // Fix: Provide JWT secret parameter
    oauthService := NewOAuthService(db, "test-jwt-secret-key-32-characters")
    require.NotNil(t, oauthService)
    
    // CRITICAL FIX: Test with bcrypt-hashed secret (production scenario)
    hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("test_secret"), bcrypt.DefaultCost)
    client := &models.OAuthClient{
        ID:     "test_client_id",
        Secret: string(hashedSecret), // bcrypt hash stored in database
        Domain: "http://localhost:8080",
        Scopes: "read,write",
    }
    err := db.Create(client).Error
    require.NoError(t, err)
    
    // Setup Gin for testing
    gin.SetMode(gin.TestMode)
    router := gin.New()
    
    // Add OAuth token endpoint
    router.POST("/oauth/token", func(c *gin.Context) {
        err := oauthService.GetServer().HandleTokenRequest(c.Writer, c.Request)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        }
    })
    
    // Test token request with form data (OAuth2 standard)
    // The plain text secret will be verified against the bcrypt hash
    tokenReq := "grant_type=client_credentials&client_id=test_client_id&client_secret=test_secret&scope=read"
    
    req := httptest.NewRequest("POST", "/oauth/token", bytes.NewBufferString(tokenReq))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err = json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    
    assert.Contains(t, response, "access_token")
    assert.Contains(t, response, "token_type")
    assert.Equal(t, "Bearer", response["token_type"])
    
    // Verify the token is a JWT
    accessToken := response["access_token"].(string)
    assert.Contains(t, accessToken, ".") // JWT format
}

func TestClientCredentialsInvalidSecret(t *testing.T) {
    db := setupTestDB(t)
    oauthService := NewOAuthService(db, "test-jwt-secret-key-32-characters")
    require.NotNil(t, oauthService)
    
    // Create test client
    hashedSecret, _ := bcrypt.GenerateFromPassword([]byte("correct_secret"), bcrypt.DefaultCost)
    client := &models.OAuthClient{
        ID:     "test_client_id",
        Secret: string(hashedSecret),
        Domain: "http://localhost:8080",
        Scopes: "read,write",
    }
    err := db.Create(client).Error
    require.NoError(t, err)
    
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.POST("/oauth/token", func(c *gin.Context) {
        err := oauthService.GetServer().HandleTokenRequest(c.Writer, c.Request)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        }
    })
    
    // Test with wrong secret
    tokenReq := "grant_type=client_credentials&client_id=test_client_id&client_secret=wrong_secret&scope=read"
    
    req := httptest.NewRequest("POST", "/oauth/token", bytes.NewBufferString(tokenReq))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Should return error for invalid credentials
    assert.True(t, w.Code >= 400)
}
