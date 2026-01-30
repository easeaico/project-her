// Package tool provides custom ADK tools for Project Her.
package tool

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/adk/memory"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/utils"
)

const (
	defaultPreloadMemoryToolName        = "preload_memory"
	defaultPreloadMemoryToolDescription = "Preloads relevant memories into the system instruction before each turn."
)

// PreloadMemoryTool injects retrieved memories into the system instruction.
type PreloadMemoryTool struct {
	name        string
	description string
	maxEntries  int
}

// NewPreloadMemoryTool creates a PreloadMemoryTool with optional configuration.
func NewPreloadMemoryTool(cfg *config.Config) *PreloadMemoryTool {
	return &PreloadMemoryTool{
		name:        defaultPreloadMemoryToolName,
		description: defaultPreloadMemoryToolDescription,
		maxEntries:  cfg.TopK,
	}
}

// Name implements tool.Tool.
func (t *PreloadMemoryTool) Name() string {
	return t.name
}

// Description implements tool.Tool.
func (t *PreloadMemoryTool) Description() string {
	return t.description
}

// IsLongRunning implements tool.Tool.
func (t *PreloadMemoryTool) IsLongRunning() bool {
	return false
}

// ProcessRequest injects retrieved memories into the system instruction.
func (t *PreloadMemoryTool) ProcessRequest(ctx tool.Context, req *model.LLMRequest) error {
	if ctx == nil || req == nil {
		return nil
	}

	query := strings.TrimSpace(utils.ExtractContentText(ctx.UserContent()))
	if query == "" {
		return nil
	}

	resp, err := ctx.SearchMemory(ctx, query)
	if err != nil {
		slog.Error("failed to search memory", "error", err.Error())
		return fmt.Errorf("failed to search memory: %w", err)
	}

	if resp == nil || len(resp.Memories) == 0 {
		return nil
	}

	instruction := buildMemoryInstruction(resp.Memories, t.maxEntries)
	if instruction == "" {
		return nil
	}
	appendInstruction(req, instruction)
	return nil
}

func buildMemoryInstruction(memories []memory.Entry, maxEntries int) string {
	if len(memories) == 0 {
		return ""
	}

	if maxEntries > 0 && len(memories) > maxEntries {
		memories = memories[:maxEntries]
	}

	var instruction strings.Builder
	instruction.WriteString(`The following content is from your previous conversations with the user.
They may be useful for answering the user's current query.
<PAST_CONVERSATIONS>
`)
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

	instruction.WriteString(`</PAST_CONVERSATIONS>
	`)
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

func appendInstruction(req *model.LLMRequest, instruction string) {
	if strings.TrimSpace(instruction) == "" {
		return
	}
	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	if req.Config.SystemInstruction == nil {
		req.Config.SystemInstruction = genai.NewContentFromText(instruction, genai.RoleUser)
		return
	}
	req.Config.SystemInstruction.Parts = append(req.Config.SystemInstruction.Parts, genai.NewPartFromText(instruction))
}
