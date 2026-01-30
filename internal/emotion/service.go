package emotion

import (
	"context"
	"fmt"

	"github.com/easeaico/project-her/internal/types"
)

// CharacterRepo defines emotion update and fetch behavior.
type CharacterRepo interface {
	GetByID(ctx context.Context, id int) (*types.Character, error)
	GetDefault(ctx context.Context) (*types.Character, error)
	UpdateEmotion(ctx context.Context, id int, affection int, mood string, lastLabel string, moodTurns int) error
}

// Service updates emotion state based on labels.
type Service struct {
	stateMachine *StateMachine
	characters   CharacterRepo
	characterID  int
}

// NewService returns a new emotion service.
func NewService(stateMachine *StateMachine, characters CharacterRepo, characterID int) *Service {
	return &Service{
		stateMachine: stateMachine,
		characters:   characters,
		characterID:  characterID,
	}
}

// UpdateFromLabel updates affection and mood based on sentiment label.
func (s *Service) UpdateFromLabel(ctx context.Context, label EmotionLabel) error {
	if s == nil || s.stateMachine == nil {
		return fmt.Errorf("emotion service not configured")
	}
	if s.characters == nil {
		return fmt.Errorf("character repo is nil")
	}

	var character *types.Character
	if s.characterID > 0 {
		char, err := s.characters.GetByID(ctx, s.characterID)
		if err != nil {
			return fmt.Errorf("failed to get character by id: %w", err)
		}
		character = char
	} else {
		char, err := s.characters.GetDefault(ctx)
		if err != nil {
			return fmt.Errorf("failed to get default character: %w", err)
		}
		character = char
	}

	if character == nil {
		return fmt.Errorf("character not found")
	}

	next := s.stateMachine.Update(EmotionState{
		Affection:   character.Affection,
		CurrentMood: character.CurrentMood,
		MoodTurns:   character.MoodTurns,
		LastLabel:   character.LastLabel,
	}, label)

	if err := s.characters.UpdateEmotion(ctx, character.ID, next.Affection, next.CurrentMood, next.LastLabel, next.MoodTurns); err != nil {
		return fmt.Errorf("failed to update emotion: %w", err)
	}
	return nil
}
