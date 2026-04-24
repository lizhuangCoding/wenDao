package handler

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
	messages []model.ChatMessage
}

func (r *stubChatMessageRepo) Create(msg *model.ChatMessage) error { return nil }
func (r *stubChatMessageRepo) GetByConversationID(conversationID int64) ([]model.ChatMessage, error) {
	return r.messages, nil
}
func (r *stubChatMessageRepo) DeleteByConversationID(conversationID int64) error { return nil }

type stubConversationRunStepRepo struct {
	steps []model.ConversationRunStep
}

func (r *stubConversationRunStepRepo) Create(step *model.ConversationRunStep) error { return nil }
func (r *stubConversationRunStepRepo) Update(step *model.ConversationRunStep) error { return nil }
func (r *stubConversationRunStepRepo) GetByConversationID(conversationID int64) ([]model.ConversationRunStep, error) {
	return r.steps, nil
}
func (r *stubConversationRunStepRepo) GetByRunID(runID int64) ([]model.ConversationRunStep, error) {
	return r.steps, nil
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
