package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
	pkgjwt "wenDao/internal/pkg/jwt"
)

func TestBuildRouter_RegistersRequiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Server.Mode = gin.TestMode
	cfg.Upload.StoragePath = "uploads"
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	defer func() {
		_ = rdb.Close()
	}()

	router := buildRouter(
		cfg,
		zap.NewNop(),
		rdb,
		&appHandlers{
			user:              &handler.UserHandler{},
			auth:              &handler.AuthHandler{},
			category:          &handler.CategoryHandler{},
			collection:        &handler.CollectionHandler{},
			article:           &handler.ArticleHandler{},
			comment:           &handler.CommentHandler{},
			upload:            &handler.UploadHandler{},
			ai:                &handler.AIHandler{},
			site:              &handler.SiteHandler{},
			stat:              &handler.StatHandler{},
			chat:              &handler.ChatHandler{},
			knowledgeDocument: &handler.KnowledgeDocumentHandler{},
			aiObservability:   &handler.AIObservabilityHandler{},
		},
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	required := []string{
		"GET /api/articles",
		"GET /api/articles/orbit",
		"GET /api/articles/:id",
		"GET /api/articles/slug/:slug",
		"GET /api/articles/:id/interaction",
		"POST /api/articles/:id/like",
		"DELETE /api/articles/:id/like",
		"POST /api/articles/:id/favorite",
		"DELETE /api/articles/:id/favorite",
		"GET /api/categories/:id/articles",
		"GET /api/collections",
		"GET /api/comments/article/:id",
		"POST /api/auth/register/code",
		"POST /api/auth/password-reset/code",
		"POST /api/auth/password-reset/confirm",
		"POST /api/auth/refresh",
		"GET /api/auth/me",
		"GET /api/users/me/liked-articles",
		"GET /api/users/me/favorite-articles",
		"POST /api/users/me/avatar",
		"PUT /api/users/me/preferences",
		"POST /api/ai/chat",
		"POST /api/ai/chat/stream/resume",
		"GET /api/admin/articles/:id",
		"GET /api/admin/collections",
		"GET /api/admin/comments",
		"POST /api/admin/comments/:id/restore",
		"GET /api/admin/knowledge-documents",
		"GET /api/admin/knowledge-documents/:id",
		"POST /api/admin/knowledge-documents/:id/approve",
		"POST /api/admin/knowledge-documents/:id/reject",
		"GET /api/admin/ai-observability/runs",
		"POST /api/admin/ai-observability/runs/batch-delete",
		"GET /health",
	}

	for _, route := range required {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func TestRateLimitMessageIncludesActionLimitAndWindow(t *testing.T) {
	message := rateLimitMessage("评论发布过于频繁", 5, time.Minute)

	expected := "评论发布过于频繁：每分钟最多 5 次，请稍后再试"
	if message != expected {
		t.Fatalf("expected %q, got %q", expected, message)
	}
}

func TestAllowedCORSOriginsIncludesSiteURLAndLocalDev(t *testing.T) {
	origins := allowedCORSOrigins(&config.Config{Site: config.SiteConfig{URL: "https://example.com"}})

	required := []string{"http://localhost:3000", "http://127.0.0.1:3000", "https://example.com"}
	for _, expected := range required {
		found := false
		for _, origin := range origins {
			if origin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected origin %q in %#v", expected, origins)
		}
	}
}

func TestBootstrapHTTPDelegatesRouteRegistrationToFocusedHelpers(t *testing.T) {
	source, err := os.ReadFile("bootstrap_http.go")
	if err != nil {
		t.Fatalf("failed to read bootstrap_http.go: %v", err)
	}

	text := string(source)
	required := []string{
		"registerAuthRoutes(",
		"registerArticleRoutes(",
		"registerCommentRoutes(",
		"registerChatRoutes(",
		"registerAdminRoutes(",
		"registerUserSelfRoutes(",
		"registerSiteRoutes(",
	}
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("expected bootstrap_http.go to delegate via %q", token)
		}
	}
}

func TestRouteAccessGroupsEnforcePublicAuthAndAdminBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.JWT.Secret = "test-secret"
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	})
	defer func() {
		_ = rdb.Close()
	}()

	router := gin.New()
	api := router.Group("/api")
	access := newRouteAccessGroups(api, cfg, rdb)

	access.public.GET("/public", func(c *gin.Context) { c.Status(http.StatusOK) })
	access.optionalAuth.POST("/vote", func(c *gin.Context) { c.Status(http.StatusOK) })
	access.authenticated.POST("/profile", func(c *gin.Context) { c.Status(http.StatusOK) })
	access.admin.POST("/admin-only", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		name           string
		method         string
		target         string
		tokenRole      string
		expectedStatus int
	}{
		{name: "public route stays open", method: http.MethodGet, target: "/api/public", expectedStatus: http.StatusOK},
		{name: "optional auth route accepts anonymous requests", method: http.MethodPost, target: "/api/vote", expectedStatus: http.StatusOK},
		{name: "authenticated route blocks anonymous requests", method: http.MethodPost, target: "/api/profile", expectedStatus: http.StatusUnauthorized},
		{name: "admin route blocks non admin user", method: http.MethodPost, target: "/api/admin-only", tokenRole: "user", expectedStatus: http.StatusForbidden},
		{name: "admin route accepts admin user", method: http.MethodPost, target: "/api/admin-only", tokenRole: "admin", expectedStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			if tc.tokenRole != "" {
				token, err := pkgjwt.GenerateAccessToken(1, tc.tokenRole, cfg.JWT.Secret, 1)
				if err != nil {
					t.Fatalf("failed to generate token: %v", err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, resp.Code)
			}
		})
	}
}

func TestRouteRateLimitFactoryBuildsScopedConfigs(t *testing.T) {
	rateLimits := newRouteRateLimitFactory()

	ipConfig := rateLimits.ip("auth-login", 5, time.Minute, "登录尝试过于频繁")
	if ipConfig.Name != "auth-login" || ipConfig.Type != middleware.IPLimit || ipConfig.Limit != 5 || ipConfig.Window != time.Minute {
		t.Fatalf("unexpected ip rate limit config: %#v", ipConfig)
	}
	if ipConfig.Message != "登录尝试过于频繁：每分钟最多 5 次，请稍后再试" {
		t.Fatalf("unexpected ip rate limit message: %q", ipConfig.Message)
	}

	userConfig := rateLimits.user("comment-create", 3, time.Minute, "评论发布过于频繁")
	if userConfig.Name != "comment-create" || userConfig.Type != middleware.UserLimit || userConfig.Limit != 3 || userConfig.Window != time.Minute {
		t.Fatalf("unexpected user rate limit config: %#v", userConfig)
	}
	if userConfig.Message != "评论发布过于频繁：每分钟最多 3 次，请稍后再试" {
		t.Fatalf("unexpected user rate limit message: %q", userConfig.Message)
	}
}

func TestRouteFilesUseRateLimitFactoryAndDomainRegistrations(t *testing.T) {
	files := []string{
		"routes_auth.go",
		"routes_comment.go",
		"routes_chat.go",
	}

	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		text := string(source)
		if !strings.Contains(text, "rateLimits.") {
			t.Fatalf("expected %s to use route rate limit factory", file)
		}
		if strings.Contains(text, "middleware.RateLimitConfig{") {
			t.Fatalf("expected %s to avoid inline RateLimitConfig literals", file)
		}
	}
}
