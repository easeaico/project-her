package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"iter"

	adkmemory "google.golang.org/adk/memory"
	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	internalagent "github.com/easeaico/project-her/internal/agent"
	"github.com/easeaico/project-her/internal/types"
)

const testMemoryTrunkSize = 50

func TestServiceAddSessionCreatesNewWindow(t *testing.T) {
	chatRepo := &mockChatHistoryRepo{}
	memRepo := &mockMemoryRepo{}
	embedder := &mockEmbedder{}
	svc := NewService(embedder, memRepo, chatRepo, nil, 3, 0.5, testMemoryTrunkSize)

	sess := newMockSession(1, "user-1", "app-1", []sessionEvent{{role: RoleUser, text: "你好"}})

	if err := svc.AddSession(context.Background(), sess); err != nil {
		t.Fatalf("AddSession returned error: %v", err)
	}

	if len(chatRepo.created) != 1 {
		t.Fatalf("expected 1 window to be created, got %d", len(chatRepo.created))
	}
	created := chatRepo.created[0]
	if created.UserID != "user-1" || created.AppName != "app-1" || created.CharacterID != 1 {
		t.Fatalf("unexpected window metadata: %+v", created)
	}
	expectedContent := fmt.Sprintf("%s: %s", RoleUser, "你好")
	if created.Content != expectedContent {
		t.Fatalf("expected content %q, got %q", expectedContent, created.Content)
	}
	if created.TurnCount != 1 {
		t.Fatalf("expected turn count 1, got %d", created.TurnCount)
	}
}

func TestServiceAddSessionSummarizesAndStoresMemory(t *testing.T) {
	window := &types.ChatHistory{
		ID:          42,
		UserID:      "user-1",
		AppName:     "app-1",
		CharacterID: 1,
		Content:     formatMessage(RoleAssistant, "上一条"),
		TurnCount:   testMemoryTrunkSize - 1,
		Summarized:  false,
	}
	chatRepo := &mockChatHistoryRepo{latestWindow: window}
	embedder := &mockEmbedder{documentVec: []float32{0.1, 0.2}}
	memRepo := &mockMemoryRepo{}
	summarizer := &mockSummarizer{
		summary: types.MemorySummary{
			Summary:       "结构化摘要",
			Facts:         []string{"fact"},
			Commitments:   []string{"promise"},
			Emotions:      []string{"感受"},
			TimeRange:     types.TimeRange{Start: "T1", End: "T2"},
			SalienceScore: 0.8,
		},
	}
	svc := NewService(embedder, memRepo, chatRepo, summarizer, 5, 0.7, testMemoryTrunkSize)

	sess := newMockSession(1, "user-1", "app-1", []sessionEvent{{role: RoleUser, text: "最后一条"}})

	if err := svc.AddSession(context.Background(), sess); err != nil {
		t.Fatalf("AddSession returned error: %v", err)
	}

	if len(chatRepo.appended) != 1 {
		t.Fatalf("expected append to be called once, got %d", len(chatRepo.appended))
	}
	appendCall := chatRepo.appended[0]
	if appendCall.id != window.ID {
		t.Fatalf("expected append on window %d, got %d", window.ID, appendCall.id)
	}
	if appendCall.turnCount != testMemoryTrunkSize {
		t.Fatalf("expected turn count %d, got %d", testMemoryTrunkSize, appendCall.turnCount)
	}

	if len(memRepo.added) != 1 {
		t.Fatalf("expected 1 memory to be stored, got %d", len(memRepo.added))
	}
	stored := memRepo.added[0]
	if stored.UserID != "user-1" || stored.AppName != "app-1" || stored.CharacterID != 1 {
		t.Fatalf("unexpected memory metadata: %+v", stored)
	}
	if stored.Summary != "结构化摘要" {
		t.Fatalf("expected summary to use structured result, got %q", stored.Summary)
	}
	if stored.Salience != 0.8 {
		t.Fatalf("expected salience 0.8, got %f", stored.Salience)
	}
	if len(stored.Embedding) != len(embedder.documentVec) {
		t.Fatalf("expected embedding to be set, got %v", stored.Embedding)
	}

	if len(chatRepo.marked) != 1 || chatRepo.marked[0] != window.ID {
		t.Fatalf("expected window to be marked summarized, got %v", chatRepo.marked)
	}
	if len(summarizer.requests) != 1 {
		t.Fatalf("expected summarizer to be invoked, got %d calls", len(summarizer.requests))
	}
}

