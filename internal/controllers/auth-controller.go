package controllers

import (
	"net/http"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthController struct {
	userService services.UserService
	jwtSecret   []byte
}

func NewAuthController(userService services.UserService, jwtSecret string) *AuthController {
	return &AuthController{
		userService: userService,
		jwtSecret:   []byte(jwtSecret),
	}
}

func (ac *AuthController) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &models.User{
		Email: req.Email,
		Password: req.Password,
		Name:  req.Name,
	}

	if err := user.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password_hashing_failed"})
		return
	}

	if err := ac.userService.CreateUser(user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user_already_exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user_created"})
}

func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ac.userService.GetUserByEmail(req.Email)
	if err != nil || !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString(ac.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_generation_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   86400,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}