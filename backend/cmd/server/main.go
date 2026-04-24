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
)

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

	// 初始化数据库
	db, err := database.InitMySQL(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}
	logger.Info("MySQL connected successfully")

	// 自动迁移
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database migrated successfully")

	// 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	logger.Info("Redis connected successfully")

	// 初始化 Redis Vector (Redis Stack)
	rdbVector := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisVector.Host, cfg.RedisVector.Port),
		Password: cfg.RedisVector.Password,
		DB:       cfg.RedisVector.DB,
		PoolSize: cfg.RedisVector.PoolSize,
	})
	logger.Info("Redis Vector connected successfully")

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	chatMessageRepo := repository.NewChatMessageRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	conversationRunRepo := repository.NewConversationRunRepository(db)
	conversationRunStepRepo := repository.NewConversationRunStepRepository(db)
	conversationMemoryRepo := repository.NewConversationMemoryRepository(db)
	knowledgeDocumentRepo := repository.NewKnowledgeDocumentRepository(db)
	knowledgeDocumentSourceRepo := repository.NewKnowledgeDocumentSourceRepository(db)
	uploadRepo := repository.NewUploadRepository(db)
	settingRepo := repository.NewSettingRepository(db)

	// 初始化 AI 核心组件
	embedder, err := eino.NewDoubaoEmbedder(&cfg.AI)
	if err != nil {
		logger.Fatal("Failed to create Doubao Embedder", zap.Error(err))
	}
	logger.Info("Doubao Embedder initialized successfully")

	llmClient, err := eino.NewDoubaoLLMClient(&cfg.AI)
	if err != nil {
		logger.Fatal("Failed to create Doubao LLM client", zap.Error(err))
	}
	logger.Info("Doubao LLM Client initialized successfully")

	// ---------------------------------------------------------
	// RAG 核心修复：强制使用 V4 版本索引，彻底物理隔离
	// ---------------------------------------------------------
	const CurrentIndexName = "idx_wendao_v4"

	vectorStore := eino.NewRedisVectorStore(rdbVector, CurrentIndexName, logger)

	// 动态探测模型维度
	logger.Info("Detecting embedding model dimension...")
	testVec, err := embedder.Embed("dimension test")
	if err != nil {
		logger.Fatal("Failed to detect model dimension", zap.Error(err))
	}
	actualDim := len(testVec)
	logger.Info("Model dimension detected", zap.Int("dimension", actualDim), zap.String("using_index", CurrentIndexName))

	// 初始化索引
	if err := vectorStore.InitIndex(CurrentIndexName, actualDim); err != nil {
		logger.Fatal("Failed to initialize vector index", zap.Error(err))
	}
	logger.Info("Redis Vector index initialized successfully")

	// 初始化 Service
	oauthService := service.NewOAuthService(cfg)
	userService := service.NewUserService(userRepo, oauthService, cfg, rdb)
	categoryService := service.NewCategoryService(categoryRepo)
	settingService := service.NewSettingService(settingRepo)

	// AI/RAG Services
	vectorService := service.NewVectorService(vectorStore, embedder, logger)
	if err := syncPublishedArticleVectors(articleRepo, vectorService, logger); err != nil {
		logger.Fatal("Failed to sync published article vectors", zap.Error(err))
	}
	knowledgeDocumentService := service.NewKnowledgeDocumentService(knowledgeDocumentRepo, knowledgeDocumentSourceRepo, vectorService, articleRepo, categoryRepo, logger)
	aiEventLogger, err := service.NewAILogger(filepath.Dir(cfg.Log.Output))
	if err != nil {
		logger.Fatal("Failed to initialize AI event logger", zap.Error(err))
	}
	defer aiEventLogger.Close()
	retriever := eino.NewRedisRetriever(vectorStore, embedder, cfg.AI.TopK)
	ragChain := eino.NewRAGChain(retriever, llmClient.GetModel(), cfg.AI.RAGMinScore, logger)
	librarian := service.NewLibrarianService(ragChain)
	journalist := service.NewJournalist(&cfg.AI)
	synthesizer := service.NewThinkTankSynthesizer(llmClient)
	memorySummarizer := service.NewConversationMemorySummarizer(llmClient)
	adkRunner, err := service.NewThinkTankADKRunner(context.Background(), llmClient, librarian, knowledgeDocumentService, service.ResearchConfig{
		Endpoint:       cfg.AI.ResearchEndpoint,
		APIKey:         cfg.AI.ResearchAPIKey,
		MaxResults:     cfg.AI.ResearchMaxResults,
		TimeoutSeconds: cfg.AI.ResearchTimeoutSeconds,
	})
	if err != nil {
		logger.Fatal("Failed to initialize ThinkTank ADK runner", zap.Error(err))
	}
	thinkTankService := service.NewThinkTankService(librarian, journalist, synthesizer, conversationRunRepo, conversationRunStepRepo, conversationMemoryRepo, conversationRepo, chatMessageRepo, knowledgeDocumentService, aiEventLogger, adkRunner, memorySummarizer)
	aiService := service.NewAIService(llmClient, thinkTankService, logger)

	// ArticleService
	articleService := service.NewArticleService(articleRepo, categoryRepo, rdb, vectorService, logger)
	commentService := service.NewCommentService(commentRepo, articleRepo)
	uploadService := service.NewUploadService(uploadRepo, cfg)

	// StatService
	statRepo := repository.NewStatRepository(db)
	statService := service.NewStatService(statRepo, rdb)

	// 初始化 Handler
	userHandler := handler.NewUserHandler(userService, uploadService, oauthService, cfg)
	authHandler := handler.NewAuthHandler(userService, cfg, rdb)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	articleHandler := handler.NewArticleHandler(articleService, statService, settingService)
	commentHandler := handler.NewCommentHandler(commentService, statService)
	uploadHandler := handler.NewUploadHandler(uploadService)
	aiHandler := handler.NewAIHandler(aiService)
	siteHandler := handler.NewSiteHandler(cfg)
	statHandler := handler.NewStatHandler(statService)
	chatHandler := handler.NewChatHandler(conversationRepo, chatMessageRepo, conversationRunStepRepo)
	knowledgeDocumentHandler := handler.NewKnowledgeDocumentHandler(knowledgeDocumentService)

	// 设置路由
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(middleware.Logger(logger), middleware.Recovery(logger))
	router.Use(middleware.CORS())

	// 注册路由
	registerRoutes(router, cfg, rdb, userHandler, authHandler, categoryHandler, articleHandler, commentHandler, uploadHandler, aiHandler, siteHandler, statHandler, chatHandler, knowledgeDocumentHandler)

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
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
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
			ai.POST("/chat", aiHandler.Chat)
			ai.POST("/chat/stream", aiHandler.ChatStream)
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
