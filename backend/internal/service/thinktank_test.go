package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type stubConversationRepository struct {
	conversation *model.Conversation
	updated      *model.Conversation
}

func (r *stubConversationRepository) Create(conv *model.Conversation) error {
	r.conversation = conv
	if conv.ID == 0 {
		conv.ID = 1
	}
	return nil
}

func (r *stubConversationRepository) GetByID(id int64) (*model.Conversation, error) {
	if r.conversation == nil {
		return nil, errors.New("conversation not found")
	}
	return r.conversation, nil
}

func (r *stubConversationRepository) GetByUserID(userID int64) ([]model.Conversation, error) {
	if r.conversation == nil {
		return []model.Conversation{}, nil
	}
	return []model.Conversation{*r.conversation}, nil
}

func (r *stubConversationRepository) Update(conv *model.Conversation) error {
	r.updated = conv
	return nil
}

func (r *stubConversationRepository) Delete(id int64) error { return nil }

type stubChatMessageRepository struct {
	created []*model.ChatMessage
	items   []model.ChatMessage
}

func (r *stubChatMessageRepository) Create(msg *model.ChatMessage) error {
	msg.ID = int64(len(r.created) + 1)
	r.created = append(r.created, msg)
	return nil
}

func (r *stubChatMessageRepository) GetByConversationID(conversationID int64) ([]model.ChatMessage, error) {
	return r.items, nil
}

func (r *stubChatMessageRepository) DeleteByConversationID(conversationID int64) error { return nil }

type stubConversationRunRepository struct {
	saved  *model.ConversationRun
	active *model.ConversationRun
}

type stubConversationRunStepRepository struct {
	steps []model.ConversationRunStep
}

type stubConversationMemoryRepository struct {
	memories []model.ConversationMemory
}

func (r *stubConversationMemoryRepository) Upsert(memory *model.ConversationMemory) error {
	if memory == nil {
		return nil
	}
	clone := *memory
	if clone.ID == 0 {
		clone.ID = int64(len(r.memories) + 1)
	}
	memory.ID = clone.ID
	for i := range r.memories {
		if r.memories[i].ConversationID == clone.ConversationID && r.memories[i].Scope == clone.Scope {
			r.memories[i] = clone
			return nil
		}
	}
	r.memories = append(r.memories, clone)
	return nil
}

func (r *stubConversationMemoryRepository) GetByConversationID(conversationID int64) ([]model.ConversationMemory, error) {
	filtered := make([]model.ConversationMemory, 0, len(r.memories))
	for _, memory := range r.memories {
		if memory.ConversationID == conversationID {
			filtered = append(filtered, memory)
		}
	}
	return filtered, nil
}

func (r *stubConversationMemoryRepository) GetByConversationIDAndScope(conversationID int64, scope string) (*model.ConversationMemory, error) {
	for _, memory := range r.memories {
		if memory.ConversationID == conversationID && memory.Scope == scope {
			clone := memory
			return &clone, nil
		}
	}
	return nil, errors.New("memory not found")
}

func (r *stubConversationMemoryRepository) DeleteByConversationID(conversationID int64) error {
	filtered := r.memories[:0]
	for _, memory := range r.memories {
		if memory.ConversationID != conversationID {
			filtered = append(filtered, memory)
		}
	}
	r.memories = filtered
	return nil
}

func (r *stubConversationRunStepRepository) Create(step *model.ConversationRunStep) error {
	clone := *step
	if clone.ID == 0 {
		clone.ID = int64(len(r.steps) + 1)
	}
	step.ID = clone.ID
	r.steps = append(r.steps, clone)
	return nil
}

func (r *stubConversationRunStepRepository) Update(step *model.ConversationRunStep) error {
	for i := range r.steps {
		if r.steps[i].ID == step.ID {
			r.steps[i] = *step
			return nil
		}
	}
	r.steps = append(r.steps, *step)
	return nil
}

