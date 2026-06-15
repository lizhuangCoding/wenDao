package chat

import (
	"reflect"
	"testing"

	"wenDao/internal/model"
)

func TestParseMemorySummaryOutput_ValidJSON(t *testing.T) {
	input := `{
		"conversation_summary": "讨论了缓存架构优化",
		"user_preferences": ["偏好使用 Redis", "喜欢渐进式披露"],
		"project_facts": ["API 网关已上线"],
		"decisions": ["使用 Redis Cluster"],
		"open_threads": ["需要压测验证"]
	}`

	drafts, err := parseMemorySummaryOutput(input)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(drafts) != 5 {
		t.Fatalf("expected 5 drafts, got %d", len(drafts))
	}

	expected := []struct {
		scope      string
		importance int
	}{
		{ConversationMemoryScopeSummary, 2},
		{ConversationMemoryScopePreference, 3},
		{ConversationMemoryScopeProjectFact, 2},
		{ConversationMemoryScopeDecision, 3},
		{ConversationMemoryScopeOpenThread, 2},
	}
	for i, exp := range expected {
		if drafts[i].Scope != exp.scope {
			t.Fatalf("draft[%d] scope: expected %q, got %q", i, exp.scope, drafts[i].Scope)
		}
		if drafts[i].Importance != exp.importance {
			t.Fatalf("draft[%d] importance: expected %d, got %d", i, exp.importance, drafts[i].Importance)
		}
	}
}

func TestParseMemorySummaryOutput_MarkdownWrapper(t *testing.T) {
	input := "```json\n{\"conversation_summary\": \"测试\", \"user_preferences\": [], \"project_facts\": [], \"decisions\": [], \"open_threads\": []}\n```"

	drafts, err := parseMemorySummaryOutput(input)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected 1 draft (only summary had content), got %d", len(drafts))
	}
	if drafts[0].Scope != ConversationMemoryScopeSummary {
		t.Fatalf("expected scope %q, got %q", ConversationMemoryScopeSummary, drafts[0].Scope)
	}
}

func TestParseMemorySummaryOutput_EmptyFields(t *testing.T) {
	input := `{"conversation_summary": "", "user_preferences": ["", " "], "project_facts": null, "decisions": [], "open_threads": ["   \t  "]}`

	drafts, err := parseMemorySummaryOutput(input)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(drafts) != 0 {
		t.Fatalf("expected 0 drafts (all fields empty), got %d", len(drafts))
	}
}

func TestParseMemorySummaryOutput_InvalidJSON(t *testing.T) {
	_, err := parseMemorySummaryOutput("not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFilterNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect []string
	}{
		{name: "all valid", input: []string{"a", "b", "c"}, expect: []string{"a", "b", "c"}},
		{name: "mixed", input: []string{"a", "", "b", "  ", "c"}, expect: []string{"a", "b", "c"}},
		{name: "all empty", input: []string{"", "  ", "\t"}, expect: []string{}},
		{name: "nil input", input: nil, expect: []string{}},
		{name: "empty input", input: []string{}, expect: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterNonEmpty(tt.input)
			if !reflect.DeepEqual(result, tt.expect) {
				t.Fatalf("expected %#v, got %#v", tt.expect, result)
			}
		})
	}
}

func TestFormatMemorySource(t *testing.T) {
	tests := []struct {
		name     string
		history  []model.ChatMessage
		contains []string
	}{
		{
			name: "normal messages",
			history: []model.ChatMessage{
				{Role: "user", Content: "什么是缓存穿透？"},
				{Role: "assistant", Content: "缓存穿透是指查询一个不存在的数据..."},
			},
			contains: []string{"user:", "assistant:", "缓存穿透", "什么是缓存穿透"},
		},
		{
			name: "skips empty content",
			history: []model.ChatMessage{
				{Role: "user", Content: "   "},
				{Role: "assistant", Content: "有效内容"},
			},
			contains: []string{"assistant: 有效内容"},
		},
		{
			name: "truncates long content",
			history: []model.ChatMessage{
				{Role: "user", Content: repeatRune('a', 500)},
			},
			contains: nil,
		},
		{
			name:     "empty history",
			history:  nil,
			contains: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemorySource(tt.history)
			for _, substr := range tt.contains {
				if substr != "" && !containsStr(result, substr) {
					t.Errorf("result missing expected substring %q in: %s", substr, result)
				}
			}
			if tt.name == "truncates long content" {
				runes := []rune(result)
				if len(runes) > 250 {
					t.Errorf("expected truncated content, got %d runes", len(runes))
				}
			}
			if tt.name == "empty history" && result != "" {
				t.Errorf("expected empty result, got %q", result)
			}
		})
	}
}

func TestFormatExistingMemories(t *testing.T) {
	tests := []struct {
		name      string
		memories  []model.ConversationMemory
		contains  []string
		isEmpty   bool
	}{
		{
			name: "empty",
			memories: nil,
			isEmpty: true,
		},
		{
			name: "single",
			memories: []model.ConversationMemory{
				{Scope: "preference", Content: "用户喜欢深色模式"},
			},
			contains: []string{"- preference: 用户喜欢深色模式"},
		},
		{
			name: "skips empty content",
			memories: []model.ConversationMemory{
				{Scope: "preference", Content: "有效"},
				{Scope: "fact", Content: ""},
				{Scope: "decision", Content: "  "},
			},
			contains: []string{"- preference: 有效"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatExistingMemories(tt.memories)
			if tt.isEmpty && result != "无" {
				t.Errorf("expected '无', got %q", result)
			}
			for _, substr := range tt.contains {
				if !containsStr(result, substr) {
					t.Errorf("result missing expected substring %q in: %s", substr, result)
				}
			}
		})
	}
}

func TestNewConversationMemorySummarizer_NilLLM(t *testing.T) {
	s := NewConversationMemorySummarizer(nil)
	if s != nil {
		t.Fatal("expected nil summarizer when llm is nil")
	}
}

func repeatRune(r rune, count int) string {
	runes := make([]rune, count)
	for i := range runes {
		runes[i] = r
	}
	return string(runes)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
