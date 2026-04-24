package chat

import (
	"context"
	"testing"

	"wenDao/internal/model"
)

func TestThinkTankService_ChatStream_AllowsInjectedADKRunner(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "站内资料充足"}}
	synthesizer := &stubSynthesizer{answer: "最终回答", sources: []string{"文章标题"}}
	service := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, &thinkTankADKRunner{}).(*thinkTankService)

	eventCh, errCh := service.ChatStream(context.Background(), "调研一下李小龙", nil, nil)
	var sawCompleted bool
	for event := range eventCh {
		if event.Type == StreamEventDone {
			sawCompleted = true
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}
	if !sawCompleted {
		t.Fatalf("expected stream to complete when ADK runner is injected")
	}
}

func TestParseADKPendingContext_RoundTripsWaitingUserCheckpoint(t *testing.T) {
	run := &model.ConversationRun{
		Status:         "waiting_user",
		PendingContext: `{"type":"adk_interrupt","checkpoint_id":"thinktank-41-123-8"}`,
	}

	ctxInfo, ok := parseADKPendingContext(run)
	if !ok {
		t.Fatal("expected waiting_user run with ADK checkpoint to be resumable")
	}
	if ctxInfo.Type != "adk_interrupt" {
		t.Fatalf("expected adk interrupt type, got %#v", ctxInfo)
	}
	if ctxInfo.Checkpoint != "thinktank-41-123-8" {
		t.Fatalf("expected checkpoint round-trip, got %#v", ctxInfo)
	}
}
