// Package agent provides agent initialization.
package agent

import (
	"context"
	"fmt"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/genai"

	"github.com/easeaico/adk-memory-agent/internal/config"
	"github.com/easeaico/adk-memory-agent/internal/models"
	"github.com/easeaico/adk-memory-agent/internal/prompt"
	"github.com/easeaico/adk-memory-agent/internal/repository"
)

// NewRolePlayAgent builds the companion agent.
func NewRolePlayAgent(
	ctx context.Context,
	store *repository.Store,
	cfg *config.Config,
) (agent.Agent, error) {
	if store == nil || cfg == nil {
		return nil, fmt.Errorf("store and config are required")
	}

	llmModel, err := models.NewGrokModel(ctx, cfg.LLMModel, &genai.ClientConfig{
		APIKey: cfg.XAIAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grok model: %w", err)
	}

	builder := prompt.NewBuilder(cfg.HistoryLimit)
	character, err := store.Characters.GetByID(ctx, cfg.CharacterID)
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

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "project_her_roleplay",
		Description: "高情商、有记忆的 AI 伴侣",
		Model:       llmModel,
		Instruction: instruction,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create girlfriend agent: %w", err)
	}

	return llmAgent, nil
}
