// Package models provides OpenAI-compatible adapters.
package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// openaiModel wraps an OpenAI-compatible chat client.
type openaiModel struct {
	client             *openai.Client
	name               string
	versionHeaderValue string
}

type toolCallBuilder struct {
	Index int64
	ID    string
	Name  string
	Args  strings.Builder
}

func NewOpenAIModel(ctx context.Context, modelName string, cfg *genai.ClientConfig) (model.LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	// Create OpenAI client with x.ai configuration
	client := openai.NewClient(option.WithAPIKey(cfg.APIKey))

	// Create header value once, when the model is created
	headerValue := fmt.Sprintf("openai-go/%s go/%s",
		"1.0.0", strings.TrimPrefix(runtime.Version(), "go"))

	return &openaiModel{
		name:               modelName,
		client:             &client,
		versionHeaderValue: headerValue,
	}, nil
}

func (m *openaiModel) Name() string {
	return m.name
}

func (m *openaiModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	m.maybeAppendUserContent(req)

	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	if req.Config.HTTPOptions == nil {
		req.Config.HTTPOptions = &genai.HTTPOptions{}
	}
	if req.Config.HTTPOptions.Headers == nil {
		req.Config.HTTPOptions.Headers = make(http.Header)
	}
	m.addHeaders(req.Config.HTTPOptions.Headers)

	if stream {
		return m.generateStream(ctx, req)
	}

	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.generate(ctx, req)
		yield(resp, err)
	}
}

func (m *openaiModel) addHeaders(headers http.Header) {
	headers.Set("user-agent", m.versionHeaderValue)
}

func (m *openaiModel) generate(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	params := buildOpenAIParams(req, m.name)

	resp, err := m.client.Chat.Completions.New(ctx, *params)
	if err != nil {
		slog.Error("failed to call llm API", "error", err.Error())
		return nil, fmt.Errorf("failed to call Grok API: %w", err)
	}

	if resp == nil || len(resp.Choices) == 0 {
		return &model.LLMResponse{}, nil
	}

	message := resp.Choices[0].Message
	content := &genai.Content{
		Role:  string(message.Role),
		Parts: []*genai.Part{},
	}

	if message.Content != "" {
		content.Parts = append(content.Parts, &genai.Part{
			Text: message.Content,
		})
	}

	if len(message.ToolCalls) > 0 {
		builder := &toolCallBuilder{}

		for _, v := range message.ToolCalls {
			// The type of the tool. Currently, only `function` is supported.
			if v.Type == "function" {
				if v.ID != "" {
					builder.ID = v.ID
				}

				if v.Function.Name != "" {
					builder.Name = v.Function.Name
				}

				if v.Function.Arguments != "" {
					builder.Args.WriteString(v.Function.Arguments)
				}
			}
		}

		if builder.ID != "" && builder.Name != "" {
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   builder.ID,
					Name: builder.Name,
					Args: parseFunctionArgs(builder.Args.String()),
				},
			})
		}
	}

	llmResp := &model.LLMResponse{
		Content: content,
	}
	return llmResp, nil
}

func (m *openaiModel) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params := buildOpenAIParams(req, m.name)
		if params == nil {
			yield(nil, fmt.Errorf("invalid request parameters"))
			return
		}

		stream := m.client.Chat.Completions.NewStreaming(ctx, *params)
		defer stream.Close()

		pendingTools := make(map[int64]*toolCallBuilder)
		for stream.Next() {
			chunk := stream.Current()

			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]
			isFinished := choice.FinishReason != ""

			if choice.Delta.Content != "" {
				llmResp := &model.LLMResponse{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{Text: choice.Delta.Content},
						},
					},
					Partial:      true,
					TurnComplete: isFinished && len(pendingTools) == 0,
				}
				if !yield(llmResp, nil) {
					return
				}
			}

			for _, tc := range choice.Delta.ToolCalls {
				index := tc.Index
				builder, exists := pendingTools[index]
				if !exists {
					builder = &toolCallBuilder{Index: index}
					pendingTools[index] = builder
				}

				if tc.ID != "" {
					builder.ID = tc.ID
				}
				if tc.Function.Name != "" {
					builder.Name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					builder.Args.WriteString(tc.Function.Arguments)
				}
			}

			if isFinished && len(pendingTools) > 0 {
				var parts []*genai.Part

				var indices []int64
				for k := range pendingTools {
					indices = append(indices, k)
				}
				sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

				for _, idx := range indices {
					builder := pendingTools[idx]
					parts = append(parts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							ID:   builder.ID,
							Name: builder.Name,
							Args: parseFunctionArgs(builder.Args.String()),
						},
					})
				}

				llmResp := &model.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: parts,
					},
					Partial:      false,
					TurnComplete: true,
				}
				if !yield(llmResp, nil) {
					return
				}
			}
		}

		if err := stream.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				yield(nil, fmt.Errorf("context cancelled: %w", err))
				return
			}
			slog.Error("failed to stream call llm API", "error", err.Error())
			yield(nil, fmt.Errorf("stream error: %w", err))
		}
	}
}

func (m *openaiModel) maybeAppendUserContent(req *model.LLMRequest) {
	if len(req.Contents) == 0 {
		req.Contents = append(req.Contents, genai.NewContentFromText("Handle the requests as specified in the System Instruction.", "user"))
	}

	if last := req.Contents[len(req.Contents)-1]; last != nil && last.Role != "user" {
		req.Contents = append(req.Contents, genai.NewContentFromText("Continue processing previous requests as instructed.", "user"))
	}
}

// parseFunctionArgs returns an empty map on parse failure.
func parseFunctionArgs(jsonStr string) map[string]any {
	if jsonStr == "" {
		return make(map[string]any)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &args); err != nil {
		return make(map[string]any)
	}
	return args
}
