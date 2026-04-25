package main

import (
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
	article           *handler.ArticleHandler
	comment           *handler.CommentHandler
	upload            *handler.UploadHandler
	ai                *handler.AIHandler
	site              *handler.SiteHandler
	stat              *handler.StatHandler
	chat              *handler.ChatHandler
	knowledgeDocument *handler.KnowledgeDocumentHandler
}

func initHandlers(cfg *config.Config, repos *repositories, services *appServices, rdb *redis.Client) *appHandlers {
	return &appHandlers{
		user:              handler.NewUserHandler(services.user, services.upload, services.oauth, cfg),
		auth:              handler.NewAuthHandler(services.user, cfg, rdb),
		category:          handler.NewCategoryHandler(services.category),
		article:           handler.NewArticleHandler(services.article, services.stat, services.setting),
		comment:           handler.NewCommentHandler(services.comment, services.stat),
		upload:            handler.NewUploadHandler(services.upload),
		ai:                handler.NewAIHandler(services.ai),
		site:              handler.NewSiteHandler(cfg),
		stat:              handler.NewStatHandler(services.stat),
		chat:              handler.NewChatHandler(repos.conversation, repos.chatMessage, repos.conversationRun, repos.conversationRunStep, repos.conversationMemory),
		knowledgeDocument: handler.NewKnowledgeDocumentHandler(services.knowledgeDocument),
	}
}

func buildRouter(cfg *config.Config, logger *zap.Logger, rdb *redis.Client, handlers *appHandlers) *gin.Engine {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(middleware.Logger(logger), middleware.Recovery(logger), middleware.CORS())

	registerRoutes(
		router,
		cfg,
		rdb,
		handlers.user,
		handlers.auth,
		handlers.category,
		handlers.article,
		handlers.comment,
		handlers.upload,
		handlers.ai,
		handlers.site,
		handlers.stat,
		handlers.chat,
		handlers.knowledgeDocument,
	)

	return router
}

func registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	rdb *redis.Client,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	articleHandler *handler.ArticleHandler,
	commentHandler *handler.CommentHandler,
	uploadHandler *handler.UploadHandler,
	aiHandler *handler.AIHandler,
	siteHandler *handler.SiteHandler,
	statHandler *handler.StatHandler,
	chatHandler *handler.ChatHandler,
	knowledgeDocumentHandler *handler.KnowledgeDocumentHandler,
) {
	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.Use(middleware.RateLimit(rdb, middleware.RateLimitConfig{
			Type:   middleware.IPLimit,
			Limit:  cfg.RateLimit.Global,
			Window: time.Second,
		}))
		{
			auth.POST("/register", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.Register,
				Window: time.Minute,
			}), userHandler.Register)
			auth.POST("/login", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Type:   middleware.IPLimit,
				Limit:  cfg.RateLimit.Login,
				Window: time.Minute,
			}), userHandler.Login)
			auth.GET("/github", userHandler.GitHubLogin)
			auth.POST("/refresh", authHandler.Refresh)
			auth.GET("/github/callback", userHandler.GitHubCallback)
		}

		api.GET("/articles", articleHandler.List)
		api.GET("/articles/:id", articleHandler.GetByID)
		api.GET("/articles/slug/:slug", articleHandler.GetBySlug)
		api.GET("/categories", categoryHandler.List)
		api.GET("/categories/:id/articles", articleHandler.List)
		api.GET("/comments/article/:id", commentHandler.GetByArticleID)
		api.GET("/slogan", siteHandler.GetSlogan)
		api.GET("/settings/sort-mode", articleHandler.GetSortMode)

		authRequired := api.Group("")
		authRequired.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
			authRequired.POST("/auth/logout", authHandler.Logout)
			authRequired.GET("/auth/me", authHandler.GetUserInfo)
			authRequired.POST("/users/me/avatar", userHandler.UploadAvatar)
			authRequired.PUT("/users/me/username", userHandler.UpdateUsername)
			authRequired.POST("/comments", commentHandler.Create)
			authRequired.DELETE("/comments/:id", commentHandler.Delete)
		}

		ai := api.Group("/ai")
		ai.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
			ai.POST("/chat", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.Chat)
			ai.POST("/chat/stream", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.ChatStream)
			ai.POST("/chat/stream/resume", middleware.RateLimit(rdb, middleware.RateLimitConfig{
				Type:   middleware.UserLimit,
				Limit:  cfg.RateLimit.AIChat,
				Window: time.Minute,
			}), aiHandler.ResumeChatStream)
			ai.POST("/summary", middleware.AdminRequired(cfg.JWT.Secret, rdb), aiHandler.GenerateSummary)
		}

		conversations := api.Group("/chat/conversations")
		conversations.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
		{
			conversations.GET("", chatHandler.List)
			conversations.POST("", chatHandler.Create)
			conversations.GET("/:id", chatHandler.Get)
			conversations.PATCH("/:id", chatHandler.Update)
			conversations.DELETE("/:id", chatHandler.Delete)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb), middleware.AdminRequired(cfg.JWT.Secret, rdb))
		{
			articles := admin.Group("/articles")
			{
				articles.GET("", articleHandler.AdminList)
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
				categories.POST("", categoryHandler.Create)
				categories.PUT("/:id", categoryHandler.Update)
				categories.DELETE("/:id", categoryHandler.Delete)
			}
			comments := admin.Group("/comments")
			{
				comments.GET("", commentHandler.AdminList)
				comments.DELETE("/:id", commentHandler.Delete)
				comments.POST("/:id/restore", commentHandler.Restore)
			}
			knowledgeDocs := admin.Group("/knowledge-documents")
			{
				knowledgeDocs.GET("", knowledgeDocumentHandler.List)
				knowledgeDocs.GET("/:id", knowledgeDocumentHandler.Get)
				knowledgeDocs.POST("/:id/approve", knowledgeDocumentHandler.Approve)
				knowledgeDocs.POST("/:id/reject", knowledgeDocumentHandler.Reject)
				knowledgeDocs.DELETE("/:id", knowledgeDocumentHandler.Delete)
			}
			admin.POST("/upload/image", uploadHandler.UploadImage)
			admin.GET("/stats/dashboard", statHandler.GetDashboardStats)
			admin.PUT("/settings/sort-mode", articleHandler.SetSortMode)
		}
	}

	router.Static("/uploads", cfg.Upload.StoragePath)
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})
}
