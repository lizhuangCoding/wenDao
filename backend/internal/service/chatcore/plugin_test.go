package chatcore

import (
	"context"
	"testing"
)

type recordingThinkTankService struct {
	chatMessage string

	seenQuestion       string
	seenConversationID *int64
	seenUserID         *int64
}

func (s *recordingThinkTankService) Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	s.seenQuestion = question
	s.seenConversationID = conversationID
	s.seenUserID = userID
	return &ThinkTankChatResponse{Message: s.chatMessage}, nil
}

func (s *recordingThinkTankService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	s.seenQuestion = question
	s.seenConversationID = conversationID
	s.seenUserID = userID
	eventCh := make(chan StreamEvent, 1)
	errCh := make(chan error)
	eventCh <- StreamEvent{Type: StreamEventChunk, Message: "stream ok"}
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func (s *recordingThinkTankService) ResumeChatStream(ctx context.Context, conversationID int64, runID int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	s.seenConversationID = &conversationID
	s.seenUserID = userID
	eventCh := make(chan StreamEvent, 1)
	errCh := make(chan error)
	eventCh <- StreamEvent{Type: StreamEventResume, RunID: runID}
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func TestThinkTankPluginDelegatesChatAndStream(t *testing.T) {
	convID := int64(11)
	userID := int64(22)
	inner := &recordingThinkTankService{chatMessage: "plugin ok"}
	plugin := NewThinkTankPlugin(inner)

	response, err := plugin.Run(context.Background(), AgentRunInput{
		Question:       "hello",
		ConversationID: &convID,
		UserID:         &userID,
	})
	if err != nil {
		t.Fatalf("expected delegated chat response, got error %v", err)
	}
	if response == nil || response.Message != "plugin ok" {
		t.Fatalf("expected delegated chat response, got %#v", response)
	}
	if inner.seenQuestion != "hello" || inner.seenConversationID != &convID || inner.seenUserID != &userID {
		t.Fatalf("expected run input to reach thinktank, got question=%q conversation=%v user=%v", inner.seenQuestion, inner.seenConversationID, inner.seenUserID)
	}

	events, errs := plugin.RunStream(context.Background(), AgentRunInput{Question: "stream"})
	if event := <-events; event.Type != StreamEventChunk || event.Message != "stream ok" {
		t.Fatalf("expected delegated stream event, got %#v", event)
	}
	if err := <-errs; err != nil {
		t.Fatalf("expected stream err channel to close cleanly, got %v", err)
	}
}

func TestThinkTankPluginDelegatesResumeStream(t *testing.T) {
	userID := int64(22)
	inner := &recordingThinkTankService{}
	plugin := NewThinkTankPlugin(inner)

	events, errs := plugin.ResumeStream(context.Background(), AgentResumeInput{
		ConversationID: 11,
		RunID:          99,
		UserID:         &userID,
	})

	if event := <-events; event.Type != StreamEventResume || event.RunID != 99 {
		t.Fatalf("expected delegated resume event, got %#v", event)
	}
	if err := <-errs; err != nil {
		t.Fatalf("expected resume err channel to close cleanly, got %v", err)
	}
	if inner.seenConversationID == nil || *inner.seenConversationID != 11 || inner.seenUserID != &userID {
		t.Fatalf("expected resume input to reach thinktank, got conversation=%v user=%v", inner.seenConversationID, inner.seenUserID)
	}
}

func TestPluginRegistryReturnsDefaultPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := NewThinkTankPlugin(&recordingThinkTankService{})

	if err := registry.Register(plugin, WithDefaultPlugin()); err != nil {
		t.Fatalf("expected register success, got %v", err)
	}
	got, ok := registry.Default()
	if !ok {
		t.Fatal("expected default plugin")
	}
	if got.Manifest().Name != "thinktank" {
		t.Fatalf("expected thinktank default plugin, got %q", got.Manifest().Name)
	}
}
