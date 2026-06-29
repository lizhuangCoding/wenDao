package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
)

func registerPublicRoutes(
	api *gin.RouterGroup,
	cfg *config.Config,
	rdb *redis.Client,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
	collectionHandler *handler.CollectionHandler,
	articleHandler *handler.ArticleHandler,
	commentHandler *handler.CommentHandler,
	aiHandler *handler.AIHandler,
	siteHandler *handler.SiteHandler,
	notificationHandler *handler.NotificationHandler,
) {
	api.GET("/articles", articleHandler.List)
	api.GET("/articles/orbit", articleHandler.ListOrbitArticles)
	api.GET("/articles/:id", articleHandler.GetByID)
	api.GET("/articles/slug/:slug", articleHandler.GetBySlug)
	api.GET("/search/articles", articleHandler.Search)
	api.GET("/categories", categoryHandler.List)
	api.GET("/tags", tagHandler.List)
	api.GET("/collections", collectionHandler.List)
	api.GET("/categories/:id/articles", articleHandler.List)
	api.GET("/comments/article/:id", commentHandler.GetByArticleID)
	api.GET("/slogan", siteHandler.GetSlogan)
	api.GET("/contact-links", siteHandler.GetContactLinks)
	api.GET("/settings/sort-mode", articleHandler.GetSortMode)
	api.GET("/models", aiHandler.GetModels)

	commentVotes := api.Group("")
	commentVotes.Use(middleware.AuthOptional(cfg.JWT.Secret, rdb), middleware.CSRFProtection())
	commentVotes.POST("/comments/:id/like", commentHandler.Like)
	commentVotes.DELETE("/comments/:id/like", commentHandler.Unlike)
	commentVotes.POST("/comments/:id/dislike", commentHandler.Dislike)
	commentVotes.DELETE("/comments/:id/dislike", commentHandler.Undislike)

	authRequired := api.Group("")
	authRequired.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.CSRFProtection())
	authRequired.POST("/auth/logout", authHandler.Logout)
	authRequired.GET("/auth/me", authHandler.GetUserInfo)
	authRequired.POST("/users/me/avatar", userHandler.UploadAvatar)
	authRequired.PUT("/users/me/username", userHandler.UpdateUsername)
	authRequired.PUT("/users/me/preferences", userHandler.UpdatePreferences)
	authRequired.GET("/users/me/liked-articles", articleHandler.ListLikedArticles)
	authRequired.GET("/users/me/favorite-articles", articleHandler.ListFavoriteArticles)
	authRequired.GET("/articles/:id/interaction", articleHandler.GetInteraction)
	authRequired.POST("/articles/:id/like", articleHandler.Like)
	authRequired.DELETE("/articles/:id/like", articleHandler.Unlike)
	authRequired.POST("/articles/:id/favorite", articleHandler.Favorite)
	authRequired.DELETE("/articles/:id/favorite", articleHandler.Unfavorite)
	authRequired.POST("/comments", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "comment-create",
		Type:    middleware.UserLimit,
		Limit:   cfg.RateLimit.CommentCreate,
		Window:  time.Minute,
		Message: rateLimitMessage("评论发布过于频繁", cfg.RateLimit.CommentCreate, time.Minute),
	}), commentHandler.Create)
	authRequired.DELETE("/comments/:id", commentHandler.Delete)

	notifications := authRequired.Group("/notifications")
	notifications.GET("", notificationHandler.List)
	notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
	notifications.PUT("/:id/read", notificationHandler.MarkRead)
	notifications.PUT("/read-all", notificationHandler.MarkAllRead)
}
