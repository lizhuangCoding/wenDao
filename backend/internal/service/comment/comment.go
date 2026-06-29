package comment

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	articlesvc "wenDao/internal/service/article"
	asyncjobsvc "wenDao/internal/service/asyncjob"
	"wenDao/internal/svcerrors"
)

// CommentService 评论服务接口
type CommentService interface {
	Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error)
	GetByArticleID(articleID int64) ([]*model.Comment, error)
	GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error)
	ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error)
	Delete(id, userID int64, isAdmin bool) error
	DeleteBatch(ids []int64, userID int64, isAdmin bool) error
	Restore(id int64) error
	Like(commentID, userID int64) error
	Unlike(commentID, userID int64) error
	Dislike(commentID, userID int64) error
	Undislike(commentID, userID int64) error
}

// commentService 评论服务实现
type commentService struct {
	commentRepo             repository.CommentRepository
	articleRepo             repository.ArticleRepository
	userRepo                repository.UserRepository
	replyNotificationSender CommentReplyNotificationSender
	notificationService     NotificationService
	articleCacheRdb         *redis.Client
	writeTxRunner           WriteTransactionRunner
}

// NotificationService 站内通知服务接口（用于评论回复时创建通知）
type NotificationService interface {
	Create(userID int64, notifType, title, content, linkURL string) error
}

type CommentServiceOption func(*commentService)

func WithReplyNotificationSender(sender CommentReplyNotificationSender) CommentServiceOption {
	return func(s *commentService) {
		s.replyNotificationSender = sender
	}
}

func WithNotificationService(notifSvc NotificationService) CommentServiceOption {
	return func(s *commentService) {
		s.notificationService = notifSvc
	}
}

func WithUserRepository(userRepo repository.UserRepository) CommentServiceOption {
	return func(s *commentService) {
		s.userRepo = userRepo
	}
}

func WithArticleCacheInvalidation(rdb *redis.Client) CommentServiceOption {
	return func(s *commentService) {
		s.articleCacheRdb = rdb
	}
}

func WithWriteTransactionRunner(runner WriteTransactionRunner) CommentServiceOption {
	return func(s *commentService) {
		s.writeTxRunner = runner
	}
}

// NewCommentService 创建评论服务实例
func NewCommentService(
	commentRepo repository.CommentRepository,
	articleRepo repository.ArticleRepository,
	options ...CommentServiceOption,
) CommentService {
	svc := &commentService{
		commentRepo: commentRepo,
		articleRepo: articleRepo,
	}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	return svc
}

// Create 创建评论
func (s *commentService) Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error) {
	// 验证文章是否存在
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrArticleNotFound
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// 验证文章是否已发布
	if article.Status != "published" {
		return nil, svcerrors.ErrCannotCommentOnUnpublishedArticle
	}

	var rootID *int64
	var effectiveParentID *int64 = parentID
	var parentComment *model.Comment

	// 如果是回复评论，处理层级和回复目标
	if parentID != nil && *parentID > 0 {
		parentComment, err = s.commentRepo.GetByID(*parentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, svcerrors.ErrParentCommentNotFound
			}
			return nil, fmt.Errorf("failed to get parent comment: %w", err)
		}

		// 验证父评论是否属于同一文章
		if parentComment.ArticleID != articleID {
			return nil, svcerrors.ErrParentCommentNotBelongToArticle
		}

		// 验证父评论是否已删除
		if parentComment.Status == "deleted" {
			return nil, svcerrors.ErrCannotReplyToDeletedComment
		}

		// 抖音模式：所有二级、三级评论都挂在同一个一级评论（Root）下
		if parentComment.ParentID == nil {
			// 如果父评论是一级评论，那么它就是 Root
			id := parentComment.ID
			rootID = &id
			effectiveParentID = &id
		} else {
			// 如果父评论是二级评论，我们要找到它的父评论作为 Root
			effectiveParentID = parentComment.ParentID
			rootID = parentComment.ParentID
		}

		// 如果没有明确传 reply_to_user_id，默认回复父评论作者
		if replyToUserID == nil || *replyToUserID == 0 {
			uid := parentComment.UserID
			replyToUserID = &uid
		}
	}

	// 创建评论
	comment := &model.Comment{
		ArticleID:     articleID,
		UserID:        userID,
		Content:       content,
		ParentID:      effectiveParentID,
		RootID:        rootID,
		ReplyToUserID: replyToUserID,
		Status:        "normal",
	}

	if s.writeTxRunner != nil {
		if err := s.createWithTransaction(article, comment, parentComment); err != nil {
			return nil, err
		}
	} else {
		if err := s.commentRepo.Create(comment); err != nil {
			return nil, fmt.Errorf("failed to create comment: %w", err)
		}
		s.articleRepo.IncrementCommentCount(articleID)
		articlesvc.InvalidateArticleCaches(s.articleCacheRdb, article.ID, article.Slug)
		articlesvc.BumpArticleCollectionCacheVersions(s.articleCacheRdb)
	}

	// 重新查询以获取关联的用户信息（包括被回复人的信息）
	comment, err = s.commentRepo.GetByID(comment.ID)
	if err != nil {
		// 即使查询失败，评论已创建成功
		return comment, nil
	}

	if s.writeTxRunner == nil {
		s.notifyReplyRecipient(context.Background(), article, comment, parentComment)
	}

	return comment, nil
}

