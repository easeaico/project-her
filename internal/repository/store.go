package repository

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Store holds the DB pool and repositories.
type Store struct {
	db          *gorm.DB
	Characters  *CharacterRepo
	ChatHistory *ChatHistoryRepo
}

// NewStore initializes the PostgreSQL pool and repositories.
func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql db: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{
		db:          db,
		Characters:  NewCharacterRepo(db),
		ChatHistory: NewChatHistoryRepo(db),
	}
	return store, nil
}

func (s *Store) Pool() *gorm.DB {
	return s.db
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Close() {
	if s.db == nil {
		return
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}
