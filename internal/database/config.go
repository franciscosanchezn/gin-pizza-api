package database

import (
	"fmt"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	// Driver specifies the database driver (postgres, sqlite)
	Driver string

	// PostgreSQL-specific configuration
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	// SQLite-specific configuration
	Path string
}

// String returns a string representation with sensitive data masked
func (c *DatabaseConfig) String() string {
	return fmt.Sprintf("DatabaseConfig{Driver: %s, Host: %s, Port: %s, User: %s, Password: [REDACTED], Name: %s, SSLMode: %s, Path: %s}",
		c.Driver, c.Host, c.Port, c.User, c.Name, c.SSLMode, c.Path)
}

// DSN builds a Data Source Name string based on the driver
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode)
	case "sqlite", "":
		return c.Path
	default:
		return ""
	}
}
