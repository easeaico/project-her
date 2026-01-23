package memory

import (
	"context"
	"fmt"
	"strings"

	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	internalagent "github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

// Service implements ADK memory.Service and RAG helpers.
type service struct {
	embedder            Embedder
	memories            MemoryRepo
	chatHistories       ChatHistoryRepo
	summarizer          internalagent.Summarizer
	topK                int
	similarityThreshold float64
}

const (
	// MemoryTrunkSize is the hard cap for a single memory window (turn-based).
	MemoryTrunkSize = 50
	RoleAssistant     = "assistant"
	RoleUser          = "user"
)

type MemoryRepo interface {
	AddMemory(ctx context.Context, mem types.Memory) error
	SearchSimilar(ctx context.Context, userID, appName, memoryType string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error)
}

type ChatHistoryRepo interface {
	GetLatestWindow(ctx context.Context, characterID int, userID, appName string) (*types.ChatHistory, error)
	CreateWindow(ctx context.Context, history types.ChatHistory) error
	AppendToWindow(ctx context.Context, id int, content string, turnCount int) error
	MarkSummarized(ctx context.Context, id int) error
	GetRecent(ctx context.Context, characterID int, userID, appName string, limit int) ([]types.ChatHistory, error)
}

// NewService returns a memory service.
func NewService(embedder Embedder, memories MemoryRepo, chatHistories ChatHistoryRepo, summarizer internalagent.Summarizer, topK int, threshold float64) adkmemory.Service {
	return &service{
		embedder:            embedder,
		memories:            memories,
		chatHistories:       chatHistories,
		summarizer:          summarizer,
		topK:                topK,
		similarityThreshold: threshold,
	}
}

// AddSession ingests the latest event and maintains rolling memory windows.
func (s *service) AddSession(ctx context.Context, session session.Session) error {
	characterID, err := session.State().Get("character_id")
	if err != nil {
		return err
	}
	cid, ok := characterID.(int)
	if !ok {
		return fmt.Errorf("character_id is not set")
	}

	events := session.Events()
	if events.Len() == 0 {
		return nil
	}

	event := events.At(events.Len() - 1)
	content := strings.TrimSpace(utils.ExtractContentText(event.Content))
	if len(content) == 0 {
		return nil
	}

	role := event.Content.Role
	if role != RoleUser && role != RoleAssistant {
		return nil
	}

	userID := session.UserID()
	appName := session.AppName()

	window, err := s.chatHistories.GetLatestWindow(ctx, cid, userID, appName)
	if err != nil {
		return err
	}

	if window == nil || window.TurnCount >= MemoryTrunkSize || window.Summarized {
		newWindow := types.ChatHistory{
			UserID:      userID,
			AppName:     appName,
			CharacterID: cid,
			Content:     formatMessage(role, content),
			TurnCount:   1,
			Summarized:  false,
		}
		if err := s.chatHistories.CreateWindow(ctx, newWindow); err != nil {
			return err
		}
		return nil
	}

	newContent := appendContent(window.Content, formatMessage(role, content))
	newTurnCount := window.TurnCount + 1
	if err := s.chatHistories.AppendToWindow(ctx, window.ID, newContent, newTurnCount); err != nil {
		return err
	}
	if newTurnCount < MemoryTrunkSize {
		return nil
	}

	windowText := newContent
	windowSalience := aggregateSalienceFromContent(windowText)

	// Default fallback when structured summarizer is unavailable.
	summary := summarizeContent(windowText)
	summaryResult := types.MemorySummary{}
	if s.summarizer != nil {
		summarized, err := s.summarizer.Summarize(ctx, windowText)
		if err != nil {
			return err
		}
		summaryResult = summarized
		if strings.TrimSpace(summarized.Summary) != "" {
			summary = summarized.Summary
		}
	}
	// Embeddings combine summary + durable facts/commitments for better recall.
	embeddingText := buildEmbeddingText(summary, summaryResult.Facts, summaryResult.Commitments)
	embedding, err := s.embedder.EmbedDocument(ctx, embeddingText)
	if err != nil {
		return err
	}

	memory := types.Memory{
		UserID:      userID,
		AppName:     appName,
		CharacterID: cid,
		Type:        types.MemoryTypeChat,
		Summary:     summary,
		Facts:       summaryResult.Facts,
		Commitments: summaryResult.Commitments,
		Emotions:    summaryResult.Emotions,
		TimeRange:   summaryResult.TimeRange,
		Salience:    windowSalience,
		Embedding:   embedding,
	}
	if summaryResult.SalienceScore > 0 {
		memory.Salience = summaryResult.SalienceScore
	}
	if err := s.memories.AddMemory(ctx, memory); err != nil {
		return err
	}
	return s.chatHistories.MarkSummarized(ctx, window.ID)
}

func (s *service) Search(ctx context.Context, req *adkmemory.SearchRequest) (*adkmemory.SearchResponse, error) {
	if req == nil || req.Query == "" {
		return &adkmemory.SearchResponse{Memories: nil}, nil
	}

	vec, err := s.embedder.EmbedQuery(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	memories, err := s.memories.SearchSimilar(ctx, req.UserID, req.AppName, types.MemoryTypeChat, vec, s.topK, s.similarityThreshold)
	if err != nil {
		return nil, err
	}
	return &adkmemory.SearchResponse{Memories: ToMemoryEntries(memories)}, nil
}

func ToMemoryEntries(memories []types.RetrievedMemory) []adkmemory.Entry {
	if len(memories) == 0 {
		return nil
	}
	results := make([]adkmemory.Entry, 0, len(memories))
	for _, m := range memories {
		results = append(results, adkmemory.Entry{
			Content:   genai.NewContentFromText(m.Content, genai.Role(m.Role)),
			Author:    m.Role,
			Timestamp: m.CreatedAt,
		})
	}
	return results
}

func summarizeContent(content string) string {
	const maxRunes = 500
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

// updateSalience accumulates importance with per-message weighting.
func updateSalience(current float64, text string, weight float64) float64 {
	score := messageSalience(text) * weight
	return clamp01(current + score)
}

// messageSalience uses lightweight heuristics to detect important content.
func messageSalience(text string) float64 {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	runes := []rune(trimmed)
	lengthScore := float64(len(runes)) / 400.0
	if lengthScore > 0.5 {
		lengthScore = 0.5
	}

	keywords := []string{
		"喜欢", "讨厌", "害怕", "梦想", "目标", "计划", "约定", "承诺",
		"生日", "纪念日", "地址", "电话", "工作", "学校", "家人",
	}
	var keywordScore float64
	for _, kw := range keywords {
		if strings.Contains(trimmed, kw) {
			keywordScore += 0.1
		}
	}
	if keywordScore > 0.5 {
		keywordScore = 0.5
	}
	return clamp01(lengthScore + keywordScore)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func formatMessage(role, content string) string {
	return fmt.Sprintf("%s: %s", role, content)
}

func appendContent(existing, content string) string {
	if strings.TrimSpace(existing) == "" {
		return content
	}
	return existing + "\n" + content
}

func aggregateSalienceFromContent(content string) float64 {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var salience float64
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		weight := 0.4
		if strings.HasPrefix(trimmed, RoleUser+":") {
			weight = 0.6
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, RoleUser+":"))
		} else if strings.HasPrefix(trimmed, RoleAssistant+":") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, RoleAssistant+":"))
		}
		salience = updateSalience(salience, trimmed, weight)
	}
	return salience
}

// buildEmbeddingText concatenates high-value fields for vector search.
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
