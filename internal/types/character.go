package types

import "time"

// Character is the persisted profile.
type Character struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Appearance      string    `json:"appearance"`
	Personality     string    `json:"personality"`
	Scenario        string    `json:"scenario"`
	FirstMessage    string    `json:"first_message"`
	ExampleDialogue string    `json:"example_dialogue"`
	SystemPrompt    string    `json:"system_prompt"`
	SystemPromptRaw string    `json:"system_prompt_raw"`
	AvatarPath      string    `json:"avatar_path"`
	AvatarURL       string    `json:"avatar_url"`
	Affection       int       `json:"affection"`
	CurrentMood     string    `json:"current_mood"`
	LastLabel       string    `json:"last_label"`
	MoodTurns       int       `json:"mood_turns"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	// MemoryTypeChat is chunked chat memory.
	MemoryTypeChat = "chat"
	// MemoryTypePersona stores persona-like memory.
	MemoryTypePersona = "persona"
	// MemoryTypeFacts stores extracted facts or preferences.
	MemoryTypeFacts = "facts"
	// MemoryTypeEvents stores notable events.
	MemoryTypeEvents = "events"
)

// Memory is a stored memory record, designed for retrieval and summarization.
type Memory struct {
	ID      int    `json:"id"`
	UserID  string `json:"user_id"`
	AppName string `json:"app_name"`
	Type    string `json:"type"`
	// Summary stores the final summarized text used as memory body.
	Summary string `json:"summary"`
	// Facts captures durable facts or preferences extracted from the window.
	Facts []string `json:"facts"`
	// Commitments captures promises, plans, or agreements.
	Commitments []string `json:"commitments"`
	// Emotions captures relationship or emotional shifts.
	Emotions []string `json:"emotions"`
	// TimeRange describes the period covered by the window.
	TimeRange TimeRange `json:"time_range"`
	// Salience is a 0-1 score indicating memory importance.
	Salience  float64   `json:"salience_score"`
	Embedding []float32 `json:"-"` // embedding vectors, not serialized
	CreatedAt time.Time `json:"created_at"`
}

// ChatHistory is a bundled chat window stored separately from memories.
type ChatHistory struct {
	ID         int       `json:"id"`
	UserID     string    `json:"user_id"`
	AppName    string    `json:"app_name"`
	Content    string    `json:"content"`
	TurnCount  int       `json:"turn_count"`
	Summarized bool      `json:"summarized"`
	CreatedAt  time.Time `json:"created_at"`
}

// TimeRange describes the covered period of a memory window.
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// MemorySummary is the structured output of the summarizer.
type MemorySummary struct {
	Summary string `json:"summary"`
	// Facts, Commitments, Emotions are extracted lists to improve retrieval.
	Facts       []string  `json:"facts"`
	Commitments []string  `json:"commitments"`
	Emotions    []string  `json:"emotions"`
	TimeRange   TimeRange `json:"time_range"`
	// SalienceScore is normalized to [0,1] by the caller.
	SalienceScore float64 `json:"salience_score"`
}

// RetrievedMemory is a retrieved memory snippet.
type RetrievedMemory struct {
	Content    string    `json:"content"`
	Role       string    `json:"role"`
	Type       string    `json:"type"`
	Similarity float64   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
}
