package callback

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/easeaico/project-her/internal/utils"
)

const (
	relationshipLevelDistant  = "Distant"
	relationshipLevelNeutral  = "Neutral"
	relationshipLevelFriendly = "Friendly"
	relationshipLevelClose    = "Close"
	relationshipLevelIntimate = "Intimate"
)

var (
	strongPositiveKeywords = []string{
		"爱你",
		"好爱",
		"想你",
		"亲亲",
		"拥抱",
		"love you",
		"adore you",
		"miss you",
	}
	positiveKeywords = []string{
		"喜欢",
		"开心",
		"谢谢",
		"感激",
		"欣赏",
		"温柔",
		"可爱",
		"贴心",
		"谢谢你",
		"thank you",
		"thanks",
		"great",
		"good",
		"sweet",
	}
	negativeKeywords = []string{
		"失望",
		"难过",
		"冷淡",
		"不喜欢",
		"讨厌",
		"烦",
		"生气",
		"annoy",
		"upset",
		"sad",
		"bad",
	}
	strongNegativeKeywords = []string{
		"恨你",
		"讨厌你",
		"滚",
		"闭嘴",
		"恶心",
		"hate you",
		"fuck",
	}
)

// NewRelationshipLevelCallback updates relationship score/level based on user input.
func NewRelationshipLevelCallback() agent.AfterAgentCallback {
	return func(ctx agent.CallbackContext) (*genai.Content, error) {
		userText := strings.TrimSpace(utils.ExtractContentText(ctx.UserContent()))
		if userText == "" {
			return nil, nil
		}

		delta := relationshipScoreDelta(userText)
		if delta == 0 {
			return nil, nil
		}

		score, err := readIntState(ctx.State(), "RelationshipScore")
		if err != nil {
			return nil, fmt.Errorf("failed to read RelationshipScore: %w", err)
		}

		newScore := score + delta
		if err := ctx.State().Set("RelationshipScore", newScore); err != nil {
			return nil, fmt.Errorf("failed to set RelationshipScore: %w", err)
		}

		level := mapRelationshipLevel(newScore)
		if err := ctx.State().Set("RelationshipLevel", level); err != nil {
			return nil, fmt.Errorf("failed to set RelationshipLevel: %w", err)
		}

		return nil, nil
	}
}

func relationshipScoreDelta(text string) int {
	lowered := strings.ToLower(text)
	delta := 0
	if containsAny(lowered, strongPositiveKeywords) {
		delta += 3
	}
	if containsAny(lowered, positiveKeywords) {
		delta += 2
	}
	if containsAny(lowered, negativeKeywords) {
		delta -= 2
	}
	if containsAny(lowered, strongNegativeKeywords) {
		delta -= 3
	}
	return delta
}

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func mapRelationshipLevel(score int) string {
	switch {
	case score <= -3:
		return relationshipLevelDistant
	case score <= 1:
		return relationshipLevelNeutral
	case score <= 4:
		return relationshipLevelFriendly
	case score <= 7:
		return relationshipLevelClose
	default:
		return relationshipLevelIntimate
	}
}

func readIntState(state session.State, key string) (int, error) {
	val, err := state.Get(key)
	if err != nil {
		if errors.Is(err, session.ErrStateKeyNotExist) {
			return 0, nil
		}
		return 0, err
	}
	switch cast := val.(type) {
	case int:
		return cast, nil
	case int8:
		return int(cast), nil
	case int16:
		return int(cast), nil
	case int32:
		return int(cast), nil
	case int64:
		return int(cast), nil
	case uint:
		return int(cast), nil
	case uint8:
		return int(cast), nil
	case uint16:
		return int(cast), nil
	case uint32:
		return int(cast), nil
	case uint64:
		if cast > uint64(^uint(0)>>1) {
			return 0, fmt.Errorf("state value for %s overflows int", key)
		}
		return int(cast), nil
	case float32:
		return int(cast), nil
	case float64:
		return int(cast), nil
	case json.Number:
		parsed, err := cast.Int64()
		if err != nil {
			return 0, fmt.Errorf("state value for %s is not an int: %w", key, err)
		}
		return int(parsed), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(cast))
		if err != nil {
			return 0, fmt.Errorf("state value for %s is not an int: %w", key, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("state value for %s has unsupported type %T", key, val)
	}
}
