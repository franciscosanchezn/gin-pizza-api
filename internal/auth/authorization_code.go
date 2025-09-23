package auth

import (
	"net/http"
	"net/url"
	"time"
	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
)

func (o *OAuthService) HandleAuthorize(c *gin.Context) {
	clientID := c.Query("client_id")
	// responseType := c.Query("response_type")
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