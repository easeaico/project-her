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

func NewOpenRouterModel(ctx context.Context, modelName string, cfg *genai.ClientConfig) (model.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	// 使用 OpenRouter Base URL 创建 OpenAI 兼容客户端。
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL("https://openrouter.ai/api/v1"),
	)

	// 创建时一次性生成 UA 头，避免每次请求重复拼接。
	headerValue := fmt.Sprintf("openrouter-go/%s go/%s",
		"1.0.0", strings.TrimPrefix(runtime.Version(), "go"))

	modelName = fmt.Sprintf("openrouter/%s", modelName)

	return &openaiModel{
		name:               modelName,
		client:             &client,
		versionHeaderValue: headerValue,
	}, nil

}
