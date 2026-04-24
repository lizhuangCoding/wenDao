package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"wenDao/internal/model"
)

type stubConversationMemorySummarizer struct {
	drafts []ConversationMemoryDraft
	err    error
	called int
}

func (s *stubConversationMemorySummarizer) Summarize(ctx context.Context, history []model.ChatMessage, existing []model.ConversationMemory) ([]ConversationMemoryDraft, error) {
	s.called++
	if s.err != nil {
		return nil, s.err
	}
	return s.drafts, nil
}

func TestCompressConversationMemory_KeepsRecentKeyContext(t *testing.T) {
	history := []model.ChatMessage{
		{Role: "user", Content: "第一轮问题"},
		{Role: "assistant", Content: "第一轮回答：博主在《李小龙的功夫哲学》提到了李小龙。"},
		{Role: "user", Content: "第二轮问题"},
		{Role: "assistant", Content: "第二轮回答：原文是“我无法教你，只能帮你探索自己，仅此而已”。"},
	}
	memory := compressConversationMemory(history)
	if !strings.Contains(memory, "李小龙的功夫哲学") {
		t.Fatalf("expected compressed memory to retain key article reference, got %q", memory)
	}
	if !strings.Contains(memory, "我无法教你") {
		t.Fatalf("expected compressed memory to retain quoted original text, got %q", memory)
	}
}

func TestBuildConversationMemory_UsesStoredSummaryAndRecentMessages(t *testing.T) {
	oldVerbosePrompt := "第一轮问题：请调研 AI Agent 的产品形态以及 Manus 的交互方式，内容很长很长，需要被压缩。"
	history := []model.ChatMessage{
		{ID: 1, Role: "user", Content: oldVerbosePrompt},
		{ID: 2, Role: "assistant", Content: "第一轮回答：Manus 的过程展示包含任务拆解、工具调用和结果整合，这些旧细节不应完整进入 prompt。"},
		{ID: 3, Role: "user", Content: "第二轮问题：继续分析多 Agent 协作。"},
		{ID: 4, Role: "assistant", Content: "第二轮回答：多 Agent 需要过程透明、可展开日志和最终结果区分。"},
		{ID: 5, Role: "user", Content: "第三轮问题：如何保存到 MySQL？"},
		{ID: 6, Role: "assistant", Content: "第三轮回答：可以保存 run step 和工具日志。"},
		{ID: 7, Role: "user", Content: "第四轮问题：现在继续优化记忆。"},
		{ID: 8, Role: "assistant", Content: "第四轮回答：久远记忆应压缩，近期记忆保留原文。"},
		{ID: 9, Role: "user", Content: "第五轮问题：把记忆摘要和近期上下文组合起来。"},
		{ID: 10, Role: "assistant", Content: "第五轮回答：近期消息保留原文，较早消息交给摘要或长期记忆。"},
	}
	memories := []model.ConversationMemory{
		{Scope: ConversationMemoryScopeSummary, Content: "用户此前关注 Manus 风格多 Agent 过程展示、工具调用日志和 MySQL 持久化。"},
	}

	memory := buildConversationMemory(history, memories)
	if !strings.Contains(memory, "长期记忆") || !strings.Contains(memory, "用户此前关注 Manus 风格") {
		t.Fatalf("expected stored memory summary, got %q", memory)
	}
	if !strings.Contains(memory, "最近对话") || !strings.Contains(memory, "第五轮问题：把记忆摘要和近期上下文组合起来。") {
		t.Fatalf("expected recent raw messages, got %q", memory)
	}
	if strings.Contains(memory, oldVerbosePrompt) {
		t.Fatalf("expected old verbose prompt to be summarized away, got %q", memory)
	}
}

