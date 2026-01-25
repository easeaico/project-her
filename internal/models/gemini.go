package models

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type ImageGenerator struct {
	client      *genai.Client
	model       string
	aspectRatio string
}

func NewGeminiImageGenerator(ctx context.Context, apiKey, model, aspectRatio string) (*ImageGenerator, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &ImageGenerator{
		client:      client,
		model:       strings.TrimSpace(model),
		aspectRatio: normalizeAspectRatio(aspectRatio),
	}, nil
}

func (g *ImageGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	if g == nil || g.client == nil {
		return "", fmt.Errorf("image generator not configured")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	config := &genai.GenerateContentConfig{
		ResponseModalities: []string{"IMAGE", "TEXT"},
		ImageConfig: &genai.ImageConfig{
			AspectRatio: g.aspectRatio,
		},
	}
	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), config)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0] == nil || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("empty image response")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
			continue
		}
		mimeType := strings.TrimSpace(part.InlineData.MIMEType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		encoded := base64.StdEncoding.EncodeToString(part.InlineData.Data)
		imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
		return imageURL, nil
	}
	return "", fmt.Errorf("image data missing in response")
}

func normalizeAspectRatio(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "1:1", "3:4", "4:3", "9:16", "16:9":
		return value
	default:
		return "9:16"
	}
}
