package services

import (
	"errors"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
}

type userService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) UserService {
	return &userService{db: db}
}

func (s *userService) CreateUser(user *models.User) error {
	var existing models.User
	if err := s.db.Where("email = ?", user.Email).First(&existing).Error; err == nil {
		return errors.New("user_already_exists")
	}

	return s.db.Create(user).Error
}

func (s *userService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}