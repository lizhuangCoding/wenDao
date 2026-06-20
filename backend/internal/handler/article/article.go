package article

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/pkg/async"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

// ArticleHandler 文章处理器
type ArticleHandler struct {
	articleService    service.ArticleService
	collectionService service.CollectionService
	statService       *service.StatService
	settingService    service.SettingService
}

// NewArticleHandler 创建文章处理器实例
func NewArticleHandler(articleService service.ArticleService, statService *service.StatService, settingService service.SettingService, collectionServices ...service.CollectionService) *ArticleHandler {
	var collectionService service.CollectionService
	if len(collectionServices) > 0 {
		collectionService = collectionServices[0]
	}
	return &ArticleHandler{
		articleService:    articleService,
		collectionService: collectionService,
		statService:       statService,
		settingService:    settingService,
	}
}

func isAdminRequest(c *gin.Context) bool {
	role, exists := c.Get("user_role")
	return exists && role == "admin"
}

func currentUserID(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := userID.(int64)
	return id, ok
}

func parseArticleIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return 0, false
	}
	return id, true
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
		response.InvalidParams(c, "排序设置参数不正确：enabled 必须是布尔值")
		return
	}

	if err := h.settingService.SetSortByPopularity(req.Enabled); err != nil {
		response.InternalErrorWithErr(c, "保存文章排序设置失败，请稍后重试", err)
		return
	}

	response.Success(c, nil)
}

// CreateArticleRequest 创建文章请求
type CreateArticleRequest struct {
	Title              string  `json:"title" binding:"required,min=1,max=200"`
	Content            string  `json:"content" binding:"required,min=10"`
	Summary            string  `json:"summary" binding:"max=500"`
	CategoryID         int64   `json:"category_id" binding:"required"`
	CoverImage         *string `json:"cover_image"`
	Status             string  `json:"status" binding:"required,oneof=draft published"`
	ScheduledPublishAt *string `json:"scheduled_publish_at"`
	CollectionID       *int64  `json:"collection_id"`
	CollectionPosition int     `json:"collection_position"`
	TagIDs             []int64 `json:"tag_ids"`
}

// UpdateArticleRequest 更新文章请求
type UpdateArticleRequest struct {
	Title              string  `json:"title" binding:"required,min=1,max=200"`
	Content            string  `json:"content" binding:"required,min=10"`
	Summary            string  `json:"summary" binding:"max=500"`
	CategoryID         int64   `json:"category_id" binding:"required"`
	CoverImage         *string `json:"cover_image"`
	ScheduledPublishAt *string `json:"scheduled_publish_at"`
	CollectionID       *int64  `json:"collection_id"`
	CollectionPosition int     `json:"collection_position"`
	TagIDs             []int64 `json:"tag_ids"`
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

func parseScheduledPublishAt(value *string) (*time.Time, bool, string) {
	if value == nil {
		return nil, false, ""
	}
	if *value == "" {
		return nil, true, ""
	}

	scheduledTime, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return nil, true, "定时发布时间格式不正确，请使用 RFC3339 格式"
	}
	if scheduledTime.Before(time.Now()) {
		return nil, true, "定时发布时间必须晚于当前时间"
	}
	return &scheduledTime, true, ""
}

func (h *ArticleHandler) hydrateArticleCollection(c *gin.Context, article *model.Article) bool {
	if h.collectionService == nil || article == nil {
		return true
	}
	includeNavigation := article.Status == "published" && !isAdminRequest(c)
	if err := h.collectionService.HydrateArticleCollectionData(article, includeNavigation); err != nil {
		response.InternalErrorWithErr(c, "加载文章所属合集信息失败，请稍后重试", err)
		return false
	}
	return true
}

func (h *ArticleHandler) setArticleCollectionPlacement(c *gin.Context, articleID int64, collectionID *int64, position int) bool {
	if h.collectionService == nil {
		return true
	}
	if collectionID != nil && *collectionID <= 0 {
		collectionID = nil
	}
	if err := h.collectionService.SetPrimaryArticlePlacement(articleID, collectionID, position); err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
		} else if errors.Is(err, svcerrors.ErrCollectionNotFound) {
			response.NotFound(c, "选择的合集不存在或已被删除")
		} else {
			response.InternalErrorWithErr(c, "保存文章合集设置失败，请稍后重试", err)
		}
		return false
	}
	return true
}

