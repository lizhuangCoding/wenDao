package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/handler"
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
		"GET /health",
	}

	for _, route := range required {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s to be registered", route)
		}
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
