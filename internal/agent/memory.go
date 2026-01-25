// Package agent provides agent initialization.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/types"
	"github.com/easeaico/project-her/internal/utils"
)

// Summarizer defines memory summary behavior.
type Summarizer interface {
	Summarize(ctx context.Context, content string) (types.MemorySummary, error)
}

const (
	memorySummarizerAppName = "project_her_memory"
	memorySummarizerUserID  = "memory_summarizer"
)

// memorySummaryInstruction instructs the model to return structured JSON only.
const memorySummaryInstruction = `You are a professional dialogue memory summarizer.
Your task is to compress the conversation history into a concise summary while preserving the most important information.

Extract and retain:
1. Key events and important decisions
2. Emotional shifts and intimate moments
3. User-revealed personal info (preferences, habits, important dates, etc.)
4. Promises or agreements made by either party
5. The overall emotional tone

Output requirements:
- Use third-person narration
- Organize chronologically
- Keep the summary within 200-300 Chinese characters
- Return a valid JSON object that matches the output schema
- Do not include any extra keys or text outside the JSON object`

// MemorySummarizer uses an ADK agent to summarize memory content.
type MemorySummarizer struct {
	// agent is the underlying LLM agent.
	agent agent.Agent
	// runner executes the agent in an isolated in-memory session.
	runner *runner.Runner
	// sessionService stores transient sessions for summaries.
	sessionService session.Service
	// counter generates unique session IDs.
	counter uint64
}

// NewMemorySummarizer creates a summarizer based on ADK llmagent.
func NewMemorySummarizer(ctx context.Context, llm model.LLM) (*MemorySummarizer, error) {
	if llm == nil {
		return nil, fmt.Errorf("llm model is required")
	}

	llmAgent, err := llmagent.New(llmagent.Config{
		Name:            "memory_summarizer",
		Description:     "对话记忆摘要智能体",
		Model:           llm,
		Instruction:     memorySummaryInstruction,
		OutputSchema:    summaryOutputSchema(),
		IncludeContents: llmagent.IncludeContentsNone,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory summarizer agent: %w", err)
	}

	sessionService := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        memorySummarizerAppName,
		Agent:          llmAgent,
		SessionService: sessionService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create memory summarizer runner: %w", err)
	}

	return &MemorySummarizer{
		agent:          llmAgent,
		runner:         r,
		sessionService: sessionService,
	}, nil
}

// Summarize returns a structured summary of the input content.
func (s *MemorySummarizer) Summarize(ctx context.Context, content string) (types.MemorySummary, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return types.MemorySummary{}, nil
	}

	sessionID := fmt.Sprintf("summary-%d", atomic.AddUint64(&s.counter, 1))
	msg := genai.NewContentFromText(trimmed, "user")
	events := s.runner.Run(ctx, memorySummarizerUserID, sessionID, msg, agent.RunConfig{
		StreamingMode: agent.StreamingModeNone,
	})

	var last string
	for event, err := range events {
		if err != nil {
			return types.MemorySummary{}, err
		}
		if event == nil || event.Content == nil {
			continue
		}
		if event.Author == "user" {
			continue
		}
		text := strings.TrimSpace(utils.ExtractContentText(event.Content))
		if text == "" {
			continue
		}
		last = text
		if event.IsFinalResponse() {
			break
		}
	}
	if last == "" {
		return types.MemorySummary{}, fmt.Errorf("empty summary response")
	}

	summary, err := parseSummaryJSON(last)
	if err != nil {
		return types.MemorySummary{}, err
	}
	summary.SalienceScore = clamp01(summary.SalienceScore)
	return summary, nil
}

func summaryOutputSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"summary": {
				Type: genai.TypeString,
			},
			"facts": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"commitments": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"emotions": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
			"time_range": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"start": {Type: genai.TypeString},
					"end":   {Type: genai.TypeString},
				},
			},
			"salience_score": {
				Type: genai.TypeNumber,
			},
		},
		Required: []string{"summary"},
	}
}

// parseSummaryJSON extracts JSON from model output and decodes it.
func parseSummaryJSON(raw string) (types.MemorySummary, error) {
	clean := strings.TrimSpace(raw)
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		clean = clean[start : end+1]
	}
	var summary types.MemorySummary
	if err := json.Unmarshal([]byte(clean), &summary); err != nil {
		return types.MemorySummary{}, fmt.Errorf("failed to parse summary json: %w", err)
	}
	return summary, nil
}

// clamp01 keeps a float in [0,1].
func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
