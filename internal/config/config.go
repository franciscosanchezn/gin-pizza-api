package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Create a new instance of the logger
// Configure it to log at the desired level
// and format it as JSON for structured logging
var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	environment := GetEnvWithDefault("APP_ENV", "development")
	switch environment {
	case "development":
		log.SetLevel(logrus.DebugLevel)
	case "production":
		log.SetLevel(logrus.ErrorLevel)
	default:
		// Default to info level for other environments
		log.SetLevel(logrus.InfoLevel)
	}
}

// Config used for the application configuration, loading the input from environment variables
type Config struct {
	// Server Configuration
	Port int    `json:"port"`
	Host string `json:"host"`

	// Logging configuration
	LogLevel string `json:"log_level"`

	// Security Configuration
	JWTSecret string `json:"jwt_secret"`
}

// String returns a string representation of Config with sensitive data masked
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, Host: %s, LogLevel: %s, JWTSecret: [REDACTED]}",
		c.Port, c.Host, c.LogLevel)
}

// LoadConfig read the proper configuration from environment variables and returns a Config struct
// Returns an error if any required environment variable is missing or invalid
func LoadConfig() (*Config, error) {
	log.Info("Loading configuration from environment variables")
	port, err := strconv.Atoi(GetEnvWithDefault("APP_PORT", "8080"))
	if err != nil {
		return nil, err
	}

	config := &Config{
		Port:      port,
		Host:      GetEnvWithDefault("APP_HOST", "localhost"),
		LogLevel:  GetEnvWithDefault("LOG_LEVEL", "info"),
		JWTSecret: GetEnvWithDefault("JWT_SECRET", "secret"),
	}
	log.Infof("Configuration loaded: %s", config.String())
	return config, nil
}

// Helper to get environment with default values
func GetEnvWithDefault(key, defaultValue string) string {
	log.Tracef("Getting environment variable: %s", key)
	value := os.Getenv(key)
	if value == "" {
		log.Warnf("Environment variable %s not set, using default value: %s", key, defaultValue)
		return defaultValue
	}
	return value
}
