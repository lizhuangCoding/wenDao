package article

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	articleService service.ArticleService
	statService    *service.StatService
	settingService service.SettingService
}

// NewArticleHandler 创建文章处理器实例
func NewArticleHandler(articleService service.ArticleService, statService *service.StatService, settingService service.SettingService) *ArticleHandler {
	return &ArticleHandler{
		articleService: articleService,
		statService:    statService,
		settingService: settingService,
	}
}

func isAdminRequest(c *gin.Context) bool {
	role, exists := c.Get("user_role")
	return exists && role == "admin"
}

// GetSortMode 获取全站排序模式
func (h *ArticleHandler) GetSortMode(c *gin.Context) {
	enabled := h.settingService.GetSortByPopularity()
	response.Success(c, gin.H{"enabled": enabled})
}

// SetSortMode 设置全站排序模式（管理员）
func (h *ArticleHandler) SetSortMode(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := h.settingService.SetSortByPopularity(req.Enabled); err != nil {
		response.InternalErrorWithErr(c, "Failed to set sort mode", err)
		return
	}

	response.Success(c, nil)
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title      string  `json:"title" binding:"required,min=1,max=200"`
	Content    string  `json:"content" binding:"required,min=10"`
	Summary    string  `json:"summary" binding:"max=500"`
	CategoryID int64   `json:"category_id" binding:"required"`
	CoverImage *string `json:"cover_image"`
	Status     string  `json:"status" binding:"required,oneof=draft published"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Title      string  `json:"title" binding:"required,min=1,max=200"`
	Content    string  `json:"content" binding:"required,min=10"`
	Summary    string  `json:"summary" binding:"max=500"`
	CategoryID int64   `json:"category_id" binding:"required"`
	CoverImage *string `json:"cover_image"`
}

type BatchDeleteArticleRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

// AutoSaveRequest 自动保存请求
type AutoSaveRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Summary string `json:"summary"`
}

// Create 创建文章（管理员）
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	authorID, _ := c.Get("user_id")

	article, err := h.articleService.Create(
		req.Title,
		req.Content,
		req.Summary,
		req.CategoryID,
		authorID.(int64),
		req.CoverImage,
		req.Status,
	)
	if err != nil {
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to create article", err)
		return
	}

	response.Success(c, article)
}

// GetByID 根据 ID 获取文章（公开）
func (h *ArticleHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	article, err := h.articleService.GetByID(id)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get article", err)
		return
	}

	if article.Status != "published" && !isAdminRequest(c) {
		response.NotFound(c, "Article not found")
		return
	}

	if article.Status == "published" && !isAdminRequest(c) {
		_ = h.articleService.IncrViewCount(article.ID)
	}

	response.Success(c, article)
}

// GetBySlug 根据 slug 获取文章（公开）
func (h *ArticleHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleService.GetBySlug(slug)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get article", err)
		return
	}

	if article.Status != "published" {
		response.NotFound(c, "Article not found")
		return
	}

	_ = h.articleService.IncrViewCount(article.ID)

	if h.statService != nil {
		ip := c.ClientIP()
		go func(clientIP string) {
			_ = h.statService.RecordPV()
			_ = h.statService.RecordUV(clientIP)
		}(ip)
	}

	response.Success(c, article)
}

// List 获取文章列表
func (h *ArticleHandler) List(c *gin.Context) {
	status := c.Query("status")
	categoryIDStr := c.Query("category_id")
	keyword := c.Query("keyword")
	sortByPopularityStr := c.Query("sort_by_popularity")
	var sortByPopularity bool
	if sortByPopularityStr == "" {
		sortByPopularity = h.settingService.GetSortByPopularity()
	} else {
		sortByPopularity = sortByPopularityStr == "true"
	}
	p := pagination.FromQuery(c)
	categoryID, _ := strconv.ParseInt(categoryIDStr, 10, 64)

	if status == "" {
		status = "published"
	}

	articles, total, err := h.articleService.List(status, categoryID, keyword, sortByPopularity, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list articles", err)
		return
	}

	response.Success(c, gin.H{
		"data":       articles,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

// AdminList 获取所有文章列表（管理员，包含草稿）
func (h *ArticleHandler) AdminList(c *gin.Context) {
	status := c.Query("status")
	categoryIDStr := c.Query("category_id")
	keyword := c.Query("keyword")
	sortByPopularityStr := c.Query("sort_by_popularity")
	var sortByPopularity bool
	if sortByPopularityStr == "" {
		sortByPopularity = h.settingService.GetSortByPopularity()
	} else {
		sortByPopularity = sortByPopularityStr == "true"
	}
	p := pagination.FromQuery(c)
	categoryID, _ := strconv.ParseInt(categoryIDStr, 10, 64)

	articles, total, err := h.articleService.List(status, categoryID, keyword, sortByPopularity, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list articles", err)
		return
	}

	response.Success(c, gin.H{
		"data":       articles,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

// ToggleTop 切换文章置顶状态（管理员）
func (h *ArticleHandler) ToggleTop(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	article, err := h.articleService.ToggleTop(id)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to toggle top status", err)
		return
	}

	response.Success(c, article)
}

// UpdatePopularityScores 手动触发更新文章活跃度分数（管理员）
func (h *ArticleHandler) UpdatePopularityScores(c *gin.Context) {
	if err := h.articleService.UpdatePopularityScores(); err != nil {
		response.InternalErrorWithErr(c, "Failed to update popularity scores", err)
		return
	}
	response.Success(c, gin.H{"message": "Popularity scores updated successfully"})
}

// Update 更新文章（管理员）
func (h *ArticleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	article, err := h.articleService.Update(
		id,
		req.Title,
		req.Content,
		req.Summary,
		req.CategoryID,
		req.CoverImage,
	)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to update article", err)
		return
	}

	response.Success(c, article)
}

// Delete 删除文章（管理员）
func (h *ArticleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	if err := h.articleService.Delete(id); err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to delete article", err)
		return
	}

	response.Success(c, gin.H{
		"message": "Article deleted successfully",
	})
}

// BatchDelete 批量删除文章（管理员）
func (h *ArticleHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的文章")
		return
	}
	ids, ok := normalizeIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "文章 ID 无效")
		return
	}
	if err := h.articleService.DeleteBatch(ids); err != nil {
		response.InternalErrorWithErr(c, "批量删除文章失败", err)
		return
	}
	response.Success(c, gin.H{"message": "Articles deleted successfully", "deleted_count": len(ids)})
}

func normalizeIDs(ids []int64) ([]int64, bool) {
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

// Publish 发布文章（管理员）
func (h *ArticleHandler) Publish(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	if err := h.articleService.Publish(id); err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		if err.Error() == "article is already published" {
			response.Error(c, response.CodeInvalidParams, "Article is already published")
			return
		}
		response.InternalErrorWithErr(c, "Failed to publish article", err)
		return
	}

	response.Success(c, gin.H{
		"message": "Article published successfully",
	})
}

// Draft 转为草稿（管理员）
func (h *ArticleHandler) Draft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	if err := h.articleService.Draft(id); err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		if err.Error() == "article is already draft" {
			response.Error(c, response.CodeInvalidParams, "Article is already draft")
			return
		}
		response.InternalErrorWithErr(c, "Failed to draft article", err)
		return
	}

	response.Success(c, gin.H{
		"message": "Article drafted successfully",
	})
}

// AutoSave 自动保存文章（管理员）
func (h *ArticleHandler) AutoSave(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid article ID")
		return
	}

	var req AutoSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	if err := h.articleService.AutoSave(id, req.Title, req.Content, req.Summary); err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to auto-save article", err)
		return
	}

	response.Success(c, nil)
}
