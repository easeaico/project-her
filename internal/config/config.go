// Package config provides configuration loading and validation.
package config

import (
	"log"
	"os"
)

// Config holds the application configuration loaded from environment variables.
// All fields are required except WorkDir, which defaults to the current working directory.
type Config struct {
	DatabaseURL string // PostgreSQL connection string (required)
	APIKey      string // Google GenAI API key (required)
	WorkDir     string // Working directory for file operations (optional, defaults to current directory)
}

// Load loads configuration from environment variables.
func Load() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		APIKey:      os.Getenv("GOOGLE_API_KEY"),
		WorkDir:     os.Getenv("WORK_DIR"),
	}

	// Set defaults
	if cfg.WorkDir == "" {
		cfg.WorkDir, _ = os.Getwd()
	}

	// Validate required config
	if cfg.APIKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is required")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required (e.g., postgres://user:pass@localhost:5432/dbname)")
	}

	return cfg
}