func TestBuildConversationMemory_CombinesStructuredMemoriesWithRelevantHistory(t *testing.T) {
	padding := strings.Repeat("不相关的背景说明。", 30)
	history := []model.ChatMessage{
		{Role: "user", Content: "我们之前决定在 API 网关里用 Redis 做限流和热点缓存。"},
		{Role: "assistant", Content: "是的，Redis 负责共享计数器、短时状态和热点数据缓存。"},
		{Role: "user", Content: "中间有一大段关于前端主题色的讨论。" + padding},
		{Role: "assistant", Content: "主题色讨论先不影响后端架构。" + padding},
		{Role: "user", Content: "最近我们又加了监控埋点。"},
		{Role: "assistant", Content: "监控会接 Prometheus 和告警。"},
		{Role: "user", Content: "最后再确认一下 Redis 方案是否继续沿用？"},
		{Role: "assistant", Content: "可以沿用，但要补充降级策略。"},
	}
	memories := []model.ConversationMemory{
		{Scope: ConversationMemoryScopeSummary, Content: "项目正在重构 API 网关的缓存与限流链路。"},
		{Scope: ConversationMemoryScopePreference, Content: "团队偏好把共享限流状态放进 Redis。"},
		{Scope: ConversationMemoryScopeDecision, Content: "已决定网关统一负责降级、熔断和监控埋点。"},
		{Scope: ConversationMemoryScopeOpenThread, Content: ""},
	}

	memory := buildConversationMemory(history, memories)
	if !strings.Contains(memory, "长期记忆") {
		t.Fatalf("expected structured memories section, got %q", memory)
	}
	if !strings.Contains(memory, "团队偏好把共享限流状态放进 Redis") {
		t.Fatalf("expected preference memory to be included, got %q", memory)
	}
	if !strings.Contains(memory, "已决定网关统一负责降级、熔断和监控埋点") {
		t.Fatalf("expected decision memory to be included, got %q", memory)
	}
	if strings.Contains(memory, "OpenThread") || strings.Contains(memory, "\n- \n") {
		t.Fatalf("expected empty memory content to be ignored, got %q", memory)
	}
	if !strings.Contains(memory, "我们之前决定在 API 网关里用 Redis 做限流和热点缓存") {
		t.Fatalf("expected Redis context to remain available in conversation memory, got %q", memory)
	}
	if !strings.Contains(memory, "最近对话") || !strings.Contains(memory, "最后再确认一下 Redis 方案是否继续沿用") {
		t.Fatalf("expected recent raw messages to remain, got %q", memory)
	}
}

func TestBuildConversationMemoryForQuestion_RetrievesRelevantOlderMessages(t *testing.T) {
	padding := strings.Repeat("补充说明", 40)
	history := []model.ChatMessage{
		{Role: "user", Content: "我们要模仿 Manus 的多 Agent 过程展示。"},
		{Role: "assistant", Content: "核心是任务拆解、工具调用和结果整合。"},
		{Role: "user", Content: "第二轮讨论普通登录问题。" + padding},
		{Role: "assistant", Content: "登录问题暂时不处理。" + padding},
		{Role: "user", Content: "第三轮讨论文章置顶。" + padding},
		{Role: "assistant", Content: "文章置顶字段是 is_top。" + padding},
		{Role: "user", Content: "第四轮讨论评论回复。" + padding},
		{Role: "assistant", Content: "评论回复使用 parent_id。" + padding},
		{Role: "user", Content: "第五轮讨论构建错误。" + padding},
		{Role: "assistant", Content: "构建错误已经修复。" + padding},
	}

	memory := buildConversationMemoryForQuestion("Manus 那种过程展示还需要什么？", history, nil)
	if !strings.Contains(memory, "相关历史片段") {
		t.Fatalf("expected relevant older section, got %q", memory)
	}
	if !strings.Contains(memory, "模仿 Manus") {
		t.Fatalf("expected old Manus context to be retrieved, got %q", memory)
	}
}

func TestSelectRecentMemoryStart_UsesMoreContextForReferentialQuestion(t *testing.T) {
	history := []model.ChatMessage{
		{Role: "user", Content: "第一轮短消息"},
		{Role: "assistant", Content: "第一轮回答"},
		{Role: "user", Content: "第二轮短消息"},
		{Role: "assistant", Content: "第二轮回答"},
		{Role: "user", Content: "第三轮短消息"},
		{Role: "assistant", Content: "第三轮回答"},
		{Role: "user", Content: "第四轮短消息"},
		{Role: "assistant", Content: "第四轮回答"},
	}

	referentialStart := selectRecentMemoryStart("继续", history)
	explicitStart := selectRecentMemoryStart(strings.Repeat("这是一个明确的新问题，", 20), history)
	if referentialStart > explicitStart {
		t.Fatalf("expected referential question to keep at least as much recent context, got referential start %d explicit start %d", referentialStart, explicitStart)
	}
}

