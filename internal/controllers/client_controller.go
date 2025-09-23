package controllers

import (
	"net/http"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
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