func (h *ArticleHandler) setArticleTags(c *gin.Context, article *model.Article, tagIDs []int64) (*model.Article, bool) {
	if tagIDs == nil {
		return article, true
	}
	updated, err := h.articleService.SetTags(article.ID, tagIDs)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
		} else if errors.Is(err, svcerrors.ErrTagNotFound) {
			response.NotFound(c, "选择的标签不存在或已被删除")
		} else {
			response.InternalErrorWithErr(c, "保存文章标签失败，请稍后重试", err)
		}
		return nil, false
	}
	return updated, true
}

// Create 创建文章（管理员）
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "文章参数不正确：标题、正文、分类和状态为必填项，请检查后重试")
		return
	}

	// 处理定时发布
	effectiveStatus := req.Status
	scheduledAt, _, scheduleErr := parseScheduledPublishAt(req.ScheduledPublishAt)
	if scheduleErr != "" {
		response.InvalidParams(c, scheduleErr)
		return
	}
	if scheduledAt != nil {
		effectiveStatus = "draft"
	}

	authorID, _ := c.Get("user_id")

	article, err := h.articleService.Create(
		req.Title,
		req.Content,
		req.Summary,
		req.CategoryID,
		authorID.(int64),
		req.CoverImage,
		effectiveStatus,
	)
	if err != nil {
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "选择的分类不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "创建文章失败，请稍后重试", err)
		return
	}

	// 如果有定时发布，需要单独设置
	if scheduledAt != nil {
		if updateErr := h.articleService.SetScheduledPublishAt(article.ID, scheduledAt); updateErr != nil {
			// 已成功创建文章，但定时设置失败，仅记录
			c.Error(updateErr)
		}
	}

	if updated, ok := h.setArticleTags(c, article, req.TagIDs); !ok {
		return
	} else {
		article = updated
	}

	if !h.setArticleCollectionPlacement(c, article.ID, req.CollectionID, req.CollectionPosition) {
		return
	}
	if !h.hydrateArticleCollection(c, article) {
		return
	}

	response.Success(c, article)
}

// GetByID 根据 ID 获取文章（公开）
func (h *ArticleHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	article, err := h.articleService.GetByID(id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "加载文章详情失败，请稍后重试", err)
		return
	}

	if article.Status != "published" && !isAdminRequest(c) {
		response.NotFound(c, "文章不存在、未发布或已被删除")
		return
	}

	if article.Status == "published" && !isAdminRequest(c) {
		_ = h.articleService.IncrViewCount(article.ID)
	}

	if !h.hydrateArticleCollection(c, article) {
		return
	}

	response.Success(c, article)
}

// GetBySlug 根据 slug 获取文章（公开）
func (h *ArticleHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleService.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "加载文章详情失败，请稍后重试", err)
		return
	}

	if article.Status != "published" {
		response.NotFound(c, "文章不存在、未发布或已被删除")
		return
	}

	_ = h.articleService.IncrViewCount(article.ID)

	if h.statService != nil {
		ip := c.ClientIP()
		ctx := context.WithoutCancel(c.Request.Context())
		async.Go(ctx, zap.L(), "record article view stats", func(ctx context.Context) error {
			if err := h.statService.RecordPVContext(ctx); err != nil {
				return err
			}
			return h.statService.RecordUVContext(ctx, ip)
		})
	}

	if !h.hydrateArticleCollection(c, article) {
		return
	}

	response.Success(c, article)
}

