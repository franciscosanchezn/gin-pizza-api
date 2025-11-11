package auth

import (
	"context"

	internalmodels "github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/go-oauth2/oauth2/v4"
	"gorm.io/gorm"
)

type GormClientStore struct {
	db *gorm.DB
}

func NewGormClientStore(db *gorm.DB) *GormClientStore {
	return &GormClientStore{db: db}
}

func (s *GormClientStore) GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error) {
	var client internalmodels.OAuthClient
	if err := s.db.Where("id = ?", id).First(&client).Error; err != nil {
		return nil, err
	}

	// Return our custom OAuthClient which implements ClientPasswordVerifier
	return &client, nil
}
