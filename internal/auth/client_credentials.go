package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

	if grantType != "client_credentials" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	o.handleClientCredentials(c)
}

func (o *OAuthService) handleClientCredentials(c *gin.Context) {
	// Let the OAuth2 library handle the entire flow
	// It will:
	// 1. Get the client from the store
	// 2. Verify the client secret using VerifyPassword
	// 3. Generate the token
	gt, tgr, err := o.server.ValidationTokenRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": err.Error(),
		})
		return
	}

	ti, err := o.server.GetAccessToken(c, gt, tgr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":             "server_error",
			"error_description": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, o.server.GetTokenData(ti))
}
