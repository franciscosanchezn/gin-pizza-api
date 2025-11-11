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

func TestLoadConfig(t *testing.T) {
	// Helper function to set multiple env vars
	setTestEnv := func() {
		os.Setenv("APP_PORT", "9000")
		os.Setenv("APP_HOST", "0.0.0.0")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("JWT_SECRET", "super_secret_jwt_key")
	}

	// Helper function to cleanup env vars
	cleanupTestEnv := func() {
		vars := []string{
			"APP_PORT", "APP_HOST", "LOG_LEVEL", "JWT_SECRET",
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
		if config.LogLevel != "debug" {
			t.Errorf("LogLevel = %s, expected debug", config.LogLevel)
		}
	})

	t.Run("should fail with invalid port", func(t *testing.T) {
		cleanupTestEnv()
		os.Setenv("APP_PORT", "not_a_number")
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
