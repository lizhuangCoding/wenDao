package comment

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/async"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

// CommentHandler 评论处理器
type CommentHandler struct {
	commentService service.CommentService
	statService    commentStatRecorder
	taskRunner     async.Runner
}

type commentStatRecorder interface {
	RecordCommentCountContext(ctx context.Context) error
}

// NewCommentHandler 创建评论处理器实例
func NewCommentHandler(commentService service.CommentService, statService commentStatRecorder) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		statService:    statService,
	}
}

func (h *CommentHandler) SetTaskRunner(runner async.Runner) {
	if h != nil {
		h.taskRunner = runner
	}
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	ArticleID     int64  `json:"article_id" binding:"required"`
	Content       string `json:"content" binding:"required,min=1,max=1000"`
	ParentID      *int64 `json:"parent_id"`
	ReplyToUserID *int64 `json:"reply_to_user_id"`
}

type BatchDeleteCommentRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

// Create 发表评论（需要认证）
func (h *CommentHandler) Create(c *gin.Context) {
	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "评论参数不正确：文章 ID 和评论内容必填，内容长度需在 1-1000 字之间")
		return
	}

	// 从 context 获取当前用户 ID
	userID, _ := c.Get("user_id")

	comment, err := h.commentService.Create(req.ArticleID, userID.(int64), req.Content, req.ParentID, req.ReplyToUserID)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法发表评论")
		} else if errors.Is(err, svcerrors.ErrCannotCommentOnUnpublishedArticle) {
			response.Forbidden(c, "文章尚未发布，不能发表评论")
		} else if errors.Is(err, svcerrors.ErrParentCommentNotFound) {
			response.NotFound(c, "要回复的评论不存在或已被删除")
		} else if errors.Is(err, svcerrors.ErrParentCommentNotBelongToArticle) {
			response.InvalidParams(c, "要回复的评论不属于当前文章")
		} else if errors.Is(err, svcerrors.ErrCannotReplyToDeletedComment) {
			response.InvalidParams(c, "该评论已删除，不能继续回复")
		} else if errors.Is(err, svcerrors.ErrCannotReplyToReplyComment) {
			response.InvalidParams(c, "当前只支持两级评论，不能继续回复子评论")
		} else {
			response.InternalErrorWithErr(c, "发表评论失败，请稍后重试", err)
		}
		return
	}

	// 记录评论数统计（异步）
	if h.statService != nil {
		ctx := context.WithoutCancel(c.Request.Context())
		if h.taskRunner != nil {
			_ = h.taskRunner.Submit(
				ctx,
				"record comment count",
				h.statService.RecordCommentCountContext,
				async.WithTimeout(3*time.Second),
				async.WithRetries(1),
				async.WithRetryDelay(func(attempt int) time.Duration { return 100 * time.Millisecond }),
			)
		} else {
			_ = h.statService.RecordCommentCountContext(ctx)
		}
	}

	response.Success(c, comment)
}

// GetByArticleID 获取文章评论列表（公开）
func (h *CommentHandler) GetByArticleID(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	sort := c.Query("sort")
	if sort != "hottest" {
		sort = "newest"
	}

	comments, err := h.commentService.GetByArticleIDSorted(articleID, sort)
	if err != nil {
		response.InternalErrorWithErr(c, "评论列表加载失败，请稍后重试", err)
		return
	}

	response.Success(c, comments)
}

// AdminList 获取所有评论列表（管理员）
func (h *CommentHandler) AdminList(c *gin.Context) {
	p := pagination.FromQuery(c)

	comments, total, err := h.commentService.ListAll(repository.CommentFilter{
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "评论管理列表加载失败，请稍后重试", err)
		return
	}

	response.Success(c, gin.H{
		"data":       comments,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

// Delete 删除评论（本人或管理员）
func (h *CommentHandler) Delete(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	// 从 context 获取当前用户信息
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	isAdmin := userRole.(string) == "admin"

	if err := h.commentService.Delete(commentID, userID.(int64), isAdmin); err != nil {
		if errors.Is(err, svcerrors.ErrCommentNotFound) {
			response.NotFound(c, "评论不存在或已被删除")
		} else if errors.Is(err, svcerrors.ErrPermissionDenied) {
			response.Forbidden(c, "你没有权限删除这条评论")
		} else if errors.Is(err, svcerrors.ErrCommentAlreadyDeleted) {
			response.InvalidParams(c, "该评论已经删除，无需重复操作")
		} else {
			response.InternalErrorWithErr(c, "删除评论失败，请稍后重试", err)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "评论删除成功",
	})
}

// BatchDelete 批量删除评论（管理员）
func (h *CommentHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的评论")
		return
	}
	ids, ok := normalizeCommentIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "评论 ID 无效")
		return
	}

	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	isAdmin := userRole.(string) == "admin"
	if err := h.commentService.DeleteBatch(ids, userID.(int64), isAdmin); err != nil {
		response.InternalErrorWithErr(c, "批量删除评论失败：请确认所选评论仍存在且你有权限操作", err)
		return
	}

	response.Success(c, gin.H{"message": "评论批量删除成功", "deleted_count": len(ids)})
}

func normalizeCommentIDs(ids []int64) ([]int64, bool) {
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, len(normalized) > 0
}

// Like 点赞评论
func (h *CommentHandler) Like(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	var userID int64
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(int64)
	}

	if err := h.commentService.Like(commentID, userID); err != nil {
		response.InternalErrorWithErr(c, "点赞评论失败，请稍后重试", err)
		return
	}

	response.Success(c, nil)
}

// Unlike 取消点赞评论
func (h *CommentHandler) Unlike(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	var userID int64
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(int64)
	}

	if err := h.commentService.Unlike(commentID, userID); err != nil {
		response.InternalErrorWithErr(c, "取消评论点赞失败，请稍后重试", err)
		return
	}

	response.Success(c, nil)
}

// Dislike 点踩评论
func (h *CommentHandler) Dislike(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	var userID int64
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(int64)
	}

	if err := h.commentService.Dislike(commentID, userID); err != nil {
		response.InternalErrorWithErr(c, "点踩评论失败，请稍后重试", err)
		return
	}

	response.Success(c, nil)
}

// Undislike 取消点踩评论
func (h *CommentHandler) Undislike(c *gin.Context) {
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	var userID int64
	if value, ok := c.Get("user_id"); ok {
		userID, _ = value.(int64)
	}

	if err := h.commentService.Undislike(commentID, userID); err != nil {
		response.InternalErrorWithErr(c, "取消评论点踩失败，请稍后重试", err)
		return
	}

	response.Success(c, nil)
}

// Restore 恢复评论（管理员）
func (h *CommentHandler) Restore(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "评论 ID 无效，请刷新页面后重试")
		return
	}

	if err := h.commentService.Restore(commentID); err != nil {
		if errors.Is(err, svcerrors.ErrCommentNotFound) {
			response.NotFound(c, "评论不存在，无法恢复")
		} else if errors.Is(err, svcerrors.ErrCommentIsNotDeleted) {
			response.InvalidParams(c, "该评论未被删除，无需恢复")
		} else {
			response.InternalErrorWithErr(c, "恢复评论失败，请稍后重试", err)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "评论恢复成功",
	})
}
