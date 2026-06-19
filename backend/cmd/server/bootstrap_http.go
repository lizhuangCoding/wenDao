package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
	"wenDao/internal/pkg/response"
)

type appHandlers struct {
	user              *handler.UserHandler
	auth              *handler.AuthHandler
	category          *handler.CategoryHandler
	collection        *handler.CollectionHandler
	article           *handler.ArticleHandler
	comment           *handler.CommentHandler
	upload            *handler.UploadHandler
	ai                *handler.AIHandler
	site              *handler.SiteHandler
	stat              *handler.StatHandler
	chat              *handler.ChatHandler
	aiObservability   *handler.AIObservabilityHandler
	knowledgeDocument *handler.KnowledgeDocumentHandler
	notification      *handler.NotificationHandler
}

func initHandlers(cfg *config.Config, repos *repositories, services *appServices, rdb *redis.Client) *appHandlers {
	notificationHandler := handler.NewNotificationHandler(services.notification)
	notificationHandler.SetUserIDProvider(func() ([]int64, error) {
		return services.user.GetAllActiveUserIDs()
	})
	return &appHandlers{
		user:              handler.NewUserHandler(services.user, services.upload, services.oauth, services.verification, cfg),
		auth:              handler.NewAuthHandler(services.user, cfg, rdb),
		category:          handler.NewCategoryHandler(services.category),
		collection:        handler.NewCollectionHandler(services.collection),
		article:           handler.NewArticleHandler(services.article, services.stat, services.setting, services.collection),
		comment:           handler.NewCommentHandler(services.comment, services.stat),
		upload:            handler.NewUploadHandler(services.upload),
		ai:                handler.NewAIHandler(services.ai, cfg, repos.conversationRun),
		site:              handler.NewSiteHandler(cfg, services.article, services.setting),
		stat:              handler.NewStatHandler(services.stat),
		chat:              handler.NewChatHandler(cfg, repos.conversation, repos.chatMessage, repos.conversationRun, repos.conversationRunStep, repos.conversationMemory),
		aiObservability:   handler.NewAIObservabilityHandler(repos.conversationRun, repos.conversationRunStep),
		knowledgeDocument: handler.NewKnowledgeDocumentHandler(services.knowledgeDocument),
		notification:      notificationHandler,
	}
}

func buildRouter(cfg *config.Config, logger *zap.Logger, rdb *redis.Client, handlers *appHandlers) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.SecurityHeaders(),
		middleware.CORS(allowedCORSOrigins(cfg)...),
	)

	registerRoutes(
		router,
		cfg,
		rdb,
		handlers.user,
		handlers.auth,
		handlers.category,
		handlers.collection,
		handlers.article,
		handlers.comment,
		handlers.upload,
		handlers.ai,
		handlers.site,
		handlers.stat,
		handlers.chat,
		handlers.aiObservability,
		handlers.knowledgeDocument,
		handlers.notification,
	)

	return router
}

func allowedCORSOrigins(cfg *config.Config) []string {
	origins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}
	if cfg != nil && strings.TrimSpace(cfg.Site.URL) != "" {
		origins = append(origins, cfg.Site.URL)
	}
	return origins
}

func registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	rdb *redis.Client,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	collectionHandler *handler.CollectionHandler,
	articleHandler *handler.ArticleHandler,
	commentHandler *handler.CommentHandler,
	uploadHandler *handler.UploadHandler,
	aiHandler *handler.AIHandler,
	siteHandler *handler.SiteHandler,
	statHandler *handler.StatHandler,
	chatHandler *handler.ChatHandler,
	aiObservabilityHandler *handler.AIObservabilityHandler,
	knowledgeDocumentHandler *handler.KnowledgeDocumentHandler,
	notificationHandler *handler.NotificationHandler,
) {
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.Use(middleware.RateLimit(rdb, middleware.RateLimitConfig{
			Name:   "auth-global",
			Type:   middleware.IPLimit,
			Limit:  cfg.RateLimit.Global,
			Window: time.Second,
		}))
		{
			auth.POST("/register/code", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-register-code",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.VerificationCode,
				Window: time.Minute,
			}), userHandler.RequestRegisterCode)
			auth.POST("/register", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-register",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.Register,
				Window: time.Minute,
			}), userHandler.Register)
			auth.POST("/login", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-login",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.Login,
				Window: time.Minute,
			}), userHandler.Login)
			auth.POST("/password-reset/code", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-password-reset-code",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.PasswordReset,
				Window: time.Minute,
			}), userHandler.RequestPasswordResetCode)
			auth.POST("/password-reset/confirm", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-password-reset-confirm",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.PasswordReset,
				Window: time.Minute,
			}), userHandler.ConfirmPasswordReset)
			auth.GET("/github", userHandler.GitHubLogin)
			auth.POST("/refresh", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "auth-refresh",
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.Refresh,
				Window: time.Minute,
			}), authHandler.Refresh)
			auth.GET("/github/callback", userHandler.GitHubCallback)
		}

		api.GET("/articles", articleHandler.List)
		api.GET("/articles/orbit", articleHandler.ListOrbitArticles)
		api.GET("/articles/:id", articleHandler.GetByID)
		api.GET("/articles/slug/:slug", articleHandler.GetBySlug)
		api.GET("/categories", categoryHandler.List)
		api.GET("/collections", collectionHandler.List)
		api.GET("/categories/:id/articles", articleHandler.List)
		api.GET("/comments/article/:id", commentHandler.GetByArticleID)
		commentVotes := api.Group("")
		commentVotes.Use(middleware.AuthOptional(cfg.JWT.Secret, rdb))
		commentVotes.POST("/comments/:id/like", commentHandler.Like)
		commentVotes.DELETE("/comments/:id/like", commentHandler.Unlike)
		commentVotes.POST("/comments/:id/dislike", commentHandler.Dislike)
		commentVotes.DELETE("/comments/:id/dislike", commentHandler.Undislike)
		api.GET("/slogan", siteHandler.GetSlogan)
		api.GET("/contact-links", siteHandler.GetContactLinks)
		api.GET("/settings/sort-mode", articleHandler.GetSortMode)
		api.GET("/models", aiHandler.GetModels)

		authRequired := api.Group("")
		authRequired.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
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
			authRequired.POST("/comments", commentHandler.Create)
			authRequired.DELETE("/comments/:id", commentHandler.Delete)

			// 通知相关
			notifications := authRequired.Group("/notifications")
			{
				notifications.GET("", notificationHandler.List)
				notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
				notifications.PUT("/:id/read", notificationHandler.MarkRead)
				notifications.PUT("/read-all", notificationHandler.MarkAllRead)
			}
		}

		ai := api.Group("/ai")
		ai.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
			ai.POST("/chat", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "ai-chat",
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.Chat)
			ai.POST("/chat/stream", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "ai-chat-stream",
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.ChatStream)
			ai.POST("/chat/stream/resume", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "ai-chat-stream-resume",
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.ResumeChatStream)
			ai.POST("/summary", middleware.AdminRequired(cfg.JWT.Secret, rdb), aiHandler.GenerateSummary)
			ai.POST("/writing", middleware.AdminRequired(cfg.JWT.Secret, rdb), middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Name:   "ai-writing",
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.GenerateWriting)
		}

		conversations := api.Group("/chat/conversations")
		conversations.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
			conversations.GET("", chatHandler.List)
			conversations.POST("", chatHandler.Create)
			conversations.GET("/:id", chatHandler.Get)
			conversations.PATCH("/:id", chatHandler.Update)
			conversations.DELETE("/:id", chatHandler.Delete)
			conversations.POST("/:id/share", chatHandler.Share)
			conversations.GET("/:id/export", chatHandler.Export)
		}

		// 公开分享的对话
		api.GET("/shared/conversations/:token", chatHandler.GetShared)

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.AdminRequired(cfg.JWT.Secret, rdb))
		{
			// 用户管理
			users := admin.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.PUT("/:id/role", userHandler.UpdateUserRole)
				users.PUT("/:id/status", userHandler.UpdateUserStatus)
			}
			articles := admin.Group("/articles")
			{
				articles.GET("", articleHandler.AdminList)
				articles.POST("/batch-delete", articleHandler.BatchDelete)
				articles.GET("/:id", articleHandler.GetByID)
				articles.POST("", articleHandler.Create)
				articles.PUT("/:id", articleHandler.Update)
				articles.PUT("/:id/autosave", articleHandler.AutoSave)
				articles.DELETE("/:id", articleHandler.Delete)
				articles.PATCH("/:id/publish", articleHandler.Publish)
				articles.PATCH("/:id/draft", articleHandler.Draft)
				articles.PATCH("/:id/top", articleHandler.ToggleTop)
				articles.POST("/refresh-scores", articleHandler.UpdatePopularityScores)
			}
			categories := admin.Group("/categories")
			{
				categories.GET("", categoryHandler.AdminList)
				categories.POST("/batch-delete", categoryHandler.BatchDelete)
				categories.POST("", categoryHandler.Create)
				categories.PUT("/:id", categoryHandler.Update)
				categories.DELETE("/:id", categoryHandler.Delete)
			}
			collections := admin.Group("/collections")
			{
				collections.GET("", collectionHandler.AdminList)
				collections.POST("/batch-delete", collectionHandler.BatchDelete)
				collections.POST("", collectionHandler.Create)
				collections.PUT("/:id", collectionHandler.Update)
				collections.DELETE("/:id", collectionHandler.Delete)
			}
			comments := admin.Group("/comments")
			{
				comments.GET("", commentHandler.AdminList)
				comments.POST("/batch-delete", commentHandler.BatchDelete)
				comments.DELETE("/:id", commentHandler.Delete)
				comments.POST("/:id/restore", commentHandler.Restore)
			}
			knowledgeDocs := admin.Group("/knowledge-documents")
			{
				knowledgeDocs.GET("", knowledgeDocumentHandler.List)
				knowledgeDocs.POST("/batch-delete", knowledgeDocumentHandler.BatchDelete)
				knowledgeDocs.GET("/:id", knowledgeDocumentHandler.Get)
				knowledgeDocs.POST("/:id/approve", knowledgeDocumentHandler.Approve)
				knowledgeDocs.POST("/:id/reject", knowledgeDocumentHandler.Reject)
				knowledgeDocs.DELETE("/:id", knowledgeDocumentHandler.Delete)
			}
			admin.POST("/upload/image", uploadHandler.UploadImage)
			admin.GET("/stats/dashboard", statHandler.GetDashboardStats)
			admin.GET("/ai-observability/runs", aiObservabilityHandler.ListRuns)
			admin.POST("/ai-observability/runs/batch-delete", aiObservabilityHandler.BatchDeleteRuns)
			admin.PUT("/settings/sort-mode", articleHandler.SetSortMode)
			admin.PUT("/settings/slogan", siteHandler.SetSlogan)
			admin.PUT("/settings/contact-links", siteHandler.SetContactLinks)

			// 消息广播
			admin.POST("/notifications/broadcast", notificationHandler.Broadcast)
		}
	}

	router.Static("/uploads", cfg.Upload.StoragePath)
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})
	router.GET("/robots.txt", siteHandler.RobotsTxt)
	router.GET("/sitemap.xml", siteHandler.SitemapXml)
}
