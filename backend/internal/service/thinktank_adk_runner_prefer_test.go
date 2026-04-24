package service

import (
	"context"
	"testing"
)

func TestThinkTankService_Chat_PrefersADKRunnerAnswerWhenAvailable(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "站内资料充足"}}
	synthesizer := &stubSynthesizer{answer: "手工汇总答案", sources: []string{"文章标题"}}
	svc := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}).(*thinkTankService)

	svc.adkRunner = &thinkTankADKRunner{runner: nil, agent: nil}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		return "ADK 最终答案", nil
	}

	resp, err := svc.Chat(context.Background(), "调研一下李小龙", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "ADK 最终答案" {
		t.Fatalf("expected ADK answer to be the complete response, got %q", resp.Message)
	}
}
