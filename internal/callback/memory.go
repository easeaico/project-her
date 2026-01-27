package callback

import (
	"log/slog"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// NewMemoryCallback 返回在每轮结束后写入记忆的回调。
// 通过 sessionService 获取完整会话，再调用 MemoryService.AddSession 进行记忆写入。
func NewMemoryCallback(sessionService session.Service, memoryService memory.Service) agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		resp, err := sessionService.Get(ctx, &session.GetRequest{
			AppName:   ctx.AppName(),
			UserID:    ctx.UserID(),
			SessionID: ctx.SessionID()})

		if err != nil {
			slog.Error("failed to get completed session", "error", err.Error())
			return nil, err
		}

		if err := memoryService.AddSession(ctx, resp.Session); err != nil {
			slog.Error("failed to add session to memory", "error", err.Error())
			return nil, err
		}

		return nil, nil
	}
}
