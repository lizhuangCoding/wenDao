package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
)

type stubConversationRepo struct {
	conversation *model.Conversation
}

func (r *stubConversationRepo) Create(conv *model.Conversation) error { return nil }
func (r *stubConversationRepo) GetByID(id int64) (*model.Conversation, error) {
	return r.conversation, nil
}
func (r *stubConversationRepo) GetByUserID(userID int64) ([]model.Conversation, error) {
	if r.conversation == nil {
		return nil, nil
	}
	return []model.Conversation{*r.conversation}, nil
}
func (r *stubConversationRepo) Update(conv *model.Conversation) error { return nil }
func (r *stubConversationRepo) Delete(id int64) error                 { return nil }

type stubChatMessageRepo struct {
	messages              []model.ChatMessage
	deletedConversationID int64
}

func (r *stubChatMessageRepo) Create(msg *model.ChatMessage) error { return nil }
func (r *stubChatMessageRepo) GetByConversationID(conversationID int64) ([]model.ChatMessage, error) {
	return r.messages, nil
}
func (r *stubChatMessageRepo) DeleteByConversationID(conversationID int64) error {
	r.deletedConversationID = conversationID
	return nil
}

type stubConversationRunRepo struct {
	active                *model.ConversationRun
	deletedConversationID int64
}

func (r *stubConversationRunRepo) Create(run *model.ConversationRun) error { return nil }
func (r *stubConversationRunRepo) GetByID(id int64) (*model.ConversationRun, error) {
	if r.active != nil && r.active.ID == id {
		return r.active, nil
	}
	return nil, nil
}
func (r *stubConversationRunRepo) GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error) {
	return r.active, nil
}
func (r *stubConversationRunRepo) Update(run *model.ConversationRun) error { return nil }
func (r *stubConversationRunRepo) DeleteByConversationID(conversationID int64) error {
	r.deletedConversationID = conversationID
	return nil
}

type stubConversationRunStepRepo struct {
	steps                 []model.ConversationRunStep
	deletedConversationID int64
}

func (r *stubConversationRunStepRepo) Create(step *model.ConversationRunStep) error { return nil }
func (r *stubConversationRunStepRepo) Update(step *model.ConversationRunStep) error { return nil }
func (r *stubConversationRunStepRepo) GetByConversationID(conversationID int64) ([]model.ConversationRunStep, error) {
	return r.steps, nil
}
func (r *stubConversationRunStepRepo) GetByRunID(runID int64) ([]model.ConversationRunStep, error) {
	filtered := make([]model.ConversationRunStep, 0, len(r.steps))
	for _, step := range r.steps {
		if step.RunID == runID {
			filtered = append(filtered, step)
		}
	}
	return filtered, nil
}
func (r *stubConversationRunStepRepo) DeleteByConversationID(conversationID int64) error {
	r.deletedConversationID = conversationID
	return nil
}

type stubConversationMemoryRepo struct {
	deletedConversationID int64
}

func (r *stubConversationMemoryRepo) Upsert(memory *model.ConversationMemory) error { return nil }
func (r *stubConversationMemoryRepo) GetByConversationID(conversationID int64) ([]model.ConversationMemory, error) {
	return nil, nil
}
func (r *stubConversationMemoryRepo) GetByConversationIDAndScope(conversationID int64, scope string) (*model.ConversationMemory, error) {
	return nil, nil
}
func (r *stubConversationMemoryRepo) DeleteByConversationID(conversationID int64) error {
	r.deletedConversationID = conversationID
	return nil
}

func TestChatHandler_GetIncludesRunStepsForHistoricalReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "研究会话",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{{
			ID:             1,
			ConversationID: 21,
			Role:           "assistant",
			Content:        "最终回答",
			CreatedAt:      now,
		}}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{
			{ID: 1, ConversationID: 21, RunID: 3, AgentName: "Librarian", Type: "thinking", Summary: "正在检索站内知识", Detail: "站内摘要", Status: "completed", CreatedAt: now},
			{ID: 2, ConversationID: 21, RunID: 3, AgentName: "Journalist", Type: "research", Summary: "正在进行外部调研", Detail: "外部结果", Status: "completed", CreatedAt: now},
		}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected response data object, got %#v", body["data"])
	}

	runSteps, ok := data["steps"].([]any)
	if !ok {
		t.Fatalf("expected steps array in response, got %#v", data["steps"])
	}
	if len(runSteps) != 2 {
		t.Fatalf("expected 2 run steps, got %d", len(runSteps))
	}
}

