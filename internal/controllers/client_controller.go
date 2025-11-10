package controllers

import (
	"net/http"

	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ClientController struct {
	clientService services.ClientService
}

func NewClientController(clientService services.ClientService) *ClientController {
	return &ClientController{clientService: clientService}
}

// CreateClient godoc
// @Summary Create OAuth2 client
// @Description Create a new OAuth2 client for API access
// @Tags OAuth2 Clients
// @Accept json
// @Produce json
// @Param client body object{name=string,domain=string,scopes=string,grant_types=string,redirect_uri=string} true "Client details"
// @Success 201 {object} map[string]interface{} "Client created with client_id and client_secret"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Client creation failed"
// @Security BearerAuth
// @Router /api/v1/protected/clients [post]
func (cc *ClientController) CreateClient(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Domain      string `json:"domain"`
		Scopes      string `json:"scopes"`
		GrantTypes  string `json:"grant_types"`
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
		ID:          uuid.New().String(),
		Secret:      string(hashedSecret),
		Name:        req.Name,
		Domain:      req.Domain,
		Scopes:      req.Scopes,
		GrantTypes:  req.GrantTypes,
		RedirectURI: req.RedirectURI,
		UserID:      c.GetUint("userID"),
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

// ListClients godoc
// @Summary List OAuth2 clients
// @Description Get all OAuth2 clients owned by the authenticated user
// @Tags OAuth2 Clients
// @Accept json
// @Produce json
// @Success 200 {array} object "List of clients"
// @Failure 500 {object} map[string]string "Failed to retrieve clients"
// @Security BearerAuth
// @Router /api/v1/protected/clients [get]
func (cc *ClientController) ListClients(c *gin.Context) {
	userID := c.GetUint("userID")
	clients, err := cc.clientService.GetClientsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_retrieve_clients"})
		return
	}

	c.JSON(http.StatusOK, clients)
}

// DeleteClient godoc
// @Summary Delete OAuth2 client
// @Description Delete an OAuth2 client owned by the authenticated user
// @Tags OAuth2 Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Success 204 "Client deleted successfully"
// @Failure 404 {object} map[string]string "Client not found"
// @Security BearerAuth
// @Router /api/v1/protected/clients/{id} [delete]
func (cc *ClientController) DeleteClient(c *gin.Context) {
	clientID := c.Param("id")
	userID := c.GetUint("userID")

	if err := cc.clientService.DeleteClient(clientID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "client_not_found"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