func (r *stubConversationRunStepRepository) GetByConversationID(conversationID int64) ([]model.ConversationRunStep, error) {
	filtered := make([]model.ConversationRunStep, 0, len(r.steps))
	for _, step := range r.steps {
		if step.ConversationID == conversationID {
			filtered = append(filtered, step)
		}
	}
	return filtered, nil
}

func (r *stubConversationRunStepRepository) GetByRunID(runID int64) ([]model.ConversationRunStep, error) {
	filtered := make([]model.ConversationRunStep, 0, len(r.steps))
	for _, step := range r.steps {
		if step.RunID == runID {
			filtered = append(filtered, step)
		}
	}
	return filtered, nil
}

func (r *stubConversationRunStepRepository) DeleteByConversationID(conversationID int64) error {
	filtered := r.steps[:0]
	for _, step := range r.steps {
		if step.ConversationID != conversationID {
			filtered = append(filtered, step)
		}
	}
	r.steps = filtered
	return nil
}

func (r *stubConversationRunRepository) Create(run *model.ConversationRun) error {
	clone := *run
	if clone.ID == 0 {
		clone.ID = 1
	}
	run.ID = clone.ID
	r.saved = &clone
	r.active = &clone
	return nil
}

func (r *stubConversationRunRepository) GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error) {
	return r.active, nil
}

func (r *stubConversationRunRepository) Update(run *model.ConversationRun) error {
	clone := *run
	r.saved = &clone
	r.active = &clone
	return nil
}

func (r *stubConversationRunRepository) DeleteByConversationID(conversationID int64) error {
	if r.active != nil && r.active.ConversationID == conversationID {
		r.active = nil
	}
	if r.saved != nil && r.saved.ConversationID == conversationID {
		r.saved = nil
	}
	return nil
}

type stubLibrarian struct {
	result LibrarianResult
	err    error
}

func (l *stubLibrarian) Search(ctx context.Context, question string) (LibrarianResult, error) {
	return l.result, l.err
}

type stubJournalist struct {
	result *JournalistResult
	err    error
	called int
}

func (j *stubJournalist) Research(ctx context.Context, question string, local LibrarianResult) (*JournalistResult, error) {
	j.called++
	return j.result, j.err
}

type stubSynthesizer struct {
	answer  string
	sources []string
	err     error
}

func (s *stubSynthesizer) Compose(ctx context.Context, question string, local LibrarianResult, web *JournalistResult) (string, []string, error) {
	return s.answer, s.sources, s.err
}

type stubAILogger struct {
	entries []AILogEntry
}

func (l *stubAILogger) LogStage(entry AILogEntry) { l.entries = append(l.entries, entry) }
func (l *stubAILogger) LogError(entry AILogEntry) { l.entries = append(l.entries, entry) }
func (l *stubAILogger) Close() error              { return nil }

func ptrInt64(v int64) *int64 { return &v }

func createdMessageRoles(messages []*model.ChatMessage) []string {
	roles := make([]string, 0, len(messages))
	for _, msg := range messages {
		if msg != nil {
			roles = append(roles, msg.Role)
		}
	}
	return roles
}

func TestThinkTankService_UsesLibrarianAndSynthesizerWhenLocalKnowledgeIsEnough(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "站内文章总结", Sources: []SourceRef{{Kind: "article", ID: 2, Title: "文章标题", URL: "/article/article-title"}}}}
	synthesizer := &stubSynthesizer{answer: "这是基于站内知识的最终回答", sources: []string{"文章标题"}}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 12, UserID: 9, Title: "新会话"}}
	msgRepo := &stubChatMessageRepository{}
	runRepo := &stubConversationRunRepository{}
	svc := NewThinkTankService(librarian, nil, synthesizer, runRepo, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, convRepo, msgRepo, nil, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "站内文章讲了什么", ptrInt64(12), ptrInt64(9))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "这是基于站内知识的最终回答" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
	if resp.Stage != "completed" {
		t.Fatalf("expected completed stage, got %q", resp.Stage)
	}
	if len(msgRepo.created) != 2 {
		t.Fatalf("expected persisted user and assistant messages, got %d", len(msgRepo.created))
	}
	if runRepo.saved == nil || runRepo.saved.Status != "completed" {
		t.Fatalf("expected completed run to be persisted, got %#v", runRepo.saved)
	}
}

