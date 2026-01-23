package prompt

import (
	"bytes"
	"fmt"
	"time"

	"github.com/easeaico/project-her/internal/types"
)

// BuildContext holds prompt inputs.
type BuildContext struct {
	Character   *types.Character
	Affection   int
	Mood        string
	Memories    []types.RetrievedMemory
	History     []types.ChatHistory
	UserMessage string
}

// Builder assembles prompt contents.
type Builder struct {
	historyLimit int
	nowFunc      func() time.Time
}

// NewBuilder returns a prompt builder.
func NewBuilder(historyLimit int) *Builder {
	if historyLimit <= 0 {
		historyLimit = 10
	}
	return &Builder{
		historyLimit: historyLimit,
		nowFunc:      time.Now,
	}
}

func (b *Builder) BuildInstruction(ctx BuildContext) (string, error) {
	if ctx.Character == nil {
		return "", fmt.Errorf("character is required")
	}

	data := struct {
		Character       *types.Character
		Affection       int
		Mood            string
		Now             string
		ExampleDialogue string
	}{
		Character:       ctx.Character,
		Affection:       ctx.Affection,
		Mood:            ctx.Mood,
		Now:             b.nowFunc().Format(time.RFC3339),
		ExampleDialogue: replaceVars(ctx.Character.ExampleDialogue, ctx.Character.Name, "user"),
	}

	var buf bytes.Buffer
	if err := promptTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	return buf.String(), nil
}
