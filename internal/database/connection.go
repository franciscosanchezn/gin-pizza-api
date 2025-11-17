package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var log = logrus.New()

func init() {
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
}

// InitDatabase initializes the database connection based on the provided configuration
// It supports both PostgreSQL and SQLite drivers with automatic retry logic and connection pooling
func InitDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Normalize driver name
	driver := strings.ToLower(cfg.Driver)

	log.WithFields(logrus.Fields{
		"db_driver": driver,
		"db_host":   cfg.Host,
		"db_name":   cfg.Name,
		"db_path":   cfg.Path,
	}).Info("Initializing database connection")

	// Retry logic: max 5 attempts with exponential backoff
	maxRetries := 5
	retryDelays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.WithFields(logrus.Fields{
			"attempt":     attempt,
			"max_retries": maxRetries,
		}).Info("Attempting database connection")

		// Select driver based on configuration
		switch driver {
		case "postgres", "postgresql":
			dsn := cfg.DSN()
			log.WithField("dsn_host", cfg.Host).Debug("Connecting to PostgreSQL")
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

		case "sqlite", "":
			dsn := cfg.DSN()
			log.WithField("db_path", cfg.Path).Debug("Connecting to SQLite")
			db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})

		default:
			return nil, fmt.Errorf("unsupported database driver: %s (supported: postgres, sqlite)", cfg.Driver)
		}

		if err == nil {
			// Connection successful, verify with ping
			sqlDB, sqlErr := db.DB()
			if sqlErr != nil {
				log.WithError(sqlErr).Error("Failed to get database instance")
				err = sqlErr
			} else {
				pingErr := sqlDB.Ping()
				if pingErr != nil {
					log.WithError(pingErr).Error("Failed to ping database")
					err = pingErr
				} else {
					// Success! Configure connection pool
					log.Info("Database connection successful, configuring connection pool")
					configureConnectionPool(sqlDB)

					log.WithFields(logrus.Fields{
						"db_driver": driver,
						"attempt":   attempt,
					}).Info("Database initialized successfully")

					return db, nil
				}
			}
		}

		// Connection failed
		log.WithFields(logrus.Fields{
			"attempt": attempt,
			"error":   err.Error(),
		}).Warn("Database connection attempt failed")

		// Don't wait after the last attempt
		if attempt < maxRetries {
			delay := retryDelays[attempt-1]
			log.WithField("delay", delay).Info("Retrying database connection")
			time.Sleep(delay)
		}
	}

	// All retries exhausted
	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

// configureConnectionPool sets up connection pool parameters for optimal performance
func configureConnectionPool(sqlDB *sql.DB) {
	// SetMaxOpenConns sets the maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(25)

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqlDB.SetMaxIdleConns(5)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.WithFields(logrus.Fields{
		"max_open_conns":    25,
		"max_idle_conns":    5,
		"conn_max_lifetime": "5m",
	}).Debug("Connection pool configured")
}
