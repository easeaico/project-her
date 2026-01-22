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
	AvatarPath      string    `json:"avatar_path"`
	Affection       int       `json:"affection"`
	CurrentMood     string    `json:"current_mood"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const (
	MemoryTypeChat    = "chat"
	MemoryTypePersona = "persona"
	MemoryTypeFacts   = "facts"
	MemoryTypeEvents  = "events"
)

// Memory is a stored memory record.
type Memory struct {
	ID          int       `json:"id"`
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id"`
	CharacterID int       `json:"character_id"`
	Type        string    `json:"type"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Embedding   []float32 `json:"-"` // embedding vectors, not serialized
	CreatedAt   time.Time `json:"created_at"`
}

// RetrievedMemory is a retrieved memory snippet.
type RetrievedMemory struct {
	Content    string    `json:"content"`
	Role       string    `json:"role"`
	Type       string    `json:"type"`
	Similarity float64   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
}
