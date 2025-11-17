package main

import (
	"fmt"
	"net/http"
	"time"

	_ "github.com/franciscosanchezn/gin-pizza-api/docs" // Import generated docs
	"github.com/franciscosanchezn/gin-pizza-api/internal/auth"
	"github.com/franciscosanchezn/gin-pizza-api/internal/config"
	"github.com/franciscosanchezn/gin-pizza-api/internal/controllers"
	"github.com/franciscosanchezn/gin-pizza-api/internal/database"
	"github.com/franciscosanchezn/gin-pizza-api/internal/middleware"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// APIVersion is the current version of the API
	APIVersion = "1.0.0"
)

var (
	db              *gorm.DB
	pizzaService    services.PizzaService
	pizzaController controllers.PizzaController
	configuration   *config.Config
)

// @title Pizza API
// @version 1.0
// @description A simple Pizza API
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load environment variables
	loadDotenvFile()

	// Initialize logger
	setUpLogger()

	// Load configuration
	configuration = loadConfig()

	// Initialize database connection
	setupDatabase()

	// Bootstrap OAuth client for K8s/production deployments
	bootstrapOAuthClient()

	// Initialize services and controllers
	pizzaService = services.NewPizzaService(db)
	pizzaController = controllers.NewPizzaController(pizzaService)

	// Initialize Gin router
	var router *gin.Engine = setupRouter()

	// Start the server
	log.Infof("Starting server on %s:%d", configuration.Host, configuration.Port)
	if err := router.Run(fmt.Sprintf("%v:%d", configuration.Host, configuration.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// checkPanicErr checks if an error occurred and panics if it did
func checkPanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// loadDotenvFile loads environment variables from a .env file
// If the file is not found, it will log a warning and use system environment variables
func loadDotenvFile() {
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, using system environment variables")
	}
}

// setUpLogger initializes the logger with a JSON formatter and sets the log level based on the environment
func setUpLogger() {
	log.SetFormatter(&log.JSONFormatter{})
	environment := config.GetEnvWithDefault("APP_ENV", "development")
	switch environment {
	case "development":
		log.SetLevel(log.DebugLevel)
	case "production":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}

// loadConfig loads the application configuration from environment variables
// It returns a Config struct or panics if there is an error
func loadConfig() *config.Config {
	log.Info("Loading configuration from environment variables")
	conf, err := config.LoadConfig()
	checkPanicErr(err)
	log.Infof("Configuration loaded: %+v", conf)
	return conf
}

// setupDatabase initializes the database connection and returns a gorm.DB instance
func setupDatabase() *gorm.DB {
	// Build database configuration from app config
	dbConfig := database.DatabaseConfig{
		Driver:   configuration.DBDriver,
		Host:     configuration.DBHost,
		Port:     configuration.DBPort,
		User:     configuration.DBUser,
		Password: configuration.DBPassword,
		Name:     configuration.DBName,
		SSLMode:  configuration.DBSSLMode,
		Path:     configuration.DBPath,
	}

	// Initialize database connection
	var err error
	db, err = database.InitDatabase(dbConfig)
	checkPanicErr(err)

	log.Infof("Database initialized: driver=%s", configuration.DBDriver)

	// Migrate the schema
	if err := db.AutoMigrate(&models.Pizza{}); err != nil {
		log.Fatalf("Failed to migrate Pizza schema: %v", err)
	}
	// Add OAuth models
	if err := db.AutoMigrate(
		&models.User{},
		&models.Pizza{},
		&models.OAuthClient{},
	); err != nil {
		log.Fatalf("Failed to migrate OAuth schemas: %v", err)
	}

	// Create only if is empty
	var count int64
	db.Model(&models.Pizza{}).Count(&count)
	if count == 0 {
		log.Info("Database is empty, seeding initial data")
		seedDatabase()
	} else {
		log.Info("Database already seeded with initial data")
	}
	return db
}

// bootstrapOAuthClient creates an admin OAuth client on first startup if it doesn't exist
// This enables automatic credential provisioning for K8s deployments
func bootstrapOAuthClient() {
	log.Info("Checking OAuth client bootstrap requirements")

	clientID := configuration.BootstrapClientID

	// Check if the specific bootstrap client already exists
	var existing models.OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		log.WithField("client_id", clientID).Info("Bootstrap OAuth client already exists, skipping")
		return
	}

	// Bootstrap client doesn't exist, create it
	log.WithField("client_id", clientID).Info("Creating bootstrap OAuth client")

	// Ensure system user exists
	systemUser := models.User{
		Email: "system@pizza.com",
		Name:  "System User",
		Role:  "admin",
	}

	var existingUser models.User
	if err := db.Where("email = ?", systemUser.Email).First(&existingUser).Error; err != nil {
		// System user doesn't exist, create it
		if err := db.Create(&systemUser).Error; err != nil {
			log.WithError(err).Error("Failed to create system user for OAuth bootstrap")
			return
		}
		log.Info("✓ System user created for OAuth bootstrap")
	} else {
		systemUser.ID = existingUser.ID
		log.Info("Using existing system user for OAuth bootstrap")
	}

	// Get client secret from configuration
	clientSecret := configuration.BootstrapClientSecret

	// Generate random secret if not provided
	if clientSecret == "" {
		clientSecret = fmt.Sprintf("bootstrap-secret-%d", time.Now().UnixNano())
		log.Warn("No BOOTSTRAP_CLIENT_SECRET provided, generated random secret")
	}

	// Hash the client secret using bcrypt
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Failed to hash bootstrap client secret")
		return
	}

	oauthClient := models.OAuthClient{
		ID:     clientID,
		Secret: string(hashedSecret),
		UserID: systemUser.ID,
		Scopes: "read write",
	}

	if err := db.Create(&oauthClient).Error; err != nil {
		log.WithError(err).Error("Failed to create bootstrap OAuth client")
		return
	}

	log.WithFields(log.Fields{
		"client_id": clientID,
		"user_id":   systemUser.ID,
		"scopes":    oauthClient.Scopes,
	}).Info("✓ Bootstrap OAuth client created successfully")

	// Security note: Do NOT log the client secret in production
	log.Warn("IMPORTANT: Save the bootstrap client credentials securely")
}

