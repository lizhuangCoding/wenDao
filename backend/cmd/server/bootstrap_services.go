package main

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/async"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/service"
	aisvc "wenDao/internal/service/ai"
	articlesvc "wenDao/internal/service/article"
	asyncjobsvc "wenDao/internal/service/asyncjob"
)

type appServices struct {
	oauth             service.OAuthService
	verification      service.VerificationService
	user              service.UserService
	category          service.CategoryService
	tag               service.TagService
	collection        service.CollectionService
	setting           service.SettingService
	vector            service.VectorService
	knowledgeDocument service.KnowledgeDocumentService
	ai                service.AIService
	article           service.ArticleService
	comment           service.CommentService
	upload            service.UploadService
	stat              *service.StatService
	notification      service.NotificationService
	asyncJob          service.AsyncJobService
	taskRunner        async.Runner
}

func initServices(cfg *config.Config, logger *zap.Logger, repos *repositories, infra *infrastructure, aiCore *aiComponents) (*appServices, func(), error) {
	oauthService := service.NewOAuthService(cfg)
	var rdb = infra.rdb
	logVerificationEmailConfig(cfg, logger)
	verificationService := service.NewVerificationService(cfg, rdb, nil)
	userService := service.NewUserService(repos.user, oauthService, cfg, rdb)
	commentReplyEmailSender := service.NewSMTPCommentReplyEmailSender(cfg.Email, cfg.Site.URL)
	notifSvc := service.NewNotificationService(repos.notification)
	taskRunner := async.NewTaskRunner(context.Background(), logger, async.WithDefaultTimeout(5*time.Second))
	var vectorService service.VectorService
	asyncJobService := service.NewAsyncJobService(
		repos.asyncJob,
		logger,
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeNotificationCreate, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.NotificationCreatePayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			return notifSvc.Create(payload.UserID, payload.NotificationType, payload.Title, payload.Content, payload.LinkURL)
		}),
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeCommentReplyEmail, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.CommentReplyEmailPayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			if commentReplyEmailSender == nil {
				return nil
			}
			return commentReplyEmailSender.SendCommentReplyNotification(ctx, service.CommentReplyNotification{
				RecipientEmail:      payload.RecipientEmail,
				RecipientUsername:   payload.RecipientUsername,
				ReplyAuthorUsername: payload.ReplyAuthorUsername,
				ArticleTitle:        payload.ArticleTitle,
				ArticleSlug:         payload.ArticleSlug,
				CommentPreview:      payload.CommentPreview,
			})
		}),
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeArticleCacheInvalidation, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.ArticleCacheInvalidationPayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			articlesvc.InvalidateArticleCaches(infra.rdb, payload.ArticleID, payload.ArticleSlug)
			if payload.BumpCollectionVersions {
				articlesvc.BumpArticleCollectionCacheVersions(infra.rdb)
			}
			return nil
		}),
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeArticleVectorize, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.ArticleVectorizePayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			if vectorService == nil {
				return nil
			}
			if err := repos.article.UpdateAIIndexStatus(payload.ArticleID, "pending"); err != nil {
				logger.Warn("Failed to mark article AI index pending", zap.Int64("article_id", payload.ArticleID), zap.Error(err))
			}
			if err := vectorService.VectorizeArticle(payload.ArticleID, payload.Title, payload.Content, payload.Slug); err != nil {
				_ = repos.article.UpdateAIIndexStatus(payload.ArticleID, "failed")
				return err
			}
			return repos.article.UpdateAIIndexStatus(payload.ArticleID, "success")
		}),
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeArticleVectorDelete, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.ArticleVectorDeletePayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			if vectorService == nil {
				return nil
			}
			return vectorService.DeleteArticleVector(payload.ArticleID)
		}),
	)
	categoryService := service.NewCategoryService(repos.category)
	tagService := service.NewTagService(repos.tag)
	collectionService := service.NewCollectionService(repos.collection, repos.article)
	settingService := service.NewSettingService(repos.setting)
	knowledgeDocumentService := service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, nil, repos.article, repos.category, logger)
	aiService := service.NewDisabledAIService("AI initialization failed")
	cleanup := func() {}

	if aiCore != nil && aiCore.llmClient != nil {
		var librarian service.Librarian
		if aiCore.vectorStore != nil && aiCore.embedder != nil {
			vectorService = service.NewVectorService(aiCore.vectorStore, aiCore.embedder, logger, repos.articleSemanticProfile)
			if err := aisvc.SyncPublishedArticleVectors(repos.article, repos.articleSemanticProfile, vectorService, logger); err != nil {
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
		adkRunner, err := service.NewThinkTankADKRunner(context.Background(), aiCore.llmClient, librarian, service.ResearchConfig{
			Endpoint:       cfg.AI.ResearchEndpoint,
			APIKey:         cfg.AI.ResearchAPIKey,
			MaxResults:     cfg.AI.ResearchMaxResults,
			TimeoutSeconds: cfg.AI.ResearchTimeoutSeconds,
		})
		options := service.ThinkTankServiceOptions{
			MemorySummarizer: memorySummarizer,
			Metrics:          service.NewRunMetricsConfig(cfg.AI),
		}
		if err != nil {
			logger.Warn("ThinkTank runner unavailable, continuing with manual ThinkTank flow", zap.Error(err))
		} else if adkRunner != nil {
			options.ADKRunner = adkRunner
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
			options,
		)
		pluginRegistry := service.NewPluginRegistry()
		if err := pluginRegistry.Register(service.NewThinkTankPlugin(thinkTankService), service.WithDefaultPlugin()); err != nil {
			logger.Warn("ThinkTank plugin registration failed, continuing with disabled AI service", zap.Error(err))
		} else if defaultAgent, ok := pluginRegistry.Default(); ok {
			aiService = service.NewAIService(aiCore.llmClient, defaultAgent, logger)
		} else {
			logger.Warn("ThinkTank plugin registry has no default plugin, continuing with disabled AI service")
		}

		return &appServices{
			oauth:             oauthService,
			verification:      verificationService,
			user:              userService,
			category:          categoryService,
			tag:               tagService,
			collection:        collectionService,
			setting:           settingService,
			vector:            vectorService,
			knowledgeDocument: knowledgeDocumentService,
			ai:                aiService,
			article:           service.NewArticleService(repos.article, repos.category, infra.rdb, vectorService, logger, repos.articleSemanticProfile, repos.tag, service.WithArticleWriteTransactionRunner(service.NewArticleWriteTransactionRunner(infra.db)), service.WithArticleTaskRunner(taskRunner)),
			comment:           service.NewCommentService(repos.comment, repos.article, service.WithReplyNotificationSender(commentReplyEmailSender), service.WithCommentNotificationService(notifSvc), service.WithCommentUserRepository(repos.user), service.WithCommentAsyncJobRepository(repos.asyncJob), service.WithArticleCacheInvalidation(infra.rdb), service.WithCommentWriteTransactionRunner(service.NewCommentWriteTransactionRunner(infra.db))),
			upload:            service.NewUploadService(repos.upload, cfg),
			stat:              service.NewStatService(repos.stat, infra.rdb),
			notification:      notifSvc,
			asyncJob:          asyncJobService,
			taskRunner:        taskRunner,
		}, cleanup, nil
	}

	return &appServices{
		oauth:             oauthService,
		verification:      verificationService,
		user:              userService,
		category:          categoryService,
		tag:               tagService,
		collection:        collectionService,
		setting:           settingService,
		vector:            nil,
		knowledgeDocument: knowledgeDocumentService,
		ai:                aiService,
		article:           service.NewArticleService(repos.article, repos.category, infra.rdb, nil, logger, repos.articleSemanticProfile, repos.tag, service.WithArticleWriteTransactionRunner(service.NewArticleWriteTransactionRunner(infra.db)), service.WithArticleTaskRunner(taskRunner)),
		comment:           service.NewCommentService(repos.comment, repos.article, service.WithReplyNotificationSender(commentReplyEmailSender), service.WithCommentNotificationService(notifSvc), service.WithCommentUserRepository(repos.user), service.WithCommentAsyncJobRepository(repos.asyncJob), service.WithArticleCacheInvalidation(infra.rdb), service.WithCommentWriteTransactionRunner(service.NewCommentWriteTransactionRunner(infra.db))),
		upload:            service.NewUploadService(repos.upload, cfg),
		stat:              service.NewStatService(repos.stat, infra.rdb),
		notification:      notifSvc,
		asyncJob:          asyncJobService,
		taskRunner:        taskRunner,
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
