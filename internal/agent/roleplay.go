// Package agent provides agent initialization.
package agent

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/handler"
	"github.com/easeaico/project-her/internal/models"
	"github.com/easeaico/project-her/internal/prompt"
	"github.com/easeaico/project-her/internal/types"
)

// CharacterRepo exposes persistence operations for companion personas.
// Implementations live under internal/storage and back the ADK agent.
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
	llmModel, err := models.NewGrokModel(ctx, cfg.ChatModel, &genai.ClientConfig{
		APIKey: cfg.XAIAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grok model: %w", err)
	}

	character, err := characters.GetByID(ctx, cfg.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	instruction, err := prompt.BuildRoleplayInstruction(character)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	imgHandler, err := handler.NewImageCommandHandler(ctx, cfg, character)
	if err != nil {
		return nil, fmt.Errorf("failed to create image command handler: %w", err)
	}

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        fmt.Sprintf("project_her_roleplay_%d", cfg.CharacterID),
		Description: "高情商、有记忆的 AI 伴侣",
		Model:       llmModel,
		Instruction: instruction,
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{
			handler.EnsureSessionStateCallback(character),
			imgHandler.Handle,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create girlfriend agent: %w", err)
	}

	return llmAgent, nil
}