func TestThinkTankServiceChat_PersistsConversationStateAfterADKAnswer(t *testing.T) {
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 41, UserID: 8, Title: "新会话"}}
	msgRepo := &stubChatMessageRepository{items: []model.ChatMessage{
		{ID: 1, ConversationID: 41, Role: "user", Content: "我们要做一个 API 网关。"},
		{ID: 2, ConversationID: 41, Role: "assistant", Content: "可以先梳理缓存、限流和监控方案。"},
		{ID: 3, ConversationID: 41, Role: "user", Content: "限流状态我倾向放在 Redis。"},
		{ID: 4, ConversationID: 41, Role: "assistant", Content: "Redis 适合做共享计数器和短期状态。"},
		{ID: 5, ConversationID: 41, Role: "user", Content: "另外还要考虑降级和熔断。"},
		{ID: 6, ConversationID: 41, Role: "assistant", Content: "可以在网关层统一做降级与熔断策略。"},
	}}
	runRepo := &stubConversationRunRepository{}
	memoryRepo := &stubConversationMemoryRepository{}
	svc := NewThinkTankService(nil, nil, &stubSynthesizer{}, runRepo, &stubConversationRunStepRepository{}, memoryRepo, convRepo, msgRepo, nil, &stubAILogger{}).(*thinkTankService)
	svc.adkRunner = &thinkTankADKRunner{}
	svc.adkAnswerFetcher = func(ctx context.Context, question string) (string, error) {
		if !strings.Contains(question, "Redis 适合做共享计数器和短期状态") {
			t.Fatalf("expected ADK query to include recent conversation memory, got %q", question)
		}
		if !strings.Contains(question, "帮我总结一下 Redis 在这个网关方案里的作用") {
			t.Fatalf("expected ADK query to include latest user question, got %q", question)
		}
		return "ADK 最终答案：Redis 适合承载限流计数、热点缓存和短时共享状态。", nil
	}

	resp, err := svc.Chat(context.Background(), "帮我总结一下 Redis 在这个网关方案里的作用", ptrInt64(41), ptrInt64(8))
	if err != nil {
		t.Fatalf("expected chat success, got %v", err)
	}
	if strings.TrimSpace(resp.Message) == "" {
		t.Fatal("expected non-empty assistant message")
	}
	if resp.Stage != "completed" {
		t.Fatalf("expected completed stage, got %q", resp.Stage)
	}
	if got := createdMessageRoles(msgRepo.created); len(got) != 2 || got[0] != "user" || got[1] != "assistant" {
		t.Fatalf("expected persisted user/assistant messages, got %#v", got)
	}
	if convRepo.updated == nil {
		t.Fatal("expected conversation metadata update")
	}
	if convRepo.updated.Title == "" || convRepo.updated.Title == "新会话" {
		t.Fatalf("expected conversation title to be refreshed, got %#v", convRepo.updated)
	}
	if runRepo.saved == nil || runRepo.saved.Status != "completed" || runRepo.saved.CurrentStage != "completed" {
		t.Fatalf("expected completed run to be persisted, got %#v", runRepo.saved)
	}
	if !strings.Contains(runRepo.saved.PendingContext, "Redis 适合承载限流计数") {
		t.Fatalf("expected completed answer snapshot in run pending context, got %#v", runRepo.saved)
	}
	if len(memoryRepo.memories) != 1 {
		t.Fatalf("expected a summarized conversation memory to be persisted, got %#v", memoryRepo.memories)
	}
	if memoryRepo.memories[0].Scope != ConversationMemoryScopeSummary {
		t.Fatalf("expected summary memory scope, got %#v", memoryRepo.memories[0])
	}
	if !strings.Contains(memoryRepo.memories[0].Content, "API 网关") {
		t.Fatalf("expected persisted summary memory to retain older context, got %#v", memoryRepo.memories[0])
	}
}

