package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	pkgjwt "wenDao/internal/pkg/jwt"
)

func TestAuthOptionalInjectsUserContextWhenTokenPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := pkgjwt.GenerateAccessToken(42, "user", "secret", 1)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer func() { _ = rdb.Close() }()

	router := gin.New()
	router.Use(AuthOptional("secret", rdb))
	router.GET("/ping", func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			t.Fatalf("expected user_id in context")
		}
		if userID.(int64) != 42 {
			t.Fatalf("expected user_id 42, got %v", userID)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestAuthOptionalAllowsAnonymousRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer func() { _ = rdb.Close() }()

	router := gin.New()
	router.Use(AuthOptional("secret", rdb))
	router.GET("/ping", func(c *gin.Context) {
		if _, exists := c.Get("user_id"); exists {
			t.Fatalf("did not expect user_id for anonymous request")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}
