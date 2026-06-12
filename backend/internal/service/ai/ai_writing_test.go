package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/model"

	"wenDao/internal/pkg/eino"
)

type writingLLMStub struct {
	response string
	messages []eino.ChatMessage
}

func (s *writingLLMStub) Chat(ctx context.Context, messages []eino.ChatMessage) (string, error) {
	s.messages = append([]eino.ChatMessage(nil), messages...)
	return s.response, nil
}

func (s *writingLLMStub) ChatStream(ctx context.Context, messages []eino.ChatMessage) (<-chan string, <-chan error) {
	textCh := make(chan string)
	errCh := make(chan error)
	close(textCh)
	close(errCh)
	return textCh, errCh
}

func (s *writingLLMStub) GetModel() model.ChatModel {
	return nil
}

func TestGenerateWritingPolishesSelectedMarkdown(t *testing.T) {
	llm := &writingLLMStub{response: "润色后的段落"}
	svc := NewAIService(llm, nil, nil)

	result, err := svc.GenerateWriting(context.Background(), WritingRequest{
		Action:  WritingActionPolish,
		Content: "这段话有点口语化，需要更自然。",
		Title:   "写作体验",
	})
	if err != nil {
		t.Fatalf("expected writing generation to succeed, got %v", err)
	}
	if result.Result != "润色后的段落" {
		t.Fatalf("expected trimmed result, got %q", result.Result)
	}
	if len(result.Suggestions) != 0 {
		t.Fatalf("expected no suggestions for polish action, got %#v", result.Suggestions)
	}
	if len(llm.messages) != 2 {
		t.Fatalf("expected system and user messages, got %#v", llm.messages)
	}
	if !strings.Contains(llm.messages[1].Content, "保持 Markdown 结构") {
		t.Fatalf("expected prompt to preserve markdown structure, got %q", llm.messages[1].Content)
	}
}

func TestGenerateWritingParsesSEOTitleSuggestions(t *testing.T) {
	llm := &writingLLMStub{response: "1. Go 并发实践指南\n2. 写给后端工程师的 Go 并发\n3. Go 协程与通道最佳实践"}
	svc := NewAIService(llm, nil, nil)

	result, err := svc.GenerateWriting(context.Background(), WritingRequest{
		Action:  WritingActionSEOTitle,
		Content: "本文介绍 goroutine、channel 和并发控制。",
		Title:   "Go 并发",
		Summary: "Go 并发模型介绍",
	})
	if err != nil {
		t.Fatalf("expected seo title generation to succeed, got %v", err)
	}
	want := []string{"Go 并发实践指南", "写给后端工程师的 Go 并发", "Go 协程与通道最佳实践"}
	if strings.Join(result.Suggestions, "|") != strings.Join(want, "|") {
		t.Fatalf("expected suggestions %#v, got %#v", want, result.Suggestions)
	}
	if result.Result != want[0] {
		t.Fatalf("expected first suggestion as result, got %q", result.Result)
	}
}

func TestGenerateWritingRejectsInvalidAction(t *testing.T) {
	svc := NewAIService(&writingLLMStub{response: "unused"}, nil, nil)

	_, err := svc.GenerateWriting(context.Background(), WritingRequest{
		Action:  "translate",
		Content: "正文",
	})
	if err == nil {
		t.Fatal("expected invalid action error")
	}
}

func TestDisabledAIServiceGenerateWritingReturnsUnavailableError(t *testing.T) {
	svc := NewDisabledAIService("writing backend unavailable")

	_, err := svc.GenerateWriting(context.Background(), WritingRequest{
		Action:  WritingActionPolish,
		Content: "正文",
	})
	if !errors.Is(err, ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from GenerateWriting, got %v", err)
	}
}