func TestThinkTankService_AppendsArticleReferenceMarkdown(t *testing.T) {
	synthesizer := &thinkTankSynthesizer{llm: nil}
	answer, _, err := synthesizer.Compose(context.Background(), "站内文章讲了什么", LibrarianResult{
		Summary: "已有回答",
		Sources: []SourceRef{{Kind: "article", Title: "深入 Go 并发", URL: "/article/go-concurrency"}},
	}, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(answer, "参考博主文章") {
		t.Fatalf("expected reference article section, got %q", answer)
	}
	if !strings.Contains(answer, "[深入 Go 并发](/article/go-concurrency)") {
		t.Fatalf("expected markdown link to article, got %q", answer)
	}
}

func TestThinkTankService_AppendsGroupedBlogAndExternalReferences(t *testing.T) {
	synthesizer := &thinkTankSynthesizer{llm: nil}
	answer, _, err := synthesizer.Compose(context.Background(), "我要学习 k8s", LibrarianResult{
		Summary: "站内回答",
		Sources: []SourceRef{{Kind: "article", Title: "k8s学习笔记", URL: "/article/d4735e3a26"}},
	}, &JournalistResult{
		Summary: "外部补充",
		Sources: []SourceRef{{Kind: "web", Title: "Kubernetes 官方教程", URL: "https://kubernetes.io/zh-cn/docs/tutorials/kubernetes-basics/"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(answer, "参考博主文章") {
		t.Fatalf("expected blog reference section, got %q", answer)
	}
	if !strings.Contains(answer, "- [k8s学习笔记](/article/d4735e3a26)") {
		t.Fatalf("expected blog article link, got %q", answer)
	}
	if !strings.Contains(answer, "参考外部文章") {
		t.Fatalf("expected external reference section, got %q", answer)
	}
	if !strings.Contains(answer, "- [Kubernetes 官方教程](https://kubernetes.io/zh-cn/docs/tutorials/kubernetes-basics/)") {
		t.Fatalf("expected external article link, got %q", answer)
	}
}

func TestAppendArticleReferences_DoesNotDuplicateExistingReference(t *testing.T) {
	answer := "已有回答\n\n参考文章\n- [深入 Go 并发](/article/go-concurrency)"
	got := appendArticleReferences(answer, []SourceRef{{Kind: "article", Title: "深入 Go 并发", URL: "/article/go-concurrency"}})
	if strings.Count(got, "/article/go-concurrency") != 1 {
		t.Fatalf("expected existing article reference to stay unique, got %q", got)
	}
}

func TestExtractLocalSearchArticleSourcesFromToolResult(t *testing.T) {
	content := `{"coverage_status":"sufficient","summary":"摘要","sources":[{"Kind":"article","Title":"李小龙的功夫哲学","URL":"/article/lee-philosophy"},{"Kind":"web","Title":"外部来源","URL":"https://example.com"}]}`
	sources := extractLocalSearchArticleSources(content)
	if len(sources) != 1 {
		t.Fatalf("expected one article source, got %#v", sources)
	}
	if sources[0].Title != "李小龙的功夫哲学" || sources[0].URL != "/article/lee-philosophy" {
		t.Fatalf("unexpected article source %#v", sources[0])
	}
}

func TestExtractWebSearchSourcesFromToolResult(t *testing.T) {
	content := `{"organic":[{"title":"Kubernetes 官方教程","link":"https://kubernetes.io/zh-cn/docs/tutorials/kubernetes-basics/"},{"title":"无链接"}]}`
	sources := extractWebSearchSources(content)
	if len(sources) != 1 {
		t.Fatalf("expected one external source, got %d: %#v", len(sources), sources)
	}
	if sources[0].Kind != "web" {
		t.Fatalf("expected web source kind, got %q", sources[0].Kind)
	}
	if sources[0].Title != "Kubernetes 官方教程" || sources[0].URL != "https://kubernetes.io/zh-cn/docs/tutorials/kubernetes-basics/" {
		t.Fatalf("unexpected source: %#v", sources[0])
	}
}

func TestFormatJournalistStepDetail_IncludesSummaryAndExternalSources(t *testing.T) {
	detail := formatJournalistStepDetail(&JournalistResult{
		Summary: "外部调研发现：行业正在加速落地。",
		Sources: []SourceRef{
			{Kind: "web", Title: "行业报告", URL: "https://example.com/report"},
			{Kind: "web", Title: "新闻来源", URL: "https://example.com/news"},
		},
	})

	if !strings.Contains(detail, "外部调研发现：行业正在加速落地。") {
		t.Fatalf("expected research summary in detail, got %q", detail)
	}
	if !strings.Contains(detail, "行业报告") || !strings.Contains(detail, "https://example.com/report") {
		t.Fatalf("expected source title and URL in detail, got %q", detail)
	}
	if !strings.Contains(detail, "新闻来源") || !strings.Contains(detail, "https://example.com/news") {
		t.Fatalf("expected second source title and URL in detail, got %q", detail)
	}
}

func TestThinkTankADKRunner_DoesNotPreWrapSupervisorSubAgents(t *testing.T) {
	content, err := os.ReadFile("thinktank_adk.go")
	if err != nil {
		t.Fatalf("expected to read ADK runner source, got %v", err)
	}

	if strings.Contains(string(content), "AgentWithDeterministicTransferTo(ctx, &adk.DeterministicTransferConfig") {
		t.Fatalf("supervisor.New already wraps sub-agents with deterministic transfer; do not pre-wrap them in thinktank_adk.go")
	}
}

func TestPlanExecuteInstructions_ExposeClarificationAndAvoidSupervisorTransfer(t *testing.T) {
	if !strings.Contains(thinkTankPlannerInstruction, "complete plan") {
		t.Fatalf("planner instruction must request a complete plan")
	}
	if !strings.Contains(thinkTankPlannerInstruction, "ask_for_clarification") {
		t.Fatalf("planner instruction must route ambiguous requests to ask_for_clarification")
	}
	if !strings.Contains(thinkTankExecutorInstruction, "LocalSearch") || !strings.Contains(thinkTankExecutorInstruction, "WebSearch") {
		t.Fatalf("executor instruction must list direct tools")
	}
	if !strings.Contains(thinkTankExecutorInstruction, "Do not call transfer_to_agent") {
		t.Fatalf("executor instruction must forbid supervisor transfer calls")
	}
	if !strings.Contains(thinkTankReplannerInstruction, "RespondTool") || !strings.Contains(thinkTankReplannerInstruction, "PlanTool") {
		t.Fatalf("replanner instruction must choose between respond and replan")
	}
}

func TestAppendStepDetail_TruncatesLargeRawLogs(t *testing.T) {
	step := &model.ConversationRunStep{}
	appendStepDetail(step, strings.Repeat("x", maxStepDetailRunes+100))

	if got := len([]rune(step.Detail)); got > maxStepDetailRunes+80 {
		t.Fatalf("expected step detail to be bounded, got %d runes", got)
	}
	if !strings.Contains(step.Detail, "内容过长，已截断") {
		t.Fatalf("expected truncation note in detail, got %q", step.Detail)
	}
}

func TestThinkTankService_ChatStream_StreamsChunksInsteadOfSingleFinalPayload(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "第一段\n第二段", Sources: []SourceRef{{Kind: "article", Title: "文章标题", URL: "/article/article-title"}}}}
	synthesizer := &stubSynthesizer{answer: "第一段\n第二段\n\n参考文章\n- [文章标题](/article/article-title)", sources: []string{"文章标题"}}
	svc := NewThinkTankService(librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{})

	eventCh, errCh := svc.ChatStream(context.Background(), "站内文章讲了什么", nil, nil)
	var chunkMessages []string
	for event := range eventCh {
		if event.Type == StreamEventChunk {
			chunkMessages = append(chunkMessages, event.Message)
		}
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}
	if len(chunkMessages) < 2 {
		t.Fatalf("expected multiple streamed chunks, got %d with %#v", len(chunkMessages), chunkMessages)
	}
}

func TestThinkTankService_UsesJournalistWhenLocalKnowledgeIsInsufficient(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient"}}
	journalist := &stubJournalist{result: &JournalistResult{Summary: "联网调研结果", Sources: []SourceRef{{Kind: "web", Title: "外部来源", URL: "https://example.com"}}, KnowledgeDraftTitle: "调研草稿", KnowledgeDraftSummary: "草稿摘要", KnowledgeDraftBody: "草稿正文"}}
	synthesizer := &stubSynthesizer{answer: "整合后的最终回答", sources: []string{"外部来源"}}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 13, UserID: 10, Title: "研究会话"}}
	msgRepo := &stubChatMessageRepository{}
	runRepo := &stubConversationRunRepository{}
	knowledgeSvc := &stubKnowledgeDocumentService{}
	svc := NewThinkTankService(librarian, journalist, synthesizer, runRepo, &stubConversationRunStepRepository{}, &stubConversationMemoryRepository{}, convRepo, msgRepo, knowledgeSvc, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "调研工业大模型落地", ptrInt64(13), ptrInt64(10))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "整合后的最终回答" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
	if journalist.called == 0 {
		t.Fatalf("expected journalist to be called")
	}
	if knowledgeSvc.created == nil || knowledgeSvc.created.Title != "调研草稿" {
		t.Fatalf("expected knowledge draft to be created, got %#v", knowledgeSvc.created)
	}
}

type stubKnowledgeDocumentService struct {
	created *CreateKnowledgeDocumentInput
}

func (s *stubKnowledgeDocumentService) CreateResearchDraft(input CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error) {
	s.created = &input
	return &model.KnowledgeDocument{ID: 1, Title: input.Title}, nil
}
func (s *stubKnowledgeDocumentService) Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	return nil, nil
}
func (s *stubKnowledgeDocumentService) Reject(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	return nil, nil
}
func (s *stubKnowledgeDocumentService) GetByID(id int64) (*model.KnowledgeDocument, []*model.KnowledgeDocumentSource, error) {
	return nil, nil, nil
}
func (s *stubKnowledgeDocumentService) List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error) {
	return nil, 0, nil
}
func (s *stubKnowledgeDocumentService) Delete(id int64) error {
	return nil
}

