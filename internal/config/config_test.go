package config

import (
	"os"
	"testing"
)

func TestGetEnvWithDefault(t *testing.T) {
	testCases := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "should return env value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "from_env",
			expected:     "from_env",
		},
		{
			name:         "should return default when env not set",
			key:          "MISSING_KEY",
			defaultValue: "default_value",
			envValue:     "",
			expected:     "default_value",
		},
		{
			name:         "should return empty string default",
			key:          "EMPTY_KEY",
			defaultValue: "",
			envValue:     "",
			expected:     "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Setup: set environment variable if provided
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key) // cleanup after test
			} else {
				os.Unsetenv(tt.key) // ensure it's not set
			}

			// Execute
			result := GetEnvWithDefault(tt.key, tt.defaultValue)

			// Assert
			if result != tt.expected {
				t.Errorf("GetEnvWithDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsType(t *testing.T) {
	t.Run("test int conversion", func(t *testing.T) {
		testCases := []struct {
			name         string
			key          string
			envValue     string
			defaultValue int
			expected     int
		}{
			{
				name:         "valid int from env",
				key:          "INT_KEY",
				envValue:     "42",
				defaultValue: 10,
				expected:     42,
			},
			{
				name:         "invalid int returns default",
				key:          "INVALID_INT",
				envValue:     "not_a_number",
				defaultValue: 100,
				expected:     100,
			},
			{
				name:         "missing env returns default",
				key:          "MISSING_INT",
				envValue:     "",
				defaultValue: 999,
				expected:     999,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				if tt.envValue != "" {
					os.Setenv(tt.key, tt.envValue)
					defer os.Unsetenv(tt.key)
				} else {
					os.Unsetenv(tt.key)
				}

				result := GetEnvAsType(tt.key, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetEnvAsType() = %v, expected %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("test string conversion", func(t *testing.T) {
		os.Setenv("STRING_KEY", "hello_world")
		defer os.Unsetenv("STRING_KEY")

		result := GetEnvAsType("STRING_KEY", "default")
		expected := "hello_world"

		if result != expected {
			t.Errorf("GetEnvAsType() = %v, expected %v", result, expected)
		}
	})

	t.Run("test bool conversion", func(t *testing.T) {
		tests := []struct {
			name         string
			envValue     string
			defaultValue bool
			expected     bool
		}{
			{"true value", "true", false, true},
			{"false value", "false", true, false},
			{"1 value", "1", false, true},
			{"0 value", "0", true, false},
			{"invalid bool", "maybe", true, true}, // should return default
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				key := "BOOL_KEY"
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)

				result := GetEnvAsType(key, tt.defaultValue)
				if result != tt.expected {
					t.Errorf("GetEnvAsType() = %v, expected %v", result, tt.expected)
				}
			})
		}
	})
}

func TestLoadConfig(t *testing.T) {
	// Helper function to set multiple env vars
	setTestEnv := func() {
		os.Setenv("APP_PORT", "9000")
		os.Setenv("APP_HOST", "0.0.0.0")
		os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
		os.Setenv("DB_NAME", "pizza_db")
		os.Setenv("DB_USER", "pizza_user")
		os.Setenv("DB_PASSWORD", "secret123")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("JWT_SECRET", "super_secret_jwt_key")
	}

	// Helper function to cleanup env vars
	cleanupTestEnv := func() {
		vars := []string{
			"APP_PORT", "APP_HOST", "DATABASE_URL", "DB_NAME",
			"DB_USER", "DB_PASSWORD", "LOG_LEVEL", "JWT_SECRET",
		}
		for _, v := range vars {
			os.Unsetenv(v)
		}
	}

	t.Run("successful config load with all env vars", func(t *testing.T) {
		setTestEnv()
		defer cleanupTestEnv()

		config, err := LoadConfig()

		// Should not return error
		if err != nil {
			t.Fatalf("LoadConfig() returned error: %v", err)
		}

		// Verify all values
		if config.Port != 9000 {
			t.Errorf("Port = %d, expected 9000", config.Port)
		}
		if config.Host != "0.0.0.0" {
			t.Errorf("Host = %s, expected 0.0.0.0", config.Host)
		}
		if config.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
			t.Errorf("DatabaseURL = %s, expected postgres://...", config.DatabaseURL)
		}
		if config.DBName != "pizza_db" {
			t.Errorf("DBName = %s, expected pizza_db", config.DBName)
		}
		if config.LogLevel != "debug" {
			t.Errorf("LogLevel = %s, expected debug", config.LogLevel)
		}
	})

	t.Run("should fail when DATABASE_URL is missing", func(t *testing.T) {
		cleanupTestEnv()
		// Don't set DATABASE_URL

		config, err := LoadConfig()

		// Should return error
		if err == nil {
			t.Error("LoadConfig() should return error when DATABASE_URL is missing")
		}
		if config != nil {
			t.Error("Config should be nil when error occurs")
		}

		expectedError := "DATABASE_URL environment variable is required"
		if err.Error() != expectedError {
			t.Errorf("Error = %v, expected %v", err.Error(), expectedError)
		}
	})

	t.Run("should fail with invalid port", func(t *testing.T) {
		cleanupTestEnv()
		os.Setenv("APP_PORT", "not_a_number")
		os.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
		defer cleanupTestEnv()

		config, err := LoadConfig()

		if err == nil {
			t.Error("LoadConfig() should return error when APP_PORT is invalid")
		}
		if config != nil {
			t.Error("Config should be nil when error occurs")
		}
	})

	t.Run("should use defaults when optional env vars not set", func(t *testing.T) {
		cleanupTestEnv()
		// Only set required DATABASE_URL
		os.Setenv("DATABASE_URL", "postgres://localhost:5432/test")
		defer cleanupTestEnv()

		config, err := LoadConfig()

		if err != nil {
			t.Fatalf("LoadConfig() returned unexpected error: %v", err)
		}

		// Check defaults
		if config.Port != 8080 {
			t.Errorf("Port = %d, expected default 8080", config.Port)
		}
		if config.Host != "localhost" {
			t.Errorf("Host = %s, expected default localhost", config.Host)
		}
		if config.DBName != "mydb" {
			t.Errorf("DBName = %s, expected default mydb", config.DBName)
		}
		if config.LogLevel != "info" {
			t.Errorf("LogLevel = %s, expected default info", config.LogLevel)
		}
	})
}

// Benchmark tests (optional but good practice)
func BenchmarkGetEnvWithDefault(b *testing.B) {
	os.Setenv("BENCH_KEY", "test_value")
	defer os.Unsetenv("BENCH_KEY")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetEnvWithDefault("BENCH_KEY", "default")
	}
}
