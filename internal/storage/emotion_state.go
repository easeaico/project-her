package storage

import (
	"context"
	"fmt"

	"github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/memory"
	"github.com/easeaico/project-her/internal/types"
)

// EmotionStateProvider implements memory.EmotionStateProvider using character data.
type EmotionStateProvider struct {
	characters  agent.CharacterRepo
	characterID int
}

// NewEmotionStateProvider returns a provider for emotion state.
func NewEmotionStateProvider(characters agent.CharacterRepo, characterID int) *EmotionStateProvider {
	return &EmotionStateProvider{
		characters:  characters,
		characterID: characterID,
	}
}

// GetEmotionState returns the current emotion state for the configured character.
func (p *EmotionStateProvider) GetEmotionState(ctx context.Context, userID, appName string) (memory.EmotionState, error) {
	if p.characters == nil {
		return memory.EmotionState{}, fmt.Errorf("character repo is nil")
	}

	var character *types.Character
	if p.characterID > 0 {
		char, getErr := p.characters.GetByID(ctx, p.characterID)
		if getErr != nil {
			return memory.EmotionState{}, fmt.Errorf("failed to get character by id: %w", getErr)
		}
		character = char
	} else {
		char, getErr := p.characters.GetDefault(ctx)
		if getErr != nil {
			return memory.EmotionState{}, fmt.Errorf("failed to get default character: %w", getErr)
		}
		character = char
	}

	if character == nil {
		return memory.EmotionState{}, fmt.Errorf("character not found")
	}

	return memory.EmotionState{
		Affection: character.Affection,
		Mood:      character.CurrentMood,
	}, nil
}
