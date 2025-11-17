package controllers

import (
	"net/http"
	"strconv"

	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
	"github.com/gin-gonic/gin"
)

// PizzaController handles HTTP requests related to pizzas
type PizzaController interface {
	// GetAllPizzas retrieves all pizzas
	GetAllPizzas(c *gin.Context)
	// GetPizzaByID retrieves a pizza by its ID
	GetPizzaByID(c *gin.Context)
	// CreatePizza creates a new pizza
	CreatePizza(c *gin.Context)
	// UpdatePizza updates an existing pizza
	UpdatePizza(c *gin.Context)
	// DeletePizza deletes a pizza by its ID
	DeletePizza(c *gin.Context)
}

type controller struct {
	service services.PizzaService
}

// NewPizzaController creates a new instance of PizzaController
func NewPizzaController(service services.PizzaService) *controller {
	return &controller{service: service}
}

// GetAllPizzas godoc
// @Summary Get all pizzas
// @Description Get a list of all pizzas with optional filtering
// @Tags pizzas
// @Accept json
// @Produce json
// @Param created_by query string false "Filter by creator user ID"
// @Param name query string false "Filter by pizza name (partial match)"
// @Success 200 {array} models.Pizza
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/public/pizzas [get]
func (c *controller) GetAllPizzas(ctx *gin.Context) {
	// Get query parameters
	createdBy := ctx.Query("created_by")
	name := ctx.Query("name")

	pizzas, err := c.service.GetAllPizzas(createdBy, name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pizzas"})
		return
	}
	ctx.JSON(http.StatusOK, pizzas)
}

// GetPizzaByID godoc
// @Summary Get pizza by ID
// @Description Get a single pizza by its ID
// @Tags pizzas
// @Accept json
// @Produce json
// @Param id path int true "Pizza ID"
// @Success 200 {object} models.Pizza
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/public/pizzas/{id} [get]
func (c *controller) GetPizzaByID(ctx *gin.Context) {
	id, existId := ctx.Params.Get("id")
	if !existId {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID"})
	}

	pizzaId, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID format"})
		return
	}

	pizza, err := c.service.GetPizzaByID(pizzaId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Pizza not found"})
		return
	}
	ctx.JSON(http.StatusOK, pizza)
}

// CreatePizza godoc
// @Summary Create a new pizza
// @Description Create a new pizza with the input payload
// @Tags pizzas
// @Accept json
// @Produce json
// @Param pizza body models.Pizza true "Pizza object"
// @Success 201 {object} models.Pizza
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/pizzas [post]
func (c *controller) CreatePizza(ctx *gin.Context) {
	var pizza models.Pizza
	if err := ctx.ShouldBindJSON(&pizza); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get the authenticated user ID from context
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Set the creator based on the type of userID
	switch v := userID.(type) {
	case uint:
		pizza.CreatedBy = v
	case int:
		pizza.CreatedBy = uint(v)
	case string:
		// For OAuth2 tokens with string userID, we need to parse or handle differently
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format for this operation"})
		return
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected user ID type"})
		return
	}

	createdPizza, err := c.service.CreatePizza(pizza)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pizza"})
		return
	}
	ctx.JSON(http.StatusCreated, createdPizza)
}

// UpdatePizza godoc
// @Summary Update a pizza
// @Description Update a pizza with the input payload
// @Tags pizzas
// @Accept json
// @Produce json
// @Param id path int true "Pizza ID"
// @Param pizza body models.Pizza true "Pizza object"
// @Success 200 {object} models.Pizza
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/pizzas/{id} [put]
func (c *controller) UpdatePizza(ctx *gin.Context) {
	id, existId := ctx.Params.Get("id")
	if !existId {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID"})
		return
	}

	pizzaId, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID format"})
		return
	}

	// Get the existing pizza to check ownership
	existingPizza, err := c.service.GetPizzaByID(pizzaId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Pizza not found"})
		return
	}

	// Get the authenticated user ID and role
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := ctx.Get("userRole")
	isAdmin := userRole == "admin"

	// Check if user is the creator or an admin
	var currentUserID uint
	switch v := userID.(type) {
	case uint:
		currentUserID = v
	case int:
		currentUserID = uint(v)
	case string:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format for this operation"})
		return
	}

	if existingPizza.CreatedBy != currentUserID && !isAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error":       "You can only update your own pizzas",
			"pizza_owner": existingPizza.CreatedBy,
			"your_id":     currentUserID,
		})
		return
	}

	var pizza models.Pizza
	if err := ctx.ShouldBindJSON(&pizza); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Ensure the ID from URL is used
	pizza.ID = pizzaId
	// Preserve the original creator
	pizza.CreatedBy = existingPizza.CreatedBy

	updatedPizza, err := c.service.UpdatePizza(pizza)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pizza"})
		return
	}
	ctx.JSON(http.StatusOK, updatedPizza)
}

// DeletePizza godoc
// @Summary Delete a pizza
// @Description Delete a pizza by its ID
// @Tags pizzas
// @Accept json
// @Produce json
// @Param id path int true "Pizza ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/v1/pizzas/{id} [delete]
func (c *controller) DeletePizza(ctx *gin.Context) {
	id, existId := ctx.Params.Get("id")
	if !existId {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID"})
		return
	}

	pizzaId, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pizza ID format"})
		return
	}

	// Get the existing pizza to check ownership
	existingPizza, err := c.service.GetPizzaByID(pizzaId)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Pizza not found"})
		return
	}

	// Get the authenticated user ID and role
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := ctx.Get("userRole")
	isAdmin := userRole == "admin"

	// Check if user is the creator or an admin
	var currentUserID uint
	switch v := userID.(type) {
	case uint:
		currentUserID = v
	case int:
		currentUserID = uint(v)
	case string:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format for this operation"})
		return
	}

	if existingPizza.CreatedBy != currentUserID && !isAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error":       "You can only delete your own pizzas",
			"pizza_owner": existingPizza.CreatedBy,
			"your_id":     currentUserID,
		})
		return
	}

	if err := c.service.DeletePizza(pizzaId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pizza"})
		return
	}
	ctx.JSON(http.StatusNoContent, nil)
}
