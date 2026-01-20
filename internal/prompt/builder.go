package prompt

import (
	"bytes"
	"fmt"
	"time"

	"google.golang.org/genai"

	"github.com/easeaico/adk-memory-agent/internal/types"
)

// BuildContext contains all inputs for prompt assembly.
type BuildContext struct {
	Character   *types.Character
	Affection   int
	Mood        string
	Memories    []types.RetrievedMemory
	History     []types.ChatMessage
	UserMessage string
}

// Builder assembles layered prompts for the agent.
type Builder struct {
	historyLimit int
	nowFunc      func() time.Time
}

// NewBuilder creates a prompt Builder.
func NewBuilder(historyLimit int) *Builder {
	if historyLimit <= 0 {
		historyLimit = 10
	}
	return &Builder{
		historyLimit: historyLimit,
		nowFunc:      time.Now,
	}
}

// Build assembles the full prompt into genai contents.
func (b *Builder) Build(ctx BuildContext) ([]*genai.Content, error) {
	if ctx.Character == nil {
		return nil, fmt.Errorf("character is required")
	}

	history := ctx.History
	if len(history) > b.historyLimit {
		history = history[len(history)-b.historyLimit:]
	}
	normalizedHistory := make([]types.ChatMessage, len(history))
	copy(normalizedHistory, history)
	for i := range normalizedHistory {
		if normalizedHistory[i].Role == "model" {
			normalizedHistory[i].Role = ctx.Character.Name
		}
	}

	data := struct {
		Character       *types.Character
		Affection       int
		Mood            string
		Memories        []types.RetrievedMemory
		History         []types.ChatMessage
		Now             string
		ExampleDialogue string
	}{
		Character:       ctx.Character,
		Affection:       ctx.Affection,
		Mood:            ctx.Mood,
		Memories:        ctx.Memories,
		History:         normalizedHistory,
		Now:             b.nowFunc().Format(time.RFC3339),
		ExampleDialogue: replaceVars(ctx.Character.ExampleDialogue, ctx.Character.Name, "user"),
	}

	var buf bytes.Buffer
	if err := promptTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	systemContent := genai.NewContentFromText(buf.String(), "system")
	userContent := genai.NewContentFromText(ctx.UserMessage, "user")
	return []*genai.Content{systemContent, userContent}, nil
}
