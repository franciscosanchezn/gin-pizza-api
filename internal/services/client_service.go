package services

import (
	"errors"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"gorm.io/gorm"
)

type ClientService interface {
	CreateClient(client *models.OAuthClient) error
	GetClientsByUserID(userID uint) ([]models.OAuthClient, error)
	GetClientByID(id string) (*models.OAuthClient, error)
	DeleteClient(clientID string, userID uint) error
}

type clientService struct {
	db *gorm.DB
}

func NewClientService(db *gorm.DB) ClientService {
	return &clientService{db: db}
}

func (s *clientService) CreateClient(client *models.OAuthClient) error {
	return s.db.Create(client).Error
}

func (s *clientService) GetClientsByUserID(userID uint) ([]models.OAuthClient, error) {
	var clients []models.OAuthClient
	if err := s.db.Where("user_id = ?", userID).Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

func (s *clientService) GetClientByID(id string) (*models.OAuthClient, error) {
	var client models.OAuthClient
	if err := s.db.Where("id = ?", id).First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (s *clientService) DeleteClient(clientID string, userID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", clientID, userID).Delete(&models.OAuthClient{})
	if result.RowsAffected == 0 {
		return errors.New("client_not_found")
	}
	return result.Error
}