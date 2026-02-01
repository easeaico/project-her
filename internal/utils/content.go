package utils

import (
	"strings"

	"google.golang.org/genai"
)

func ExtractContentText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range content.Parts {
		if part != nil && part.Text != "" {
			sb.WriteString(part.Text)
		}
	}
	return sb.String()
}

func NormalizePromptText(text string, charName, userName string) string {
	text = strings.ReplaceAll(text, "{{char}}", charName)
	text = strings.ReplaceAll(text, "{{user}}", userName)
	text = strings.ReplaceAll(text, "\\r\\n", "\n")
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\\"", "\"")
	return text
}
