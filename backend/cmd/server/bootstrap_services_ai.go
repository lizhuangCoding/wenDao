package main

import (
	"context"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/service"
	aisvc "wenDao/internal/service/ai"
)

type aiStack struct {
	vector            service.VectorService
	knowledgeDocument service.KnowledgeDocumentService
	ai                service.AIService
	cleanup           func()
}

func newDisabledAIStack(repos *repositories, logger *zap.Logger) *aiStack {
	return &aiStack{
		vector:            nil,
		knowledgeDocument: service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, nil, repos.article, repos.category, logger),
		ai:                service.NewDisabledAIService("AI initialization failed"),
		cleanup:           func() {},
	}
}

func buildAIStack(cfg *config.Config, logger *zap.Logger, repos *repositories, aiCore *aiComponents, base *aiStack) *aiStack {
	if aiCore == nil || aiCore.llmClient == nil {
		return base
	}

	var librarian service.Librarian
	if aiCore.vectorStore != nil && aiCore.embedder != nil {
		base.vector = service.NewVectorService(aiCore.vectorStore, aiCore.embedder, logger, repos.articleSemanticProfile)
		if err := aisvc.SyncPublishedArticleVectors(repos.article, repos.articleSemanticProfile, base.vector, logger); err != nil {
			logger.Warn("Published article vector sync skipped, continuing in degraded mode", zap.Error(err))
		}
		retriever := eino.NewRedisRetriever(aiCore.vectorStore, aiCore.embedder, cfg.AI.TopK)
		ragChain := eino.NewRAGChain(retriever, aiCore.llmClient.GetModel(), cfg.AI.RAGMinScore, logger)
		librarian = service.NewLibrarianService(ragChain)
	} else {
		logger.Warn("Vector search unavailable, ThinkTank will continue without local RAG retrieval")
		librarian = service.NewLibrarianService(nil)
	}

	base.knowledgeDocument = service.NewKnowledgeDocumentService(repos.knowledgeDocument, repos.knowledgeDocumentSource, base.vector, repos.article, repos.category, logger)

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
		base.cleanup = func() {
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
	} else if typedRunner, ok := adkRunner.(interface {
		Answer(context.Context, string) (string, error)
	}); ok && typedRunner != nil {
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
		base.knowledgeDocument,
		aiEventLogger,
		options,
	)
	pluginRegistry := service.NewPluginRegistry()
	if err := pluginRegistry.Register(service.NewThinkTankPlugin(thinkTankService), service.WithDefaultPlugin()); err != nil {
		logger.Warn("ThinkTank plugin registration failed, continuing with disabled AI service", zap.Error(err))
		return base
	}
	defaultAgent, ok := pluginRegistry.Default()
	if !ok {
		logger.Warn("ThinkTank plugin registry has no default plugin, continuing with disabled AI service")
		return base
	}

	base.ai = service.NewAIService(aiCore.llmClient, defaultAgent, logger)
	return base
}