// seedDatabase seeds the database with initial data
func seedDatabase() {
	log.Info("Seeding database with initial data")

	// Create a system/default user for seeded pizzas
	systemUser := models.User{
		Email: "system@pizza.com",
		Name:  "System User",
		Role:  "admin",
	}

	// Check if system user already exists
	var existingUser models.User
	if err := db.Where("email = ?", systemUser.Email).First(&existingUser).Error; err == nil {
		// User exists, use that ID
		systemUser.ID = existingUser.ID
		log.Info("System user already exists, using existing ID")
	} else {
		// Create new system user
		if err := db.Create(&systemUser).Error; err != nil {
			log.Errorf("Failed to create system user: %v", err)
			return
		}
		log.Infof("✓ System user created: system@pizza.com")
	}

	// Create a regular user for testing
	regularUser := models.User{
		Email: "user@pizza.com",
		Name:  "Regular User",
		Role:  "user",
	}

	var existingRegularUser models.User
	if err := db.Where("email = ?", regularUser.Email).First(&existingRegularUser).Error; err == nil {
		regularUser.ID = existingRegularUser.ID
		log.Info("Regular user already exists, using existing ID")
	} else {
		if err := db.Create(&regularUser).Error; err != nil {
			log.Errorf("Failed to create regular user: %v", err)
			return
		}
		log.Infof("✓ Regular user created: user@pizza.com")
	}

	pizzas := []models.Pizza{
		{Name: "Margherita", Price: 10.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Basil"}, CreatedBy: systemUser.ID},
		{Name: "Pepperoni", Price: 12.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Pepperoni"}, CreatedBy: systemUser.ID},
		{Name: "Vegetarian", Price: 11.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Bell Peppers", "Olives"}, CreatedBy: systemUser.ID},
	}
	for _, pizza := range pizzas {
		db.Create(&pizza)
	}

	// Create development OAuth clients for local testing
	createDevOAuthClient(systemUser.ID)
	createUserOAuthClient(regularUser.ID)

	log.Info("Database seeded successfully")
}

// createDevOAuthClient creates a dev-client for local development and testing
func createDevOAuthClient(userID uint) {
	clientID := "dev-client"
	clientSecret := "dev-secret-123"

	// Check if dev-client already exists
	var existing models.OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		log.Info("Development OAuth client already exists")
		return
	}

	// Create dev-client
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Failed to hash dev client secret")
		return
	}

	devClient := models.OAuthClient{
		ID:     clientID,
		Secret: string(hashedSecret),
		Name:   "Development Client",
		UserID: userID,
		Scopes: "read write",
	}

	if err := db.Create(&devClient).Error; err != nil {
		log.WithError(err).Error("Failed to create dev OAuth client")
		return
	}

	log.WithFields(log.Fields{
		"client_id":     clientID,
		"client_secret": clientSecret,
	}).Info("✓ Development OAuth client created (for testing only)")
}

// createUserOAuthClient creates a user-client for USER role testing
func createUserOAuthClient(userID uint) {
	clientID := "user-client"
	clientSecret := "user-secret-123"

	// Check if user-client already exists
	var existing models.OAuthClient
	if err := db.Where("id = ?", clientID).First(&existing).Error; err == nil {
		log.Info("User OAuth client already exists")
		return
	}

	// Create user-client
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Failed to hash user client secret")
		return
	}

	userClient := models.OAuthClient{
		ID:     clientID,
		Secret: string(hashedSecret),
		Name:   "User Test Client",
		UserID: userID,
		Scopes: "read write",
	}

	if err := db.Create(&userClient).Error; err != nil {
		log.WithError(err).Error("Failed to create user OAuth client")
		return
	}

	log.WithFields(log.Fields{
		"client_id":     clientID,
		"client_secret": clientSecret,
	}).Info("✓ User OAuth client created (for testing USER role)")
}

