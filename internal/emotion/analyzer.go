package emotion

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Analyzer classifies conversation sentiment.
type Analyzer struct {
	model model.LLM
}

// NewAnalyzer returns an Analyzer.
func NewAnalyzer(m model.LLM) *Analyzer {
	return &Analyzer{model: m}
}

// Analyze returns the sentiment label for text.
func (a *Analyzer) Analyze(ctx context.Context, text string) (EmotionLabel, error) {
	if a == nil || a.model == nil {
		return EmotionNeutral, fmt.Errorf("emotion analyzer not configured")
	}

	if strings.TrimSpace(text) == "" {
		return EmotionNeutral, nil
	}

	system := `你是情感分析器。仅返回以下三个标签之一：Positive、Negative、Neutral。不要输出其他内容。`
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText(system, "system"),
			genai.NewContentFromText(text, "user"),
		},
	}

	seq := a.model.GenerateContent(ctx, req, false)
	var resp *model.LLMResponse
	var err error
	seq(func(r *model.LLMResponse, e error) bool {
		resp = r
		err = e
		return false
	})
	if err != nil {
		return EmotionNeutral, err
	}

	label := extractLabel(resp)
	switch label {
	case "positive":
		return EmotionPositive, nil
	case "negative":
		return EmotionNegative, nil
	default:
		return EmotionNeutral, nil
	}
}

func extractLabel(resp *model.LLMResponse) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range resp.Content.Parts {
		if part != nil && part.Text != "" {
			sb.WriteString(part.Text)
		}
	}
	return strings.ToLower(strings.TrimSpace(sb.String()))
}
