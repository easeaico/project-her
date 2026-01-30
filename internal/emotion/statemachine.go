package emotion

// StateMachine updates affection and mood with hysteresis to avoid rapid flips.
type StateMachine struct{}

const (
	minMoodTurns      = 2
	negativeThreshold = 2
	positiveThreshold = 2
)

// NewStateMachine returns a StateMachine.
func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// Update returns the updated emotion state using label streak thresholds.
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

	labelStr := string(label)
	streak := 1
	if state.LastLabel == labelStr {
		streak = state.MoodTurns + 1
	}

	desired := deriveMood(state.Affection, label, state.CurrentMood)
	switch label {
	case EmotionPositive:
		// Only switch to Happy after consecutive positives.
		if desired != state.CurrentMood && streak >= positiveThreshold && streak >= minMoodTurns {
			state.CurrentMood = desired
		}
	case EmotionNegative:
		// Only switch to Sad/Angry after consecutive negatives.
		if desired != state.CurrentMood && streak >= negativeThreshold && streak >= minMoodTurns {
			state.CurrentMood = desired
		}
	case EmotionNeutral:
		// Keep current mood for neutral signals to stabilize.
	}

	state.LastLabel = labelStr
	state.MoodTurns = streak
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
