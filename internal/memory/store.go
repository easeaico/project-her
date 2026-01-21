package memory

import (
	"context"

	"github.com/easeaico/adk-memory-agent/internal/repository"
)

// Store wraps repository.Store for ADK compatibility.
type Store struct {
	*repository.Store
}

// NewPostgresStore initializes a PostgreSQL-backed store.
func NewPostgresStore(ctx context.Context, databaseURL string) (*Store, error) {
	store, err := repository.NewStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{Store: store}, nil
}
