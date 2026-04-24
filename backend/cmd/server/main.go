package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
	"wenDao/internal/pkg/database"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"

	"gorm.io/gorm"
)

type infrastructure struct {
	db        *gorm.DB
	rdb       *redis.Client
	rdbVector *redis.Client
}

type repositories struct {
	user                    repository.UserRepository
	article                 repository.ArticleRepository
	category                repository.CategoryRepository
	comment                 repository.CommentRepository
	chatMessage             repository.ChatMessageRepository
	conversation            repository.ConversationRepository
	conversationRun         repository.ConversationRunRepository
	conversationRunStep     repository.ConversationRunStepRepository
	conversationMemory      repository.ConversationMemoryRepository
	knowledgeDocument       repository.KnowledgeDocumentRepository
	knowledgeDocumentSource repository.KnowledgeDocumentSourceRepository
	upload                  repository.UploadRepository
	setting                 repository.SettingRepository
	stat                    *repository.StatRepository
}

type aiComponents struct {
	embedder    eino.Embedder
	llmClient   eino.LLMClient
	vectorStore eino.RedisVectorStore
}

type appServices struct {
	oauth             service.OAuthService
	user              service.UserService
	category          service.CategoryService
	setting           service.SettingService
	vector            service.VectorService
	knowledgeDocument service.KnowledgeDocumentService
	ai                service.AIService
	article           service.ArticleService
	comment           service.CommentService
	upload            service.UploadService
	stat              *service.StatService
}

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

func main() {
	// 加载环境变量
	_ = loadServerEnv()

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger := initLogger(cfg.Log)
	defer logger.Sync()

	infra, err := initInfrastructure(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize infrastructure", zap.Error(err))
	}

	repos := initRepositories(infra.db)

	aiCore, err := initAIComponents(cfg, logger, infra.rdbVector)
	if err != nil {
		logger.Warn("AI components unavailable, continuing in degraded mode", zap.Error(err))
	}

	services, cleanup, err := initServices(cfg, logger, repos, infra, aiCore)
	if err != nil {
		logger.Fatal("Failed to initialize services", zap.Error(err))
	}
	defer cleanup()

	handlers := initHandlers(cfg, repos, services, infra.rdb)

	router := buildRouter(cfg, logger, infra.rdb, handlers)

	// 启动服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	logger.Info("Server starting", zap.String("addr", ":"+cfg.Server.Port), zap.String("mode", cfg.Server.Mode))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}

func initInfrastructure(cfg *config.Config, logger *zap.Logger) (*infrastructure, error) {
	db, err := database.InitMySQL(&cfg.Database)
	if err != nil {
		return nil, err
	}
	logger.Info("MySQL connected successfully")

	if err := database.AutoMigrate(db); err != nil {
		return nil, err
	}
	logger.Info("Database migrated successfully")

	rdb := newRedisClient(cfg.Redis)
	logger.Info("Redis connected successfully")

	rdbVector := newRedisClient(cfg.RedisVector)
	logger.Info("Redis Vector connected successfully")

	return &infrastructure{db: db, rdb: rdb, rdbVector: rdbVector}, nil
}

func newRedisClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}

func initRepositories(db *gorm.DB) *repositories {
	return &repositories{
		user:                    repository.NewUserRepository(db),
		article:                 repository.NewArticleRepository(db),
		category:                repository.NewCategoryRepository(db),
		comment:                 repository.NewCommentRepository(db),
		chatMessage:             repository.NewChatMessageRepository(db),
		conversation:            repository.NewConversationRepository(db),
		conversationRun:         repository.NewConversationRunRepository(db),
		conversationRunStep:     repository.NewConversationRunStepRepository(db),
		conversationMemory:      repository.NewConversationMemoryRepository(db),
		knowledgeDocument:       repository.NewKnowledgeDocumentRepository(db),
		knowledgeDocumentSource: repository.NewKnowledgeDocumentSourceRepository(db),
		upload:                  repository.NewUploadRepository(db),
		setting:                 repository.NewSettingRepository(db),
		stat:                    repository.NewStatRepository(db),
	}
}

