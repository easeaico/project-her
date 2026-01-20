package memory

import (
	"context"
	"log/slog"
	"time"

	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/adk-memory-agent/internal/repository"
	"github.com/easeaico/adk-memory-agent/internal/types"
)

// Service implements ADK memory.Service and provides RAG helpers.
type Service struct {
	embedder            Embedder
	store               *repository.Store
	retriever           *Retriever
	topK                int
	similarityThreshold float64
	logger              *slog.Logger
}

// NewService creates a memory service with retrieval capability.
func NewService(embedder Embedder, store *repository.Store, topK int, threshold float64) *Service {
	retriever := NewRetriever(embedder, store.ChatHistory, topK, threshold)
	return &Service{
		embedder:            embedder,
		store:               store,
		retriever:           retriever,
		topK:                topK,
		similarityThreshold: threshold,
		logger:              slog.Default(),
	}
}

// AddSession is a no-op for this demo implementation.
func (s *Service) AddSession(ctx context.Context, _ session.Session) error {
	return nil
}

// Search returns memory entries relevant to the query.
func (s *Service) Search(ctx context.Context, req *adkmemory.SearchRequest) (*adkmemory.SearchResponse, error) {
	if req == nil || req.Query == "" {
		return &adkmemory.SearchResponse{Memories: nil}, nil
	}

	// No session_id in SearchRequest; return empty response for now.
	return &adkmemory.SearchResponse{Memories: nil}, nil
}

// RetrieveMemories returns retrieved memories for the given session and query.
func (s *Service) RetrieveMemories(ctx context.Context, sessionID, query string) ([]types.RetrievedMemory, error) {
	return s.retriever.Retrieve(ctx, sessionID, query)
}

// SaveMessageAsync saves a message with embedding asynchronously.
func (s *Service) SaveMessageAsync(parent context.Context, msg types.ChatMessage, embedAsQuery bool) {
	if s == nil || s.store == nil || s.embedder == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var vec []float32
		var err error
		if embedAsQuery {
			vec, err = s.embedder.EmbedQuery(ctx, msg.Content)
		} else {
			vec, err = s.embedder.EmbedDocument(ctx, msg.Content)
		}
		if err != nil {
			s.logger.Warn("failed to embed message", "error", err.Error())
			return
		}

		if err := s.store.ChatHistory.AddMessage(ctx, msg, vec); err != nil {
			s.logger.Warn("failed to save message", "error", err.Error())
		}
	}()
}

// ToMemoryEntries converts retrieved memories to ADK memory entries.
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