func TestSummarizeOlderConversationMemory_ProducesSingleSentence(t *testing.T) {
	history := []model.ChatMessage{
		{Role: "user", Content: "请调研 Manus AI 的多 Agent 过程展示。"},
		{Role: "assistant", Content: "Manus 会展示任务拆解、工具调用、外部调研和最终整合。"},
		{Role: "user", Content: "还要把过程和最终结果区分样式。"},
	}

	summary := summarizeOlderConversationMemory(history)
	if summary == "" {
		t.Fatalf("expected non-empty summary")
	}
	if strings.Count(summary, "。") > 1 {
		t.Fatalf("expected one-sentence summary, got %q", summary)
	}
	if !strings.Contains(summary, "Manus") {
		t.Fatalf("expected summary to retain key subject, got %q", summary)
	}
}

func TestThinkTankService_UsesConversationMemoryForFollowupQuestion(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{
		CoverageStatus: "sufficient",
		Summary:        "上一篇回答已经提到：博主在《李小龙的功夫哲学》一文中引用了“我无法教你，只能帮你探索自己，仅此而已”。",
		Sources:        []SourceRef{{Kind: "article", Title: "李小龙的功夫哲学", URL: "/article/lee-philosophy"}},
	}}
	synthesizer := &stubSynthesizer{answer: "原文在《李小龙的功夫哲学》这篇文章里。", sources: []string{"李小龙的功夫哲学"}}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 33, UserID: 7, Title: "李小龙调研"}}
	msgRepo := &stubChatMessageRepository{items: []model.ChatMessage{
		{ID: 1, ConversationID: 33, Role: "user", Content: "博主对于李小龙的看法"},
		{ID: 2, ConversationID: 33, Role: "assistant", Content: "博主认为李小龙不仅仅是一位武术家，更是一位哲学家。"},
	}}
	runRepo := &stubConversationRunRepository{}
	svc := NewThinkTankService(librarian, nil, synthesizer, runRepo, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, convRepo, msgRepo, nil, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "博主提到的原文在哪里？", ptrInt64(33), ptrInt64(7))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "原文在《李小龙的功夫哲学》这篇文章里。" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
	if len(msgRepo.created) != 2 {
		t.Fatalf("expected follow-up user and assistant messages to be persisted, got %d", len(msgRepo.created))
	}
}

func TestThinkTankService_PersistsLongConversationSummaryMemory(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "可直接回答"}}
	synthesizer := &stubSynthesizer{answer: "已继续优化记忆模块。"}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 34, UserID: 7, Title: "记忆优化"}}
	msgRepo := &stubChatMessageRepository{items: []model.ChatMessage{
		{ID: 1, ConversationID: 34, Role: "user", Content: "请调研 Manus AI 的多 Agent 过程展示。"},
		{ID: 2, ConversationID: 34, Role: "assistant", Content: "Manus 展示任务拆解和工具调用。"},
		{ID: 3, ConversationID: 34, Role: "user", Content: "还要保存到 MySQL。"},
		{ID: 4, ConversationID: 34, Role: "assistant", Content: "可以保存步骤和日志。"},
		{ID: 5, ConversationID: 34, Role: "user", Content: "过程和结果样式要区分。"},
		{ID: 6, ConversationID: 34, Role: "assistant", Content: "过程用折叠面板，结果用正文。"},
	}}
	memoryRepo := &stubConversationMemoryRepository{}
	svc := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, memoryRepo, convRepo, msgRepo, nil, &stubAILogger{})

	_, err := svc.Chat(context.Background(), "继续优化记忆模块", ptrInt64(34), ptrInt64(7))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(memoryRepo.memories) != 1 {
		t.Fatalf("expected one persisted memory, got %#v", memoryRepo.memories)
	}
	if memoryRepo.memories[0].Scope != ConversationMemoryScopeSummary {
		t.Fatalf("expected summary scope, got %#v", memoryRepo.memories[0])
	}
	if !strings.Contains(memoryRepo.memories[0].Content, "Manus") {
		t.Fatalf("expected memory content to preserve old subject, got %q", memoryRepo.memories[0].Content)
	}
}

