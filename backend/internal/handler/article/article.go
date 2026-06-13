package article

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
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
		response.InvalidParams(c, "Invalid article ID")
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
	Title              string  `json:"title" binding:"required,min=1,max=200"`
	Content            string  `json:"content" binding:"required,min=10"`
	Summary            string  `json:"summary" binding:"max=500"`
	CategoryID         int64   `json:"category_id" binding:"required"`
	CoverImage         *string `json:"cover_image"`
	Status             string  `json:"status" binding:"required,oneof=draft published"`
	ScheduledPublishAt *string `json:"scheduled_publish_at"`
	CollectionID       *int64  `json:"collection_id"`
	CollectionPosition int     `json:"collection_position"`
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
}

type BatchDeleteArticleRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

type ArticleOrbitCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ArticleOrbitCollection struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Position int    `json:"position"`
}

type ArticleOrbitSemanticPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ArticleOrbitSemanticNeighbor struct {
	ArticleID int64   `json:"article_id"`
	Score     float64 `json:"score"`
}

type ArticleOrbitItem struct {
	ID                int64                          `json:"id"`
	Title             string                         `json:"title"`
	Slug              string                         `json:"slug"`
	Summary           string                         `json:"summary"`
	CoverImage        *string                        `json:"cover_image,omitempty"`
	ViewCount         int                            `json:"view_count"`
	CommentCount      int                            `json:"comment_count"`
	IsTop             bool                           `json:"is_top"`
	SourceType        string                         `json:"source_type"`
	Category          *ArticleOrbitCategory          `json:"category,omitempty"`
	Collection        *ArticleOrbitCollection        `json:"collection,omitempty"`
	SemanticPosition  *ArticleOrbitSemanticPosition  `json:"semantic_position,omitempty"`
	SemanticNeighbors []ArticleOrbitSemanticNeighbor `json:"semantic_neighbors,omitempty"`
	CreatedAt         string                         `json:"created_at"`
	PublishedAt       string                         `json:"published_at"`
}

func toArticleOrbitItem(article *model.Article) ArticleOrbitItem {
	publishedAt := article.CreatedAt
	if article.PublishedAt != nil {
		publishedAt = *article.PublishedAt
	}
	item := ArticleOrbitItem{
		ID:           article.ID,
		Title:        article.Title,
		Slug:         article.Slug,
		Summary:      article.Summary,
		CoverImage:   article.CoverImage,
		ViewCount:    article.ViewCount,
		CommentCount: article.CommentCount,
		IsTop:        article.IsTop,
		SourceType:   article.SourceType,
		CreatedAt:    article.CreatedAt.Format(time.RFC3339),
		PublishedAt:  publishedAt.Format(time.RFC3339),
	}
	if article.Category != nil {
		item.Category = &ArticleOrbitCategory{
			ID:   article.Category.ID,
			Name: article.Category.Name,
			Slug: article.Category.Slug,
		}
	}
	if article.CollectionMembership != nil {
		item.Collection = &ArticleOrbitCollection{
			ID:       article.CollectionMembership.CollectionID,
			Name:     article.CollectionMembership.Name,
			Slug:     article.CollectionMembership.Slug,
			Position: article.CollectionMembership.Position,
		}
	}
	if article.SemanticProfile != nil {
		item.SemanticPosition = &ArticleOrbitSemanticPosition{
			X: article.SemanticProfile.MapX,
			Y: article.SemanticProfile.MapY,
			Z: article.SemanticProfile.MapZ,
		}
		if neighbors, err := article.SemanticProfile.Neighbors(); err == nil && len(neighbors) > 0 {
			item.SemanticNeighbors = make([]ArticleOrbitSemanticNeighbor, 0, len(neighbors))
			for _, neighbor := range neighbors {
				item.SemanticNeighbors = append(item.SemanticNeighbors, ArticleOrbitSemanticNeighbor{
					ArticleID: neighbor.ArticleID,
					Score:     neighbor.Score,
				})
			}
		}
	}
	return item
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
		return nil, true, "Invalid scheduled_publish_at format, use RFC3339"
	}
	if scheduledTime.Before(time.Now()) {
		return nil, true, "scheduled_publish_at must be in the future"
	}
	return &scheduledTime, true, ""
}

