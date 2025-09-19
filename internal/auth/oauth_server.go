package auth

import (
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type OAuthService struct {
	server *server.Server
	db     *gorm.DB
}

func NewOAuthService(db *gorm.DB, jwtSecret string) *OAuthService {
	manager := manage.NewDefaultManager()

	// Use JWT for access tokens
	manager.MapAccessGenerate(generates.NewJWTAccessGenerate("", []byte(jwtSecret), jwt.SigningMethodHS512))

	// Configure token store
	tokenStore := NewGormTokenStore(db)
	manager.MustTokenStorage(tokenStore, nil)

	// Configure client store
	clientStore := NewGormClientStore(db)
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	return &OAuthService{
		server: srv,
		db:     db,
	}
}

func (o *OAuthService) GetServer() *server.Server {
	return o.server
}