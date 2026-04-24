package service

import (
	"context"
	"testing"
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