// setupRouter initializes the Gin router and sets up the routes
// It returns the configured router
func setupRouter() *gin.Engine {
	// Initialize Gin router
	router := gin.Default()

	// Define routes
	setupRoutes(router)

	return router
}

// setupRoutes defines the routes for the Gin router
func setupRoutes(router *gin.Engine) {
	// Initialize OAuth service
	oauthService := auth.NewOAuthService(db, configuration.JWTSecret)

	// Health check endpoint
	router.GET("/health", healthCheckHandler)

	// Pizza routes
	v1 := router.Group("/api/v1")
	{
		publicApi := v1.Group("/public")
		{
			publicApi.GET("/pizzas", pizzaController.GetAllPizzas)
			publicApi.GET("/pizzas/:id", pizzaController.GetPizzaByID)
		}

		// Initialize client controller
		clientService := services.NewClientService(db)
		clientController := controllers.NewClientController(clientService)

		// OAuth2 routes remain separate
		oauthRoutes := v1.Group("/oauth")
		{
			oauthRoutes.POST("/token", oauthService.HandleToken)
		}

		// Pizza CRUD - requires authentication, ownership enforced in controller
		pizzaApi := v1.Group("/pizzas")
		pizzaApi.Use(middleware.OAuth2Auth([]byte(configuration.JWTSecret)))
		{
			pizzaApi.POST("", pizzaController.CreatePizza)
			pizzaApi.PUT("/:id", pizzaController.UpdatePizza)
			pizzaApi.DELETE("/:id", pizzaController.DeletePizza)
		}

		// OAuth client management - admin only
		clientApi := v1.Group("/clients")
		clientApi.Use(middleware.OAuth2Auth([]byte(configuration.JWTSecret)))
		clientApi.Use(middleware.RequireRole("admin"))
		{
			clientApi.POST("", clientController.CreateClient)
			clientApi.GET("", clientController.ListClients)
			clientApi.DELETE("/:id", clientController.DeleteClient)
		}
	}

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"healthy"`
	Version   string `json:"version" example:"1.0.0"`
	Database  string `json:"database" example:"connected"`
	DBDriver  string `json:"db_driver" example:"sqlite"`
	Timestamp string `json:"timestamp" example:"2025-11-10T12:34:56Z"`
}

// healthCheckHandler handles the health check endpoint
// @Summary Health check
// @Description Check if the service is running and database connectivity
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthCheckHandler(c *gin.Context) {
	// Check database connectivity
	dbStatus := "connected"
	sqlDB, err := db.DB()
	if err != nil {
		dbStatus = "disconnected"
	} else if err := sqlDB.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	response := HealthResponse{
		Status:    "healthy",
		Version:   APIVersion,
		Database:  dbStatus,
		DBDriver:  configuration.DBDriver,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}
