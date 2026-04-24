package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/service"
)

type stubAIService struct {
	chatAnswer   string
	chatErr      error
	streamEvents []service.StreamEvent
	streamErrs   []error
	summary      string
	summaryErr   error
}

func (s *stubAIService) Chat(question string, conversationID *int64, userID *int64) (string, error) {
	return s.chatAnswer, s.chatErr
}

func (s *stubAIService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan service.StreamEvent, <-chan error) {
	eventCh := make(chan service.StreamEvent, len(s.streamEvents))
	errCh := make(chan error, len(s.streamErrs))
	for _, event := range s.streamEvents {
		eventCh <- event
	}
	for _, err := range s.streamErrs {
		errCh <- err
	}
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func (s *stubAIService) GenerateSummary(content string) (string, error) {
	return s.summary, s.summaryErr
}

func TestAIHandlerChatStream_EmitsStageAndQuestionEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAIHandler(&stubAIService{
		streamEvents: []service.StreamEvent{
			{Type: service.StreamEventStage, Stage: "analyzing", Label: "正在理解你的问题"},
			{Type: service.StreamEventQuestion, Stage: "clarifying", Message: "你更关注国内还是海外案例？"},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat/stream", strings.NewReader(`{"message":"帮我调研大模型"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ChatStream(c)

	body := w.Body.String()
	if !strings.Contains(body, "event: stage") {
		t.Fatalf("expected stage event, got %s", body)
	}
	if !strings.Contains(body, "event: question") {
		t.Fatalf("expected question event, got %s", body)
	}
	if !strings.Contains(body, "requires_user_input") {
		t.Fatalf("expected requires_user_input flag, got %s", body)
	}
}
