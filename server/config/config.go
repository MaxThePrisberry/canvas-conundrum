package config

import (
	"flag"
	"os"
)

// Config holds the application configuration
type Config struct {
	Port        string
	Environment string
	LogLevel    string
}

// Load loads configuration from environment variables and command line flags
func Load() *Config {
	cfg := &Config{}

	// Define command line flags
	flag.StringVar(&cfg.Port, "port", DefaultPort, "Server port")
	flag.StringVar(&cfg.Environment, "env", "development", "Environment (development|production)")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug|info|warn|error)")

	// Parse command line flags
	flag.Parse()

	// Override with environment variables if they exist
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	return cfg
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
