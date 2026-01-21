package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/easeaico/adk-memory-agent/internal/types"
)

type chatMessageModel struct {
	ID          int
	SessionID   string
	CharacterID int
	Role        string
	Content     string
	Embedding   *pgvector.Vector `gorm:"type:vector"`
	CreatedAt   time.Time
}

func (chatMessageModel) TableName() string {
	return "chat_history"
}

// ChatHistoryRepo accesses chat history data.
type ChatHistoryRepo struct {
	db *gorm.DB
}

// NewChatHistoryRepo returns a ChatHistoryRepo.
func NewChatHistoryRepo(db *gorm.DB) *ChatHistoryRepo {
	return &ChatHistoryRepo{db: db}
}

func (r *ChatHistoryRepo) AddMessage(ctx context.Context, msg types.ChatMessage, embedding []float32) error {
	var vector *pgvector.Vector
	if len(embedding) > 0 {
		v := pgvector.NewVector(embedding)
		vector = &v
	}
	record := chatMessageModel{
		SessionID:   msg.SessionID,
		CharacterID: msg.CharacterID,
		Role:        msg.Role,
		Content:     msg.Content,
		Embedding:   vector,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("failed to insert chat message: %w", err)
	}
	return nil
}

func (r *ChatHistoryRepo) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]types.ChatMessage, error) {
	var records []chatMessageModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}

	results := make([]types.ChatMessage, 0, len(records))
	for _, record := range records {
		results = append(results, chatMessageFromModel(record))
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
	var results []types.RetrievedMemory
	if err := r.db.WithContext(ctx).
		Raw(query, vector, sessionID, threshold, topK).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to search similar memories: %w", err)
	}
	return results, nil
}

func chatMessageFromModel(model chatMessageModel) types.ChatMessage {
	return types.ChatMessage{
		ID:          model.ID,
		SessionID:   model.SessionID,
		CharacterID: model.CharacterID,
		Role:        model.Role,
		Content:     model.Content,
		CreatedAt:   model.CreatedAt,
	}
}
