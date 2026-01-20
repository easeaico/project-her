// Package agent provides agent initialization and configuration.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"github.com/easeaico/adk-memory-agent/internal/config"
	"github.com/easeaico/adk-memory-agent/internal/emotion"
	"github.com/easeaico/adk-memory-agent/internal/memory"
	"github.com/easeaico/adk-memory-agent/internal/models"
	"github.com/easeaico/adk-memory-agent/internal/prompt"
	"github.com/easeaico/adk-memory-agent/internal/repository"
	"github.com/easeaico/adk-memory-agent/internal/types"
)

// NewGirlfriendAgent creates the AI companion agent with RAG and emotion engine.
func NewGirlfriendAgent(
	ctx context.Context,
	embedder memory.Embedder,
	store *repository.Store,
	cfg *config.Config,
	memoryService *memory.Service,
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
	retriever := memory.NewRetriever(embedder, store.ChatHistory, cfg.TopK, cfg.SimilarityThreshold)
	analyzer := emotion.NewAnalyzer(llmModel)
	stateMachine := emotion.NewStateMachine()
	logger := slog.Default()

	before := func(cbCtx agent.CallbackContext, req *model.LLMRequest) (*model.LLMResponse, error) {
		userText := extractUserText(cbCtx.UserContent())
		sessionID := cbCtx.SessionID()
		if sessionID == "" {
			sessionID = cbCtx.InvocationID()
		}

		character, err := loadCharacter(cbCtx, store, cfg.CharacterID)
		if err != nil {
			return nil, err
		}

		history, err := store.ChatHistory.GetRecentMessages(cbCtx, sessionID, cfg.HistoryLimit)
		if err != nil {
			logger.Warn("failed to load history", "error", err.Error())
		}

		memories, err := retriever.Retrieve(cbCtx, sessionID, userText)
		if err != nil {
			logger.Warn("failed to retrieve memories", "error", err.Error())
		}

		promptContents, err := builder.Build(prompt.BuildContext{
			Character:   character,
			Affection:   character.Affection,
			Mood:        character.CurrentMood,
			Memories:    memories,
			History:     history,
			UserMessage: userText,
		})
		if err != nil {
			return nil, err
		}

		req.Contents = promptContents

		if memoryService != nil && userText != "" {
			memoryService.SaveMessageAsync(cbCtx, types.ChatMessage{
				SessionID:   sessionID,
				CharacterID: character.ID,
				Role:        "user",
				Content:     userText,
			}, true)
		}
		return nil, nil
	}

	after := func(cbCtx agent.CallbackContext, resp *model.LLMResponse, respErr error) (*model.LLMResponse, error) {
		if respErr != nil || resp == nil {
			return nil, respErr
		}

		sessionID := cbCtx.SessionID()
		if sessionID == "" {
			sessionID = cbCtx.InvocationID()
		}

		character, err := loadCharacter(cbCtx, store, cfg.CharacterID)
		if err != nil {
			return nil, err
		}

		replyText := extractResponseText(resp)
		conversation := fmt.Sprintf("user: %s\nassistant: %s", extractUserText(cbCtx.UserContent()), replyText)
		label, err := analyzer.Analyze(cbCtx, conversation)
		if err != nil {
			logger.Warn("failed to analyze emotion", "error", err.Error())
		}

		newState := stateMachine.Update(emotion.EmotionState{
			Affection:   character.Affection,
			CurrentMood: character.CurrentMood,
		}, label)

		if err := store.Characters.UpdateEmotion(cbCtx, character.ID, newState.Affection, newState.CurrentMood); err != nil {
			logger.Warn("failed to update emotion state", "error", err.Error())
		}

		if memoryService != nil && replyText != "" {
			memoryService.SaveMessageAsync(cbCtx, types.ChatMessage{
				SessionID:   sessionID,
				CharacterID: character.ID,
				Role:        "model",
				Content:     replyText,
			}, false)
		}

		return nil, nil
	}

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "project_her_girlfriend",
		Description: "高情商、有记忆的 AI 伴侣",
		Model:       llmModel,
		IncludeContents: llmagent.IncludeContentsNone,
		BeforeModelCallbacks: []llmagent.BeforeModelCallback{before},
		AfterModelCallbacks:  []llmagent.AfterModelCallback{after},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create girlfriend agent: %w", err)
	}

	return llmAgent, nil
}

func loadCharacter(ctx context.Context, store *repository.Store, characterID int) (*types.Character, error) {
	if characterID > 0 {
		character, err := store.Characters.GetByID(ctx, characterID)
		if err == nil {
			return character, nil
		}
	}
	return store.Characters.GetDefault(ctx)
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
	return strings.TrimSpace(sb.String())
}

func extractResponseText(resp *model.LLMResponse) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range resp.Content.Parts {
		if part != nil && part.Text != "" {
			sb.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}
