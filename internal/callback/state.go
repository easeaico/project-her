package callback

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
		ensureStateValue(state, "MoodInstruction", "")
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
		// 已存在键时不覆盖。
		return
	}
	if !errors.Is(err, session.ErrStateKeyNotExist) {
		slog.Warn("failed to check session state key", "key", key, "error", err.Error())
		return
	}
	// 键不存在则写入初始值。
	if err := state.Set(key, value); err != nil {
		slog.Warn("failed to set session state", "key", key, "error", err.Error())
		// 状态写入失败不阻断主流程，只记录日志。
	}
}
