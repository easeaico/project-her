package storage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/types"
)

type characterModel struct {
	ID              int
	Name            string
	Description     string
	Appearance      string
	Personality     string
	Scenario        string
	FirstMessage    string
	ExampleDialogue string
	SystemPrompt    string
	AvatarPath      string
	Affection       int
	CurrentMood     string
	LastLabel       string
	MoodTurns       int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (characterModel) TableName() string {
	return "characters"
}

// CharacterRepo accesses characters data.
type characterRepo struct {
	db *gorm.DB
}

// NewCharacterRepo returns a CharacterRepo.
func NewCharacterRepo(db *gorm.DB) agent.CharacterRepo {
	return &characterRepo{db: db}
}

func (r *characterRepo) GetByID(ctx context.Context, id int) (*types.Character, error) {
	var model characterModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get character by id: %w", err)
	}
	return characterFromModel(model), nil
}

func (r *characterRepo) GetDefault(ctx context.Context) (*types.Character, error) {
	var model characterModel
	if err := r.db.WithContext(ctx).Order("id ASC").Limit(1).First(&model).Error; err != nil {
		return nil, fmt.Errorf("failed to get default character: %w", err)
	}
	return characterFromModel(model), nil
}

func (r *characterRepo) UpdateEmotion(ctx context.Context, id int, affection int, mood string, lastLabel string, moodTurns int) error {
	updates := map[string]any{
		"affection":    affection,
		"current_mood": mood,
		"last_label":   lastLabel,
		"mood_turns":   moodTurns,
		"updated_at":   gorm.Expr("NOW()"),
	}
	if err := r.db.WithContext(ctx).
		Model(&characterModel{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update emotion: %w", err)
	}
	return nil
}

func characterFromModel(model characterModel) *types.Character {
	return &types.Character{
		ID:              model.ID,
		Name:            model.Name,
		Description:     model.Description,
		Appearance:      model.Appearance,
		Personality:     model.Personality,
		Scenario:        model.Scenario,
		FirstMessage:    model.FirstMessage,
		ExampleDialogue: model.ExampleDialogue,
		SystemPrompt:    model.SystemPrompt,
		AvatarPath:      model.AvatarPath,
		Affection:       model.Affection,
		CurrentMood:     model.CurrentMood,
		LastLabel:       model.LastLabel,
		MoodTurns:       model.MoodTurns,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
	}
}
