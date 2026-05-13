package comment

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
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

func parsePaginationQuery(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := c.Query("page_size")
	if pageSize == "" {
		pageSize = c.Query("pageSize")
	}
	return page, normalizePageSize(parsePositiveInt(pageSize, 20))
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return 20
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	ArticleID     int64  `json:"article_id" binding:"required"`
	Content       string `json:"content" binding:"required,min=1,max=1000"`
	ParentID      *int64 `json:"parent_id"`
	ReplyToUserID *int64 `json:"reply_to_user_id"`
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
			response.InternalError(c, "Failed to create comment")
		}
		return
	}

	// 记录评论数统计（异步）
	go h.statService.RecordCommentCount()

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
		response.InternalError(c, "Failed to get comments")
		return
	}

	response.Success(c, comments)
}

// AdminList 获取所有评论列表（管理员）
func (h *CommentHandler) AdminList(c *gin.Context) {
	page, pageSize := parsePaginationQuery(c)

	comments, total, err := h.commentService.ListAll(page, pageSize)
	if err != nil {
		response.InternalError(c, "Failed to get comments")
		return
	}

	response.Success(c, gin.H{
		"data":     comments,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
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
			response.InternalError(c, "Failed to delete comment")
		}
		return
	}

	response.Success(c, gin.H{
		"message": "Comment deleted successfully",
	})
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
			response.InternalError(c, "Failed to restore comment")
		}
		return
	}

	response.Success(c, gin.H{
		"message": "Comment restored successfully",
	})
}
