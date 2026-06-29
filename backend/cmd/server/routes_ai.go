package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
)

func registerAIRoutes(
	api *gin.RouterGroup,
	cfg *config.Config,
	rdb *redis.Client,
	aiHandler *handler.AIHandler,
	chatHandler *handler.ChatHandler,
) {
	ai := api.Group("/ai")
	ai.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.CSRFProtection())

	ai.POST("/chat", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "ai-chat",
		Type:    middleware.UserLimit,
		Limit:   cfg.RateLimit.AIChat,
		Window:  time.Minute,
		Message: rateLimitMessage("AI 对话请求过于频繁", cfg.RateLimit.AIChat, time.Minute),
	}), aiHandler.Chat)
	ai.POST("/chat/stream", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "ai-chat-stream",
		Type:    middleware.UserLimit,
		Limit:   cfg.RateLimit.AIChat,
		Window:  time.Minute,
		Message: rateLimitMessage("AI 流式对话请求过于频繁", cfg.RateLimit.AIChat, time.Minute),
	}), aiHandler.ChatStream)
	ai.POST("/chat/stream/resume", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "ai-chat-stream-resume",
		Type:    middleware.UserLimit,
		Limit:   cfg.RateLimit.AIChat,
		Window:  time.Minute,
		Message: rateLimitMessage("AI 流式恢复请求过于频繁", cfg.RateLimit.AIChat, time.Minute),
	}), aiHandler.ResumeChatStream)
	ai.POST("/summary", middleware.AdminRequired(cfg.JWT.Secret, rdb), aiHandler.GenerateSummary)
	ai.POST("/writing", middleware.AdminRequired(cfg.JWT.Secret, rdb), middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "ai-writing",
		Type:    middleware.UserLimit,
		Limit:   cfg.RateLimit.AIChat,
		Window:  time.Minute,
		Message: rateLimitMessage("AI 写作请求过于频繁", cfg.RateLimit.AIChat, time.Minute),
	}), aiHandler.GenerateWriting)

	conversations := api.Group("/chat/conversations")
	conversations.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.CSRFProtection())
	conversations.GET("", chatHandler.List)
	conversations.POST("", chatHandler.Create)
	conversations.GET("/:id", chatHandler.Get)
	conversations.PATCH("/:id", chatHandler.Update)
	conversations.DELETE("/:id", chatHandler.Delete)
	conversations.POST("/:id/share", chatHandler.Share)
	conversations.GET("/:id/export", chatHandler.Export)

	api.GET("/shared/conversations/:token", chatHandler.GetShared)
}
