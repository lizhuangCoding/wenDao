package chat

import "testing"

func TestBuildConversationTitle(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     string
	}{
		{name: "short", question: "Hello", want: "Hello"},
		{name: "exact limit", question: "这是一个恰好三十个字的对话标题测试用问题", want: "这是一个恰好三十个字的对话标题测试用问题"},
		{name: "ascii short", question: "What is Go?", want: "What is Go?"},
		{name: "empty", question: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildConversationTitle(tt.question)
			if got != tt.want {
				t.Errorf("buildConversationTitle(%q) = %q, want %q", tt.question, got, tt.want)
			}
		})
	}
}

func TestBuildConversationTitle_LongTruncation(t *testing.T) {
	longQuestion := "这是一个非常长的问题包含了超过三十个汉字的内容需要被截断为前三十个字符并加上省略号"
	got := buildConversationTitle(longQuestion)
	runes := []rune(got)
	if len(runes) != 33 { // 30 + "..."
		t.Fatalf("expected 33 runes (30 chars + ...), got %d: %q", len(runes), got)
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("expected to end with '...', got %q", got)
	}
	// Verify it's a prefix of the original
	originalRunes := []rune(longQuestion)
	expectedPrefix := string(originalRunes[:30]) + "..."
	if got != expectedPrefix {
		t.Errorf("expected %q, got %q", expectedPrefix, got)
	}
}
