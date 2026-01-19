package models

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// NewGrokModel creates a new Grok model instance
//
// It uses the provided context and configuration to initialize the underlying
// OpenAI-compatible client. The modelName specifies which Grok model to target
// (e.g., "grok-beta", "grok-2-1212").
func NewGrokModel(ctx context.Context, modelName string, cfg *genai.ClientConfig) (model.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	// Create OpenAI client with x.ai configuration
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	// Create header value once, when the model is created
	headerValue := fmt.Sprintf("grok-go/%s go/%s",
		"1.0.0", strings.TrimPrefix(runtime.Version(), "go"))

	return &openaiModel{
		name:               modelName,
		client:             &client,
		versionHeaderValue: headerValue,
	}, nil
}