func TestChatHandler_GetIncludesMessageRunIDForStepAssociation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	runID := int64(42)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "多轮会话",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{{
			ID:             1,
			ConversationID: 21,
			RunID:          &runID,
			Role:           "assistant",
			Content:        "第一轮回答",
			CreatedAt:      now,
		}}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{{
			ID:             1,
			ConversationID: 21,
			RunID:          runID,
			AgentName:      "planner",
			Type:           "thinking",
			Summary:        "生成任务计划",
			Detail:         "计划详情",
			Status:         "completed",
			CreatedAt:      now,
		}}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	messages := data["messages"].([]any)
	message := messages[0].(map[string]any)
	if got := int64(message["run_id"].(float64)); got != runID {
		t.Fatalf("expected message run_id %d, got %d", runID, got)
	}
	processSteps, ok := message["process_steps"].([]any)
	if !ok {
		t.Fatalf("expected message process_steps array, got %#v", message["process_steps"])
	}
	if len(processSteps) != 1 {
		t.Fatalf("expected 1 message process step, got %d", len(processSteps))
	}
}

func TestChatHandler_GetAttachesEachRunStepsToMatchingAssistantMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	firstRunID := int64(101)
	secondRunID := int64(102)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "多轮会话",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{
			{ID: 1, ConversationID: 21, Role: "user", Content: "第一个问题", CreatedAt: now},
			{ID: 2, ConversationID: 21, RunID: &firstRunID, Role: "assistant", Content: "第一轮回答", CreatedAt: now.Add(1 * time.Second)},
			{ID: 3, ConversationID: 21, Role: "user", Content: "第二个问题", CreatedAt: now.Add(2 * time.Second)},
			{ID: 4, ConversationID: 21, RunID: &secondRunID, Role: "assistant", Content: "第二轮回答", CreatedAt: now.Add(3 * time.Second)},
		}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{
			{ID: 1, ConversationID: 21, RunID: firstRunID, AgentName: "planner", Type: "thinking", Summary: "第一轮计划", Detail: "第一轮详情", Status: "completed", CreatedAt: now},
			{ID: 2, ConversationID: 21, RunID: secondRunID, AgentName: "executor", Type: "thinking", Summary: "第二轮执行", Detail: "第二轮详情", Status: "completed", CreatedAt: now.Add(2 * time.Second)},
		}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	messages := data["messages"].([]any)
	firstAssistant := messages[1].(map[string]any)
	secondAssistant := messages[3].(map[string]any)

	firstSteps := firstAssistant["process_steps"].([]any)
	secondSteps := secondAssistant["process_steps"].([]any)
	if got := firstSteps[0].(map[string]any)["summary"]; got != "第一轮计划" {
		t.Fatalf("expected first assistant to include first run step, got %#v", got)
	}
	if got := secondSteps[0].(map[string]any)["summary"]; got != "第二轮执行" {
		t.Fatalf("expected second assistant to include second run step, got %#v", got)
	}
}

func TestChatHandler_GetIncludesActiveRunSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	heartbeatAt := now.Add(15 * time.Second)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "恢复中的会话",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{{
			ID:             1,
			ConversationID: 21,
			Role:           "user",
			Content:        "帮我分析一下",
			CreatedAt:      now,
		}}},
		runRepo: &stubConversationRunRepo{active: &model.ConversationRun{
			ID:               9,
			ConversationID:   21,
			UserID:           7,
			Status:           "running",
			CurrentStage:     "web_research",
			OriginalQuestion: "帮我分析一下",
			LastPlan:         "plan",
			LastAnswer:       "这是当前快照",
			HeartbeatAt:      &heartbeatAt,
			CreatedAt:        now,
			UpdatedAt:        now,
		}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{
			{ID: 11, ConversationID: 21, RunID: 9, AgentName: "Journalist", Type: "research", Summary: "外部调研中", Detail: "正在抓取资料", Status: "running", CreatedAt: now},
			{ID: 12, ConversationID: 21, RunID: 8, AgentName: "Librarian", Type: "thinking", Summary: "旧 run", Detail: "旧步骤", Status: "completed", CreatedAt: now},
		}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	data := body["data"].(map[string]any)
	activeRun, ok := data["active_run"].(map[string]any)
	if !ok {
		t.Fatalf("expected active_run object, got %#v", data["active_run"])
	}
	if got := int64(activeRun["id"].(float64)); got != 9 {
		t.Fatalf("expected active run id 9, got %d", got)
	}
	if got := activeRun["last_answer"].(string); got != "这是当前快照" {
		t.Fatalf("expected last_answer snapshot, got %q", got)
	}
	if got := activeRun["can_resume"].(bool); !got {
		t.Fatalf("expected can_resume to be true")
	}

	activeSteps, ok := data["active_steps"].([]any)
	if !ok {
		t.Fatalf("expected active_steps array, got %#v", data["active_steps"])
	}
	if len(activeSteps) != 1 {
		t.Fatalf("expected only active run steps, got %d", len(activeSteps))
	}
}

