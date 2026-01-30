package callback

import (
	"log/slog"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/emotion"
	"github.com/easeaico/project-her/internal/utils"
)

// NewEmotionCallback parses structured JSON output, updates emotion state, and rewrites reply content
// so the user only sees the reply field.
func NewEmotionCallback(service *emotion.Service) llmagent.AfterModelCallback {
	return func(ctx agent.CallbackContext, resp *model.LLMResponse, err error) (*model.LLMResponse, error) {
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Content == nil {
			return nil, nil
		}
		if resp.Partial {
			return nil, nil
		}

		text := strings.TrimSpace(utils.ExtractContentText(resp.Content))
		if text == "" {
			return nil, nil
		}

		parsed, parseErr := utils.ParseRoleplayOutput(text)
		if parseErr != nil {
			slog.Warn("failed to parse roleplay output", "error", parseErr.Error())
			return nil, nil
		}

		if service != nil {
			label := emotion.EmotionLabel(parsed.Emotion)
			if updateErr := service.UpdateFromLabel(ctx, label); updateErr != nil {
				slog.Error("failed to update emotion state", "error", updateErr.Error())
			}
		}

		resp.Content = genai.NewContentFromText(parsed.Reply, "assistant")
		return resp, nil
	}
}
