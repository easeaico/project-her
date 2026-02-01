package callback

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/utils"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// NewMemoryCallback 返回在每轮结束后写入记忆的回调。
// 通过 sessionService 获取完整会话，再调用 MemoryService.AddSession 进行记忆写入。
func NewAddSessionToMemoryCallback(sessionService session.Service, memoryService memory.Service) agent.AfterAgentCallback {
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

// NewMemoriesStateCallback searches memories and writes them into session state.
func NewMemoriesStateCallback(memoryService memory.Service, cfg *config.Config) agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		query := strings.TrimSpace(utils.ExtractContentText(ctx.UserContent()))
		if query == "" {
			if err := ctx.State().Set("memories", ""); err != nil {
				return nil, fmt.Errorf("failed to set memories: %w", err)
			}
			return nil, nil
		}

		resp, err := memoryService.Search(ctx, &memory.SearchRequest{
			AppName: ctx.AppName(),
			UserID:  ctx.UserID(),
			Query:   query,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search memories: %w", err)
		}

		instruction := buildMemoriesBlock(resp, cfg.TopK)
		if err := ctx.State().Set("Memories", instruction); err != nil {
			return nil, fmt.Errorf("failed to set memories: %w", err)
		}

		return nil, nil
	}
}

func buildMemoriesBlock(resp *memory.SearchResponse, maxEntries int) string {
	if resp == nil || len(resp.Memories) == 0 {
		return ""
	}

	memories := resp.Memories
	if maxEntries > 0 && len(memories) > maxEntries {
		memories = memories[:maxEntries]
	}

	var instruction strings.Builder
	for _, entry := range memories {
		text := strings.TrimSpace(utils.ExtractContentText(entry.Content))
		if text == "" {
			continue
		}
		stamp := ""
		if !entry.Timestamp.IsZero() {
			stamp = entry.Timestamp.Format(time.RFC3339)
		}
		author := strings.TrimSpace(entry.Author)
		instruction.WriteString(formatMemoryLine(stamp, author, text))
		instruction.WriteString("\n")
	}

	return instruction.String()
}

func formatMemoryLine(stamp, author, text string) string {
	parts := []string{"-"}
	if stamp != "" {
		parts = append(parts, "["+stamp+"]")
	}
	if author != "" {
		parts = append(parts, author+":")
	}
	parts = append(parts, text)
	return strings.Join(parts, " ")
}
