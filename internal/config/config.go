// Package config provides configuration loading and validation.
package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds the application configuration loaded from environment variables.
// All fields are required except WorkDir, which defaults to the current working directory.
type Config struct {
	DatabaseURL          string  // PostgreSQL connection string (required)
	GoogleAPIKey         string  // Google GenAI API key for embeddings (required)
	XAIAPIKey            string  // xAI API key for Grok LLM (required)
	WorkDir              string  // Working directory for file operations (optional, defaults to current directory)
	LLMModel             string  // LLM model name (default: grok-4-fast)
	EmbeddingModel        string  // Embedding model name (default: text-embedding-004)
	TopK                 int     // RAG top-k
	SimilarityThreshold  float64 // RAG similarity threshold
	HistoryLimit         int     // recent history turns
	CharacterID          int     // character ID to use
}

// Load loads configuration from environment variables.
func Load() Config {
	cfg := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		GoogleAPIKey:  os.Getenv("GOOGLE_API_KEY"),
		XAIAPIKey:     os.Getenv("XAI_API_KEY"),
		WorkDir:       os.Getenv("WORK_DIR"),
		LLMModel:      os.Getenv("LLM_MODEL"),
		EmbeddingModel: os.Getenv("EMBEDDING_MODEL"),
	}

	cfg.TopK = getEnvInt("TOP_K", 5)
	cfg.SimilarityThreshold = getEnvFloat("SIMILARITY_THRESHOLD", 0.7)
	cfg.HistoryLimit = getEnvInt("HISTORY_LIMIT", 10)
	cfg.CharacterID = getEnvInt("CHARACTER_ID", 1)

	// Set defaults
	if cfg.WorkDir == "" {
		cfg.WorkDir, _ = os.Getwd()
	}
	if cfg.LLMModel == "" {
		cfg.LLMModel = "grok-4-fast"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-004"
	}

	// Validate required config
	if cfg.GoogleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is required")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required (e.g., postgres://user:pass@localhost:5432/dbname)")
	}
	if cfg.XAIAPIKey == "" {
		log.Fatal("XAI_API_KEY environment variable is required")
	}

	return cfg
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}
