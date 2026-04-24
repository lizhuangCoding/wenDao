package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

// StatHandler 统计处理器
type StatHandler struct {
	statService *service.StatService
}

// NewStatHandler 创建统计处理器
func NewStatHandler(statService *service.StatService) *StatHandler {
	return &StatHandler{
		statService: statService,
	}
}

// GetDashboardStats 获取后台统计数据
// 支持两种查询方式：
// 1. days=7 表示最近7天
// 2. start_date=2024-04-01&end_date=2024-04-07 表示日期范围
func (h *StatHandler) GetDashboardStats(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var stats *service.DashboardStats
	var err error

	// 如果提供了日期范围，使用日期范围查询
	if startDate != "" && endDate != "" {
		stats, err = h.statService.GetDashboardStatsByRange(startDate, endDate)
	} else {
		// 否则使用天数查询
		daysStr := c.DefaultQuery("days", "7")
		days, err := strconv.Atoi(daysStr)
		if err != nil || days <= 0 {
			days = 7
		}
		stats, err = h.statService.GetDashboardStats(days)
	}

	if err != nil {
		response.InternalError(c, "获取统计数据失败")
		return
	}

	response.Success(c, stats)
}

// GetArticleStats 获取文章访问统计
func (h *StatHandler) GetArticleStats(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseInt(articleIDStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "无效的文章ID")
		return
	}

	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 7
	}

	stats, err := h.statService.GetArticleStats(articleID, days)
	if err != nil {
		response.InternalError(c, "获取文章统计失败")
		return
	}

	response.Success(c, gin.H{
		"data": stats,
	})
}