func TestThinkTankService_PersistsStructuredDynamicMemories(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "可直接回答"}}
	synthesizer := &stubSynthesizer{answer: "已继续优化记忆模块。"}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 35, UserID: 7, Title: "记忆优化"}}
	msgRepo := &stubChatMessageRepository{items: []model.ChatMessage{
		{ID: 1, ConversationID: 35, Role: "user", Content: "我希望多 Agent 过程像 Manus 一样展示。"},
		{ID: 2, ConversationID: 35, Role: "assistant", Content: "可以使用折叠过程面板。"},
		{ID: 3, ConversationID: 35, Role: "user", Content: "过程和最终结果样式要区分。"},
		{ID: 4, ConversationID: 35, Role: "assistant", Content: "最终结果用正文样式。"},
		{ID: 5, ConversationID: 35, Role: "user", Content: "还要存到 MySQL。"},
		{ID: 6, ConversationID: 35, Role: "assistant", Content: "可以保存 run step。"},
	}}
	memoryRepo := &stubConversationMemoryRepository{}
	summarizer := &stubConversationMemorySummarizer{drafts: []ConversationMemoryDraft{
		{Scope: ConversationMemoryScopeSummary, Content: "用户正在优化多 Agent 过程展示和记忆压缩。", Importance: 2},
		{Scope: ConversationMemoryScopePreference, Content: "用户偏好类似 Manus 的渐进式披露效果。", Importance: 3},
		{Scope: ConversationMemoryScopeDecision, Content: "过程日志和最终回答必须区分样式并持久化。", Importance: 3},
	}}
	svc := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, memoryRepo, convRepo, msgRepo, nil, &stubAILogger{}, summarizer)

	_, err := svc.Chat(context.Background(), "继续优化记忆模块", ptrInt64(35), ptrInt64(7))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if summarizer.called != 1 {
		t.Fatalf("expected dynamic summarizer to be called once, got %d", summarizer.called)
	}
	if len(memoryRepo.memories) != 3 {
		t.Fatalf("expected three structured memories, got %#v", memoryRepo.memories)
	}
	if _, ok := findMemoryByScope(memoryRepo.memories, ConversationMemoryScopePreference); !ok {
		t.Fatalf("expected preference memory in %#v", memoryRepo.memories)
	}
	if _, ok := findMemoryByScope(memoryRepo.memories, ConversationMemoryScopeDecision); !ok {
		t.Fatalf("expected decision memory in %#v", memoryRepo.memories)
	}
}

func TestThinkTankService_FallsBackWhenDynamicMemorySummarizerFails(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "可直接回答"}}
	synthesizer := &stubSynthesizer{answer: "已继续优化记忆模块。"}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 36, UserID: 7, Title: "记忆优化"}}
	msgRepo := &stubChatMessageRepository{items: []model.ChatMessage{
		{ID: 1, ConversationID: 36, Role: "user", Content: "请调研 Manus AI 的多 Agent 过程展示。"},
		{ID: 2, ConversationID: 36, Role: "assistant", Content: "Manus 展示任务拆解和工具调用。"},
		{ID: 3, ConversationID: 36, Role: "user", Content: "还要保存到 MySQL。"},
		{ID: 4, ConversationID: 36, Role: "assistant", Content: "可以保存步骤和日志。"},
		{ID: 5, ConversationID: 36, Role: "user", Content: "过程和结果样式要区分。"},
		{ID: 6, ConversationID: 36, Role: "assistant", Content: "过程用折叠面板，结果用正文。"},
	}}
	memoryRepo := &stubConversationMemoryRepository{}
	summarizer := &stubConversationMemorySummarizer{err: errors.New("llm unavailable")}
	svc := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, memoryRepo, convRepo, msgRepo, nil, &stubAILogger{}, summarizer)

	_, err := svc.Chat(context.Background(), "继续优化记忆模块", ptrInt64(36), ptrInt64(7))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	memory, ok := findMemoryByScope(memoryRepo.memories, ConversationMemoryScopeSummary)
	if !ok {
		t.Fatalf("expected fallback summary memory in %#v", memoryRepo.memories)
	}
	if !strings.Contains(memory.Content, "Manus") {
		t.Fatalf("expected fallback memory to preserve old subject, got %q", memory.Content)
	}
}

func findMemoryByScope(memories []model.ConversationMemory, scope string) (model.ConversationMemory, bool) {
	for _, memory := range memories {
		if memory.Scope == scope {
			return memory, true
		}
	}
	return model.ConversationMemory{}, false
}
