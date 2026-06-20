package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_AllowsRequestWhenRedisClientIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimit(nil, RateLimitConfig{
		Type:   IPLimit,
		Limit:  1,
		Window: time.Minute,
	}))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected request to pass through without redis, got status %d", w.Code)
	}
}

func TestGenerateRateLimitKeyIncludesConfiguredName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)
	c.Request.RemoteAddr = "203.0.113.8:1234"

	key := generateRateLimitKey(c, RateLimitConfig{
		Name: "auth-register",
		Type: IPLimit,
	})

	if key != "ratelimit:auth-register:ip:203.0.113.8" {
		t.Fatalf("expected named IP key, got %q", key)
	}
}

func TestRateLimitExceededMessageUsesConfiguredDetail(t *testing.T) {
	message := rateLimitExceededMessage(RateLimitConfig{
		Message: "评论发布过于频繁：每分钟最多 5 条，请稍后再试",
	})

	if message != "评论发布过于频繁：每分钟最多 5 条，请稍后再试" {
		t.Fatalf("expected configured message, got %q", message)
	}
}

func TestRateLimitExceededMessageFallsBackToClearDefault(t *testing.T) {
	message := rateLimitExceededMessage(RateLimitConfig{})

	if message == "" || message == "Too many requests, please try again later" {
		t.Fatalf("expected localized non-empty fallback message, got %q", message)
	}
}
