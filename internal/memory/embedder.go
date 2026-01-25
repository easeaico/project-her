// Package memory provides embedding helpers and services for conversational memories.
package memory

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// Embedder generates embeddings for text.
type Embedder interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
	EmbedDocument(ctx context.Context, text string) ([]float32, error)
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}

type GenAIEmbedder struct {
	client *genai.Client
	model  string
}

// NewEmbedder creates a GenAI embedder.
func NewEmbedder(ctx context.Context, apiKey, modelName string) (*GenAIEmbedder, error) {
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
