package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
)

func TestRegisterRoutes_RegistersRequiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	cfg := &config.Config{}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	registerRoutes(
		router,
		cfg,
		rdb,
		&handler.UserHandler{},
		&handler.AuthHandler{},
		&handler.CategoryHandler{},
		&handler.ArticleHandler{},
		&handler.CommentHandler{},
		&handler.UploadHandler{},
		&handler.AIHandler{},
		&handler.SiteHandler{},
		&handler.StatHandler{},
		&handler.ChatHandler{},
		&handler.KnowledgeDocumentHandler{},
	)

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	required := []string{
		"GET /api/articles",
		"GET /api/articles/:id",
		"GET /api/articles/slug/:slug",
		"GET /api/categories/:id/articles",
		"GET /api/comments/article/:id",
		"POST /api/auth/refresh",
		"GET /api/auth/me",
		"POST /api/users/me/avatar",
		"GET /api/admin/articles/:id",
		"GET /api/admin/comments",
		"POST /api/admin/comments/:id/restore",
		"GET /api/admin/knowledge-documents",
		"GET /api/admin/knowledge-documents/:id",
		"POST /api/admin/knowledge-documents/:id/approve",
		"POST /api/admin/knowledge-documents/:id/reject",
	}

	for _, route := range required {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}
