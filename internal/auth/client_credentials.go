package auth

import (
	"net/http"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"github.com/gin-gonic/gin"
	"github.com/go-oauth2/oauth2/v4"
	"time"
)

// HandleToken handles the token endpoint for both client credentials and authorization code grants
// @Summary Token Endpoint
// @Description Obtain an access token using client credentials or authorization code grant
// @Tags OAuth2
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param grant_type formData string true "Grant type: client_credentials or authorization_code"
// @Param client_id formData string true "Client ID"
// @Param client_secret formData string true "Client Secret"
// @Param code formData string false "Authorization code (required for authorization_code grant)"
// @Param redirect_uri formData string false "Redirect URI (required for authorization_code grant)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /oauth/token [post]
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
	// redirectURI := c.PostForm("redirect_uri")

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
	ti, err := o.server.Manager.GenerateAccessToken(c, oauth2.AuthorizationCode, &oauth2.TokenGenerateRequest{
		ClientID: clientID,
		UserID:   authCode.UserID,
		Scope:    authCode.Scopes,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	// Delete used authorization code
	o.db.Delete(&authCode)

	c.JSON(http.StatusOK, gin.H{
		"access_token": ti.GetAccess(),
		"token_type":   "Bearer",
		"expires_in":   int64(ti.GetAccessExpiresIn()),
		"scope":        ti.GetScope(),
	})
}

func (o *OAuthService) handleClientCredentials(c *gin.Context) {
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
	ti, err := o.server.Manager.GenerateAccessToken(c, oauth2.ClientCredentials, &oauth2.TokenGenerateRequest{
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
		"expires_in":   int64(ti.GetAccessExpiresIn()),
		"scope":        ti.GetScope(),
	})
}