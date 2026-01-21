package memory

import (
	"context"
	"fmt"

	"github.com/easeaico/adk-memory-agent/internal/repository"
	"github.com/easeaico/adk-memory-agent/internal/types"
)

// Retriever performs semantic search over chat history.
type Retriever struct {
	embedder             Embedder
	chatHistoryRepo      *repository.ChatHistoryRepo
	topK                 int
	similarityThreshold  float64
}

// NewRetriever returns a retriever.
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
