package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

// CommentService 评论服务接口
type CommentService interface {
	Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error)
	GetByArticleID(articleID int64) ([]*model.Comment, error)
	ListAll(page, pageSize int) ([]*model.Comment, int64, error)
	Delete(id, userID int64, isAdmin bool) error
	Restore(id int64) error
}

// commentService 评论服务实现
type commentService struct {
	commentRepo  repository.CommentRepository
	articleRepo  repository.ArticleRepository
}

// NewCommentService 创建评论服务实例
func NewCommentService(
	commentRepo repository.CommentRepository,
	articleRepo repository.ArticleRepository,
) CommentService {
	return &commentService{
		commentRepo:  commentRepo,
		articleRepo:  articleRepo,
	}
}

// Create 创建评论
func (s *commentService) Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error) {
	// 验证文章是否存在
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// 验证文章是否已发布
	if article.Status != "published" {
		return nil, errors.New("cannot comment on unpublished article")
	}

	var rootID *int64
	var effectiveParentID *int64 = parentID

	// 如果是回复评论，处理层级和回复目标
	if parentID != nil && *parentID > 0 {
		parentComment, err := s.commentRepo.GetByID(*parentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("parent comment not found")
			}
			return nil, fmt.Errorf("failed to get parent comment: %w", err)
		}

		// 验证父评论是否属于同一文章
		if parentComment.ArticleID != articleID {
			return nil, errors.New("parent comment does not belong to this article")
		}

		// 验证父评论是否已删除
		if parentComment.Status == "deleted" {
			return nil, errors.New("cannot reply to deleted comment")
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

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// 增加文章的评论数
	s.articleRepo.IncrementCommentCount(articleID)

	// 重新查询以获取关联的用户信息（包括被回复人的信息）
	comment, err = s.commentRepo.GetByID(comment.ID)
	if err != nil {
		// 即使查询失败，评论已创建成功
		return comment, nil
	}

	return comment, nil
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
func (s *commentService) ListAll(page, pageSize int) ([]*model.Comment, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	return s.commentRepo.ListAll(page, pageSize)
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
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 验证权限：只有本人或管理员可以删除
	if !isAdmin && comment.UserID != userID {
		return errors.New("permission denied")
	}

	// 验证评论是否已删除
	if comment.Status == "deleted" {
		return errors.New("comment already deleted")
	}

	// 删除评论
	if err := s.commentRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// 减少文章的评论数
	s.articleRepo.DecrementCommentCount(comment.ArticleID)

	return nil
}

// Restore 恢复评论（将已删除的评论恢复）
func (s *commentService) Restore(id int64) error {
	// 获取评论
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 验证评论是否已删除
	if comment.Status != "deleted" {
		return errors.New("comment is not deleted")
	}

	// 恢复评论
	if err := s.commentRepo.Restore(id); err != nil {
		return fmt.Errorf("failed to restore comment: %w", err)
	}

	// 增加文章的评论数
	s.articleRepo.IncrementCommentCount(comment.ArticleID)

	return nil
}
