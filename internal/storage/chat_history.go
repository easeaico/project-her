package storage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/easeaico/project-her/internal/types"
)

// chatHistoryModel maps to the chat_histories table.
type chatHistoryModel struct {
	ID          int
	UserID      string
	AppName     string
	Content     string
	TurnCount   int
	Summarized  bool
	CreatedAt   time.Time
}

func (chatHistoryModel) TableName() string {
	return "chat_histories"
}

// ChatHistoryRepo accesses chat history data.
type ChatHistoryRepo struct {
	db *gorm.DB
}

// NewChatHistoryRepo returns a ChatHistoryRepo.
func NewChatHistoryRepo(db *gorm.DB) *ChatHistoryRepo {
	return &ChatHistoryRepo{db: db}
}

func (r *ChatHistoryRepo) CreateWindow(ctx context.Context, history types.ChatHistory) error {
	record := chatHistoryModel{
		UserID:     history.UserID,
		AppName:    history.AppName,
		Content:    history.Content,
		TurnCount:  history.TurnCount,
		Summarized: history.Summarized,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("failed to insert chat history: %w", err)
	}
	return nil
}

func (r *ChatHistoryRepo) GetLatestWindow(ctx context.Context, userID, appName string) (*types.ChatHistory, error) {
	query := r.db.WithContext(ctx).
		Where("summarized = ?", false).
		Order("created_at DESC").
		Limit(1)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if appName != "" {
		query = query.Where("app_name = ?", appName)
	}

	var record chatHistoryModel
	if err := query.Find(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to query latest chat window: %w", err)
	}
	if record.ID == 0 {
		return nil, nil
	}
	result := chatHistoryFromModel(record)
	return &result, nil
}

func (r *ChatHistoryRepo) AppendToWindow(ctx context.Context, id int, content string, turnCount int) error {
	if err := r.db.WithContext(ctx).
		Model(&chatHistoryModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"content":    content,
			"turn_count": turnCount,
		}).Error; err != nil {
		return fmt.Errorf("failed to append chat window: %w", err)
	}
	return nil
}

func (r *ChatHistoryRepo) GetRecent(ctx context.Context, userID, appName string, limit int) ([]types.ChatHistory, error) {
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if appName != "" {
		query = query.Where("app_name = ?", appName)
	}

	var records []chatHistoryModel
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query chat histories: %w", err)
	}

	results := make([]types.ChatHistory, 0, len(records))
	for _, record := range records {
		results = append(results, chatHistoryFromModel(record))
	}

	// Oldest -> newest
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}
	return results, nil
}

func (r *ChatHistoryRepo) MarkSummarized(ctx context.Context, id int) error {
	if err := r.db.WithContext(ctx).
		Model(&chatHistoryModel{}).
		Where("id = ?", id).
		Update("summarized", true).Error; err != nil {
		return fmt.Errorf("failed to mark chat history summarized: %w", err)
	}
	return nil
}

func chatHistoryFromModel(model chatHistoryModel) types.ChatHistory {
	return types.ChatHistory{
		ID:         model.ID,
		UserID:     model.UserID,
		AppName:    model.AppName,
		Content:    model.Content,
		TurnCount:  model.TurnCount,
		Summarized: model.Summarized,
		CreatedAt:  model.CreatedAt,
	}
}
