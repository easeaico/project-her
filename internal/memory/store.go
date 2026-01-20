package memory

import (
	"context"

	"github.com/easeaico/adk-memory-agent/internal/repository"
)

// Store is a thin wrapper around repository.Store for compatibility.
type Store struct {
	*repository.Store
}

// NewPostgresStore initializes a Store backed by PostgreSQL.
func NewPostgresStore(ctx context.Context, databaseURL string) (*Store, error) {
	store, err := repository.NewStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{Store: store}, nil
}
