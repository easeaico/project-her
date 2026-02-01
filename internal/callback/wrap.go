package callback

import (
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/genai"
)

func WrapBeforeCallback(name string, cb agent.BeforeAgentCallback) agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("before callback panic", "name", name, "error", err)
			}
		}()

		slog.Info("before callback start", "name", name)
		content, err := cb(ctx)
		if err != nil {
			slog.Error("before callback error", "name", name, "error", err.Error())
			return content, err
		}
		slog.Info("before callback done", "name", name, "has_content", content != nil)
		return content, nil
	}
}

func WrapAfterCallback(name string, cb agent.AfterAgentCallback) agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("after callback panic", "name", name, "error", err)
			}
		}()

		slog.Info("after callback start", "name", name)
		content, err := cb(ctx)
		if err != nil {
			slog.Error("after callback error", "name", name, "error", err.Error())
			return content, err
		}
		slog.Info("after callback done", "name", name, "has_content", content != nil)
		return content, nil
	}
}
