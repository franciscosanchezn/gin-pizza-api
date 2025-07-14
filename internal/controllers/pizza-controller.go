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

func (c *controller) GetAllPizzas(ctx *gin.Context) {
	pizzas, err := c.service.GetAllPizzas()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve pizzas"})
		return
	}
	ctx.JSON(http.StatusOK, pizzas)
}

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

func (c *controller) CreatePizza(ctx *gin.Context) {
	var pizza models.Pizza
	if err := ctx.ShouldBindJSON(&pizza); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	createdPizza, err := c.service.CreatePizza(pizza)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pizza"})
		return
	}
	ctx.JSON(http.StatusCreated, createdPizza)
}

func (c *controller) UpdatePizza(ctx *gin.Context) {
	var pizza models.Pizza
	if err := ctx.ShouldBindJSON(&pizza); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updatedPizza, err := c.service.UpdatePizza(pizza)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pizza"})
		return
	}
	ctx.JSON(http.StatusOK, updatedPizza)
}

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

	if err := c.service.DeletePizza(pizzaId); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pizza"})
		return
	}
	ctx.JSON(http.StatusNoContent, nil)
}
