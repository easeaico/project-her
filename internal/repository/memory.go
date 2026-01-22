package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/easeaico/project-her/internal/types"
)

type memoryModel struct {
	ID          int
	UserID      string
	SessionID   string
	CharacterID int
	Type        string
	Role        string
	Content     string
	Embedding   *pgvector.Vector `gorm:"type:vector"`
	CreatedAt   time.Time
}

func (memoryModel) TableName() string {
	return "memories"
}

// MemoriesRepo accesses memory data.
type MemoryRepo struct {
	db *gorm.DB
}

// NewMemoriesRepo returns a MemoriesRepo.
func NewMemoryRepo(db *gorm.DB) *MemoryRepo {
	return &MemoryRepo{db: db}
}

func (r *MemoryRepo) AddMemory(ctx context.Context, mem types.Memory) error {
	var vector *pgvector.Vector
	if len(mem.Embedding) > 0 {
		v := pgvector.NewVector(mem.Embedding)
		vector = &v
	}
	record := memoryModel{
		UserID:      mem.UserID,
		SessionID:   mem.SessionID,
		CharacterID: mem.CharacterID,
		Type:        mem.Type,
		Role:        mem.Role,
		Content:     mem.Content,
		Embedding:   vector,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("failed to insert memory: %w", err)
	}
	return nil
}

func (r *MemoryRepo) GetRecentMemories(ctx context.Context, sessionID, memoryType string, limit int) ([]types.Memory, error) {
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if memoryType != "" {
		query = query.Where("type = ?", memoryType)
	}

	var records []memoryModel
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}

	results := make([]types.Memory, 0, len(records))
	for _, record := range records {
		results = append(results, memoryFromModel(record))
	}

	// Oldest -> newest
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results, nil
}

func (r *MemoryRepo) SearchSimilar(ctx context.Context, userID, appName, memoryType string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	conditions := "embedding IS NOT NULL AND 1 - (embedding <=> $1) > $2"
	args := []any{pgvector.NewVector(embedding), threshold}
	argIndex := 3

	if userID != "" {
		conditions += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}
	if memoryType != "" {
		conditions += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, memoryType)
		argIndex++
	}

	query := fmt.Sprintf(`
		SELECT role, content, type, created_at, 1 - (embedding <=> $1) AS similarity
		FROM memories
		WHERE %s
		ORDER BY similarity DESC
		LIMIT $%d`, conditions, argIndex)

	args = append(args, topK)

	var results []types.RetrievedMemory
	if err := r.db.WithContext(ctx).
		Raw(query, args...).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to search similar memories: %w", err)
	}
	return results, nil
}

func memoryFromModel(model memoryModel) types.Memory {
	return types.Memory{
		ID:          model.ID,
		UserID:      model.UserID,
		SessionID:   model.SessionID,
		CharacterID: model.CharacterID,
		Type:        model.Type,
		Role:        model.Role,
		Content:     model.Content,
		CreatedAt:   model.CreatedAt,
	}
}