func (s *commentService) notifyReplyRecipient(ctx context.Context, article *model.Article, comment *model.Comment, parentComment *model.Comment) {
	if s == nil || article == nil || comment == nil || comment.ReplyToUserID == nil {
		return
	}

	recipient := comment.ReplyToUser
	if recipient == nil && parentComment != nil && parentComment.UserID == *comment.ReplyToUserID {
		recipient = parentComment.User
	}
	if recipient == nil || recipient.ID == comment.UserID {
		return
	}

	replyAuthor := "读者"
	if comment.User != nil && strings.TrimSpace(comment.User.Username) != "" {
		replyAuthor = comment.User.Username
	}

	// 创建站内通知
	if s.notificationService != nil {
		_ = s.notificationService.Create(
			recipient.ID,
			model.NotificationTypeCommentReply,
			fmt.Sprintf("%s 回复了你的评论", replyAuthor),
			fmt.Sprintf("在《%s》中，%s 回复了你的评论：%s", article.Title, replyAuthor, commentPreview(comment.Content)),
			fmt.Sprintf("/article/%s", article.Slug),
		)
	}

	if s.replyNotificationSender == nil || !recipient.CommentReplyEmailEnabled || strings.TrimSpace(recipient.Email) == "" {
		return
	}

	_ = s.replyNotificationSender.SendCommentReplyNotification(ctx, CommentReplyNotification{
		RecipientEmail:      recipient.Email,
		RecipientUsername:   recipient.Username,
		ReplyAuthorUsername: replyAuthor,
		ArticleTitle:        article.Title,
		ArticleSlug:         article.Slug,
		CommentPreview:      commentPreview(comment.Content),
	})
}

// GetByArticleID 获取文章的评论列表
func (s *commentService) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	comments, err := s.commentRepo.GetByArticleID(articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// 组织成树形结构（一级评论 + 二级评论）
	return s.buildCommentTree(comments), nil
}

// ListAll 获取所有评论（管理员）
func (s *commentService) ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	return s.commentRepo.ListAll(filter)
}

// buildCommentTree 构建评论树（两级）
func (s *commentService) buildCommentTree(comments []*model.Comment) []*model.Comment {
	// 一级评论列表
	var topLevelComments []*model.Comment
	// 二级评论映射：parent_id -> [replies]
	replyMap := make(map[int64][]*model.Comment)

	// 分离一级评论和二级评论
	for _, comment := range comments {
		if comment.ParentID == nil {
			// 初始化 Replies 切片（确保前端收到的是空数组而不是 null）
			comment.Replies = make([]*model.Comment, 0)
			topLevelComments = append(topLevelComments, comment)
		} else {
			// 二级评论
			replyMap[*comment.ParentID] = append(replyMap[*comment.ParentID], comment)
		}
	}

	// 将二级评论附加到对应的一级评论
	for _, topLevel := range topLevelComments {
		if replies, ok := replyMap[topLevel.ID]; ok {
			topLevel.Replies = replies
		}
	}

	return topLevelComments
}

