package main

import (
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/async"
	"wenDao/internal/service"
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
	core := buildCoreServices(cfg, logger, repos, infra)
	ai := newDisabledAIStack(repos, logger)
	ai = buildAIStack(cfg, logger, repos, aiCore, ai)
	asyncJob := buildAsyncJobService(logger, repos, infra, core, func() service.VectorService {
		return ai.vector
	})

	return assembleAppServices(repos, infra, logger, core, ai, asyncJob), ai.cleanup, nil
}

func assembleAppServices(
	repos *repositories,
	infra *infrastructure,
	logger *zap.Logger,
	core *coreServices,
	ai *aiStack,
	asyncJob service.AsyncJobService,
) *appServices {
	return &appServices{
		oauth:             core.oauth,
		verification:      core.verification,
		user:              core.user,
		category:          core.category,
		tag:               core.tag,
		collection:        core.collection,
		setting:           core.setting,
		vector:            ai.vector,
		knowledgeDocument: ai.knowledgeDocument,
		ai:                ai.ai,
		article:           newArticleService(repos, infra, logger, ai.vector, core.taskRunner),
		comment:           newCommentService(repos, infra, core),
		upload:            core.upload,
		stat:              core.stat,
		notification:      core.notification,
		asyncJob:          asyncJob,
		taskRunner:        core.taskRunner,
	}
}

func newArticleService(repos *repositories, infra *infrastructure, logger *zap.Logger, vector service.VectorService, taskRunner async.Runner) service.ArticleService {
	return service.NewArticleService(
		repos.article,
		repos.category,
		infra.rdb,
		vector,
		logger,
		repos.articleSemanticProfile,
		repos.tag,
		service.WithArticleWriteTransactionRunner(service.NewArticleWriteTransactionRunner(infra.db)),
		service.WithArticleTaskRunner(taskRunner),
	)
}

func newCommentService(repos *repositories, infra *infrastructure, core *coreServices) service.CommentService {
	return service.NewCommentService(
		repos.comment,
		repos.article,
		service.WithReplyNotificationSender(core.commentReplyEmailSender),
		service.WithCommentNotificationService(core.notification),
		service.WithCommentUserRepository(repos.user),
		service.WithCommentAsyncJobRepository(repos.asyncJob),
		service.WithArticleCacheInvalidation(infra.rdb),
		service.WithCommentWriteTransactionRunner(service.NewCommentWriteTransactionRunner(infra.db)),
	)
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
