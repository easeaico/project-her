package emotion

import (
	"context"
	"testing"

	"github.com/easeaico/project-her/internal/types"
)

type fakeCharacterRepo struct {
	character *types.Character
	updated   *EmotionState
	lastLabel string
	moodTurns int
}

func (r *fakeCharacterRepo) GetByID(ctx context.Context, id int) (*types.Character, error) {
	return r.character, nil
}

func (r *fakeCharacterRepo) GetDefault(ctx context.Context) (*types.Character, error) {
	return r.character, nil
}

func (r *fakeCharacterRepo) UpdateEmotion(ctx context.Context, id int, affection int, mood string, lastLabel string, moodTurns int) error {
	r.updated = &EmotionState{Affection: affection, CurrentMood: mood}
	r.lastLabel = lastLabel
	r.moodTurns = moodTurns
	return nil
}

func TestServiceUpdateFromLabelPositive(t *testing.T) {
	repo := &fakeCharacterRepo{character: &types.Character{ID: 1, Affection: 50, CurrentMood: "Neutral"}}
	service := NewService(NewStateMachine(), repo, 1)

	if err := service.UpdateFromLabel(context.Background(), EmotionPositive); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updated == nil || repo.updated.Affection != 55 || repo.updated.CurrentMood != "Neutral" {
		t.Fatalf("unexpected update: %#v", repo.updated)
	}
	if repo.lastLabel != "Positive" || repo.moodTurns != 1 {
		t.Fatalf("unexpected label tracking: %s/%d", repo.lastLabel, repo.moodTurns)
	}
}

func TestServiceUpdateFromLabelNegativeLowAffection(t *testing.T) {
	repo := &fakeCharacterRepo{character: &types.Character{ID: 1, Affection: 20, CurrentMood: "Neutral"}}
	service := NewService(NewStateMachine(), repo, 1)

	if err := service.UpdateFromLabel(context.Background(), EmotionNegative); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updated == nil || repo.updated.Affection != 10 || repo.updated.CurrentMood != "Neutral" {
		t.Fatalf("unexpected update: %#v", repo.updated)
	}
	if repo.lastLabel != "Negative" || repo.moodTurns != 1 {
		t.Fatalf("unexpected label tracking: %s/%d", repo.lastLabel, repo.moodTurns)
	}
}

func TestServiceUpdateFromLabelNeutralKeepsMood(t *testing.T) {
	repo := &fakeCharacterRepo{character: &types.Character{ID: 1, Affection: 50, CurrentMood: "Sad"}}
	service := NewService(NewStateMachine(), repo, 1)

	if err := service.UpdateFromLabel(context.Background(), EmotionNeutral); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updated == nil || repo.updated.Affection != 51 || repo.updated.CurrentMood != "Sad" {
		t.Fatalf("unexpected update: %#v", repo.updated)
	}
	if repo.lastLabel != "Neutral" || repo.moodTurns != 1 {
		t.Fatalf("unexpected label tracking: %s/%d", repo.lastLabel, repo.moodTurns)
	}
}

func TestServiceUpdateFromLabelNegativeTwiceFlipsMood(t *testing.T) {
	repo := &fakeCharacterRepo{character: &types.Character{ID: 1, Affection: 40, CurrentMood: "Neutral", LastLabel: "Negative", MoodTurns: 1}}
	service := NewService(NewStateMachine(), repo, 1)

	if err := service.UpdateFromLabel(context.Background(), EmotionNegative); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updated == nil || repo.updated.CurrentMood != "Sad" {
		t.Fatalf("expected mood to change to Sad, got %#v", repo.updated)
	}
	if repo.moodTurns != 2 {
		t.Fatalf("expected moodTurns 2, got %d", repo.moodTurns)
	}
}

func TestServiceUpdateFromLabelPositiveTwiceFlipsMood(t *testing.T) {
	repo := &fakeCharacterRepo{character: &types.Character{ID: 1, Affection: 60, CurrentMood: "Sad", LastLabel: "Positive", MoodTurns: 1}}
	service := NewService(NewStateMachine(), repo, 1)

	if err := service.UpdateFromLabel(context.Background(), EmotionPositive); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updated == nil || repo.updated.CurrentMood != "Happy" {
		t.Fatalf("expected mood to change to Happy, got %#v", repo.updated)
	}
	if repo.moodTurns != 2 {
		t.Fatalf("expected moodTurns 2, got %d", repo.moodTurns)
	}
}
