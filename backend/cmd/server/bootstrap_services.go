package main

import (
	"context"
	"fmt"
	"path/filepath"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

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
