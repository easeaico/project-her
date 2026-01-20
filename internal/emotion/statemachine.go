package emotion

// StateMachine updates affection and mood based on sentiment label.
type StateMachine struct{}

// NewStateMachine creates a new StateMachine.
func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// Update returns the updated emotion state.
func (s *StateMachine) Update(state EmotionState, label EmotionLabel) EmotionState {
	switch label {
	case EmotionPositive:
		state.Affection += 5
	case EmotionNegative:
		state.Affection -= 10
	case EmotionNeutral:
		state.Affection += 1
	}

	state.Affection = ClampAffection(state.Affection)
	state.CurrentMood = deriveMood(state.Affection, label, state.CurrentMood)
	return state
}

func deriveMood(affection int, label EmotionLabel, current string) string {
	switch label {
	case EmotionNegative:
		if affection <= 30 {
			return "Angry"
		}
		return "Sad"
	case EmotionPositive:
		return "Happy"
	case EmotionNeutral:
		if current != "" {
			return current
		}
		return "Neutral"
	default:
		return "Neutral"
	}
}
