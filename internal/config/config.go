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

	// Database Configuration
	DBDriver   string `json:"db_driver"` // postgres or sqlite
	DBHost     string `json:"db_host"`
	DBPort     string `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"-"` // Masked in String()
	DBName     string `json:"db_name"`
	DBSSLMode  string `json:"db_sslmode"`
	DBPath     string `json:"db_path"` // SQLite only

	// Bootstrap OAuth Client (for K8s auto-provisioning)
	BootstrapClientID     string `json:"bootstrap_client_id"`
	BootstrapClientSecret string `json:"-"` // Masked
}

// String returns a string representation of Config with sensitive data masked
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, Host: %s, LogLevel: %s, JWTSecret: [REDACTED], DBDriver: %s, DBHost: %s, DBPort: %s, DBUser: %s, DBPassword: [REDACTED], DBName: %s, DBSSLMode: %s, DBPath: %s, BootstrapClientID: %s, BootstrapClientSecret: [REDACTED]}",
		c.Port, c.Host, c.LogLevel, c.DBDriver, c.DBHost, c.DBPort, c.DBUser, c.DBName, c.DBSSLMode, c.DBPath, c.BootstrapClientID)
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

		// Database Configuration
		DBDriver:   GetEnvWithDefault("DB_DRIVER", "sqlite"),
		DBHost:     GetEnvWithDefault("DB_HOST", "localhost"),
		DBPort:     GetEnvWithDefault("DB_PORT", "5432"),
		DBUser:     GetEnvWithDefault("DB_USER", "postgres"),
		DBPassword: os.Getenv("DB_PASSWORD"), // No default for security
		DBName:     GetEnvWithDefault("DB_NAME", "pizza_api"),
		DBSSLMode:  GetEnvWithDefault("DB_SSLMODE", "disable"),
		DBPath:     GetEnvWithDefault("DB_PATH", "test.sqlite"),

		// Bootstrap OAuth Client Configuration
		BootstrapClientID:     GetEnvWithDefault("BOOTSTRAP_CLIENT_ID", "admin-client"),
		BootstrapClientSecret: GetEnvWithDefault("BOOTSTRAP_CLIENT_SECRET", ""),
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
