package callback

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// EnsureUserStateCallback writes required user state fields for prompt injection.
func EnsureUserStateCallback() agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		if err := ctx.State().Set("UserName", ctx.UserID()); err != nil {
			return nil, fmt.Errorf("failed to set UserName: %w", err)
		}
		if err := ctx.State().Set("Now", time.Now().Format(time.RFC3339)); err != nil {
			return nil, fmt.Errorf("failed to set Now: %w", err)
		}
		if err := ctx.State().Set("Location", "Unknown"); err != nil {
			return nil, fmt.Errorf("failed to set Location: %w", err)
		}
		if _, err := ctx.State().Get("RelationshipScore"); err != nil {
			if errors.Is(err, session.ErrStateKeyNotExist) {
				if err := ctx.State().Set("RelationshipScore", 0); err != nil {
					return nil, fmt.Errorf("failed to set RelationshipScore: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to read RelationshipScore: %w", err)
			}
		}
		if _, err := ctx.State().Get("RelationshipLevel"); err != nil {
			if errors.Is(err, session.ErrStateKeyNotExist) {
				if err := ctx.State().Set("RelationshipLevel", "Neutral"); err != nil {
					return nil, fmt.Errorf("failed to set RelationshipLevel: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to read RelationshipLevel: %w", err)
			}
		}

		return nil, nil
	}
}
