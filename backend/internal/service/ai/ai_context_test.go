package ai

import (
	"context"
	"errors"
	"testing"

	chatcore "wenDao/internal/service/chatcore"
)

type contextRecordingThinkTank struct {
	seenErr            error
	seenQuestion       string
	seenConversationID *int64
	seenRunID          int64
}

func (t *contextRecordingThinkTank) Manifest() chatcore.PluginManifest {
	return chatcore.PluginManifest{Name: "test-agent"}
}

func (t *contextRecordingThinkTank) Run(ctx context.Context, input chatcore.AgentRunInput) (*chatcore.ThinkTankChatResponse, error) {
	t.seenErr = ctx.Err()
	t.seenQuestion = input.Question
	t.seenConversationID = input.ConversationID
	return &chatcore.ThinkTankChatResponse{Message: "ok"}, nil
}

func (t *contextRecordingThinkTank) RunStream(ctx context.Context, input chatcore.AgentRunInput) (<-chan chatcore.StreamEvent, <-chan error) {
	t.seenErr = ctx.Err()
	t.seenQuestion = input.Question
	t.seenConversationID = input.ConversationID
	eventCh := make(chan chatcore.StreamEvent, 1)
	errCh := make(chan error)
	eventCh <- chatcore.StreamEvent{Type: chatcore.StreamEventChunk, Message: "ok"}
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func (t *contextRecordingThinkTank) ResumeStream(ctx context.Context, input chatcore.AgentResumeInput) (<-chan chatcore.StreamEvent, <-chan error) {
	t.seenErr = ctx.Err()
	t.seenRunID = input.RunID
	eventCh := make(chan chatcore.StreamEvent, 1)
	errCh := make(chan error)
	eventCh <- chatcore.StreamEvent{Type: chatcore.StreamEventResume, RunID: input.RunID}
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

func TestAIServiceStreamsThroughAgentPlugin(t *testing.T) {
	agent := &contextRecordingThinkTank{}
	svc := NewAIService(nil, agent, nil)
	convID := int64(7)

	eventCh, errCh := svc.ChatStream(context.Background(), "stream question", &convID, nil)

	if event := <-eventCh; event.Type != chatcore.StreamEventChunk || event.Message != "ok" {
		t.Fatalf("expected stream event from plugin, got %#v", event)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("expected clean stream close, got %v", err)
	}
	if agent.seenQuestion != "stream question" || agent.seenConversationID != &convID {
		t.Fatalf("expected stream input to reach plugin, got question=%q conversation=%v", agent.seenQuestion, agent.seenConversationID)
	}
}

func TestAIServiceResumeStreamsThroughAgentPlugin(t *testing.T) {
	agent := &contextRecordingThinkTank{}
	svc := NewAIService(nil, agent, nil)

	eventCh, errCh := svc.ResumeChatStream(context.Background(), 7, 99, nil)

	if event := <-eventCh; event.Type != chatcore.StreamEventResume || event.RunID != 99 {
		t.Fatalf("expected resume event from plugin, got %#v", event)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("expected clean resume stream close, got %v", err)
	}
	if agent.seenRunID != 99 {
		t.Fatalf("expected resume input to reach plugin, got run id %d", agent.seenRunID)
	}
}
