package memory

import (
	"context"
	"fmt"

	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/types"
)

// Service implements ADK memory.Service and RAG helpers.
type service struct {
	embedder            Embedder
	memories            MemoryRepo
	topK                int
	similarityThreshold float64
}

type MemoryRepo interface {
	AddMemory(ctx context.Context, mem types.Memory) error
	SearchSimilar(ctx context.Context, userID, appName, memoryType string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error)
}

// NewService returns a memory service.
func NewService(embedder Embedder, memories MemoryRepo, topK int, threshold float64) adkmemory.Service {
	return &service{
		embedder:            embedder,
		memories:            memories,
		topK:                topK,
		similarityThreshold: threshold,
	}
}

// AddSession is a no-op.
func (s *service) AddSession(ctx context.Context, session session.Session) error {
	characterID, err := session.State().Get("character_id")
	if err != nil {
		return err
	}
	if characterID == nil {
		return fmt.Errorf("character_id is not set")
	}

	events := session.Events()
	if events.Len() == 0 {
		return nil
	}

	event := events.At(events.Len() - 1)
	content := event.Content.Parts[0].Text
	embedding, err := s.embedder.EmbedDocument(ctx, content)
	if err != nil {
		return err
	}

	mem := types.Memory{
		UserID:      session.UserID(),
		SessionID:   session.ID(),
		CharacterID: characterID.(int),
		Type:        types.MemoryTypeChat,
		Role:        "assistant",
		Content:     content,
		Embedding:   embedding,
	}
	err = s.memories.AddMemory(ctx, mem)
	if err != nil {
		return err
	}

	return nil
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
