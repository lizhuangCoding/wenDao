package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/async"
	"wenDao/internal/service"
)

type coreServices struct {
	oauth                   service.OAuthService
	verification            service.VerificationService
	user                    service.UserService
	category                service.CategoryService
	tag                     service.TagService
	collection              service.CollectionService
	setting                 service.SettingService
	notification            service.NotificationService
	upload                  service.UploadService
	stat                    *service.StatService
	taskRunner              async.Runner
	commentReplyEmailSender service.CommentReplyNotificationSender
}

func buildCoreServices(cfg *config.Config, logger *zap.Logger, repos *repositories, infra *infrastructure) *coreServices {
	oauthService := service.NewOAuthService(cfg)
	logVerificationEmailConfig(cfg, logger)
	verificationService := service.NewVerificationService(cfg, infra.rdb, nil)
	userService := service.NewUserService(repos.user, oauthService, cfg, infra.rdb)
	notificationService := service.NewNotificationService(repos.notification)

	return &coreServices{
		oauth:                   oauthService,
		verification:            verificationService,
		user:                    userService,
		category:                service.NewCategoryService(repos.category),
		tag:                     service.NewTagService(repos.tag),
		collection:              service.NewCollectionService(repos.collection, repos.article),
		setting:                 service.NewSettingService(repos.setting),
		notification:            notificationService,
		upload:                  service.NewUploadService(repos.upload, cfg),
		stat:                    service.NewStatService(repos.stat, infra.rdb),
		taskRunner:              async.NewTaskRunner(context.Background(), logger, async.WithDefaultTimeout(5*time.Second)),
		commentReplyEmailSender: service.NewSMTPCommentReplyEmailSender(cfg.Email, cfg.Site.URL),
	}
}
