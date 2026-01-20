package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store holds database connection pool and repositories.
type Store struct {
	pool        *pgxpool.Pool
	Characters  *CharacterRepo
	ChatHistory *ChatHistoryRepo
}

// NewStore initializes a PostgreSQL connection pool and repositories.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		pool:        pool,
		Characters:  NewCharacterRepo(pool),
		ChatHistory: NewChatHistoryRepo(pool),
	}
	return store, nil
}

// Pool exposes the underlying connection pool.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// Close closes the connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
