package chat

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"wenDao/internal/model"
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

func TestThinkTankService_ChatStream_EmitsClarifierAndAcceptanceSteps(t *testing.T) {
	question := "帮我分析一下 AI Agent 的发展趋势"
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "分析 AI Agent 的发展趋势",
		Intent:             "判断 AI Agent 的机会与风险",
		NeedSummary:        "需要一份面向产品负责人的趋势分析",
		TargetDimensions:   []string{"机会", "风险", "落地建议"},
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:           acceptanceVerdictPass,
		Score:             88,
		MatchedDimensions: []string{"机会", "风险", "落地建议"},
		Summary:           "覆盖了关键维度",
		Available:         true,
	}}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		return "趋势分析正文", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), question, nil, nil)
	seenSteps := map[string]bool{}
	var finalChunk string
	for event := range eventCh {
		if event.Type == StreamEventStep {
			seenSteps[event.AgentName] = true
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
	for _, want := range []string{"ClarifierAgent", "AcceptanceAgent"} {
		if !seenSteps[want] {
			t.Fatalf("expected stream step for %s, got %#v", want, seenSteps)
		}
	}
	for _, want := range []string{"验收摘要", "评分 88/100"} {
		if !strings.Contains(finalChunk, want) {
			t.Fatalf("expected final chunk to contain %q, got %q", want, finalChunk)
		}
	}
}

func TestThinkTankService_ChatStream_ClarifierQuestionIsStructured(t *testing.T) {
	clarifier := &stubClarifier{decision: ClarifierDecision{
		NormalizedQuestion: "帮我做增长方案",
		Intent:             "制定增长方案",
		ShouldAskUser:      true,
		NeedSummary:        "为新产品制定增长方案",
		MissingDimensions:  []string{"目标用户是谁", "预算范围是多少"},
		WhyNeeded:          "不同用户和预算会改变渠道组合。",
		SuggestedReply:     "目标用户是中小企业主，预算 10 万以内。",
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier)

	eventCh, errCh := svc.ChatStream(context.Background(), "帮我做增长方案", nil, nil)
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
	for _, want := range []string{"我理解你是想：", "为了后续回答更精确，需要确认：", "1. 目标用户是谁", "2. 预算范围是多少", "为什么需要这些信息：", "你可以这样回复："} {
		if !strings.Contains(question, want) {
			t.Fatalf("expected structured clarifier question to contain %q, got %q", want, question)
		}
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

func TestThinkTankService_ChatStream_ReturnsInitialAnswerWhenRevisionFails(t *testing.T) {
	question := "帮我分析一下 AI Agent 的发展趋势"
	clarifier := &stubClarifier{decision: defaultClarifierDecision(question)}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:             acceptanceVerdictRevise,
		MissingDimensions:   []string{"风险限制"},
		RevisionInstruction: "补充风险限制",
		Reason:              "缺少风险边界",
	}}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	var calls int
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		calls++
		if calls == 1 {
			return "初版答案", nil
		}
		return "", errors.New("revision unavailable")
	}

	eventCh, errCh := svc.ChatStream(context.Background(), question, nil, nil)
	var finalChunk string
	var sawDone bool
	for event := range eventCh {
		if event.Type == StreamEventChunk {
			finalChunk += event.Message
		}
		if event.Type == StreamEventDone {
			sawDone = true
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error when revision fails after an answer exists, got %v", err)
		}
	}
	if calls != 2 {
		t.Fatalf("expected initial and revision attempts, got %d", calls)
	}
	for _, want := range []string{"初版答案", "回答限制", "风险限制"} {
		if !strings.Contains(finalChunk, want) {
			t.Fatalf("expected final chunk to contain %q, got %q", want, finalChunk)
		}
	}
	if !sawDone {
		t.Fatalf("expected stream to finish with done event")
	}
}

func TestThinkTankService_ChatStream_AcceptanceCanAskUserAndPersistWaitingRun(t *testing.T) {
	question := "帮我制定迁移方案"
	followUp := "需要确认目标数据库和停机窗口。"
	runRepo := &stubConversationRunRepository{}
	msgRepo := &stubChatMessageRepository{}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 51, UserID: 12, Title: "迁移会话"}}
	clarifier := &stubClarifier{decision: defaultClarifierDecision(question)}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:      acceptanceVerdictAskUser,
		UserQuestion: followUp,
		Reason:       "缺少关键约束",
	}}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, runRepo, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, convRepo, msgRepo, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		return "初版方案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), question, ptrInt64(51), ptrInt64(12))
	var emittedQuestion string
	var chunks []string
	for event := range eventCh {
		if event.Type == StreamEventQuestion {
			emittedQuestion = event.Message
		}
		if event.Type == StreamEventChunk {
			chunks = append(chunks, event.Message)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	for _, want := range []string{"验收时发现还缺少一个关键信息", followUp, "为什么需要", "缺少关键约束"} {
		if !strings.Contains(emittedQuestion, want) {
			t.Fatalf("expected formatted acceptance follow-up to contain %q, got %q", want, emittedQuestion)
		}
	}
	if emittedQuestion == followUp {
		t.Fatalf("expected acceptance follow-up, got %q", emittedQuestion)
	}
	if len(chunks) != 0 {
		t.Fatalf("did not expect obsolete answer chunks before ask_user, got %#v", chunks)
	}
	if runRepo.active == nil || runRepo.active.Status != "waiting_user" {
		t.Fatalf("expected waiting_user run, got %#v", runRepo.active)
	}
	if !strings.Contains(runRepo.active.PendingContext, "acceptance_interrupt") || !strings.Contains(runRepo.active.PendingContext, question) {
		t.Fatalf("expected acceptance pending context with original question, got %q", runRepo.active.PendingContext)
	}
}

func TestThinkTankService_ChatStream_ManualFlowAcceptanceCanAskUser(t *testing.T) {
	question := "帮我制定迁移方案"
	followUp := "需要确认目标数据库和停机窗口。"
	runRepo := &stubConversationRunRepository{}
	msgRepo := &stubChatMessageRepository{}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 53, UserID: 12, Title: "迁移会话"}}
	clarifier := &stubClarifier{decision: defaultClarifierDecision(question)}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{{
		Verdict:      acceptanceVerdictAskUser,
		UserQuestion: followUp,
		Reason:       "缺少关键约束",
	}}}
	svc := NewThinkTankService(
		&stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "站内资料充足"}},
		nil,
		&stubSynthesizer{answer: "初版方案"},
		runRepo,
		&stubConversationRunStepRepository{},
		&stubConversationMemoryRepository{},
		convRepo,
		msgRepo,
		nil,
		&stubAILogger{},
		clarifier,
		reviewer,
	)

	eventCh, errCh := svc.ChatStream(context.Background(), question, ptrInt64(53), ptrInt64(12))
	var emittedQuestion string
	var chunks []string
	for event := range eventCh {
		if event.Type == StreamEventQuestion {
			emittedQuestion = event.Message
		}
		if event.Type == StreamEventChunk {
			chunks = append(chunks, event.Message)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	if reviewer.calls != 1 {
		t.Fatalf("expected manual stream to run acceptance review, got %d calls", reviewer.calls)
	}
	for _, want := range []string{"验收时发现还缺少一个关键信息", followUp, "为什么需要", "缺少关键约束"} {
		if !strings.Contains(emittedQuestion, want) {
			t.Fatalf("expected formatted acceptance follow-up to contain %q, got %q", want, emittedQuestion)
		}
	}
	if emittedQuestion == followUp {
		t.Fatalf("expected formatted acceptance follow-up, got raw question %q", emittedQuestion)
	}
	if len(chunks) != 0 {
		t.Fatalf("did not expect obsolete answer chunks before ask_user, got %#v", chunks)
	}
	if runRepo.active == nil || runRepo.active.Status != "waiting_user" {
		t.Fatalf("expected waiting_user run, got %#v", runRepo.active)
	}
	if !strings.Contains(runRepo.active.PendingContext, "acceptance_interrupt") || !strings.Contains(runRepo.active.PendingContext, question) {
		t.Fatalf("expected acceptance pending context with original question, got %q", runRepo.active.PendingContext)
	}
}

func TestThinkTankService_ChatStream_FollowUpCombinesOriginalQuestionAndSkipsClarifier(t *testing.T) {
	originalQuestion := "帮我规划数据库迁移"
	systemQuestion := "你要迁移到哪个数据库？"
	userSupplement := "从 MySQL 迁移到 PostgreSQL，停机窗口 30 分钟"
	runRepo := &stubConversationRunRepository{active: &model.ConversationRun{
		ID:               77,
		ConversationID:   52,
		UserID:           12,
		Status:           "waiting_user",
		CurrentStage:     "clarifying",
		OriginalQuestion: originalQuestion,
		PendingQuestion:  &systemQuestion,
		PendingContext:   `{"type":"clarifier_interrupt","original_question":"帮我规划数据库迁移","system_question":"你要迁移到哪个数据库？"}`,
	}}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 52, UserID: 12, Title: "迁移会话"}}
	clarifier := &stubClarifier{decision: ClarifierDecision{
		ShouldAskUser:         true,
		ClarificationQuestion: "不应该再次澄清",
	}}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{defaultAcceptanceReview()}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, runRepo, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, convRepo, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: nil}
	var query string
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		query = question
		return "迁移方案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), userSupplement, ptrInt64(52), ptrInt64(12))
	for range eventCh {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	if clarifier.calls != 0 {
		t.Fatalf("expected follow-up resume to skip clarifier, got %d calls", clarifier.calls)
	}
	for _, want := range []string{originalQuestion, systemQuestion, userSupplement} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected resumed query to contain %q, got %q", want, query)
		}
	}
}

