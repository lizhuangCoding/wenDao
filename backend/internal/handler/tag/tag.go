package tag

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

// TagHandler 标签处理器
type TagHandler struct {
	tagService service.TagService
}

func NewTagHandler(tagService service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

type CreateTagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
	Slug string `json:"slug" binding:"required,min=1,max=50"`
}

type UpdateTagRequest struct {
	Name string `json:"name" binding:"required,min=1,max=50"`
	Slug string `json:"slug" binding:"required,min=1,max=50"`
}

type BatchDeleteTagRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

func (h *TagHandler) Create(c *gin.Context) {
	var req CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	tag, err := h.tagService.Create(req.Name, req.Slug)
	if err != nil {
		if errors.Is(err, svcerrors.ErrSlugAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalErrorWithErr(c, "Failed to create tag", err)
		return
	}
	response.Success(c, tag)
}

func (h *TagHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid tag ID")
		return
	}
	tag, err := h.tagService.GetByID(id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrTagNotFound) {
			response.NotFound(c, "Tag not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get tag", err)
		return
	}
	response.Success(c, tag)
}

func (h *TagHandler) GetBySlug(c *gin.Context) {
	tag, err := h.tagService.GetBySlug(c.Param("slug"))
	if err != nil {
		if errors.Is(err, svcerrors.ErrTagNotFound) {
			response.NotFound(c, "Tag not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get tag", err)
		return
	}
	response.Success(c, tag)
}

func (h *TagHandler) List(c *gin.Context) {
	tags, err := h.tagService.List()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list tags", err)
		return
	}
	response.Success(c, tags)
}

func (h *TagHandler) AdminList(c *gin.Context) {
	p := pagination.FromQuery(c)
	tags, total, err := h.tagService.ListPaginated(repository.TagFilter{
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list tags", err)
		return
	}
	response.Success(c, gin.H{
		"data":       tags,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

func (h *TagHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid tag ID")
		return
	}
	var req UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}
	tag, err := h.tagService.Update(id, req.Name, req.Slug)
	if err != nil {
		if errors.Is(err, svcerrors.ErrTagNotFound) {
			response.NotFound(c, "Tag not found")
			return
		}
		if errors.Is(err, svcerrors.ErrSlugAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalErrorWithErr(c, "Failed to update tag", err)
		return
	}
	response.Success(c, tag)
}

func (h *TagHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid tag ID")
		return
	}
	if err := h.tagService.Delete(id); err != nil {
		if errors.Is(err, svcerrors.ErrTagNotFound) {
			response.NotFound(c, "Tag not found")
			return
		}
		if errors.Is(err, svcerrors.ErrCannotDeleteTagWithArticles) {
			response.Error(c, response.CodeInvalidParams, "Cannot delete tag with articles")
			return
		}
		response.InternalErrorWithErr(c, "Failed to delete tag", err)
		return
	}
	response.Success(c, gin.H{"message": "Tag deleted successfully"})
}

func (h *TagHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的标签")
		return
	}
	ids, ok := normalizeTagIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "标签 ID 无效")
		return
	}
	if err := h.tagService.DeleteBatch(ids); err != nil {
		if errors.Is(err, svcerrors.ErrCannotDeleteTagWithArticles) {
			response.Error(c, response.CodeInvalidParams, "Cannot delete tag with articles")
			return
		}
		response.InternalErrorWithErr(c, "批量删除标签失败", err)
		return
	}
	response.Success(c, gin.H{"message": "Tags deleted successfully", "deleted_count": len(ids)})
}

func normalizeTagIDs(ids []int64) ([]int64, bool) {
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
