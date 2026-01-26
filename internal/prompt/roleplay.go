package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/easeaico/project-her/internal/types"
)

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
请保持回复在50个字以内、自然，避免列表式输出。`

var roleplayPromptTemplate = template.Must(template.New("prompt").Parse(roleplayPromptTemplateText))

func BuildRoleplayInstruction(character *types.Character) (string, error) {
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
