package main

import (
	"time"

	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
)

func registerCommentRoutes(
	access routeAccessGroups,
	cfg *config.Config,
	rdb *redis.Client,
	rateLimits routeRateLimitFactory,
	commentHandler *handler.CommentHandler,
) {
	access.public.GET("/comments/article/:id", commentHandler.GetByArticleID)

	access.optionalAuth.POST("/comments/:id/like", commentHandler.Like)
	access.optionalAuth.DELETE("/comments/:id/like", commentHandler.Unlike)
	access.optionalAuth.POST("/comments/:id/dislike", commentHandler.Dislike)
	access.optionalAuth.DELETE("/comments/:id/dislike", commentHandler.Undislike)

	access.authenticated.POST(
		"/comments",
		middleware.RateLimit(rdb, rateLimits.user("comment-create", cfg.RateLimit.CommentCreate, time.Minute, "评论发布过于频繁")),
		commentHandler.Create,
	)
	access.authenticated.DELETE("/comments/:id", commentHandler.Delete)
}
