package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

func newTestAIHandler(aiService service.AIService) *AIHandler {
	return NewAIHandler(aiService, &config.Config{})
}

type quotaRunRepo struct {
	usage repository.ConversationRunDailyUsage
}

func (r *quotaRunRepo) Create(run *model.ConversationRun) error          { return nil }
func (r *quotaRunRepo) GetByID(id int64) (*model.ConversationRun, error) { return nil, nil }
func (r *quotaRunRepo) GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error) {
	return nil, nil
}
func (r *quotaRunRepo) ListRecent(filter repository.ConversationRunFilter) ([]model.ConversationRun, int64, error) {
	return nil, 0, nil
}
func (r *quotaRunRepo) GetDailyUsageByUser(userID int64, day time.Time) (repository.ConversationRunDailyUsage, error) {
	return r.usage, nil
}
func (r *quotaRunRepo) Update(run *model.ConversationRun) error { return nil }
func (r *quotaRunRepo) DeleteBatch(ids []int64) error           { return nil }
func (r *quotaRunRepo) DeleteByConversationID(conversationID int64) error {
	return nil
}

func TestAIHandlerChat_ReturnsServiceUnavailableWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newTestAIHandler(service.NewDisabledAIService("research backend unavailable"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat", strings.NewReader(`{"message":"帮我总结一下"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d with body %s", w.Code, w.Body.String())
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected valid response body, got %v", err)
	}
	if resp.Code != response.CodeServiceUnavailable {
		t.Fatalf("expected service unavailable code, got %d", resp.Code)
	}
}

func TestAIHandlerChat_ReturnsTooManyRequestsWhenDailyRunLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAIHandler(&stubAIService{chatAnswer: "ok"}, &config.Config{
		AI: config.AIConfig{DailyRunLimit: 1, DailyTokenLimit: 10000},
	}, &quotaRunRepo{usage: repository.ConversationRunDailyUsage{RunCount: 1}})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", int64(42))
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat", strings.NewReader(`{"message":"帮我总结一下"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestAIHandlerGenerateSummary_ReturnsServiceUnavailableWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newTestAIHandler(service.NewDisabledAIService("summary backend unavailable"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/summary", strings.NewReader(`{"content":"正文"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateSummary(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestAIHandlerGenerateWriting_ReturnsServiceUnavailableWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newTestAIHandler(service.NewDisabledAIService("writing backend unavailable"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/writing", strings.NewReader(`{"action":"polish","content":"正文"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateWriting(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d with body %s", w.Code, w.Body.String())
	}
}

func TestAIHandlerGenerateWriting_ReturnsInvalidParamsForUnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newTestAIHandler(&stubAIService{writingErr: service.ErrUnsupportedWritingAction})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/writing", strings.NewReader(`{"action":"translate","content":"正文"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateWriting(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", w.Code, w.Body.String())
	}
}
