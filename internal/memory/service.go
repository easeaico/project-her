// Package memory 实现对话记忆的写入、摘要与检索服务。
package memory

import (
	"context"
	"fmt"
	"log"
	"strings"

	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

// service 实现 ADK memory.Service，并提供检索增强所需的辅助能力。
type memoryService struct {
	cfg                 *config.Config
	embedder            Embedder
	memories            MemoryRepo
	chatHistories       ChatHistoryRepo
	summarizer          Summarizer
	topK                int
	similarityThreshold float64
	memoryTrunkSize     int
}

const (
	RoleAssistant = "assistant"
	RoleUser      = "user"
)

// Summarizer 定义记忆摘要行为。
type Summarizer interface {
	SummarizeLatestWindow(ctx context.Context, userID, appName string) error
}

// MemoryRepo 负责持久化摘要后的对话窗口并提供相似度检索。
// 生产实现通过 internal/storage 使用 GORM。
type MemoryRepo interface {
	AddMemory(ctx context.Context, mem types.Memory) error
	SearchSimilar(ctx context.Context, userID, appName, memoryType string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error)
}

// ChatHistoryRepo 维护滚动对话窗口，最终用于生成记忆。
// 它存储原始对话片段，并提供追加与窗口轮转能力。
type ChatHistoryRepo interface {
	GetLatestWindow(ctx context.Context, userID, appName string) (*types.ChatHistory, error)
	CreateWindow(ctx context.Context, history types.ChatHistory) error
	UpdateWindow(ctx context.Context, id int, content string, turnCount int) error
	MarkSummarized(ctx context.Context, id int) error
	GetRecent(ctx context.Context, userID, appName string, limit int) ([]types.ChatHistory, error)
}

// NewService 构建默认依赖的记忆服务。
func NewService(ctx context.Context, cfg *config.Config, memories MemoryRepo, chatHistories ChatHistoryRepo) adkmemory.Service {
	embedder, err := newEmbedder(ctx, cfg.GoogleAPIKey, cfg.EmbeddingModel)
	if err != nil {
		log.Fatalf("failed to create embedder service: %v", err)
	}

	summarizer, err := NewMemorySummarizer(ctx, cfg, chatHistories, memories, embedder)
	if err != nil {
		log.Fatalf("failed to create memory summarizer: %v", err)
	}
	return &memoryService{
		cfg:           cfg,
		embedder:      embedder,
		memories:      memories,
		chatHistories: chatHistories,
		summarizer:    summarizer,
	}
}

// AddSession 读取会话最新事件并维护滚动记忆窗口。
func (s *memoryService) AddSession(ctx context.Context, session session.Session) error {
	events := session.Events()
	if events.Len() == 0 {
		return nil
	}

	userID := session.UserID()
	appName := session.AppName()

	assistantText, userText := extractLatestPair(events)
	if assistantText == "" || userText == "" {
		return nil
	}
	newContent := fmt.Sprintf("%s: %s\n%s: %s\n", RoleUser, userText, RoleAssistant, assistantText)

	window, err := s.chatHistories.GetLatestWindow(ctx, userID, appName)
	if err != nil {
		return err
	}

	if window == nil || window.TurnCount >= s.memoryTrunkSize || window.Summarized {
		newWindow := types.ChatHistory{
			UserID:     userID,
			AppName:    appName,
			Content:    newContent,
			TurnCount:  2,
			Summarized: false,
		}
		if err := s.chatHistories.CreateWindow(ctx, newWindow); err != nil {
			return err
		}
		return nil
	}

	if err := s.chatHistories.UpdateWindow(ctx, window.ID, newContent, 2); err != nil {
		return err
	}
	return nil
}

func (s *memoryService) Search(ctx context.Context, req *adkmemory.SearchRequest) (*adkmemory.SearchResponse, error) {
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

func extractLatestPair(events session.Events) (assistantText, userText string) {
	if events == nil || events.Len() == 0 {
		return "", ""
	}
	assistantIndex := -1
	for i := events.Len() - 1; i >= 0; i-- {
		event := events.At(i)
		if event == nil || event.Content == nil {
			continue
		}
		if event.Content.Role != RoleAssistant {
			continue
		}
		text := strings.TrimSpace(utils.ExtractContentText(event.Content))
		if text == "" {
			continue
		}
		assistantText = text
		assistantIndex = i
		break
	}
	if assistantIndex == -1 {
		return "", ""
	}
	for i := assistantIndex - 1; i >= 0; i-- {
		event := events.At(i)
		if event == nil || event.Content == nil {
			continue
		}
		if event.Content.Role != RoleUser {
			continue
		}
		text := strings.TrimSpace(utils.ExtractContentText(event.Content))
		if text == "" {
			continue
		}
		userText = text
		break
	}
	return assistantText, userText
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
