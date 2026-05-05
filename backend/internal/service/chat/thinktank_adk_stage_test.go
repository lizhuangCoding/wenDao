package chat

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

func TestThinkTankService_ChatStream_EmitsFullStageLifecycle(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient", Summary: "站内资料不足", Sources: []SourceRef{{Kind: "article", Title: "李小龙的功夫哲学", URL: "/article/lee-philosophy"}}}}
	journalist := &stubJournalist{result: &JournalistResult{Summary: "外部调研结果", Sources: []SourceRef{{Kind: "web", Title: "Wikipedia: Bruce Lee", URL: "https://en.wikipedia.org/wiki/Bruce_Lee"}}}}
	synthesizer := &stubSynthesizer{answer: "最终回答", sources: []string{"李小龙的功夫哲学", "Wikipedia: Bruce Lee"}}
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "调研一下李小龙",
		Intent:             "了解李小龙的人物背景和影响",
		AnswerGoal:         "research",
		TargetDimensions:   []string{"生平背景", "核心成就", "武术思想"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{defaultAcceptanceReview()}}
	svc := NewThinkTankService(librarian, journalist, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer)

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

	expected := []string{"analyzing", "clarifying_intent", "local_search", "web_research", "integration", "completed"}
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

func TestThinkTankService_ChatStream_EmitsQuestionWhenClarifierNeedsUser(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion:    "帮我看看这个报错怎么修",
		Intent:                "定位报错",
		ShouldAskUser:         true,
		ClarificationQuestion: "请把完整报错信息发我。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier)

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我看看这个报错怎么修", nil, nil)
	var question string
	for event := range eventCh {
		if event.Type == StreamEventQuestion {
			question = event.Message
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	if !strings.Contains(question, "完整报错信息") {
		t.Fatalf("expected clarifier question event, got %q", question)
	}
}

func TestThinkTankService_ChatStream_EmitsReviewingAndRevisionStages(t *testing.T) {
	clarifier := &stubClarifier{decision: defaultClarifierDecision("帮我分析一下 AI Agent 的发展趋势")}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{
		{
			Verdict:             acceptanceVerdictRevise,
			MissingDimensions:   []string{"风险限制"},
			RevisionInstruction: "补充风险限制",
		},
		defaultAcceptanceReview(),
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	var calls int
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		calls++
		if calls == 2 && !strings.Contains(question, "补充风险限制") {
			t.Fatalf("expected revision instruction in second run, got %q", question)
		}
		return "第" + strconv.Itoa(calls) + "版答案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我分析一下 AI Agent 的发展趋势", nil, nil)
	var stages []string
	var finalChunk string
	for event := range eventCh {
		if event.Type == StreamEventStage {
			stages = append(stages, event.Stage)
		}
		if event.Type == StreamEventChunk {
			finalChunk += event.Message
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	for _, want := range []string{"reviewing", "revising", "completed"} {
		if !containsStage(stages, want) {
			t.Fatalf("expected stage %q in %#v", want, stages)
		}
	}
	if calls != 2 {
		t.Fatalf("expected one revision, got %d calls", calls)
	}
	if !strings.Contains(finalChunk, "第2版答案") {
		t.Fatalf("expected revised answer chunk, got %q", finalChunk)
	}
}
