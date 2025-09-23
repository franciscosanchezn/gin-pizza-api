package auth

import (
	"context"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/models"
	"gorm.io/gorm"
	"time"
	internalmodels "github.com/franciscosanchezn/gin-pizza-api/internal/models"
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

type GormTokenStore struct {
	db *gorm.DB
}

func NewGormTokenStore(db *gorm.DB) *GormTokenStore {
	return &GormTokenStore{db: db}
}

func (s *GormTokenStore) Create(ctx context.Context, info oauth2.TokenInfo) error {

	userId := info.GetUserID()
	refreshToken := info.GetRefresh()
	expiresAt := info.GetAccessExpiresIn()

	token := &internalmodels.OAuthToken{
		ClientID:    info.GetClientID(),
		UserID:      &userId,
		AccessToken: info.GetAccess(),
		RefreshToken: &refreshToken,
		Scopes:      info.GetScope(),
		ExpiresAt:   time.Now().Add(expiresAt),
	}

	return s.db.Create(token).Error
}

func (s *GormTokenStore) RemoveByAccess(ctx context.Context, access string) error {
	return s.db.Where("access_token = ?", access).Delete(&internalmodels.OAuthToken{}).Error
}

func (s *GormTokenStore) RemoveByRefresh(ctx context.Context, refresh string) error {
	return s.db.Where("refresh_token = ?", refresh).Delete(&internalmodels.OAuthToken{}).Error
}

func (s *GormTokenStore) GetByAccess(ctx context.Context, access string) (oauth2.TokenInfo, error) {
	var token internalmodels.OAuthToken
	if err := s.db.Where("access_token = ?", access).First(&token).Error; err != nil {
		return nil, err
	}
	return &models.Token{
		ClientID:         token.ClientID,
		UserID:           *token.UserID,
		Access:           token.AccessToken,
		Refresh:          *token.RefreshToken,
		AccessExpiresIn:  time.Until(token.ExpiresAt),
		Scope:            token.Scopes,
	}, nil
}

func (s *GormTokenStore) GetByRefresh(ctx context.Context, refresh string) (oauth2.TokenInfo, error) {
	var token internalmodels.OAuthToken
	if err := s.db.Where("refresh_token = ?", refresh).First(&token).Error; err != nil {
		return nil, err
	}

	return &models.Token{
		ClientID:         token.ClientID,
		UserID:           *token.UserID,
		Access:           token.AccessToken,
		Refresh:          *token.RefreshToken,
		AccessExpiresIn:  time.Until(token.ExpiresAt),
		Scope:            token.Scopes,
	}, nil
}

func (s *GormTokenStore) GetByCode(ctx context.Context, code string) (oauth2.TokenInfo, error) {
	var oauthCode internalmodels.OAuthCode
	if err := s.db.Where("code = ?", code).First(&oauthCode).Error; err != nil {
		return nil, err
	}

	// Check if the code has expired
	if time.Now().After(oauthCode.ExpiresAt) {
		return nil, gorm.ErrRecordNotFound
	}

	return &models.Token{
		ClientID:         oauthCode.ClientID,
		UserID:           oauthCode.UserID,
		Code:             oauthCode.Code,
		CodeCreateAt:     oauthCode.CreatedAt,
		CodeExpiresIn:    oauthCode.ExpiresAt.Sub(oauthCode.CreatedAt),
		CodeChallenge:    oauthCode.CodeChallenge,
		CodeChallengeMethod: oauthCode.CodeChallengeMethod,
		RedirectURI:      oauthCode.RedirectURI,
		Scope:            oauthCode.Scopes,
	}, nil
}

func (s *GormTokenStore) RemoveByCode(ctx context.Context, code string) error {
	return s.db.Where("code = ?", code).Delete(&internalmodels.OAuthCode{}).Error
}

func (s *GormTokenStore) CreateCode(ctx context.Context, info oauth2.TokenInfo) error {
	code := &internalmodels.OAuthCode{
		ClientID:          info.GetClientID(),
		UserID:            info.GetUserID(),
		Code:              info.GetCode(),
		CodeChallenge:     info.GetCodeChallenge(),
		CodeChallengeMethod: info.GetCodeChallengeMethod().String(),
		RedirectURI:       info.GetRedirectURI(),
		Scopes:           info.GetScope(),
		ExpiresAt:        info.GetCodeCreateAt().Add(info.GetCodeExpiresIn()),
	}

	return s.db.Create(code).Error
}

