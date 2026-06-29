package main

import (
	"time"

	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
)

func registerChatRoutes(
	access routeAccessGroups,
	cfg *config.Config,
	rdb *redis.Client,
	rateLimits routeRateLimitFactory,
	aiHandler *handler.AIHandler,
	chatHandler *handler.ChatHandler,
) {
	access.public.GET("/models", aiHandler.GetModels)
	access.public.GET("/shared/conversations/:token", chatHandler.GetShared)

	ai := access.authenticated.Group("/ai")
	ai.POST("/chat", middleware.RateLimit(rdb, rateLimits.user("ai-chat", cfg.RateLimit.AIChat, time.Minute, "AI 对话请求过于频繁")), aiHandler.Chat)
	ai.POST("/chat/stream", middleware.RateLimit(rdb, rateLimits.user("ai-chat-stream", cfg.RateLimit.AIChat, time.Minute, "AI 流式对话请求过于频繁")), aiHandler.ChatStream)
	ai.POST("/chat/stream/resume", middleware.RateLimit(rdb, rateLimits.user("ai-chat-stream-resume", cfg.RateLimit.AIChat, time.Minute, "AI 流式恢复请求过于频繁")), aiHandler.ResumeChatStream)

	adminAI := access.admin.Group("/ai")
	adminAI.POST("/summary", aiHandler.GenerateSummary)
	adminAI.POST("/writing", middleware.RateLimit(rdb, rateLimits.user("ai-writing", cfg.RateLimit.AIChat, time.Minute, "AI 写作请求过于频繁")), aiHandler.GenerateWriting)

	conversations := access.authenticated.Group("/chat/conversations")
	conversations.GET("", chatHandler.List)
	conversations.POST("", chatHandler.Create)
	conversations.GET("/:id", chatHandler.Get)
	conversations.PATCH("/:id", chatHandler.Update)
	conversations.DELETE("/:id", chatHandler.Delete)
	conversations.POST("/:id/share", chatHandler.Share)
	conversations.GET("/:id/export", chatHandler.Export)
}
