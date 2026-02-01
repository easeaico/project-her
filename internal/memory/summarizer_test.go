package memory

import (
	"context"
	"iter"
	"strings"
	"testing"
	"time"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/types"
)

type fakeRunner struct {
	sessionService session.Service
	response       string
}

func (r *fakeRunner) Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		if _, err := r.sessionService.Get(ctx, &session.GetRequest{
			AppName:   memorySummarizerAppName,
			UserID:    userID,
			SessionID: sessionID,
		}); err != nil {
			yield(nil, err)
			return
		}

		event := session.NewEvent("summarizer-test")
		event.Author = "assistant"
		event.LLMResponse.Content = genai.NewContentFromText(r.response, "assistant")
		_ = yield(event, nil)
	}
}

type fakeChatHistoryRepo struct {
	window *types.ChatHistory
	err    error
}

func (r *fakeChatHistoryRepo) GetLatestWindow(ctx context.Context, userID, appName string) (*types.ChatHistory, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.window, nil
}

func (r *fakeChatHistoryRepo) CreateWindow(ctx context.Context, history *types.ChatHistory) error {
	return nil
}

func (r *fakeChatHistoryRepo) UpdateWindow(ctx context.Context, history *types.ChatHistory, content string, turnCount int) error {
	return nil
}

func (r *fakeChatHistoryRepo) MarkSummarized(ctx context.Context, id int) error {
	return nil
}

func (r *fakeChatHistoryRepo) GetRecent(ctx context.Context, userID, appName string, limit int) ([]types.ChatHistory, error) {
	return nil, nil
}

type fakeMemoryRepo struct {
	last types.Memory
	err  error
}

func (r *fakeMemoryRepo) AddMemory(ctx context.Context, mem types.Memory) error {
	if r.err != nil {
		return r.err
	}
	r.last = mem
	return nil
}

func (r *fakeMemoryRepo) SearchSimilar(ctx context.Context, userID, appName, memoryType string, embedding []float32, topK int, threshold float64) ([]types.RetrievedMemory, error) {
	return nil, nil
}

type fakeEmbedder struct {
	vector []float32
	err    error
}

func (e *fakeEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.vector, e.err
}

func (e *fakeEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.vector, e.err
}

func (e *fakeEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if e.err != nil {
		return nil, e.err
	}
	results := make([][]float32, 0, len(texts))
	for range texts {
		results = append(results, e.vector)
	}
	return results, nil
}

func TestSummarizeLatestWindowCreatesSessionAndWritesMemory(t *testing.T) {
	sessionService := session.InMemoryService()
	window := &types.ChatHistory{
		ID:         1,
		UserID:     "user",
		AppName:    "project_her_roleplay_1",
		Content:    "User: hi\nAssistant: hello\n",
		TurnCount:  2,
		Summarized: false,
		CreatedAt:  time.Now(),
	}

	histories := &fakeChatHistoryRepo{window: window}
	memories := &fakeMemoryRepo{}
	embedder := &fakeEmbedder{vector: make([]float32, embeddingDimensions)}
	runner := &fakeRunner{
		sessionService: sessionService,
		response:       `{"summary":"对话摘要","facts":["用户喜欢旅行"],"commitments":["下次一起去看海"],"emotions":["开心"]}`,
	}

	summarizer := &memorySummarizer{
		runner:         runner,
		sessionService: sessionService,
		charHistories:  histories,
		memoryRepo:     memories,
		embedder:       embedder,
	}

	if err := summarizer.SummarizeLatestWindow(context.Background(), window.UserID, window.AppName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if memories.last.Summary == "" {
		t.Fatalf("expected memory summary to be saved")
	}
	if got := len(memories.last.Embedding); got != embeddingDimensions {
		t.Fatalf("expected embedding dims %d, got %d", embeddingDimensions, got)
	}
	if memories.last.UserID != window.UserID || memories.last.AppName != window.AppName {
		t.Fatalf("expected memory user/app to match window, got user=%s app=%s", memories.last.UserID, memories.last.AppName)
	}
	if memories.last.Type != types.MemoryTypeChat {
		t.Fatalf("expected memory type %s, got %s", types.MemoryTypeChat, memories.last.Type)
	}
	if memories.last.Summary != "对话摘要" {
		t.Fatalf("expected summary to match parsed content, got %s", memories.last.Summary)
	}
	if len(memories.last.Facts) != 1 || memories.last.Facts[0] != "用户喜欢旅行" {
		t.Fatalf("expected facts to be parsed, got %#v", memories.last.Facts)
	}
	if memories.last.Salience != 0.55 {
		t.Fatalf("expected salience 0.55, got %v", memories.last.Salience)
	}
}

func TestComputeSalienceClampsToRange(t *testing.T) {
	summary := types.MemorySummary{
		Summary:     strings.Repeat("很重要", 120),
		Facts:       []string{"a", "b", "c", "d"},
		Commitments: []string{"x", "y", "z"},
		Emotions:    []string{"e1", "e2", "e3"},
		TimeRange:   types.TimeRange{Start: "2026-01-01"},
	}
	score := ComputeSalience(summary)
	if score != 1 {
		t.Fatalf("expected clamp to 1, got %v", score)
	}
}

var _ Summarizer = (*memorySummarizer)(nil)
var _ ChatHistoryRepo = (*fakeChatHistoryRepo)(nil)
var _ MemoryRepo = (*fakeMemoryRepo)(nil)
var _ Embedder = (*fakeEmbedder)(nil)
