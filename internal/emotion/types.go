package emotion

// EmotionLabel is a sentiment label.
type EmotionLabel string

const (
	EmotionPositive EmotionLabel = "Positive"
	EmotionNegative EmotionLabel = "Negative"
	EmotionNeutral  EmotionLabel = "Neutral"
)

// EmotionState is current affection and mood.
type EmotionState struct {
	Affection   int
	CurrentMood string
	MoodTurns   int
	LastLabel   string
}

// ClampAffection bounds affection to 0-100.
func ClampAffection(score int) int {
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}
