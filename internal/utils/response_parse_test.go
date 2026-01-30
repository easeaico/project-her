package utils

import "testing"

func TestParseRoleplayOutput(t *testing.T) {
	got, err := ParseRoleplayOutput(`{"reply":"你好","emotion":"Positive"}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Reply != "你好" {
		t.Fatalf("unexpected reply: %s", got.Reply)
	}
	if got.Emotion != "Positive" {
		t.Fatalf("unexpected emotion: %s", got.Emotion)
	}
}

func TestParseRoleplayOutputWithWrapper(t *testing.T) {
	got, err := ParseRoleplayOutput("prefix {\"reply\":\"嗨\",\"emotion\":\"neutral\"} suffix")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Emotion != "Neutral" {
		t.Fatalf("expected normalized emotion, got %s", got.Emotion)
	}
}

func TestParseRoleplayOutputInvalid(t *testing.T) {
	_, err := ParseRoleplayOutput(`{"reply":"ok","emotion":"excited"}`)
	if err == nil {
		t.Fatalf("expected error for invalid emotion")
	}
}