func initAIComponents(cfg *config.Config, logger *zap.Logger, rdbVector *redis.Client) (*aiComponents, error) {
	embedder, err := eino.NewDoubaoEmbedder(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("create doubao embedder: %w", err)
	}
	logger.Info("Doubao Embedder initialized successfully")

	llmClient, err := eino.NewDoubaoLLMClient(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("create doubao llm client: %w", err)
	}
	logger.Info("Doubao LLM Client initialized successfully")

	const currentIndexName = "idx_wendao_v4"
	vectorStore := eino.NewRedisVectorStore(rdbVector, currentIndexName, logger)

	logger.Info("Detecting embedding model dimension...")
	testVec, err := embedder.Embed("dimension test")
	if err != nil {
		return nil, fmt.Errorf("detect embedding model dimension: %w", err)
	}
	actualDim := len(testVec)
	logger.Info("Model dimension detected", zap.Int("dimension", actualDim), zap.String("using_index", currentIndexName))

	if err := vectorStore.InitIndex(currentIndexName, actualDim); err != nil {
		return nil, fmt.Errorf("initialize vector index: %w", err)
	}
	logger.Info("Redis Vector index initialized successfully")

	return &aiComponents{
		embedder:    embedder,
		llmClient:   llmClient,
		vectorStore: vectorStore,
	}, nil
}

func initServices(cfg *config.Config, logger *zap.Logger, repos *repositories, infra *infrastructure, aiCore *aiComponents) (*appServices, func(), error) {
	oauthService := service.NewOAuthService(cfg)
	userService := service.NewUserService(repos.user, oauthService, cfg, infra.rdb)
	categoryService := service.NewCategoryService(repos.category)
	settingService := service.NewSettingService(repos.setting)
	knowledgeDocumentService := service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, nil, repos.article, repos.category, logger)
	aiService := service.NewDisabledAIService("AI initialization failed")
	cleanup := func() {}

	if aiCore != nil {
		vectorService := service.NewVectorService(aiCore.vectorStore, aiCore.embedder, logger)
		if err := syncPublishedArticleVectors(repos.article, vectorService, logger); err != nil {
			logger.Warn("Published article vector sync skipped, continuing in degraded mode", zap.Error(err))
		} else {
			knowledgeDocumentService = service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, vectorService, repos.article, repos.category, logger)

			aiEventLogger, err := service.NewAILogger(filepath.Dir(cfg.Log.Output))
			if err != nil {
				logger.Warn("AI event logger unavailable, continuing in degraded mode", zap.Error(err))
			} else {
				retriever := eino.NewRedisRetriever(aiCore.vectorStore, aiCore.embedder, cfg.AI.TopK)
				ragChain := eino.NewRAGChain(retriever, aiCore.llmClient.GetModel(), cfg.AI.RAGMinScore, logger)
				librarian := service.NewLibrarianService(ragChain)
				journalist := service.NewJournalist(&cfg.AI)
				synthesizer := service.NewThinkTankSynthesizer(aiCore.llmClient)
				memorySummarizer := service.NewConversationMemorySummarizer(aiCore.llmClient)
				adkRunner, err := service.NewThinkTankADKRunner(context.Background(), aiCore.llmClient, librarian, knowledgeDocumentService, service.ResearchConfig{
					Endpoint:       cfg.AI.ResearchEndpoint,
					APIKey:         cfg.AI.ResearchAPIKey,
					MaxResults:     cfg.AI.ResearchMaxResults,
					TimeoutSeconds: cfg.AI.ResearchTimeoutSeconds,
				})
				if err != nil {
					logger.Warn("ThinkTank runner unavailable, continuing in degraded mode", zap.Error(err))
					_ = aiEventLogger.Close()
				} else {
					thinkTankService := service.NewThinkTankService(
						librarian,
						journalist,
						synthesizer,
						repos.conversationRun,
						repos.conversationRunStep,
						repos.conversationMemory,
						repos.conversation,
						repos.chatMessage,
						knowledgeDocumentService,
						aiEventLogger,
						adkRunner,
						memorySummarizer,
					)
					aiService = service.NewAIService(aiCore.llmClient, thinkTankService, logger)
					cleanup = func() {
						_ = aiEventLogger.Close()
					}
				}
			}

			return &appServices{
				oauth:             oauthService,
				user:              userService,
				category:          categoryService,
				setting:           settingService,
				vector:            vectorService,
				knowledgeDocument: knowledgeDocumentService,
				ai:                aiService,
				article:           service.NewArticleService(repos.article, repos.category, infra.rdb, vectorService, logger),
				comment:           service.NewCommentService(repos.comment, repos.article),
				upload:            service.NewUploadService(repos.upload, cfg),
				stat:              service.NewStatService(repos.stat, infra.rdb),
			}, cleanup, nil
		}
	}

	return &appServices{
		oauth:             oauthService,
		user:              userService,
		category:          categoryService,
		setting:           settingService,
		vector:            nil,
		knowledgeDocument: knowledgeDocumentService,
		ai:                aiService,
		article:           service.NewArticleService(repos.article, repos.category, infra.rdb, nil, logger),
		comment:           service.NewCommentService(repos.comment, repos.article),
		upload:            service.NewUploadService(repos.upload, cfg),
		stat:              service.NewStatService(repos.stat, infra.rdb),
	}, cleanup, nil
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

