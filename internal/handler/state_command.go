package handler

import (
	"errors"
	"log/slog"
	"time"

	"github.com/easeaico/project-her/internal/types"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func EnsureSessionStateCallback(character *types.Character) agent.BeforeAgentCallback {
	return func(cbCtx agent.CallbackContext) (*genai.Content, error) {
		state := cbCtx.State()
		if state == nil {
			slog.Warn("session state is nil, skipping state initialization")
			return nil, nil
		}

		ensureStateValue(state, "Affection", character.Affection)
		ensureStateValue(state, "Mood", character.CurrentMood)
		ensureStateValue(state, "Now", time.Now().Format(time.RFC3339))

		return nil, nil
	}
}

func ensureStateValue(state session.State, key string, value any) {
	if value == nil {
		return
	}
	_, err := state.Get(key)
	if err == nil {
		// Key already exists, skip setting
		return
	}
	if !errors.Is(err, session.ErrStateKeyNotExist) {
		slog.Warn("failed to check session state key", "key", key, "error", err.Error())
		return
	}
	// Key doesn't exist, set it
	if err := state.Set(key, value); err != nil {
		slog.Warn("failed to set session state", "key", key, "error", err.Error())
		// Don't return error, just log it - state operations shouldn't block agent execution
	}
}