// Delete 删除评论
func (s *commentService) Delete(id, userID int64, isAdmin bool) error {
	// 获取评论
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 验证权限：只有本人或管理员可以删除
	if !isAdmin && comment.UserID != userID {
		return svcerrors.ErrPermissionDenied
	}

	// 验证评论是否已删除
	if comment.Status == "deleted" {
		return svcerrors.ErrCommentAlreadyDeleted
	}

	// 删除评论
	if err := s.commentRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// 减少文章的评论数
	s.articleRepo.DecrementCommentCount(comment.ArticleID)
	if article, err := s.articleRepo.GetByID(comment.ArticleID); err == nil {
		articlesvc.InvalidateArticleCaches(s.articleCacheRdb, article.ID, article.Slug)
	}
	articlesvc.BumpArticleCollectionCacheVersions(s.articleCacheRdb)

	return nil
}

// DeleteBatch 批量删除评论，复用单条删除的权限和计数逻辑
func (s *commentService) DeleteBatch(ids []int64, userID int64, isAdmin bool) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid comment id: %d", id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if err := s.Delete(id, userID, isAdmin); err != nil {
			if errors.Is(err, svcerrors.ErrCommentAlreadyDeleted) {
				continue
			}
			return fmt.Errorf("failed to delete comment %d: %w", id, err)
		}
	}
	return nil
}

// Restore 恢复评论（将已删除的评论恢复）
func (s *commentService) Restore(id int64) error {
	// 获取评论
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 验证评论是否已删除
	if comment.Status != "deleted" {
		return svcerrors.ErrCommentIsNotDeleted
	}

	// 恢复评论
	if err := s.commentRepo.Restore(id); err != nil {
		return fmt.Errorf("failed to restore comment: %w", err)
	}

	// 增加文章的评论数
	s.articleRepo.IncrementCommentCount(comment.ArticleID)
	if article, err := s.articleRepo.GetByID(comment.ArticleID); err == nil {
		articlesvc.InvalidateArticleCaches(s.articleCacheRdb, article.ID, article.Slug)
	}
	articlesvc.BumpArticleCollectionCacheVersions(s.articleCacheRdb)

	return nil
}

// GetByArticleIDSorted 获取文章的评论列表（支持排序）
func (s *commentService) GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error) {
	if sort != "hottest" {
		sort = "newest"
	}
	comments, err := s.commentRepo.GetByArticleIDSorted(articleID, sort)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	return s.buildCommentTree(comments), nil
}

// Like 点赞评论
func (s *commentService) Like(commentID, userID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}
	if comment == nil || comment.Status == "deleted" {
		return svcerrors.ErrCommentNotFound
	}

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(commentRepo repository.CommentRepository, _ repository.ArticleRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := commentRepo.IncrementLike(commentID); err != nil {
				return fmt.Errorf("failed to like comment: %w", err)
			}
			return s.enqueueCommentReactionJob(jobRepo, comment, userID, model.NotificationTypeCommentLike, "点赞了你的评论", "点赞了你的评论")
		}); err != nil {
			return err
		}
		return nil
	}
	if err := s.commentRepo.IncrementLike(commentID); err != nil {
		return fmt.Errorf("failed to like comment: %w", err)
	}
	s.notifyCommentReaction(context.Background(), comment, userID, model.NotificationTypeCommentLike, "点赞了你的评论", "点赞了你的评论")
	return nil
}

// Unlike 取消点赞评论
func (s *commentService) Unlike(commentID, userID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}
	if comment == nil || comment.Status == "deleted" {
		return svcerrors.ErrCommentNotFound
	}

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(commentRepo repository.CommentRepository, _ repository.ArticleRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := commentRepo.DecrementLike(commentID); err != nil {
				return fmt.Errorf("failed to unlike comment: %w", err)
			}
			return s.enqueueCommentReactionJob(jobRepo, comment, userID, model.NotificationTypeCommentLike, "取消了对你评论的点赞", "取消了对你评论的点赞")
		}); err != nil {
			return err
		}
		return nil
	}
	if err := s.commentRepo.DecrementLike(commentID); err != nil {
		return fmt.Errorf("failed to unlike comment: %w", err)
	}
	s.notifyCommentReaction(context.Background(), comment, userID, model.NotificationTypeCommentLike, "取消了对你评论的点赞", "取消了对你评论的点赞")
	return nil
}