// List 获取文章列表
func (h *ArticleHandler) List(c *gin.Context) {
	status := c.Query("status")
	categoryIDStr := c.Query("category_id")
	tagIDStr := c.Query("tag_id")
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
	tagID, _ := strconv.ParseInt(tagIDStr, 10, 64)

	if status == "" {
		status = "published"
	}

	articles, total, err := h.articleService.List(status, categoryID, tagID, keyword, sortByPopularity, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "文章列表加载失败，请稍后重试", err)
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

// Search 站内搜索已发布文章。
func (h *ArticleHandler) Search(c *gin.Context) {
	keyword := c.Query("q")
	categoryID, _ := strconv.ParseInt(c.Query("category_id"), 10, 64)
	tagID, _ := strconv.ParseInt(c.Query("tag_id"), 10, 64)
	p := pagination.FromQuery(c)

	results, total, err := h.articleService.SearchArticles(keyword, categoryID, tagID, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "文章搜索失败，请稍后重试", err)
		return
	}

	response.Success(c, gin.H{
		"data":       results,
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
	tagIDStr := c.Query("tag_id")
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
	tagID, _ := strconv.ParseInt(tagIDStr, 10, 64)

	articles, total, err := h.articleService.List(status, categoryID, tagID, keyword, sortByPopularity, p.Page, p.PageSize)
	if err != nil {
		response.InternalErrorWithErr(c, "文章管理列表加载失败，请稍后重试", err)
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
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	article, err := h.articleService.ToggleTop(id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法切换置顶状态")
			return
		}
		response.InternalErrorWithErr(c, "切换文章置顶状态失败，请稍后重试", err)
		return
	}

	response.Success(c, article)
}

// UpdatePopularityScores 手动触发更新文章活跃度分数（管理员）
func (h *ArticleHandler) UpdatePopularityScores(c *gin.Context) {
	if err := h.articleService.UpdatePopularityScores(); err != nil {
		response.InternalErrorWithErr(c, "更新文章热度分失败，请稍后重试", err)
		return
	}
	response.Success(c, gin.H{"message": "文章热度分更新成功"})
}

// Update 更新文章（管理员）
func (h *ArticleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "文章参数不正确：标题、正文和分类为必填项，请检查后重试")
		return
	}

	scheduledAt, hasScheduledPublishAt, scheduleErr := parseScheduledPublishAt(req.ScheduledPublishAt)
	if scheduleErr != "" {
		response.InvalidParams(c, scheduleErr)
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
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法更新")
			return
		}
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "选择的分类不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "更新文章失败，请稍后重试", err)
		return
	}

	if hasScheduledPublishAt {
		if scheduledAt != nil && article.Status == "published" {
			if err := h.articleService.Draft(id); err != nil {
				response.InternalErrorWithErr(c, "设置定时发布前转为草稿失败，请稍后重试", err)
				return
			}
		}
		if err := h.articleService.SetScheduledPublishAt(id, scheduledAt); err != nil {
			response.InternalErrorWithErr(c, "保存定时发布时间失败，请稍后重试", err)
			return
		}
		article.ScheduledPublishAt = scheduledAt
		if scheduledAt != nil {
			article.Status = "draft"
		}
	}

	if updated, ok := h.setArticleTags(c, article, req.TagIDs); !ok {
		return
	} else {
		article = updated
	}

	if !h.setArticleCollectionPlacement(c, article.ID, req.CollectionID, req.CollectionPosition) {
		return
	}
	if !h.hydrateArticleCollection(c, article) {
		return
	}

	response.Success(c, article)
}

// Delete 删除文章（管理员）
func (h *ArticleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	if err := h.articleService.Delete(id); err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除")
			return
		}
		response.InternalErrorWithErr(c, "删除文章失败，请稍后重试", err)
		return
	}

	response.Success(c, gin.H{
		"message": "文章删除成功",
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
		response.InternalErrorWithErr(c, "批量删除文章失败：请确认所选文章仍存在且没有关联约束异常", err)
		return
	}
	response.Success(c, gin.H{"message": "文章批量删除成功", "deleted_count": len(ids)})
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
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	if err := h.articleService.Publish(id); err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法发布")
			return
		}
		if errors.Is(err, svcerrors.ErrArticleAlreadyPublished) {
			response.Error(c, response.CodeInvalidParams, "文章已经是发布状态，无需重复发布")
			return
		}
		response.InternalErrorWithErr(c, "发布文章失败，请稍后重试", err)
		return
	}

	response.Success(c, gin.H{
		"message": "文章发布成功",
	})
}

// Draft 转为草稿（管理员）
func (h *ArticleHandler) Draft(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	if err := h.articleService.Draft(id); err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法转为草稿")
			return
		}
		if errors.Is(err, svcerrors.ErrArticleAlreadyDraft) {
			response.Error(c, response.CodeInvalidParams, "文章已经是草稿状态，无需重复操作")
			return
		}
		response.InternalErrorWithErr(c, "文章转为草稿失败，请稍后重试", err)
		return
	}

	response.Success(c, gin.H{
		"message": "文章已转为草稿",
	})
}

// AutoSave 自动保存文章（管理员）
func (h *ArticleHandler) AutoSave(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "文章 ID 无效，请刷新页面后重试")
		return
	}

	var req AutoSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "自动保存参数不正确：标题和正文不能为空")
		return
	}

	if err := h.articleService.AutoSave(id, req.Title, req.Content, req.Summary); err != nil {
		if errors.Is(err, svcerrors.ErrArticleNotFound) {
			response.NotFound(c, "文章不存在或已被删除，无法自动保存")
			return
		}
		response.InternalErrorWithErr(c, "自动保存文章失败，请检查网络后重试", err)
		return
	}

	response.Success(c, nil)
}
