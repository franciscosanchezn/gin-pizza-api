package main

import (
	"fmt"
	_ "github.com/franciscosanchezn/gin-pizza-api/docs" // Import generated docs
	"github.com/franciscosanchezn/gin-pizza-api/internal/config"
	"github.com/franciscosanchezn/gin-pizza-api/internal/controllers"
	"github.com/franciscosanchezn/gin-pizza-api/internal/models"
	"github.com/franciscosanchezn/gin-pizza-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"time"
)

var (
	db              *gorm.DB
	pizzaService    services.PizzaService
	pizzaController controllers.PizzaController
)

// @title Pizza API
// @version 1.0
// @description A simple Pizza API
// @host localhost:8080
// @BasePath /
func main() {
	// Load environment variables
	loadDotenvFile()

	// Initialize logger
	setUpLogger()

	// Load configuration
	configuration := loadConfig()

	// Initialize database connection
	setupDatabase(configuration)

	// Initialize services and controllers
	pizzaService = services.NewPizzaService(db)
	pizzaController = controllers.NewPizzaController(pizzaService)

	// Initialize Gin router
	var router *gin.Engine = setupRouter()

	// Start the server
	log.Infof("Starting server on %s:%d", configuration.Host, configuration.Port)
	router.Run(fmt.Sprintf("%v:%d", configuration.Host, configuration.Port))
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
func setupDatabase(conf *config.Config) *gorm.DB {
	// Database connection logic here
	// This is a placeholder, actual implementation will depend on the database being used
	var err error
	db, err = gorm.Open(sqlite.Open("test.sqlite"), &gorm.Config{})
	checkPanicErr(err)
	// Migrate the schema
	db.AutoMigrate(&models.Pizza{})

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

// seedDatabase seeds the database with initial data
func seedDatabase() {
	log.Info("Seeding database with initial data")
	pizzas := []models.Pizza{
		{Name: "Margherita", Price: 10.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Basil"}},
		{Name: "Pepperoni", Price: 12.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Pepperoni"}},
		{Name: "Vegetarian", Price: 11.99, Ingredients: []string{"Tomato Sauce", "Mozzarella", "Bell Peppers", "Olives"}},
	}
	for _, pizza := range pizzas {
		db.Create(&pizza)
	}
	log.Info("Database seeded successfully")
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
	// Health check endpoint
	router.GET("/health", healthCheckHandler)
	// Pizza routes
	router.GET("/pizzas", pizzaController.GetAllPizzas)
	router.GET("/pizzas/:id", pizzaController.GetPizzaByID)
	router.POST("/pizzas", pizzaController.CreatePizza)
	router.PUT("/pizzas/:id", pizzaController.UpdatePizza)
	router.DELETE("/pizzas/:id", pizzaController.DeletePizza)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

}

// healthCheckHandler handles the health check endpoint
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "gin-pizza-api",
	})
}
