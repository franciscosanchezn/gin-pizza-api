package config

import (
	"errors"
	"fmt"
	"net/url"
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
	Port        int    `json:"port"`
	Host        string `json:"host"`
	DatabaseURL string `json:"database_url"`

	// Database configuration
	DBName     string `json:"db_name"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`

	// Logging configuration
	LogLevel string `json:"log_level"`

	// Security Configuration
	JWTSecret string `json:"jwt_secret"`
}

// String returns a string representation of Config with sensitive data masked
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, Host: %s, DatabaseURL: %s, DBName: %s, DBUser: %s, DBPassword: [REDACTED], LogLevel: %s, JWTSecret: [REDACTED]}",
		c.Port, c.Host, maskDatabaseURL(c.DatabaseURL), c.DBName, c.DBUser, c.LogLevel)
}

// maskDatabaseURL masks password in database URL
func maskDatabaseURL(dbURL string) string {
	if dbURL == "" {
		return ""
	}

	parsed, err := url.Parse(dbURL)
	if err != nil {
		return "[REDACTED_INVALID_URL]"
	}

	if parsed.User != nil {
		// Replace password with [REDACTED]
		parsed.User = url.UserPassword(parsed.User.Username(), "[REDACTED]")
	}

	return parsed.String()
}

// LoadConfig read the proper configuration from environment variables and returns a Config struct
// It also validates formats like DatabaseURL and JWTSecret
// Returns an error if any required environment variable is missing or invalid
func LoadConfig() (*Config, error) {
	log.Info("Loading configuration from environment variables")
	port, err := strconv.Atoi(GetEnvWithDefault("APP_PORT", "8080"))
	if err != nil {
		return nil, err
	}

	db_url := GetEnvWithDefault("DATABASE_URL", "")
	if db_url == "" {
		return nil, errors.New("DATABASE_URL environment variable is required")
	}
	// validate URL with net/url
	_, err = url.ParseRequestURI(db_url)
	if err != nil {
		panic("Invalid DATABASE_URL format: " + db_url)
	}

	config := &Config{
		Port:        port,
		Host:        GetEnvWithDefault("APP_HOST", "localhost"),
		DatabaseURL: GetEnvWithDefault("DATABASE_URL", ""),
		DBName:      GetEnvWithDefault("DB_NAME", "mydb"),
		DBUser:      GetEnvWithDefault("DB_USER", "user"),
		DBPassword:  GetEnvWithDefault("DB_PASSWORD", "password"),
		LogLevel:    GetEnvWithDefault("LOG_LEVEL", "info"),
		JWTSecret:   GetEnvWithDefault("JWT_SECRET", "secret"),
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

// GetEnvAsType retrieves an environment variable and converts it to the specified type
// using generic type handling.
func GetEnvAsType[T any](key string, defaultValue T) T {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	var result T
	switch any(result).(type) {
	case int:
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return any(intValue).(T)
	case string:
		return any(value).(T)
	case bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return any(boolValue).(T)
	default:
		return defaultValue // Fallback for unsupported types
	}
}
