package service

import (
	"context"
	"errors"
	"testing"
)

func TestDisabledAIService_ReturnsUnavailableError(t *testing.T) {
	svc := NewDisabledAIService("vector backend offline")

	if _, err := svc.Chat("你好", nil, nil); !errors.Is(err, ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from Chat, got %v", err)
	}

	if _, err := svc.GenerateSummary("正文"); !errors.Is(err, ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from GenerateSummary, got %v", err)
	}

	eventCh, errCh := svc.ChatStream(context.Background(), "你好", nil, nil)
	if _, ok := <-eventCh; ok {
		t.Fatal("expected disabled ChatStream event channel to be closed")
	}

	err, ok := <-errCh
	if !ok {
		t.Fatal("expected disabled ChatStream error channel to contain an error")
	}
	if !errors.Is(err, ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from ChatStream, got %v", err)
	}
}
