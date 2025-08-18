package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Reset flags for clean testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	
	// Clear environment variables that might interfere
	os.Unsetenv("PORT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("LOG_LEVEL")
	
	t.Run("Default values", func(t *testing.T) {
		// Reset flags and args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"test"}
		
		cfg := Load()
		
		assert.Equal(t, DefaultPort, cfg.Port)
		assert.Equal(t, "development", cfg.Environment)
		assert.Equal(t, "info", cfg.LogLevel)
	})
	
	t.Run("Environment variables override defaults", func(t *testing.T) {
		// Reset flags and args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"test"}
		
		// Set environment variables
		os.Setenv("PORT", "9000")
		os.Setenv("ENVIRONMENT", "production")
		os.Setenv("LOG_LEVEL", "debug")
		
		defer func() {
			os.Unsetenv("PORT")
			os.Unsetenv("ENVIRONMENT")
			os.Unsetenv("LOG_LEVEL")
		}()
		
		cfg := Load()
		
		assert.Equal(t, "9000", cfg.Port)
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, "debug", cfg.LogLevel)
	})
	
	t.Run("Command line flags", func(t *testing.T) {
		// Reset flags and set args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"test", "-port=8888", "-env=staging", "-log-level=warn"}
		
		cfg := Load()
		
		assert.Equal(t, "8888", cfg.Port)
		assert.Equal(t, "staging", cfg.Environment)
		assert.Equal(t, "warn", cfg.LogLevel)
	})
	
	t.Run("Environment variables override command line", func(t *testing.T) {
		// Reset flags and set args
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		os.Args = []string{"test", "-port=7777"}
		
		// Set environment variable to override
		os.Setenv("PORT", "6666")
		defer os.Unsetenv("PORT")
		
		cfg := Load()
		
		// Environment variable should take precedence
		assert.Equal(t, "6666", cfg.Port)
	})
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expected    bool
	}{
		{"Production environment", "production", true},
		{"Development environment", "development", false},
		{"Staging environment", "staging", false},
		{"Empty environment", "", false},
		{"Mixed case production", "Production", false}, // Case sensitive
		{"Test environment", "test", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Environment: tt.environment,
			}
			
			result := cfg.IsProduction()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigStruct(t *testing.T) {
	t.Run("Config initialization", func(t *testing.T) {
		cfg := &Config{
			Port:        "8080",
			Environment: "development",
			LogLevel:    "info",
		}
		
		assert.Equal(t, "8080", cfg.Port)
		assert.Equal(t, "development", cfg.Environment)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.False(t, cfg.IsProduction())
	})
	
	t.Run("Empty config", func(t *testing.T) {
		cfg := &Config{}
		
		assert.Equal(t, "", cfg.Port)
		assert.Equal(t, "", cfg.Environment)
		assert.Equal(t, "", cfg.LogLevel)
		assert.False(t, cfg.IsProduction())
	})
}

func TestDefaultPort(t *testing.T) {
	// Test that DefaultPort constant is accessible
	assert.Equal(t, "8080", DefaultPort)
	assert.NotEmpty(t, DefaultPort)
}