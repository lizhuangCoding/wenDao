package chat

import (
	"context"
	"strings"
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

func TestThinkTankOrchestrator_EffectiveQuestionCombinesADKPendingContext(t *testing.T) {
	originalQuestion := "帮我规划数据库迁移"
	systemQuestion := "你要迁移到哪个数据库？"
	userSupplement := "从 MySQL 迁移到 PostgreSQL，停机窗口 30 分钟"
	run := &model.ConversationRun{
		Status:           "waiting_user",
		OriginalQuestion: originalQuestion,
		PendingQuestion:  &systemQuestion,
		PendingContext:   marshalADKPendingContext("thinktank-52-123-8", originalQuestion, systemQuestion),
	}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}).(*thinkTankService)

	effectiveQuestion, skipClarifier := svc.orchestrator.effectiveQuestionFromPending(userSupplement, run)

	if !skipClarifier {
		t.Fatalf("expected ADK pending follow-up to skip ClarifierAgent")
	}
	for _, want := range []string{originalQuestion, systemQuestion, userSupplement} {
		if !strings.Contains(effectiveQuestion, want) {
			t.Fatalf("expected effective question to contain %q, got %q", want, effectiveQuestion)
		}
	}
}
