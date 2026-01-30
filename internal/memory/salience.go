package memory

import (
	"unicode/utf8"

	"github.com/easeaico/project-her/internal/types"
)

// ComputeSalience calculates a deterministic salience score in [0,1] based on key memory signals
// and optional emotion state.
func ComputeSalience(summary types.MemorySummary, state *EmotionState) float64 {
	score := 0.0

	if summary.Summary != "" {
		score += 0.10
	}

	factsCount := len(summary.Facts)
	if factsCount > 3 {
		factsCount = 3
	}
	score += float64(factsCount) * 0.15

	commitCount := len(summary.Commitments)
	if commitCount > 2 {
		commitCount = 2
	}
	score += float64(commitCount) * 0.20

	emotionCount := len(summary.Emotions)
	if emotionCount > 2 {
		emotionCount = 2
	}
	score += float64(emotionCount) * 0.10

	if summary.TimeRange.Start != "" || summary.TimeRange.End != "" {
		score += 0.05
	}

	summaryLen := utf8.RuneCountInString(summary.Summary)
	if summaryLen >= 200 {
		score += 0.10
	} else if summaryLen >= 100 {
		score += 0.05
	}

	if state != nil {
		switch state.Mood {
		case "Angry", "Sad":
			score += 0.10
		case "Happy":
			score += 0.05
		}
		switch {
		case state.Affection <= 20:
			score += 0.05
		case state.Affection >= 80:
			score += 0.03
		}
	}

	return clampScore(score)
}

func clampScore(score float64) float64 {
	score = normalizeSalience(score)
	return score
}
