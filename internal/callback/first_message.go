package callback

import (
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

// NewFirstMessageCallback returns first_mes on empty first user input.
func NewFirstMessageCallback(character *types.Character) agent.BeforeAgentCallback {
	return func(cbCtx agent.CallbackContext) (*genai.Content, error) {
		userText := utils.ExtractContentText(cbCtx.UserContent())
		trimmed := strings.TrimSpace(userText)

		state := cbCtx.State()
		if state != nil && !getStateBool(state, "HasUserInput") {
			if err := state.Set("HasUserInput", true); err != nil {
				slog.Error("failed to set session state", "key", "HasUserInput", "error", err.Error())
				return nil, fmt.Errorf("failed to set session state: %w", err)
			}
		}

		if trimmed == "0_0" && character != nil && character.FirstMessage != "" {
			firstMessage := utils.NormalizePromptText(character.FirstMessage, character.Name, cbCtx.UserID())
			return genai.NewContentFromText(firstMessage, "model"), nil
		}

		return nil, nil
	}
}

func getStateBool(state session.State, key string) bool {
	value, err := state.Get(key)
	if err != nil {
		return false
	}

	boolValue, ok := value.(bool)
	if !ok {
		slog.Warn("session state key has unexpected type", "key", key)
		return false
	}

	return boolValue
}
