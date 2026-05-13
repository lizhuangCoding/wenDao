package knowledge

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

// KnowledgeDocumentHandler 知识文档处理器
type KnowledgeDocumentHandler struct {
	service service.KnowledgeDocumentService
}

// NewKnowledgeDocumentHandler 创建知识文档处理器实例
func NewKnowledgeDocumentHandler(service service.KnowledgeDocumentService) *KnowledgeDocumentHandler {
	return &KnowledgeDocumentHandler{service: service}
}

type reviewKnowledgeDocumentRequest struct {
	ReviewNote string `json:"review_note"`
}

type batchDeleteKnowledgeDocumentRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

func parseInt64Param(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid ID")
		return 0, false
	}
	return id, true
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

// List 获取知识文档列表（管理员）
func (h *KnowledgeDocumentHandler) List(c *gin.Context) {
	status := c.Query("status")
	keyword := c.Query("keyword")
	page, pageSize := parsePaginationQuery(c)

	docs, total, err := h.service.List(repository.KnowledgeDocumentFilter{
		Status:   status,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "获取知识文档列表失败")
		return
	}

	response.Success(c, gin.H{
		"data":       docs,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

// Get 获取知识文档详情（管理员）
func (h *KnowledgeDocumentHandler) Get(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, sources, err := h.service.GetByID(id)
	if err != nil {
		response.InternalError(c, "获取知识文档详情失败")
		return
	}
	response.Success(c, gin.H{"document": doc, "sources": sources})
}

// Approve 审核通过知识文档
func (h *KnowledgeDocumentHandler) Approve(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req reviewKnowledgeDocumentRequest
	_ = c.ShouldBindJSON(&req)
	uid, _ := c.Get("user_id")
	doc, err := h.service.Approve(id, uid.(int64), req.ReviewNote)
	if err != nil {
		response.InternalError(c, "审核通过失败")
		return
	}
	response.Success(c, doc)
}

// Reject 拒绝知识文档
func (h *KnowledgeDocumentHandler) Reject(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req reviewKnowledgeDocumentRequest
	_ = c.ShouldBindJSON(&req)
	uid, _ := c.Get("user_id")
	doc, err := h.service.Reject(id, uid.(int64), req.ReviewNote)
	if err != nil {
		response.InternalError(c, "拒绝知识文档失败")
		return
	}
	response.Success(c, doc)
}

// Delete 删除知识文档，并同步删除由它生成的文章
func (h *KnowledgeDocumentHandler) Delete(c *gin.Context) {
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(id); err != nil {
		response.InternalError(c, "删除知识文档失败")
		return
	}
	response.Success(c, gin.H{"message": "知识文档已删除"})
}

// BatchDelete 批量删除知识文档，并复用单条删除的级联清理逻辑
func (h *KnowledgeDocumentHandler) BatchDelete(c *gin.Context) {
	var req batchDeleteKnowledgeDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的知识文档")
		return
	}
	ids, ok := normalizeKnowledgeDocumentIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "知识文档 ID 无效")
		return
	}
	if err := h.service.DeleteBatch(ids); err != nil {
		response.InternalError(c, "批量删除知识文档失败")
		return
	}
	response.Success(c, gin.H{"message": "知识文档已删除", "deleted_count": len(ids)})
}

func normalizeKnowledgeDocumentIDs(ids []int64) ([]int64, bool) {
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
