// Package memory 提供代理初始化与装配能力。
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"strings"
	"sync/atomic"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

const (
	memorySummarizerAppName = "project_her_memory"
	memorySummarizerUserID  = "memory_summarizer"
)

// memorySummaryInstruction 要求模型仅返回符合结构的 JSON。
const memorySummaryInstruction = `You are a professional dialogue memory summarizer.
Your task is to compress the conversation history into a concise summary while preserving the most important information.

Extract and retain:
1. Key events and important decisions
2. Emotional shifts and intimate moments
3. User-revealed personal info (preferences, habits, important dates, etc.)
4. Promises or agreements made by either party
5. The overall emotional tone

Output requirements:
- Use third-person narration
- Organize chronologically
- Keep the summary within 200-300 Chinese characters
- Return a valid JSON object that matches the output schema
- Do not include any extra keys or text outside the JSON object`

// memorySummarizer 使用 ADK agent 生成记忆摘要。
type memorySummarizer struct {
	agent           agent.Agent
	runner          summarizerRunner
	sessionService  session.Service
	charHistories   ChatHistoryRepo
	memoryRepo      MemoryRepo
	embedder        Embedder
	emotionProvider EmotionStateProvider
	counter         uint64
}

type summarizerRunner interface {
	Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig) iter.Seq2[*session.Event, error]
}

// NewMemorySummarizer 基于 ADK llmagent 构建摘要器。
func NewMemorySummarizer(ctx context.Context, cfg *config.Config, charHistories ChatHistoryRepo, memoryRepo MemoryRepo, embedder Embedder, emotionProvider EmotionStateProvider) (Summarizer, error) {
	summarizerModel, err := gemini.NewModel(ctx, cfg.MemoryModel, &genai.ClientConfig{
		APIKey: cfg.GoogleAPIKey,
	})
	if err != nil {
		slog.Error("failed to create summarizer model", "error", err)
		return nil, fmt.Errorf("failed to create summarizer model: %w", err)
	}

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:            "memory_summarizer",
		Description:     "对话记忆摘要智能体",
		Model:           summarizerModel,
		Instruction:     memorySummaryInstruction,
		OutputSchema:    summaryOutputSchema(),
		IncludeContents: llmagent.IncludeContentsNone,
	})
	if err != nil {
		slog.Error("failed to create memory summarizer agent", "error", err)
		return nil, fmt.Errorf("failed to create memory summarizer agent: %w", err)
	}

	sessionService := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        memorySummarizerAppName,
		Agent:          llmAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory summarizer runner: %w", err)
	}

	return &memorySummarizer{
		agent:           llmAgent,
		runner:          r,
		sessionService:  sessionService,
		charHistories:   charHistories,
		memoryRepo:      memoryRepo,
		embedder:        embedder,
		emotionProvider: emotionProvider,
	}, nil
}

// SummarizeWindow 对指定会话的当前窗口做摘要并写入记忆。
func (s *memorySummarizer) SummarizeLatestWindow(ctx context.Context, userID, appName string) error {
	window, err := s.charHistories.GetLatestWindow(ctx, userID, appName)
	if err != nil {
		return err
	}
	if window == nil {
		return nil
	}

	summarySessID := fmt.Sprintf("summary-%d", atomic.AddUint64(&s.counter, 1))
	if _, err := s.sessionService.Create(ctx, &session.CreateRequest{
		AppName:   memorySummarizerAppName,
		UserID:    memorySummarizerUserID,
		SessionID: summarySessID,
	}); err != nil {
		if _, getErr := s.sessionService.Get(ctx, &session.GetRequest{
			AppName:   memorySummarizerAppName,
			UserID:    memorySummarizerUserID,
			SessionID: summarySessID,
		}); getErr != nil {
			return fmt.Errorf("failed to create summarizer session: %w", err)
		}
	}
	msg := genai.NewContentFromText(window.Content, "user")
	events := s.runner.Run(ctx, memorySummarizerUserID, summarySessID, msg, agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	})

	var last string
	for event, err := range events {
		if err != nil {
			return err
		}
		if event == nil || event.Content == nil {
			continue
		}
		if event.Author == "user" {
			continue
		}
		text := strings.TrimSpace(utils.ExtractContentText(event.Content))
		if text == "" {
			continue
		}
		last = text
		if event.IsFinalResponse() {
			break
		}
	}
	if last == "" {
		return fmt.Errorf("empty summary response")
	}

	summary, err := parseSummaryJSON(last)
	if err != nil {
		return err
	}

	var emotionState *EmotionState
	if s.emotionProvider != nil {
		state, err := s.emotionProvider.GetEmotionState(ctx, userID, appName)
		if err != nil {
			slog.Warn("failed to load emotion state", "error", err.Error(), "user_id", userID, "app_name", appName)
		} else {
			emotionState = &state
		}
	}
	salience := ComputeSalience(summary, emotionState)

	embeddingText := buildEmbeddingText(summary.Summary, summary.Facts, summary.Commitments)
	embedding, err := s.embedder.EmbedDocument(ctx, embeddingText)
	if err != nil {
		return err
	}

	if err := s.memoryRepo.AddMemory(ctx, types.Memory{
		UserID:      userID,
		AppName:     appName,
		Type:        types.MemoryTypeChat,
		Summary:     summary.Summary,
		Facts:       summary.Facts,
		Commitments: summary.Commitments,
		Emotions:    summary.Emotions,
		Salience:    salience,
		Embedding:   embedding,
	}); err != nil {
		return err
	}

	return nil
}

func summaryOutputSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"summary": {
				Type: genai.TypeString,
			},
			"facts": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"commitments": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"emotions": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"time_range": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"start": {Type: genai.TypeString},
					"end":   {Type: genai.TypeString},
				},
			},
			"salience_score": {
				Type: genai.TypeNumber,
			},
		},
		Required: []string{"summary"},
	}
}

// parseSummaryJSON 从模型输出中提取 JSON 并解码。
func parseSummaryJSON(raw string) (types.MemorySummary, error) {
	clean := strings.TrimSpace(raw)
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		clean = clean[start : end+1]
	}
	var summary types.MemorySummary
	if err := json.Unmarshal([]byte(clean), &summary); err != nil {
		return types.MemorySummary{}, fmt.Errorf("failed to parse summary json: %w", err)
	}
	return summary, nil
}

// buildEmbeddingText 拼接高价值字段用于向量检索。
func buildEmbeddingText(summary string, facts, commitments []string) string {
	var sb strings.Builder
	sb.WriteString(summary)
	appendList := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		sb.WriteString("\n")
		sb.WriteString(title)
		sb.WriteString(": ")
		for i, item := range items {
			if i > 0 {
				sb.WriteString(" ; ")
			}
			sb.WriteString(item)
		}
	}
	appendList("facts", facts)
	appendList("commitments", commitments)
	return sb.String()
}

func normalizeSalience(score float64) float64 {
	if score != score || score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
