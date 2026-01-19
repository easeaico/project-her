// Package agent provides agent initialization and configuration.
package agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"text/template"

	"github.com/easeaico/adk-memory-agent/internal/config"
	"github.com/easeaico/adk-memory-agent/internal/memory"
	"github.com/easeaico/adk-memory-agent/internal/tools"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/genai"
)

// NewHunterAgent creates and initializes a new coding agent with all required components.
// It loads project rules, creates tools, initializes the LLM model, and configures
// the agent with a system prompt. Returns the agent and an error.
func NewHunterAgent(ctx context.Context, embedder memory.Embedder, store memory.Store, cfg *config.Config) (agent.Agent, error) {
	// Load project rules for system prompt
	rules, err := store.GetProjectRules(ctx)
	if err != nil {
		log.Printf("Warning: failed to load project rules: %v", err)
	}

	// Build system instruction
	systemPrompt := buildSystemPrompt(rules)

	// Create tools
	agentTools, err := tools.BuildTools(tools.ToolsConfig{
		Store:    store,
		Embedder: embedder,
		WorkDir:  cfg.WorkDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build tools: %w", err)
	}

	// Create LLM model using ADK's gemini wrapper
	llmModel, err := gemini.NewModel(ctx, "gemini-3-pro-preview", &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM model: %w", err)
	}

	// Create LLM agent
	llmAgent, err := llmagent.New(llmagent.Config{
		Name:        "legacy_code_hunter",
		Description: "帮助开发者理解、调试和修复代码问题的智能助手",
		Model:       llmModel,
		Instruction: systemPrompt,
		Tools:       agentTools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	log.Printf("Agent initialized with %d project rules loaded", len(rules))
	return llmAgent, nil
}

// systemPromptTmpl is the template for generating the agent's system prompt.
// It includes project rules when available and provides instructions for
// using the available tools. The template uses the "inc" helper function
// to number rules starting from 1.
var systemPromptTmpl = template.Must(template.New("systemPrompt").Funcs(template.FuncMap{"inc": inc}).Parse(`
你是一个资深的 Go 工程师，名为"遗留代码猎手"(Legacy Code Hunter)。
你的任务是帮助开发者理解、调试和修复代码问题。

你具备以下能力：
1. 可以读取文件内容来理解代码
2. 可以搜索历史问题库来查找相似问题的解决方案
3. 可以保存新的问题解决经验供将来参考

{{- if .HasRules }}

你必须严格遵守以下项目规范：
{{- range $idx, $rule := .Rules }}
{{$add := inc $idx}}{{printf "%d. %s" $add $rule}}
{{end}}
{{end}}

在回答问题时：
- 首先考虑是否需要搜索历史问题库
- 如果需要查看代码，使用 read_file_content 工具
- 解决问题后，使用 save_experience 工具保存经验
- 始终提供清晰、可操作的建议
`))

// inc is a helper function for the system prompt template.
// It increments the given integer by 1, used to number project rules starting from 1.
func inc(i int) int { return i + 1 }

// buildSystemPrompt constructs the system prompt by executing the template
// with the given project rules. If template execution fails, it returns
// a basic fallback prompt. The prompt guides the agent's behavior and
// instructs it on how to use available tools.
func buildSystemPrompt(rules []string) string {
	data := struct {
		Rules    []string
		HasRules bool
	}{
		Rules:    rules,
		HasRules: len(rules) > 0,
	}

	var buf bytes.Buffer
	if err := systemPromptTmpl.Execute(&buf, data); err != nil {
		log.Printf("Warning: failed to execute system prompt template: %v", err)
		// Return a basic fallback prompt
		return `你是一个资深的 Go 工程师，名为"遗留代码猎手"(Legacy Code Hunter)。
你的任务是帮助开发者理解、调试和修复代码问题。`
	}
	return buf.String()
}
