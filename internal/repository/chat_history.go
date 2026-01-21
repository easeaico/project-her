package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/easeaico/adk-memory-agent/internal/types"
)

// ChatHistoryRepo accesses chat history data.
type ChatHistoryRepo struct {
	pool *pgxpool.Pool
}

// NewChatHistoryRepo returns a ChatHistoryRepo.
func NewChatHistoryRepo(pool *pgxpool.Pool) *ChatHistoryRepo {
	return &ChatHistoryRepo{pool: pool}
}

func (r *ChatHistoryRepo) AddMessage(ctx context.Context, msg types.ChatMessage, embedding []float32) error {
	query := `
		INSERT INTO chat_history (session_id, character_id, role, content, embedding)
		VALUES ($1, $2, $3, $4, $5)`

	var vector pgvector.Vector
	if len(embedding) > 0 {
		vector = pgvector.NewVector(embedding)
	}

	_, err := r.pool.Exec(ctx, query, msg.SessionID, msg.CharacterID, msg.Role, msg.Content, vector)
	if err != nil {
		return fmt.Errorf("failed to insert chat message: %w", err)
	}
	return nil
}

func (r *ChatHistoryRepo) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]types.ChatMessage, error) {
	query := `
		SELECT id, session_id, character_id, role, content, created_at
		FROM chat_history
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}
	defer rows.Close()

	var results []types.ChatMessage
	for rows.Next() {
		var msg types.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.CharacterID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chat history row: %w", err)
		}
		results = append(results, msg)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read chat history rows: %w", rows.Err())
	}

	// Oldest -> newest
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results, nil
}

func (r *ChatHistoryRepo) SearchSimilar(ctx context.Context, sessionID string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	query := `
		SELECT role, content, created_at, 1 - (embedding <=> $1) AS similarity
		FROM chat_history
		WHERE session_id = $2
		  AND embedding IS NOT NULL
		  AND 1 - (embedding <=> $1) > $3
		ORDER BY similarity DESC
		LIMIT $4`

	vector := pgvector.NewVector(embedding)
	rows, err := r.pool.Query(ctx, query, vector, sessionID, threshold, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar memories: %w", err)
	}
	defer rows.Close()

	var results []types.RetrievedMemory
	for rows.Next() {
		var memory types.RetrievedMemory
		var createdAt time.Time
		if err := rows.Scan(&memory.Role, &memory.Content, &createdAt, &memory.Similarity); err != nil {
			return nil, fmt.Errorf("failed to scan memory row: %w", err)
		}
		memory.CreatedAt = createdAt
		results = append(results, memory)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to read memory rows: %w", rows.Err())
	}
	return results, nil
}

var _ = pgx.ErrNoRows