func TestThinkTankService_ChatStream_PersistsSummarizedRunEventsWithoutChunkSnapshots(t *testing.T) {
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient", Summary: "站内资料不足"}}
	journalist := &stubJournalist{result: &JournalistResult{Summary: "外部调研摘要", Sources: []SourceRef{{Kind: "web", Title: "外部来源", URL: "https://example.com"}}}}
	synthesizer := &stubSynthesizer{answer: "整合后的最终回答", sources: []string{"外部来源"}}
	runRepo := &stubConversationRunRepository{}
	stepRepo := &stubConversationRunStepRepository{}
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 31, UserID: 11, Title: "研究会话"}}
	svc := NewThinkTankService(librarian, journalist, synthesizer, runRepo, stepRepo, &stubConversationMemoryRepository{}, convRepo, &stubChatMessageRepository{}, nil, &stubAILogger{})

	eventCh, errCh := svc.ChatStream(context.Background(), "调研工业大模型落地", ptrInt64(31), ptrInt64(11))
	for range eventCh {
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	if len(stepRepo.steps) == 0 {
		t.Fatalf("expected summarized run steps to be persisted")
	}

	var agents []string
	for _, step := range stepRepo.steps {
		agents = append(agents, step.AgentName)
		if step.RunID == 0 {
			t.Fatalf("expected persisted step to include run_id, got %#v", step)
		}
		if step.Detail == "" {
			t.Fatalf("expected persisted step detail, got %#v", step)
		}
	}

	assertContainsEventType(t, agents, "Librarian")
	assertContainsEventType(t, agents, "Journalist")
	assertContainsEventType(t, agents, "Synthesizer")
}

func assertContainsEventType(t *testing.T, types []string, want string) {
	t.Helper()
	for _, got := range types {
		if got == want {
			return
		}
	}
	t.Fatalf("expected event type %q in %#v", want, types)
}
