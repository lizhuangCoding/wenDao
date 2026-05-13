package knowledge

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/pagination"
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

// List 获取知识文档列表（管理员）
func (h *KnowledgeDocumentHandler) List(c *gin.Context) {
	status := c.Query("status")
	keyword := c.Query("keyword")
	p := pagination.FromQuery(c)

	docs, total, err := h.service.List(repository.KnowledgeDocumentFilter{
		Status:   status,
		Keyword:  keyword,
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "获取知识文档列表失败", err)
		return
	}

	response.Success(c, gin.H{
		"data":       docs,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
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
		response.InternalErrorWithErr(c, "获取知识文档详情失败", err)
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
		response.InternalErrorWithErr(c, "审核通过失败", err)
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
		response.InternalErrorWithErr(c, "拒绝知识文档失败", err)
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
		response.InternalErrorWithErr(c, "删除知识文档失败", err)
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
		response.InternalErrorWithErr(c, "批量删除知识文档失败", err)
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
