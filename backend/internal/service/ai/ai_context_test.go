package ai

import (
	"context"
	"errors"
	"testing"

	chatcore "wenDao/internal/service/chatcore"
)

type contextRecordingThinkTank struct {
	seenErr error
}

func (t *contextRecordingThinkTank) Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*chatcore.ThinkTankChatResponse, error) {
	t.seenErr = ctx.Err()
	return &chatcore.ThinkTankChatResponse{Message: "ok"}, nil
}

func (t *contextRecordingThinkTank) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan chatcore.StreamEvent, <-chan error) {
	eventCh := make(chan chatcore.StreamEvent)
	errCh := make(chan error)
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func (t *contextRecordingThinkTank) ResumeChatStream(ctx context.Context, conversationID int64, runID int64, userID *int64) (<-chan chatcore.StreamEvent, <-chan error) {
	eventCh := make(chan chatcore.StreamEvent)
	errCh := make(chan error)
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func TestAIServiceChatPassesCallerContextToThinkTank(t *testing.T) {
	thinkTank := &contextRecordingThinkTank{}
	svc := NewAIService(nil, thinkTank, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.Chat(ctx, "你好", nil, nil); err != nil {
		t.Fatalf("expected chat to return thinktank response, got %v", err)
	}
	if !errors.Is(thinkTank.seenErr, context.Canceled) {
		t.Fatalf("expected canceled caller context to reach thinktank, got %v", thinkTank.seenErr)
	}
}