// Dislike 点踩评论
func (s *commentService) Dislike(commentID, userID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}
	if comment == nil || comment.Status == "deleted" {
		return svcerrors.ErrCommentNotFound
	}

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(commentRepo repository.CommentRepository, _ repository.ArticleRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := commentRepo.IncrementDislike(commentID); err != nil {
				return fmt.Errorf("failed to dislike comment: %w", err)
			}
			return s.enqueueCommentReactionJob(jobRepo, comment, userID, model.NotificationTypeCommentLike, "点踩了你的评论", "点踩了你的评论")
		}); err != nil {
			return err
		}
		return nil
	}
	if err := s.commentRepo.IncrementDislike(commentID); err != nil {
		return fmt.Errorf("failed to dislike comment: %w", err)
	}
	s.notifyCommentReaction(context.Background(), comment, userID, model.NotificationTypeCommentLike, "点踩了你的评论", "点踩了你的评论")
	return nil
}

// Undislike 取消点踩评论
func (s *commentService) Undislike(commentID, userID int64) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCommentNotFound
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}
	if comment == nil || comment.Status == "deleted" {
		return svcerrors.ErrCommentNotFound
	}

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(commentRepo repository.CommentRepository, _ repository.ArticleRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := commentRepo.DecrementDislike(commentID); err != nil {
				return fmt.Errorf("failed to undislike comment: %w", err)
			}
			return s.enqueueCommentReactionJob(jobRepo, comment, userID, model.NotificationTypeCommentLike, "取消了对你评论的点踩", "取消了对你评论的点踩")
		}); err != nil {
			return err
		}
		return nil
	}
	if err := s.commentRepo.DecrementDislike(commentID); err != nil {
		return fmt.Errorf("failed to undislike comment: %w", err)
	}
	s.notifyCommentReaction(context.Background(), comment, userID, model.NotificationTypeCommentLike, "取消了对你评论的点踩", "取消了对你评论的点踩")
	return nil
}

func (s *commentService) createWithTransaction(
	article *model.Article,
	comment *model.Comment,
	parentComment *model.Comment,
) error {
	if s.writeTxRunner == nil {
		return nil
	}
	return s.writeTxRunner.Run(func(commentRepo repository.CommentRepository, articleRepo repository.ArticleRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
		if err := commentRepo.Create(comment); err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}
		if err := articleRepo.IncrementCommentCount(comment.ArticleID); err != nil {
			return fmt.Errorf("failed to increment comment count: %w", err)
		}
		if err := s.enqueueArticleCacheJob(jobRepo, article, true); err != nil {
			return err
		}
		if err := s.enqueueReplyNotificationJobs(jobRepo, article, comment, parentComment); err != nil {
			return err
		}
		return nil
	})
}

func (s *commentService) enqueueArticleCacheJob(jobRepo asyncjobrepo.AsyncJobRepository, article *model.Article, bumpCollections bool) error {
	if jobRepo == nil || article == nil {
		return nil
	}
	job, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeArticleCacheInvalidation, asyncjobsvc.ArticleCacheInvalidationPayload{
		ArticleID:              article.ID,
		ArticleSlug:            article.Slug,
		BumpCollectionVersions: bumpCollections,
	})
	if err != nil {
		return err
	}
	return jobRepo.Enqueue(job)
}

func (s *commentService) enqueueReplyNotificationJobs(jobRepo asyncjobrepo.AsyncJobRepository, article *model.Article, comment *model.Comment, parentComment *model.Comment) error {
	if s == nil || jobRepo == nil || article == nil || comment == nil || comment.ReplyToUserID == nil {
		return nil
	}

	recipient := s.resolveReplyRecipient(comment.ReplyToUserID, parentComment)
	if recipient == nil || recipient.ID == comment.UserID {
		return nil
	}

	replyAuthor := s.resolveActorName(comment.UserID)
	notifJob, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeNotificationCreate, asyncjobsvc.NotificationCreatePayload{
		UserID:           recipient.ID,
		NotificationType: model.NotificationTypeCommentReply,
		Title:            fmt.Sprintf("%s 回复了你的评论", replyAuthor),
		Content:          fmt.Sprintf("在《%s》中，%s 回复了你的评论：%s", article.Title, replyAuthor, commentPreview(comment.Content)),
		LinkURL:          fmt.Sprintf("/article/%s", article.Slug),
	})
	if err != nil {
		return err
	}
	if err := jobRepo.Enqueue(notifJob); err != nil {
		return err
	}

	if !recipient.CommentReplyEmailEnabled || strings.TrimSpace(recipient.Email) == "" {
		return nil
	}

	emailJob, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeCommentReplyEmail, asyncjobsvc.CommentReplyEmailPayload{
		RecipientEmail:      recipient.Email,
		RecipientUsername:   recipient.Username,
		ReplyAuthorUsername: replyAuthor,
		ArticleTitle:        article.Title,
		ArticleSlug:         article.Slug,
		CommentPreview:      commentPreview(comment.Content),
	})
	if err != nil {
		return err
	}
	return jobRepo.Enqueue(emailJob)
}

