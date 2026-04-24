package service

import (
	"context"
	"testing"
)

func TestThinkTankService_ChatStream_EmitsFullStageLifecycle(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient", Summary: "站内资料不足", Sources: []SourceRef{{Kind: "article", Title: "李小龙的功夫哲学", URL: "/article/lee-philosophy"}}}}
	journalist := &stubJournalist{result: &JournalistResult{Summary: "外部调研结果", Sources: []SourceRef{{Kind: "web", Title: "Wikipedia: Bruce Lee", URL: "https://en.wikipedia.org/wiki/Bruce_Lee"}}}}
	synthesizer := &stubSynthesizer{answer: "最终回答", sources: []string{"李小龙的功夫哲学", "Wikipedia: Bruce Lee"}}
	svc := NewThinkTankService(librarian, journalist, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{})

	eventCh, errCh := svc.ChatStream(context.Background(), "调研一下李小龙", nil, nil)
	var stages []string
	for event := range eventCh {
		if event.Type == StreamEventStage {
			stages = append(stages, event.Stage)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}

	expected := []string{"analyzing", "local_search", "web_research", "integration", "completed"}
	for _, stage := range expected {
		if !containsStage(stages, stage) {
			t.Logf("Currently saw stages: %#v", stages)
			t.Fatalf("expected stage %q in lifecycle, but not found", stage)
		}
	}
}

func containsStage(stages []string, target string) bool {
	for _, stage := range stages {
		if stage == target {
			return true
		}
	}
	return false
}
