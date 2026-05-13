package comment

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

// CommentHandler 评论处理器
type CommentHandler struct {
	commentService service.CommentService
	statService    *service.StatService
}

// NewCommentHandler 创建评论处理器实例
func NewCommentHandler(commentService service.CommentService, statService *service.StatService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		statService:    statService,
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
		response.InvalidParams(c, err.Error())
		return
	}

	// 从 context 获取当前用户 ID
	userID, _ := c.Get("user_id")

	comment, err := h.commentService.Create(req.ArticleID, userID.(int64), req.Content, req.ParentID, req.ReplyToUserID)
	if err != nil {
		switch err.Error() {
		case "article not found":
			response.NotFound(c, "Article not found")
		case "cannot comment on unpublished article":
			response.Forbidden(c, "Cannot comment on unpublished article")
		case "parent comment not found":
			response.NotFound(c, "Parent comment not found")
		case "parent comment does not belong to this article":
			response.InvalidParams(c, "Parent comment does not belong to this article")
		case "cannot reply to deleted comment":
			response.InvalidParams(c, "Cannot reply to deleted comment")
		case "cannot reply to a reply comment (only two levels allowed)":
			response.InvalidParams(c, "Cannot reply to a reply comment (only two levels allowed)")
		default:
			response.InternalErrorWithErr(c, "Failed to create comment", err)
		}
		return
	}

	// 记录评论数统计（异步）
	if h.statService != nil {
		go h.statService.RecordCommentCount()
	}

	response.Success(c, comment)
}

// GetByArticleID 获取文章评论列表（公开）
func (h *CommentHandler) GetByArticleID(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	comments, err := h.commentService.GetByArticleID(articleID)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to get comments", err)
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
		response.InternalErrorWithErr(c, "Failed to get comments", err)
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
		response.InvalidParams(c, "Invalid comment ID")
		return
	}

	// 从 context 获取当前用户信息
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	isAdmin := userRole.(string) == "admin"

	if err := h.commentService.Delete(commentID, userID.(int64), isAdmin); err != nil {
		switch err.Error() {
		case "comment not found":
			response.NotFound(c, "Comment not found")
		case "permission denied":
			response.Forbidden(c, "Permission denied")
		case "comment already deleted":
			response.InvalidParams(c, "Comment already deleted")
		default:
			response.InternalErrorWithErr(c, "Failed to delete comment", err)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "Comment deleted successfully",
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
		response.InternalErrorWithErr(c, "批量删除评论失败", err)
		return
	}

	response.Success(c, gin.H{"message": "Comments deleted successfully", "deleted_count": len(ids)})
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

// Restore 恢复评论（管理员）
func (h *CommentHandler) Restore(c *gin.Context) {
	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid comment ID")
		return
	}

	if err := h.commentService.Restore(commentID); err != nil {
		switch err.Error() {
		case "comment not found":
			response.NotFound(c, "Comment not found")
		case "comment is not deleted":
			response.InvalidParams(c, "Comment is not deleted")
		default:
			response.InternalErrorWithErr(c, "Failed to restore comment", err)
		}
		return
	}

	response.Success(c, gin.H{
		"message": "Comment restored successfully",
	})
}
