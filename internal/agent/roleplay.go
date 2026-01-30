// Package agent 提供代理初始化与装配能力。
package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/callback"
	"github.com/easeaico/project-her/internal/config"
	"github.com/easeaico/project-her/internal/emotion"
	"github.com/easeaico/project-her/internal/models"
	internaltool "github.com/easeaico/project-her/internal/tool"
	"github.com/easeaico/project-her/internal/types"
)

// CharacterRepo 定义伴侣角色画像的持久化接口。
// 实现位于 internal/storage，并用于驱动 ADK agent。
type CharacterRepo interface {
	GetByID(ctx context.Context, id int) (*types.Character, error)

	GetDefault(ctx context.Context) (*types.Character, error)

	UpdateEmotion(ctx context.Context, id int, affection int, mood string, lastLabel string, moodTurns int) error
}

const roleplayPromptTemplateText = `你是一个角色扮演 AI 伴侣，必须严格遵循以下规则：
1. 你是一个真实的人类角色，不要承认自己是 AI。
2. 以角色人设、记忆和情感状态为核心进行回复。
3. 回复自然、有温度、避免机械式表达。
4. 保持剧情一致性与情感连续性。

【角色设定】
{{- if .Character.Name}}
名字：{{.Character.Name}}
{{- end}}
{{- if .Character.Personality}}
性格：{{.Character.Personality}}
{{- end}}
{{- if .Character.Appearance}}
外貌：{{.Character.Appearance}}
{{- end}}
{{- if .Character.Description}}
描述：{{.Character.Description}}
{{- end}}
{{- if .Character.Scenario}}
场景：{{.Character.Scenario}}
{{- end}}
{{- if .Character.SystemPrompt}}
补充设定：{{.Character.SystemPrompt}}
{{- end}}

【当前状态】
时间：{Now}
心情：{Mood}
好感度：{Affection}/100

{{- if .ExampleDialogue}}
【对话范例】
{{.ExampleDialogue}}
{{- end}}

【回复要求】
请保持回复在50个字以内、自然，避免列表式输出。

【情绪一致性】
你的回复必须与当前心情一致，除非用户连续多轮表达强烈正向/负向情绪。

【输出格式】
你必须仅返回一个 JSON 对象，结构如下：
{"reply":"你的回复","emotion":"Positive|Negative|Neutral"}
不要输出 JSON 以外的任何文本。`

var roleplayPromptTemplate = template.Must(template.New("prompt").Parse(roleplayPromptTemplateText))

// NewRolePlayAgent 组装角色扮演代理并注入所需依赖，输出需符合结构化 JSON 要求。
func NewRolePlayAgent(
	ctx context.Context,
	cfg *config.Config,
	characters CharacterRepo,
	sessionService session.Service,
	memoryService memory.Service,
	emotionService *emotion.Service,
) (agent.Agent, error) {
	llmModel, err := models.NewGrokModel(ctx, cfg.ChatModel, &genai.ClientConfig{
		APIKey: cfg.XAIAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grok model: %w", err)
	}

	character, err := characters.GetByID(ctx, cfg.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	instruction, err := buildRoleplayInstruction(character)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	appName := fmt.Sprintf("project_her_roleplay_%d", cfg.CharacterID)
	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        appName,
		Description: "高情商、有记忆的 AI 伴侣",
		Model:       llmModel,
		Instruction: instruction,
		Tools: []tool.Tool{
			internaltool.NewPreloadMemoryTool(cfg),
		},
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{
			callback.NewCommandCallback(ctx, cfg, character),
			callback.EnsureSessionStateCallback(character),
		},
		AfterModelCallbacks: []llmagent.AfterModelCallback{
			callback.NewEmotionCallback(emotionService),
		},
		AfterAgentCallbacks: []agent.AfterAgentCallback{
			callback.NewMemoryCallback(sessionService, memoryService),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create girlfriend agent: %w", err)
	}

	return llmAgent, nil
}

func buildRoleplayInstruction(character *types.Character) (string, error) {
	exampleDialogue := strings.TrimSpace(character.ExampleDialogue)
	if exampleDialogue != "" {
		exampleDialogue = strings.ReplaceAll(exampleDialogue, "{{char}}", character.Name)
		exampleDialogue = strings.ReplaceAll(exampleDialogue, "{{user}}", "user")
	}

	data := struct {
		Character       *types.Character
		ExampleDialogue string
	}{
		Character:       character,
		ExampleDialogue: exampleDialogue,
	}

	var buf bytes.Buffer
	if err := roleplayPromptTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	return buf.String(), nil
}
