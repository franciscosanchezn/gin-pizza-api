package main

import (
	"fmt"
	"github.com/franciscosanchezn/gin-pizza-api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func main() {
	// Load environment variables
	loadDotenvFile()

	// Initialize logger
	setUpLogger()

	// Load configuration
	configuration := loadConfig()

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
}

// healthCheckHandler handles the health check endpoint
// It returns a JSON response with the service status and timestamp
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "gin-pizza-api",
	})
}
