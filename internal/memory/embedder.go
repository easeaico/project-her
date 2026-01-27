// Package memory 实现对话记忆的向量化与检索能力。
package memory

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// Embedder 负责将文本转换为向量表示。
type Embedder interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
	EmbedDocument(ctx context.Context, text string) ([]float32, error)
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}

type GenAIEmbedder struct {
	client *genai.Client
	model  string
}

// newEmbedder 创建 GenAI 的向量化实现。
func newEmbedder(ctx context.Context, apiKey, modelName string) (*GenAIEmbedder, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("google api key is required for embeddings")
	}
	if modelName == "" {
		modelName = "text-embedding-004"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &GenAIEmbedder{
		client: client,
		model:  modelName,
	}, nil
}

func (e *GenAIEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, "RETRIEVAL_QUERY")
}

func (e *GenAIEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, "RETRIEVAL_DOCUMENT")
}

func (e *GenAIEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vec, err := e.EmbedDocument(ctx, text)
		if err != nil {
			return nil, err
		}
		results = append(results, vec)
	}
	return results, nil
}

func (e *GenAIEmbedder) embed(ctx context.Context, text, taskType string) ([]float32, error) {
	if text == "" {
		return nil, nil
	}

	resp, err := e.client.Models.EmbedContent(ctx, e.model, genai.Text(text), &genai.EmbedContentConfig{
		TaskType: taskType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to embed content: %w", err)
	}
	if resp == nil || len(resp.Embeddings) == 0 || resp.Embeddings[0] == nil {
		return nil, fmt.Errorf("empty embedding response")
	}
	return resp.Embeddings[0].Values, nil
}