func TestThinkTankService_ChatStream_ProductionADKFlowReviewsAndRevisesBeforeEmittingChunk(t *testing.T) {
	question := "帮我分析一下 AI Agent 的发展趋势"
	clarifier := &stubClarifier{decision: defaultClarifierDecision(question)}
	reviewer := &stubAcceptanceReviewer{reviews: []AcceptanceReview{
		{
			Verdict:             acceptanceVerdictRevise,
			MissingDimensions:   []string{"风险限制"},
			RevisionInstruction: "补充风险限制",
		},
		defaultAcceptanceReview(),
	}}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{}, clarifier, reviewer).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{runner: adk.NewRunner(context.Background(), adk.RunnerConfig{Agent: &fakeFinalADKAgent{answer: "初版答案"}})}
	var revisionCalls int
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		revisionCalls++
		if !strings.Contains(question, "补充风险限制") {
			t.Fatalf("expected revision instruction in ADK revision query, got %q", question)
		}
		return "修订答案", nil
	}

	eventCh, errCh := svc.ChatStream(context.Background(), question, nil, nil)
	var stages []string
	var chunks []string
	for event := range eventCh {
		if event.Type == StreamEventStage {
			stages = append(stages, event.Stage)
		}
		if event.Type == StreamEventChunk {
			chunks = append(chunks, event.Message)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no stream error, got %v", err)
		}
	}
	if reviewer.calls != 2 {
		t.Fatalf("expected initial and post-revision reviews, got %d", reviewer.calls)
	}
	if revisionCalls != 1 {
		t.Fatalf("expected exactly one revision ADK fetch, got %d", revisionCalls)
	}
	if len(chunks) != 1 || chunks[0] != "修订答案" {
		t.Fatalf("expected only revised answer chunk, got %#v", chunks)
	}
	for _, want := range []string{"reviewing", "revising", "completed"} {
		if !containsStage(stages, want) {
			t.Fatalf("expected stage %q in %#v", want, stages)
		}
	}
}

type fakeFinalADKAgent struct {
	answer string
}

func (a *fakeFinalADKAgent) Name(ctx context.Context) string {
	return "replanner"
}

func (a *fakeFinalADKAgent) Description(ctx context.Context) string {
	return "fake final answer agent"
}

func (a *fakeFinalADKAgent) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	gen.Send(&adk.AgentEvent{
		AgentName: "replanner",
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: schema.AssistantMessage(`{"response":"`+a.answer+`"}`, nil),
			},
		},
	})
	gen.Close()
	return iter
}
