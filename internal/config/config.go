// Package config loads configuration from environment variables.
package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds runtime settings.
type Config struct {
	DatabaseURL         string
	GoogleAPIKey        string
	XAIAPIKey           string
	WorkDir             string
	LLMModel            string
	ImageModel          string
	AspectRatio         string
	EmbeddingModel      string
	TopK                int
	SimilarityThreshold float64
	HistoryLimit        int
	CharacterID         int
}

// Load reads env vars, applies defaults, and validates required fields.
func Load() Config {
	cfg := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		GoogleAPIKey:   os.Getenv("GOOGLE_API_KEY"),
		XAIAPIKey:      os.Getenv("XAI_API_KEY"),
		WorkDir:        os.Getenv("WORK_DIR"),
		LLMModel:       os.Getenv("LLM_MODEL"),
		ImageModel:     os.Getenv("IMAGE_MODEL"),
		AspectRatio:    os.Getenv("ASPECT_RATIO"),
		EmbeddingModel: os.Getenv("EMBEDDING_MODEL"),
	}

	cfg.TopK = getEnvInt("TOP_K", 5)
	cfg.SimilarityThreshold = getEnvFloat("SIMILARITY_THRESHOLD", 0.7)
	cfg.HistoryLimit = getEnvInt("HISTORY_LIMIT", 10)
	cfg.CharacterID = getEnvInt("CHARACTER_ID", 1)

	if cfg.WorkDir == "" {
		cfg.WorkDir, _ = os.Getwd()
	}
	if cfg.LLMModel == "" {
		cfg.LLMModel = "grok-4-fast"
	}
	if cfg.ImageModel == "" {
		cfg.ImageModel = "gemini-3-pro-image-preview"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-004"
	}
	if cfg.AspectRatio == "" {
		cfg.AspectRatio = "9:16"
	}

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
