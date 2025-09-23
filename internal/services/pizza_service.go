package services

import (
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"gorm.io/gorm"
)

// PizzaService provides methods to interact with the pizza database
type PizzaService interface {
	// GetAllPizzas retrieves all pizzas from the database
	GetAllPizzas() ([]models.Pizza, error)
	// GetPizzaByID retrieves a pizza by its ID
	GetPizzaByID(id int) (models.Pizza, error)
	// CreatePizza creates a new pizza in the database
	CreatePizza(pizza models.Pizza) (models.Pizza, error)
	// UpdatePizza updates an existing pizza in the database
	UpdatePizza(pizza models.Pizza) (models.Pizza, error)
	// DeletePizza deletes a pizza from the database by its ID
	DeletePizza(id int) error
}

// pizzaService is the implementation of the PizzaService interface
type pizzaService struct {
	db *gorm.DB
}

// NewPizzaService creates a new instance of PizzaService
func NewPizzaService(db *gorm.DB) PizzaService {
	return &pizzaService{db: db}
}

func (s *pizzaService) GetAllPizzas() ([]models.Pizza, error) {
	var pizzas []models.Pizza
	if err := s.db.Find(&pizzas).Error; err != nil {
		return nil, err
	}
	return pizzas, nil
}

func (s *pizzaService) GetPizzaByID(id int) (models.Pizza, error) {
	var pizza models.Pizza
	if err := s.db.First(&pizza, id).Error; err != nil {
		return models.Pizza{}, err
	}
	return pizza, nil
}

func (s *pizzaService) CreatePizza(pizza models.Pizza) (models.Pizza, error) {
	if err := s.db.Create(&pizza).Error; err != nil {
		return models.Pizza{}, err
	}
	return pizza, nil
}

func (s *pizzaService) UpdatePizza(pizza models.Pizza) (models.Pizza, error) {
	if err := s.db.Save(&pizza).Error; err != nil {
		return models.Pizza{}, err
	}
	return pizza, nil
}

func (s *pizzaService) DeletePizza(id int) error {
	if err := s.db.Delete(&models.Pizza{}, id).Error; err != nil {
		return err
	}
	return nil
}
