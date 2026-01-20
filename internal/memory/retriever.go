package memory

import (
	"context"
	"fmt"

	"github.com/easeaico/adk-memory-agent/internal/repository"
	"github.com/easeaico/adk-memory-agent/internal/types"
)

// Retriever provides semantic search against chat history.
type Retriever struct {
	embedder             Embedder
	chatHistoryRepo      *repository.ChatHistoryRepo
	topK                 int
	similarityThreshold  float64
}

// NewRetriever creates a new Retriever.
func NewRetriever(embedder Embedder, chatHistoryRepo *repository.ChatHistoryRepo, topK int, threshold float64) *Retriever {
	if topK <= 0 {
		topK = 5
	}
	if threshold <= 0 {
		threshold = 0.7
	}
	return &Retriever{
		embedder:            embedder,
		chatHistoryRepo:     chatHistoryRepo,
		topK:                topK,
		similarityThreshold: threshold,
	}
}

// Retrieve returns top-k memories for a given query.
func (r *Retriever) Retrieve(ctx context.Context, sessionID, query string) ([]types.RetrievedMemory, error) {
	if query == "" {
		return nil, nil
	}
	if r.embedder == nil || r.chatHistoryRepo == nil {
		return nil, fmt.Errorf("retriever not properly configured")
	}

	vec, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	return r.chatHistoryRepo.SearchSimilar(ctx, sessionID, vec, r.topK, r.similarityThreshold)
}