func loadServerEnv() error {
	if err := godotenv.Load("config/.env"); err == nil {
		return nil
	}
	return godotenv.Load()
}

func syncPublishedArticleVectors(articleRepo repository.ArticleRepository, vectorService service.VectorService, logger *zap.Logger) error {
	if articleRepo == nil || vectorService == nil || logger == nil {
		return nil
	}

	const pageSize = 100
	page := 1
	totalSynced := 0

	for {
		articles, total, err := articleRepo.List(repository.ArticleFilter{
			Status:   "published",
			Page:     page,
			PageSize: pageSize,
		})
		if err != nil {
			return fmt.Errorf("failed to list published articles for vector sync: %w", err)
		}
		if len(articles) == 0 {
			break
		}

		logger.Info("Syncing published article vector batch",
			zap.Int("page", page),
			zap.Int("batch_size", len(articles)),
			zap.Int64("total", total))

		for _, article := range articles {
			if article == nil {
				continue
			}
			if err := vectorService.VectorizeArticle(article.ID, article.Title, article.Content, article.Slug); err != nil {
				return fmt.Errorf("failed to sync article %d vectors: %w", article.ID, err)
			}
			totalSynced++
		}

		if int64(page*pageSize) >= total {
			break
		}
		page++
	}

	logger.Info("Published article vector sync completed", zap.Int("article_count", totalSynced))
	return nil
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

func initLogger(cfg config.LogConfig) *zap.Logger {
	level := zap.InfoLevel
	_ = level.UnmarshalText([]byte(cfg.Level))

	// 设置通用编码配置（使用 ISO8601 时间格式，更易读）
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	if cfg.Format != "json" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var cores []zapcore.Core
	if cfg.Output == "stdout" || cfg.Output == "" {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	} else {
		// 生成以日期命名的文件名：log/2026-04-11.log
		dir := filepath.Dir(cfg.Output)
		if dir == "." {
			dir = "log" // 默认存储在 log 目录下
		}

		// 确保目录存在
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}

		todayFilename := time.Now().Format("2006-01-02") + ".log"
		fullPath := filepath.Join(dir, todayFilename)

		fileWriter := &lumberjack.Logger{
			Filename:   fullPath,
			MaxSize:    100, // megabytes
			MaxBackups: 7,
			MaxAge:     28, // days
			Compress:   true,
		}

		var encoder zapcore.Encoder
		if cfg.Format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			// 在文件中禁用颜色，但保留可读的时间格式
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level))
	}
	return zap.New(zapcore.NewTee(cores...), zap.AddCaller())
}