func TestChatHandler_GetOmitsCompletedRunFromActiveRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "已完成会话",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{{
			ID:             1,
			ConversationID: 21,
			Role:           "assistant",
			Content:        "最终回答",
			CreatedAt:      now,
		}}},
		runRepo: &stubConversationRunRepo{active: &model.ConversationRun{
			ID:               10,
			ConversationID:   21,
			UserID:           7,
			Status:           "completed",
			CurrentStage:     "completed",
			OriginalQuestion: "总结一下",
			LastAnswer:       "最终回答",
			CreatedAt:        now,
			UpdatedAt:        now,
		}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{
			{ID: 11, ConversationID: 21, RunID: 10, AgentName: "Synthesizer", Type: "thinking", Summary: "已完成", Detail: "最终整合", Status: "completed", CreatedAt: now},
		}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	data := body["data"].(map[string]any)
	if activeRun, exists := data["active_run"]; exists && activeRun != nil {
		t.Fatalf("expected completed run to be omitted from active_run, got %#v", activeRun)
	}
	if activeSteps, exists := data["active_steps"]; exists {
		if array, ok := activeSteps.([]any); ok && len(array) > 0 {
			t.Fatalf("expected completed run active_steps to be omitted, got %#v", activeSteps)
		}
	}
}

func TestChatHandler_GetOmitsStaleRunningRunWhenAssistantAnswerExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	runStartedAt := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	answerAt := runStartedAt.Add(10 * time.Second)
	h := &ChatHandler{
		convRepo: &stubConversationRepo{conversation: &model.Conversation{
			ID:        21,
			UserID:    7,
			Title:     "旧运行态会话",
			CreatedAt: runStartedAt,
			UpdatedAt: answerAt,
		}},
		msgRepo: &stubChatMessageRepo{messages: []model.ChatMessage{
			{
				ID:             1,
				ConversationID: 21,
				Role:           "user",
				Content:        "总结一下",
				CreatedAt:      runStartedAt,
			},
			{
				ID:             2,
				ConversationID: 21,
				Role:           "assistant",
				Content:        "最终回答",
				CreatedAt:      answerAt,
			},
		}},
		runRepo: &stubConversationRunRepo{active: &model.ConversationRun{
			ID:               10,
			ConversationID:   21,
			UserID:           7,
			Status:           "running",
			CurrentStage:     "streaming",
			OriginalQuestion: "总结一下",
			LastAnswer:       "最终回答",
			CreatedAt:        runStartedAt,
			UpdatedAt:        answerAt,
		}},
		runStepRepo: &stubConversationRunStepRepo{steps: []model.ConversationRunStep{
			{ID: 11, ConversationID: 21, RunID: 10, AgentName: "Synthesizer", Type: "thinking", Summary: "已完成", Detail: "最终整合", Status: "completed", CreatedAt: answerAt},
		}},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/chat/conversations/21", strings.NewReader(""))
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	if activeRun, exists := data["active_run"]; exists && activeRun != nil {
		t.Fatalf("expected stale running run with persisted assistant answer to be omitted, got %#v", activeRun)
	}
}

func TestChatHandler_DeleteCleansConversationRelatedData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	convRepo := &stubConversationRepo{conversation: &model.Conversation{ID: 21, UserID: 7, Title: "研究会话"}}
	msgRepo := &stubChatMessageRepo{}
	runRepo := &stubConversationRunRepo{}
	stepRepo := &stubConversationRunStepRepo{}
	memoryRepo := &stubConversationMemoryRepo{}
	h := NewChatHandler(convRepo, msgRepo, runRepo, stepRepo, memoryRepo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/chat/conversations/21", nil)
	c.Params = gin.Params{{Key: "id", Value: "21"}}
	c.Set("user_id", int64(7))

	h.Delete(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if msgRepo.deletedConversationID != 21 {
		t.Fatalf("expected messages to be deleted for conversation 21, got %d", msgRepo.deletedConversationID)
	}
	if runRepo.deletedConversationID != 21 {
		t.Fatalf("expected runs to be deleted for conversation 21, got %d", runRepo.deletedConversationID)
	}
	if stepRepo.deletedConversationID != 21 {
		t.Fatalf("expected run steps to be deleted for conversation 21, got %d", stepRepo.deletedConversationID)
	}
	if memoryRepo.deletedConversationID != 21 {
		t.Fatalf("expected memories to be deleted for conversation 21, got %d", memoryRepo.deletedConversationID)
	}
}
