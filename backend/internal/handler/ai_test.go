package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

func TestAIHandlerChat_ReturnsServiceUnavailableWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAIHandler(service.NewDisabledAIService("research backend unavailable"))
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

func TestAIHandlerGenerateSummary_ReturnsServiceUnavailableWhenAIDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewAIHandler(service.NewDisabledAIService("summary backend unavailable"))
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/summary", strings.NewReader(`{"content":"正文"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateSummary(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d with body %s", w.Code, w.Body.String())
	}
}
