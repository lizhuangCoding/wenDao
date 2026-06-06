package notification

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service/notification"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notifSvc    notification.NotificationService
	getUserIDs  func() ([]int64, error)
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(notifSvc notification.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// SetUserIDProvider 设置获取所有用户ID的函数（供广播功能使用）
func (h *NotificationHandler) SetUserIDProvider(fn func() ([]int64, error)) {
	h.getUserIDs = fn
}

// List 获取当前用户的通知列表
func (h *NotificationHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	p := pagination.FromQuery(c)

	notifications, total, err := h.notifSvc.ListByUser(userID.(int64), p.Page, p.PageSize)
	if err != nil {
		response.InternalError(c, "Failed to list notifications")
		return
	}

	response.Success(c, gin.H{
		"data":        notifications,
		"total":       total,
		"page":        p.Page,
		"page_size":   p.PageSize,
		"total_pages": pagination.TotalPages(total, p.PageSize),
	})
}

// GetUnreadCount 获取未读通知数
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := c.Get("user_id")

	count, err := h.notifSvc.GetUnreadCount(userID.(int64))
	if err != nil {
		response.InternalError(c, "Failed to get unread count")
		return
	}

	response.Success(c, gin.H{"unread_count": count})
}

// MarkRead 标记单条通知已读
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	notifID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, response.CodeInvalidParams, "Invalid notification ID")
		return
	}

	if err := h.notifSvc.MarkRead(userID.(int64), notifID); err != nil {
		response.InternalError(c, "Failed to mark notification as read")
		return
	}

	response.Success(c, nil)
}

// MarkAllRead 标记所有通知已读
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, _ := c.Get("user_id")

	if err := h.notifSvc.MarkAllRead(userID.(int64)); err != nil {
		response.InternalError(c, "Failed to mark all notifications as read")
		return
	}

	response.Success(c, nil)
}

// BroadcastRequest 管理员广播请求
type BroadcastRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	LinkURL string `json:"link_url"`
}

// Broadcast 管理员发送广播通知
func (h *NotificationHandler) Broadcast(c *gin.Context) {
	if h.getUserIDs == nil {
		response.InternalError(c, "Broadcast function not configured")
		return
	}

	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "Invalid request")
		return
	}

	if err := h.notifSvc.BroadcastToAllUsers(req.Title, req.Content, req.LinkURL, h.getUserIDs); err != nil {
		response.InternalError(c, "Failed to broadcast notification")
		return
	}

	response.Success(c, gin.H{"message": "Broadcast sent successfully"})
}
