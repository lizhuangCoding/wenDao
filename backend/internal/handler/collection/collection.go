package collection

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/repository"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

type CollectionHandler struct {
	collectionService service.CollectionService
}

func NewCollectionHandler(collectionService service.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionService: collectionService}
}

type CollectionRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Slug        string `json:"slug" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status" binding:"omitempty,oneof=active hidden"`
}

type BatchDeleteCollectionRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

func (h *CollectionHandler) List(c *gin.Context) {
	collections, err := h.collectionService.List()
	if err != nil {
		response.InternalErrorWithErr(c, "合集列表加载失败，请稍后重试", err)
		return
	}
	response.Success(c, collections)
}

func (h *CollectionHandler) AdminList(c *gin.Context) {
	p := pagination.FromQuery(c)
	collections, total, err := h.collectionService.ListPaginated(repository.CollectionFilter{
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "合集管理列表加载失败，请稍后重试", err)
		return
	}
	response.Success(c, gin.H{
		"data":       collections,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

func (h *CollectionHandler) Create(c *gin.Context) {
	var req CollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "合集参数不正确：名称和 slug 为必填项，状态只能为 active 或 hidden")
		return
	}
	collection, err := h.collectionService.Create(req.Name, req.Slug, req.Description, req.SortOrder, req.Status)
	if err != nil {
		handleCollectionError(c, err, "创建合集失败，请稍后重试")
		return
	}
	response.Success(c, collection)
}

func (h *CollectionHandler) Update(c *gin.Context) {
	id, ok := parseCollectionID(c)
	if !ok {
		return
	}
	var req CollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "合集参数不正确：名称和 slug 为必填项，状态只能为 active 或 hidden")
		return
	}
	collection, err := h.collectionService.Update(id, req.Name, req.Slug, req.Description, req.SortOrder, req.Status)
	if err != nil {
		handleCollectionError(c, err, "更新合集失败，请稍后重试")
		return
	}
	response.Success(c, collection)
}

func (h *CollectionHandler) Delete(c *gin.Context) {
	id, ok := parseCollectionID(c)
	if !ok {
		return
	}
	if err := h.collectionService.Delete(id); err != nil {
		handleCollectionError(c, err, "删除合集失败，请稍后重试")
		return
	}
	response.Success(c, gin.H{"message": "合集删除成功"})
}

func (h *CollectionHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的合集")
		return
	}
	ids, ok := normalizeCollectionIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "合集 ID 无效")
		return
	}
	if err := h.collectionService.DeleteBatch(ids); err != nil {
		handleCollectionError(c, err, "批量删除合集失败：请确认所选合集仍存在且没有关联文章")
		return
	}
	response.Success(c, gin.H{"message": "合集批量删除成功", "deleted_count": len(ids)})
}

func parseCollectionID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "合集 ID 无效，请刷新页面后重试")
		return 0, false
	}
	return id, true
}

func normalizeCollectionIDs(ids []int64) ([]int64, bool) {
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

func handleCollectionError(c *gin.Context, err error, fallback string) {
	if errors.Is(err, svcerrors.ErrCollectionNotFound) {
		response.NotFound(c, "合集不存在或已被删除")
	} else if errors.Is(err, svcerrors.ErrSlugAlreadyExists) {
		response.Error(c, response.CodeInvalidParams, "合集别名已存在，请更换 slug")
	} else if errors.Is(err, svcerrors.ErrInvalidCollectionStatus) {
		response.InvalidParams(c, "合集状态无效，只能设置为 active 或 hidden")
	} else if errors.Is(err, svcerrors.ErrCannotDeleteCollectionWithArticles) {
		response.Error(c, response.CodeInvalidParams, "该合集下仍有关联文章，不能删除")
	} else {
		response.InternalErrorWithErr(c, fallback, err)
	}
}
