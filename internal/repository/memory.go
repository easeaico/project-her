package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/easeaico/project-her/internal/types"
)

// memoryModel maps to the memories table.
type memoryModel struct {
	ID          int
	UserID      string
	AppName     string
	CharacterID int
	Type        string
	Summary     string
	// Facts/Commitments/Emotions/TimeRange are stored as JSONB for retrieval filters.
	Facts       json.RawMessage `gorm:"type:jsonb"`
	Commitments json.RawMessage `gorm:"type:jsonb"`
	Emotions    json.RawMessage `gorm:"type:jsonb"`
	TimeRange   json.RawMessage `gorm:"type:jsonb"`
	// Salience is a 0-1 importance score, used in ranking.
	Salience    float64 `gorm:"column:salience_score"`
	// Embedding stores vector representation for similarity search.
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
	// Marshal structured fields into JSONB.
	facts, err := marshalJSON(mem.Facts)
	if err != nil {
		return fmt.Errorf("failed to encode memory facts: %w", err)
	}
	commitments, err := marshalJSON(mem.Commitments)
	if err != nil {
		return fmt.Errorf("failed to encode memory commitments: %w", err)
	}
	emotions, err := marshalJSON(mem.Emotions)
	if err != nil {
		return fmt.Errorf("failed to encode memory emotions: %w", err)
	}
	timeRange, err := marshalJSON(mem.TimeRange)
	if err != nil {
		return fmt.Errorf("failed to encode memory time range: %w", err)
	}
	record := memoryModel{
		UserID:      mem.UserID,
		AppName:     mem.AppName,
		CharacterID: mem.CharacterID,
		Type:        mem.Type,
		Summary:     mem.Summary,
		Facts:       facts,
		Commitments: commitments,
		Emotions:    emotions,
		TimeRange:   timeRange,
		Salience:    mem.Salience,
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

	// Filter by cosine similarity and then re-rank by salience.
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
		SELECT 'assistant' AS role, summary AS content, type, created_at,
		       1 - (embedding <=> $1) AS similarity,
		       COALESCE(salience_score, 0) AS salience_score
		FROM memories
		WHERE %s
		ORDER BY (0.85 * similarity + 0.15 * COALESCE(salience_score, 0)) DESC
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

// memoryFromModel converts database model to domain struct.
func memoryFromModel(model memoryModel) types.Memory {
	var facts []string
	var commitments []string
	var emotions []string
	var timeRange types.TimeRange
	_ = unmarshalJSON(model.Facts, &facts)
	_ = unmarshalJSON(model.Commitments, &commitments)
	_ = unmarshalJSON(model.Emotions, &emotions)
	_ = unmarshalJSON(model.TimeRange, &timeRange)
	return types.Memory{
		ID:          model.ID,
		UserID:      model.UserID,
		AppName:     model.AppName,
		CharacterID: model.CharacterID,
		Type:        model.Type,
		Summary:     model.Summary,
		Facts:       facts,
		Commitments: commitments,
		Emotions:    emotions,
		TimeRange:   timeRange,
		Salience:    model.Salience,
		CreatedAt:   model.CreatedAt,
	}
}

// marshalJSON encodes a value into JSONB, returning nil for empty values.
func marshalJSON(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	return json.RawMessage(raw), nil
}

// unmarshalJSON decodes JSONB into the provided target.
func unmarshalJSON(data json.RawMessage, target any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}
