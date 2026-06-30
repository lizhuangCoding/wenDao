package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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

func TestCheckRateLimit_UsesAtomicRedisScriptForFixedWindow(t *testing.T) {
	source, err := os.ReadFile("ratelimit.go")
	if err != nil {
		t.Fatalf("failed to read ratelimit.go: %v", err)
	}

	text := string(source)
	if !strings.Contains(text, "redis.NewScript") {
		t.Fatalf("expected rate limiter to use a Redis script for atomic fixed-window counting")
	}
	if strings.Contains(text, "rdb.Incr(ctx, key)") {
		t.Fatalf("expected checkRateLimit to avoid standalone INCR calls")
	}
	if strings.Contains(text, "rdb.Expire(ctx, key, window)") {
		t.Fatalf("expected checkRateLimit to avoid standalone EXPIRE calls")
	}
}

func TestCheckRateLimit_EnforcesFixedWindowWithRedisState(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer srv.Close()

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer func() { _ = rdb.Close() }()

	allowed, err := checkRateLimit(t.Context(), rdb, "ratelimit:test:key", 2, time.Minute)
	if err != nil {
		t.Fatalf("expected first request to succeed, got err=%v", err)
	}
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}

	allowed, err = checkRateLimit(t.Context(), rdb, "ratelimit:test:key", 2, time.Minute)
	if err != nil {
		t.Fatalf("expected second request to succeed, got err=%v", err)
	}
	if !allowed {
		t.Fatal("expected second request to be allowed")
	}

	allowed, err = checkRateLimit(t.Context(), rdb, "ratelimit:test:key", 2, time.Minute)
	if err != nil {
		t.Fatalf("expected third request to return a decision, got err=%v", err)
	}
	if allowed {
		t.Fatal("expected third request to be rate limited inside the same fixed window")
	}

	if ttl := srv.TTL("ratelimit:test:key"); ttl <= 0 {
		t.Fatalf("expected rate limit key to have a positive TTL, got %v", ttl)
	}
}