func (h *ArticleHandler) hydrateArticleCollection(c *gin.Context, article *model.Article) bool {
	if h.collectionService == nil || article == nil {
		return true
	}
	includeNavigation := article.Status == "published" && !isAdminRequest(c)
	if err := h.collectionService.HydrateArticleCollectionData(article, includeNavigation); err != nil {
		response.InternalErrorWithErr(c, "Failed to get article collection data", err)
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
		switch err.Error() {
		case "article not found":
			response.NotFound(c, "Article not found")
		case "collection not found":
			response.NotFound(c, "Collection not found")
		default:
			response.InternalErrorWithErr(c, "Failed to set article collection", err)
		}
		return false
	}
	return true
}

// Create 创建文章（管理员）
func (h *ArticleHandler) Create(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
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
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to create article", err)
		return
	}

	// 如果有定时发布，需要单独设置
	if scheduledAt != nil {
		if updateErr := h.articleService.SetScheduledPublishAt(article.ID, scheduledAt); updateErr != nil {
			// 已成功创建文章，但定时设置失败，仅记录
			c.Error(updateErr)
		}
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

	if !h.hydrateArticleCollection(c, article) {
		return
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

// ListOrbitArticles 获取首页 3D 文章星球所需的轻量文章列表。
func (h *ArticleHandler) ListOrbitArticles(c *gin.Context) {
	articles, err := h.articleService.ListOrbitArticles()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list orbit articles", err)
		return
	}

	items := make([]ArticleOrbitItem, 0, len(articles))
	for _, article := range articles {
		if article == nil {
			continue
		}
		if h.collectionService != nil {
			if err := h.collectionService.HydrateArticleCollectionData(article, false); err != nil {
				response.InternalErrorWithErr(c, "Failed to get article collection data", err)
				return
			}
		}
		items = append(items, toArticleOrbitItem(article))
	}

	response.Success(c, gin.H{
		"data":  items,
		"total": len(items),
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

	if hasScheduledPublishAt {
		if scheduledAt != nil && article.Status == "published" {
			if err := h.articleService.Draft(id); err != nil {
				response.InternalErrorWithErr(c, "Failed to draft article for scheduled publish", err)
				return
			}
		}
		if err := h.articleService.SetScheduledPublishAt(id, scheduledAt); err != nil {
			response.InternalErrorWithErr(c, "Failed to set scheduled publish time", err)
			return
		}
		article.ScheduledPublishAt = scheduledAt
		if scheduledAt != nil {
			article.Status = "draft"
		}
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

func (h *ArticleHandler) GetInteraction(c *gin.Context) {
	id, ok := parseArticleIDParam(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	state, err := h.articleService.GetArticleInteractionState(userID, id)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get article interaction state", err)
		return
	}
	response.Success(c, state)
}

func (h *ArticleHandler) Like(c *gin.Context) {
	h.updateInteraction(c, h.articleService.LikeArticleForUser)
}

func (h *ArticleHandler) Unlike(c *gin.Context) {
	h.updateInteraction(c, h.articleService.UnlikeArticleForUser)
}

func (h *ArticleHandler) Favorite(c *gin.Context) {
	h.updateInteraction(c, h.articleService.FavoriteArticleForUser)
}

func (h *ArticleHandler) Unfavorite(c *gin.Context) {
	h.updateInteraction(c, h.articleService.UnfavoriteArticleForUser)
}

func (h *ArticleHandler) updateInteraction(
	c *gin.Context,
	update func(userID, articleID int64) (*model.ArticleInteractionState, error),
) {
	id, ok := parseArticleIDParam(c)
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}

	state, err := update(userID, id)
	if err != nil {
		if err.Error() == "article not found" {
			response.NotFound(c, "Article not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to update article interaction", err)
		return
	}
	response.Success(c, state)
}

func (h *ArticleHandler) ListLikedArticles(c *gin.Context) {
	h.listInteractedArticles(c, model.ArticleInteractionTypeLike)
}

func (h *ArticleHandler) ListFavoriteArticles(c *gin.Context) {
	h.listInteractedArticles(c, model.ArticleInteractionTypeFavorite)
}

func (h *ArticleHandler) listInteractedArticles(c *gin.Context, interactionType string) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	p := pagination.FromQuery(c)

	articles, total, err := h.articleService.ListArticlesByInteraction(userID, interactionType, p.Page, p.PageSize)
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
