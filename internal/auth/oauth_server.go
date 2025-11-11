package auth

import (
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type OAuthService struct {
	server *server.Server
	db     *gorm.DB
}

func NewOAuthService(db *gorm.DB, jwtSecret string) *OAuthService {
	manager := manage.NewDefaultManager()

	// Use our custom JWT generator that includes the UserID and Role claims
	// Pass the database connection so it can fetch user information
	manager.MapAccessGenerate(NewCustomJWTAccessGenerate([]byte(jwtSecret), jwt.SigningMethodHS512, db))

	// Use in-memory token store (required by OAuth2 library even with stateless JWTs)
	tokenStore, _ := store.NewMemoryTokenStore()
	manager.MapTokenStorage(tokenStore)

	// Configure client store
	clientStore := NewGormClientStore(db)
	manager.MapClientStorage(clientStore)

	srv := server.NewDefaultServer(manager)
	srv.SetAllowGetAccessRequest(true)
	srv.SetClientInfoHandler(server.ClientFormHandler)

	// The OAuth2 v4.5.4 library automatically detects that our OAuthClient
	// implements ClientPasswordVerifier and uses the VerifyPassword method
	// No additional configuration needed!

	return &OAuthService{
		server: srv,
		db:     db,
	}
}

func (o *OAuthService) GetServer() *server.Server {
	return o.server
}
