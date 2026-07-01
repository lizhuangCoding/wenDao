package main

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/service"
	articlesvc "wenDao/internal/service/article"
	asyncjobsvc "wenDao/internal/service/asyncjob"
)

func buildAsyncJobService(
	logger *zap.Logger,
	repos *repositories,
	infra *infrastructure,
	core *coreServices,
	vectorServiceProvider func() service.VectorService,
) service.AsyncJobService {
	return service.NewAsyncJobService(
		repos.asyncJob,
		logger,
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeNotificationCreate, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.NotificationCreatePayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			return core.notification.Create(payload.UserID, payload.NotificationType, payload.Title, payload.Content, payload.LinkURL)
		}),
		asyncjobsvc.WithJobHandler(asyncjobsvc.JobTypeCommentReplyEmail, func(ctx context.Context, job *model.AsyncJob) error {
			var payload asyncjobsvc.CommentReplyEmailPayload
			if err := json.Unmarshal(job.Payload, &payload); err != nil {
				return err
			}
			if core.commentReplyEmailSender == nil {
				return nil
			}
			return core.commentReplyEmailSender.SendCommentReplyNotification(ctx, service.CommentReplyNotification{
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
			vectorService := vectorServiceProvider()
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
			vectorService := vectorServiceProvider()
			if vectorService == nil {
				return nil
			}
			return vectorService.DeleteArticleVector(payload.ArticleID)
		}),
	)
}