func (s *commentService) resolveReplyRecipient(replyToUserID *int64, parentComment *model.Comment) *model.User {
	if s == nil || replyToUserID == nil || *replyToUserID == 0 {
		return nil
	}
	if parentComment != nil && parentComment.User != nil && parentComment.UserID == *replyToUserID {
		return parentComment.User
	}
	if s.userRepo == nil {
		return nil
	}
	user, err := s.userRepo.GetByID(*replyToUserID)
	if err != nil {
		return nil
	}
	return user
}

func (s *commentService) enqueueCommentReactionJob(jobRepo asyncjobrepo.AsyncJobRepository, comment *model.Comment, actorID int64, notifType, titleText, actionText string) error {
	if s == nil || jobRepo == nil || comment == nil || comment.UserID == 0 || actorID == 0 || actorID == comment.UserID {
		return nil
	}

	articleTitle := "你的评论"
	articleLink := fmt.Sprintf("/article/%d", comment.ArticleID)
	if article, err := s.articleRepo.GetByID(comment.ArticleID); err == nil && article != nil {
		articleTitle = article.Title
		if strings.TrimSpace(article.Slug) != "" {
			articleLink = fmt.Sprintf("/article/%s", article.Slug)
		}
	}

	preview := commentPreview(comment.Content)
	if preview == "" {
		preview = "你的评论"
	}
	actorName := s.resolveActorName(actorID)
	job, err := asyncjobsvc.NewJob(asyncjobsvc.JobTypeNotificationCreate, asyncjobsvc.NotificationCreatePayload{
		UserID:           comment.UserID,
		NotificationType: notifType,
		Title:            fmt.Sprintf("%s%s", actorName, titleText),
		Content:          fmt.Sprintf("在《%s》中，%s%s：%s", articleTitle, actorName, actionText, preview),
		LinkURL:          articleLink,
	})
	if err != nil {
		return err
	}
	return jobRepo.Enqueue(job)
}

func (s *commentService) notifyCommentReaction(ctx context.Context, comment *model.Comment, actorID int64, notifType, titleText, actionText string) {
	if s == nil || s.notificationService == nil || comment == nil || comment.UserID == 0 || actorID == 0 || actorID == comment.UserID {
		return
	}

	articleTitle := "你的评论"
	articleLink := ""
	if article, err := s.articleRepo.GetByID(comment.ArticleID); err == nil && article != nil {
		articleTitle = article.Title
		if strings.TrimSpace(article.Slug) != "" {
			articleLink = fmt.Sprintf("/article/%s", article.Slug)
		}
	}
	if articleLink == "" {
		articleLink = fmt.Sprintf("/article/%d", comment.ArticleID)
	}

	preview := commentPreview(comment.Content)
	if preview == "" {
		preview = "你的评论"
	}
	actorName := s.resolveActorName(actorID)

	_ = s.notificationService.Create(
		comment.UserID,
		notifType,
		fmt.Sprintf("%s%s", actorName, titleText),
		fmt.Sprintf("在《%s》中，%s%s：%s", articleTitle, actorName, actionText, preview),
		articleLink,
	)
}

func (s *commentService) resolveActorName(actorID int64) string {
	if actorID <= 0 {
		return "读者"
	}
	if s.userRepo == nil {
		return "读者"
	}
	user, err := s.userRepo.GetByID(actorID)
	if err != nil || user == nil || strings.TrimSpace(user.Username) == "" {
		return "读者"
	}
	return user.Username
}
