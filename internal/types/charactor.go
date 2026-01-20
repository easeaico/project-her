package types

// CharactorCard 对应解码后的 JSON 结构
type CharactorCard struct {
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

// 对应 V2 Spec 的包装结构 (有的卡片包了一层 data 字段)
type V2CardWrapper struct {
	Data CharactorCard `json:"data"`
}
