package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

type appServices struct {
	oauth             service.OAuthService
	verification      service.VerificationService
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
	notification      service.NotificationService
}

func initServices(cfg *config.Config, logger *zap.Logger, repos *repositories, infra *infrastructure, aiCore *aiComponents) (*appServices, func(), error) {
	oauthService := service.NewOAuthService(cfg)
	var rdb = infra.rdb
	logVerificationEmailConfig(cfg, logger)
	verificationService := service.NewVerificationService(cfg, rdb, nil)
	userService := service.NewUserService(repos.user, oauthService, cfg, rdb)
	commentReplyEmailSender := service.NewSMTPCommentReplyEmailSender(cfg.Email, cfg.Site.URL)
	notifSvc := service.NewNotificationService(repos.notification)
	categoryService := service.NewCategoryService(repos.category)
	settingService := service.NewSettingService(repos.setting)
	knowledgeDocumentService := service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, nil, repos.article, repos.category, logger)
	aiService := service.NewDisabledAIService("AI initialization failed")
	cleanup := func() {}

	if aiCore != nil && aiCore.llmClient != nil {
		var vectorService service.VectorService
		var librarian service.Librarian
		if aiCore.vectorStore != nil && aiCore.embedder != nil {
			vectorService = service.NewVectorService(aiCore.vectorStore, aiCore.embedder, logger)
			if err := syncPublishedArticleVectors(repos.article, vectorService, logger); err != nil {
				logger.Warn("Published article vector sync skipped, continuing in degraded mode", zap.Error(err))
			}
			retriever := eino.NewRedisRetriever(aiCore.vectorStore, aiCore.embedder, cfg.AI.TopK)
			ragChain := eino.NewRAGChain(retriever, aiCore.llmClient.GetModel(), cfg.AI.RAGMinScore, logger)
			librarian = service.NewLibrarianService(ragChain)
		} else {
			logger.Warn("Vector search unavailable, ThinkTank will continue without local RAG retrieval")
			librarian = service.NewLibrarianService(nil)
		}
		knowledgeDocumentService = service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, vectorService, repos.article, repos.category, logger)

		aiEventLogger, err := service.NewAILoggerWithRotation(aiLogDir(cfg.Log.Output), service.LogRotationConfig{
			MaxSizeMB:  cfg.Log.MaxSizeMB,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAgeDays: cfg.Log.MaxAgeDays,
			Compress:   cfg.Log.Compress,
		})
		if err != nil {
			logger.Warn("AI event logger unavailable, continuing without AI event logs", zap.Error(err))
			aiEventLogger = nil
		} else {
			cleanup = func() {
				_ = aiEventLogger.Close()
			}
		}

		journalist := service.NewJournalist(&cfg.AI)
		synthesizer := service.NewThinkTankSynthesizer(aiCore.llmClient)
		memorySummarizer := service.NewConversationMemorySummarizer(aiCore.llmClient)
		adkRunner, err := service.NewThinkTankADKRunner(context.Background(), aiCore.llmClient, librarian, knowledgeDocumentService, service.ResearchConfig{
			Endpoint:       cfg.AI.ResearchEndpoint,
			APIKey:         cfg.AI.ResearchAPIKey,
			MaxResults:     cfg.AI.ResearchMaxResults,
			TimeoutSeconds: cfg.AI.ResearchTimeoutSeconds,
		})
		options := []any{memorySummarizer}
		if err != nil {
			logger.Warn("ThinkTank runner unavailable, continuing with manual ThinkTank flow", zap.Error(err))
		} else if adkRunner != nil {
			options = append(options, adkRunner)
		}
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
			options...,
		)
		aiService = service.NewAIService(aiCore.llmClient, thinkTankService, logger)

		return &appServices{
			oauth:             oauthService,
			verification:      verificationService,
			user:              userService,
			category:          categoryService,
			setting:           settingService,
			vector:            vectorService,
			knowledgeDocument: knowledgeDocumentService,
			ai:                aiService,
			article:           service.NewArticleService(repos.article, repos.category, infra.rdb, vectorService, logger),
			comment:           service.NewCommentService(repos.comment, repos.article, service.WithReplyNotificationSender(commentReplyEmailSender), service.WithCommentNotificationService(notifSvc), service.WithArticleCacheInvalidation(infra.rdb)),
			upload:            service.NewUploadService(repos.upload, cfg),
			stat:              service.NewStatService(repos.stat, infra.rdb),
			notification:      notifSvc,
		}, cleanup, nil
	}

	return &appServices{
		oauth:             oauthService,
		verification:      verificationService,
		user:              userService,
		category:          categoryService,
		setting:           settingService,
		vector:            nil,
		knowledgeDocument: knowledgeDocumentService,
		ai:                aiService,
		article:           service.NewArticleService(repos.article, repos.category, infra.rdb, nil, logger),
		comment:           service.NewCommentService(repos.comment, repos.article, service.WithReplyNotificationSender(commentReplyEmailSender), service.WithCommentNotificationService(notifSvc), service.WithArticleCacheInvalidation(infra.rdb)),
		upload:            service.NewUploadService(repos.upload, cfg),
		stat:              service.NewStatService(repos.stat, infra.rdb),
		notification:      notifSvc,
	}, cleanup, nil
}

func logVerificationEmailConfig(cfg *config.Config, logger *zap.Logger) {
	if cfg == nil || logger == nil {
		return
	}
	logger.Info("Verification email config loaded",
		zap.String("smtp_host", cfg.Email.SMTPHost),
		zap.Int("smtp_port", cfg.Email.SMTPPort),
		zap.Bool("username_configured", cfg.Email.Username != ""),
		zap.Bool("password_configured", cfg.Email.Password != ""),
		zap.Bool("from_address_configured", cfg.Email.FromAddress != ""),
	)
}

func syncPublishedArticleVectors(articleRepo repository.ArticleRepository, vectorService service.VectorService, logger *zap.Logger) error {
	if articleRepo == nil || vectorService == nil || logger == nil {
		return nil
	}

	const pageSize = 100
	batch := 1
	totalSynced := 0

	for {
		articles, total, err := articleRepo.List(repository.ArticleFilter{
			Status:          "published",
			AIIndexStatuses: []string{"pending", "failed"},
			IncludeContent:  true,
			Page:            1,
			PageSize:        pageSize,
		})
		if err != nil {
			return fmt.Errorf("failed to list published articles for vector sync: %w", err)
		}
		if len(articles) == 0 {
			break
		}

		logger.Info("Syncing published article vector batch",
			zap.Int("batch", batch),
			zap.Int("batch_size", len(articles)),
			zap.Int64("total", total))

		for _, article := range articles {
			if article == nil {
				continue
			}
			if err := vectorService.VectorizeArticle(article.ID, article.Title, article.Content, article.Slug); err != nil {
				if statusErr := articleRepo.UpdateAIIndexStatus(article.ID, "failed"); statusErr != nil {
					logger.Warn("Failed to mark article vector sync as failed",
						zap.Int64("article_id", article.ID),
						zap.Error(statusErr))
				}
				return fmt.Errorf("failed to sync article %d vectors: %w", article.ID, err)
			}
			if err := articleRepo.UpdateAIIndexStatus(article.ID, "success"); err != nil {
				return fmt.Errorf("failed to mark article %d vector sync as success: %w", article.ID, err)
			}
			totalSynced++
		}

		batch++
	}

	logger.Info("Published article vector sync completed", zap.Int("article_count", totalSynced))
	return nil
}
