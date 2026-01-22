// Package agent provides agent initialization.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/models"
	"github.com/easeaico/project-her/internal/prompt"
	"github.com/easeaico/project-her/internal/types"
)

type CharacterRepo interface {
	GetByID(ctx context.Context, id int) (*types.Character, error)

	GetDefault(ctx context.Context) (*types.Character, error)

	UpdateEmotion(ctx context.Context, id int, affection int, mood string) error
}

// NewRolePlayAgent builds the companion agent.
func NewRolePlayAgent(
	ctx context.Context,
	cfg *config.Config,
	characters CharacterRepo,
) (agent.Agent, error) {
	llmModel, err := models.NewGrokModel(ctx, cfg.LLMModel, &genai.ClientConfig{
		APIKey: cfg.XAIAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grok model: %w", err)
	}

	builder := prompt.NewBuilder(cfg.HistoryLimit)
	character, err := characters.GetByID(ctx, cfg.CharacterID)
	if err != nil {
		return nil, err
	}

	instruction, err := builder.BuildInstruction(prompt.BuildContext{
		Character: character,
		Affection: character.Affection,
		Mood:      character.CurrentMood,
	})
	if err != nil {
		return nil, err
	}

	imageGenerator, err := models.NewGeminiImageGenerator(ctx, cfg.GoogleAPIKey, cfg.ImageModel, cfg.AspectRatio)
	if err != nil {
		return nil, fmt.Errorf("failed to create image generator: %w", err)
	}

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "project_her_roleplay",
		Description: "高情商、有记忆的 AI 伴侣",
		Model:       llmModel,
		Instruction: instruction,
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{
			func(cbCtx agent.CallbackContext) (*genai.Content, error) {
				userText := extractUserText(cbCtx.UserContent())
				trimmed := strings.TrimSpace(userText)
				if !strings.HasPrefix(trimmed, "/image") {
					return nil, nil
				}

				prompt := strings.TrimSpace(strings.TrimPrefix(trimmed, "/image"))
				if prompt == "" {
					return genai.NewContentFromText("请提供图片描述，例如：/image 一个在雨中撑伞的女孩", "model"), nil
				}

				dataURI, err := imageGenerator.Generate(cbCtx, prompt)
				if err != nil {
					slog.Error("failed to generate image", "error", err.Error())
					return genai.NewContentFromText("图片生成失败，请稍后重试。", "model"), nil
				}

				return genai.NewContentFromText(fmt.Sprintf("图片已生成（base64）：%s", dataURI), "model"), nil
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create girlfriend agent: %w", err)
	}

	return llmAgent, nil
}

func extractUserText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range content.Parts {
		if part != nil && part.Text != "" {
			sb.WriteString(part.Text)
		}
	}
	return sb.String()
}