func TestServiceSearchReturnsMemories(t *testing.T) {
	embedder := &mockEmbedder{queryVec: []float32{0.4, 0.6}}
	chatRepo := &mockChatHistoryRepo{}
	memRepo := &mockMemoryRepo{
		searchResult: []types.RetrievedMemory{
			{
				Content:    "记忆",
				Role:       RoleAssistant,
				Type:       types.MemoryTypeChat,
				Similarity: 0.9,
				CreatedAt:  time.Unix(10, 0),
			},
		},
	}
	svc := NewService(embedder, memRepo, chatRepo, nil, 5, 0.5, testMemoryTrunkSize)

	resp, err := svc.Search(context.Background(), &adkmemory.SearchRequest{
		Query:   "喜欢吃什么",
		UserID:  "user-1",
		AppName: "app-1",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if resp == nil || len(resp.Memories) != 1 {
		t.Fatalf("expected one memory entry, got %#v", resp)
	}
	entry := resp.Memories[0]
	if entry.Author != RoleAssistant {
		t.Fatalf("expected author %q, got %q", RoleAssistant, entry.Author)
	}
	if entry.Content == nil || len(entry.Content.Parts) == 0 || entry.Content.Parts[0].Text != "记忆" {
		t.Fatalf("unexpected entry content: %+v", entry.Content)
	}
	if len(embedder.queryInputs) != 1 || embedder.queryInputs[0] != "喜欢吃什么" {
		t.Fatalf("expected embedder to encode the query, got %v", embedder.queryInputs)
	}
	if len(memRepo.searchCalls) != 1 {
		t.Fatalf("expected search repo to be called once, got %d", len(memRepo.searchCalls))
	}
	call := memRepo.searchCalls[0]
	if call.userID != "user-1" || call.appName != "app-1" {
		t.Fatalf("search call missing filters: %+v", call)
	}
}

type mockEmbedder struct {
	documentVec []float32
	queryVec    []float32
	docInputs   []string
	queryInputs []string
}

func (m *mockEmbedder) EmbedQuery(_ context.Context, text string) ([]float32, error) {
	m.queryInputs = append(m.queryInputs, text)
	if m.queryVec == nil {
		return nil, nil
	}
	return m.queryVec, nil
}

func (m *mockEmbedder) EmbedDocument(_ context.Context, text string) ([]float32, error) {
	m.docInputs = append(m.docInputs, text)
	if m.documentVec == nil {
		return nil, nil
	}
	return m.documentVec, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := m.EmbedDocument(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

type mockMemoryRepo struct {
	added        []types.Memory
	searchResult []types.RetrievedMemory
	searchCalls  []searchCall
}

type searchCall struct {
	userID    string
	appName   string
	memType   string
	topK      int
	threshold float64
}

func (m *mockMemoryRepo) AddMemory(_ context.Context, mem types.Memory) error {
	m.added = append(m.added, mem)
	return nil
}

func (m *mockMemoryRepo) SearchSimilar(_ context.Context, userID, appName, memoryType string, _ []float32, topK int, threshold float64) ([]types.RetrievedMemory, error) {
	m.searchCalls = append(m.searchCalls, searchCall{userID: userID, appName: appName, memType: memoryType, topK: topK, threshold: threshold})
	return m.searchResult, nil
}

type mockChatHistoryRepo struct {
	latestWindow *types.ChatHistory
	created      []types.ChatHistory
	appended     []appendCall
	marked       []int
}

type appendCall struct {
	id        int
	content   string
	turnCount int
}

func (m *mockChatHistoryRepo) GetLatestWindow(context.Context, int, string, string) (*types.ChatHistory, error) {
	if m.latestWindow == nil {
		return nil, nil
	}
	copyValue := *m.latestWindow
	return &copyValue, nil
}

func (m *mockChatHistoryRepo) CreateWindow(_ context.Context, history types.ChatHistory) error {
	m.created = append(m.created, history)
	return nil
}

func (m *mockChatHistoryRepo) AppendToWindow(_ context.Context, id int, content string, turnCount int) error {
	m.appended = append(m.appended, appendCall{id: id, content: content, turnCount: turnCount})
	return nil
}

func (m *mockChatHistoryRepo) MarkSummarized(_ context.Context, id int) error {
	m.marked = append(m.marked, id)
	return nil
}

func (m *mockChatHistoryRepo) GetRecent(context.Context, int, string, string, int) ([]types.ChatHistory, error) {
	return nil, nil
}

func newMockSession(characterID int, userID, appName string, events []sessionEvent) session.Session {
	state := &mockState{data: map[string]any{"character_id": characterID}}
	evtList := make([]*session.Event, 0, len(events))
	for _, e := range events {
		evtList = append(evtList, &session.Event{
			LLMResponse: model.LLMResponse{
				Content: genai.NewContentFromText(e.text, genai.Role(e.role)),
			},
		})
	}
	return &mockSession{
		id:     "session-1",
		app:    appName,
		user:   userID,
		state:  state,
		events: &mockEvents{events: evtList},
	}
}

type sessionEvent struct {
	role string
	text string
}

type mockSession struct {
	id     string
	app    string
	user   string
	state  *mockState
	events *mockEvents
}

func (m *mockSession) ID() string                { return m.id }
func (m *mockSession) AppName() string           { return m.app }
func (m *mockSession) UserID() string            { return m.user }
func (m *mockSession) State() session.State      { return m.state }
func (m *mockSession) Events() session.Events    { return m.events }
func (m *mockSession) LastUpdateTime() time.Time { return time.Now() }

var _ session.Session = (*mockSession)(nil)

type mockState struct {
	data map[string]any
}

func (m *mockState) Get(key string) (any, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, session.ErrStateKeyNotExist
	}
	return val, nil
}

func (m *mockState) Set(key string, value any) error {
	if m.data == nil {
		m.data = map[string]any{}
	}
	m.data[key] = value
	return nil
}

func (m *mockState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range m.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

var _ session.State = (*mockState)(nil)

type mockEvents struct {
	events []*session.Event
}

func (m *mockEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, evt := range m.events {
			if !yield(evt) {
				return
			}
		}
	}
}

func (m *mockEvents) Len() int {
	return len(m.events)
}

func (m *mockEvents) At(i int) *session.Event {
	return m.events[i]
}

var _ session.Events = (*mockEvents)(nil)

type mockSummarizer struct {
	summary  types.MemorySummary
	requests []string
}

func (m *mockSummarizer) Summarize(_ context.Context, content string) (types.MemorySummary, error) {
	m.requests = append(m.requests, content)
	return m.summary, nil
}

var _ internalagent.Summarizer = (*mockSummarizer)(nil)
