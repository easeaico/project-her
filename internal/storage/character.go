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
	ID           int
	Name         string
	Description  string
	Personality  string
	Scenario     string
	FirstMessage string `gorm:"column:first_mes"`
	MesExample   string `gorm:"column:mes_example"`
	SystemPrompt string
	Avatar       string `gorm:"column:avatar"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

func characterFromModel(model characterModel) *types.Character {
	return &types.Character{
		ID:             model.ID,
		Name:           model.Name,
		Description:    model.Description,
		Personality:    model.Personality,
		Scenario:       model.Scenario,
		FirstMessage:   model.FirstMessage,
		MessageExample: model.MesExample,
		SystemPrompt:   model.SystemPrompt,
		Avatar:         model.Avatar,
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
