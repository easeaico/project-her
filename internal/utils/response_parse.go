package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RoleplayOutput is the structured response from the roleplay model.
type RoleplayOutput struct {
	Reply   string `json:"reply"`
	Emotion string `json:"emotion"`
}

// ParseRoleplayOutput extracts and validates structured roleplay output.
func ParseRoleplayOutput(raw string) (RoleplayOutput, error) {
	clean := strings.TrimSpace(raw)
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		clean = clean[start : end+1]
	}

	var output RoleplayOutput
	if err := json.Unmarshal([]byte(clean), &output); err != nil {
		return RoleplayOutput{}, fmt.Errorf("failed to parse roleplay output: %w", err)
	}

	output.Reply = strings.TrimSpace(output.Reply)
	if output.Reply == "" {
		return RoleplayOutput{}, fmt.Errorf("missing reply")
	}

	emotion := strings.ToLower(strings.TrimSpace(output.Emotion))
	switch emotion {
	case "positive", "negative", "neutral":
		// normalize casing
		switch emotion {
		case "positive":
			output.Emotion = "Positive"
		case "negative":
			output.Emotion = "Negative"
		default:
			output.Emotion = "Neutral"
		}
	default:
		return RoleplayOutput{}, fmt.Errorf("invalid emotion label: %s", output.Emotion)
	}

	return output, nil
}
