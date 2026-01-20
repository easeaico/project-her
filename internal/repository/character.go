package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/easeaico/adk-memory-agent/internal/types"
)

// CharacterRepo provides access to characters table.
type CharacterRepo struct {
	pool *pgxpool.Pool
}

// NewCharacterRepo creates a new CharacterRepo.
func NewCharacterRepo(pool *pgxpool.Pool) *CharacterRepo {
	return &CharacterRepo{pool: pool}
}

// GetByID fetches a character by ID.
func (r *CharacterRepo) GetByID(ctx context.Context, id int) (*types.Character, error) {
	query := `
		SELECT id, name, description, personality, scenario, first_message,
		       example_dialogue, system_prompt, avatar_path, affection, current_mood,
		       created_at, updated_at
		FROM characters
		WHERE id = $1`
	row := r.pool.QueryRow(ctx, query, id)

	var c types.Character
	if err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&c.Personality,
		&c.Scenario,
		&c.FirstMessage,
		&c.ExampleDialogue,
		&c.SystemPrompt,
		&c.AvatarPath,
		&c.Affection,
		&c.CurrentMood,
		&c.CreatedAt,
		&c.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get character by id: %w", err)
	}
	return &c, nil
}

// GetDefault fetches the first available character.
func (r *CharacterRepo) GetDefault(ctx context.Context) (*types.Character, error) {
	query := `
		SELECT id, name, description, personality, scenario, first_message,
		       example_dialogue, system_prompt, avatar_path, affection, current_mood,
		       created_at, updated_at
		FROM characters
		ORDER BY id ASC
		LIMIT 1`
	row := r.pool.QueryRow(ctx, query)

	var c types.Character
	if err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Description,
		&c.Personality,
		&c.Scenario,
		&c.FirstMessage,
		&c.ExampleDialogue,
		&c.SystemPrompt,
		&c.AvatarPath,
		&c.Affection,
		&c.CurrentMood,
		&c.CreatedAt,
		&c.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get default character: %w", err)
	}
	return &c, nil
}

// UpdateEmotion updates a character's affection and mood.
func (r *CharacterRepo) UpdateEmotion(ctx context.Context, id int, affection int, mood string) error {
	query := `
		UPDATE characters
		SET affection = $2, current_mood = $3, updated_at = NOW()
		WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, affection, mood)
	if err != nil {
		return fmt.Errorf("failed to update emotion: %w", err)
	}
	return nil
}
