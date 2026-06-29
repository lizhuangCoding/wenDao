package main

import (
	"strconv"
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
	tag               *handler.TagHandler
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
	articleHandler := handler.NewArticleHandler(services.article, services.stat, services.setting, services.collection)
	articleHandler.SetTaskRunner(services.taskRunner)
	commentHandler := handler.NewCommentHandler(services.comment, services.stat)
	commentHandler.SetTaskRunner(services.taskRunner)
	return &appHandlers{
		user:              handler.NewUserHandler(services.user, services.upload, services.oauth, services.verification, cfg),
		auth:              handler.NewAuthHandler(services.user, cfg, rdb),
		category:          handler.NewCategoryHandler(services.category),
		tag:               handler.NewTagHandler(services.tag),
		collection:        handler.NewCollectionHandler(services.collection),
		article:           articleHandler,
		comment:           commentHandler,
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
		handlers.tag,
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

func rateLimitMessage(action string, limit int, window time.Duration) string {
	if limit <= 0 {
		return action + "，请稍后再试"
	}
	return action + "：" + rateLimitWindowMessage(limit, window) + "，请稍后再试"
}

func rateLimitWindowMessage(limit int, window time.Duration) string {
	switch window {
	case time.Second:
		return "每秒最多 " + strconv.Itoa(limit) + " 次"
	case time.Minute:
		return "每分钟最多 " + strconv.Itoa(limit) + " 次"
	default:
		return "当前时间窗口内最多 " + strconv.Itoa(limit) + " 次"
	}
}

func registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	rdb *redis.Client,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
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
	access := newRouteAccessGroups(api, cfg, rdb)
	rateLimits := newRouteRateLimitFactory()

	registerAuthRoutes(api, cfg, rdb, rateLimits, userHandler, authHandler)
	registerArticleRoutes(
		access,
		categoryHandler,
		tagHandler,
		collectionHandler,
		articleHandler,
	)
	registerCommentRoutes(
		access,
		cfg,
		rdb,
		rateLimits,
		commentHandler,
	)
	registerUserSelfRoutes(
		access,
		userHandler,
		authHandler,
		notificationHandler,
	)
	registerSiteRoutes(
		access,
		siteHandler,
		articleHandler,
	)
	registerChatRoutes(
		access,
		cfg,
		rdb,
		rateLimits,
		aiHandler,
		chatHandler,
	)
	registerAdminRoutes(
		access,
		userHandler,
		categoryHandler,
		tagHandler,
		collectionHandler,
		articleHandler,
		commentHandler,
		uploadHandler,
		siteHandler,
		statHandler,
		aiObservabilityHandler,
		knowledgeDocumentHandler,
		notificationHandler,
	)

	router.Static("/uploads", cfg.Upload.StoragePath)
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok"})
	})
	router.GET("/robots.txt", siteHandler.RobotsTxt)
	router.GET("/sitemap.xml", siteHandler.SitemapXml)
}
