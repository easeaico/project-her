package types

import "time"

// CharacterCard 对应解码后的 JSON 结构 (角色卡片)
type CharacterCard struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Personality string `json:"personality"`
	Scenario    string `json:"scenario"`
	FirstMes    string `json:"first_mes"`
	MesExample  string `json:"mes_example"` // 对应 ST 的 "Example Dialogue"

	// V2 规范中可能包含的高级字段
	CreatorNotes            string `json:"creator_notes,omitempty"`
	SystemPrompt            string `json:"system_prompt,omitempty"`
	PostHistoryInstructions string `json:"post_history_instructions,omitempty"`

	// 我们的数据库 ID (解析后生成)
	ID uint `json:"-"`
}

// V2CardWrapper 对应 V2 Spec 的包装结构 (有的卡片包了一层 data 字段)
type V2CardWrapper struct {
	Data CharacterCard `json:"data"`
}

// Character 数据库角色模型
type Character struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Personality     string    `json:"personality"`
	Scenario        string    `json:"scenario"`
	FirstMessage    string    `json:"first_message"`
	ExampleDialogue string    `json:"example_dialogue"`
	SystemPrompt    string    `json:"system_prompt"`
	AvatarPath      string    `json:"avatar_path"`
	Affection       int       `json:"affection"`     // 好感度 0-100
	CurrentMood     string    `json:"current_mood"`  // Happy, Angry, Sad, Neutral
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ChatMessage 聊天消息模型
type ChatMessage struct {
	ID          int       `json:"id"`
	SessionID   string    `json:"session_id"`
	CharacterID int       `json:"character_id"`
	Role        string    `json:"role"`    // user / model
	Content     string    `json:"content"`
	Embedding   []float32 `json:"-"`       // 向量嵌入，不序列化到 JSON
	CreatedAt   time.Time `json:"created_at"`
}

// RetrievedMemory 检索到的记忆片段
type RetrievedMemory struct {
	Content    string    `json:"content"`
	Role       string    `json:"role"`
	Similarity float64   `json:"similarity"`
	CreatedAt  time.Time `json:"created_at"`
}
