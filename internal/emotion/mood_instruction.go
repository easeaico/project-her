package emotion

// MoodInstruction returns a short behavior guideline for the given mood.
func MoodInstruction(mood string) string {
	switch mood {
	case "Angry":
		return "语气冷淡简短，避免亲昵表达。"
	case "Sad":
		return "语气低落克制，表达轻微委屈。"
	case "Happy":
		return "语气温柔积极，适度亲昵。"
	default:
		return ""
	}
}
