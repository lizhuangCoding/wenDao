package category

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

// CategoryHandler 分类处理器
type CategoryHandler struct {
	categoryService service.CategoryService
}

// NewCategoryHandler 创建分类处理器实例
func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=50"`
	Slug        string `json:"slug" binding:"required,min=1,max=50"`
	Description string `json:"description" binding:"max=200"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=50"`
	Slug        string `json:"slug" binding:"required,min=1,max=50"`
	Description string `json:"description" binding:"max=200"`
	SortOrder   int    `json:"sort_order"`
}

type BatchDeleteCategoryRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

// Create 创建分类（管理员）
func (h *CategoryHandler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	category, err := h.categoryService.Create(req.Name, req.Slug, req.Description, req.SortOrder)
	if err != nil {
		if errors.Is(err, svcerrors.ErrSlugAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalErrorWithErr(c, "Failed to create category", err)
		return
	}

	response.Success(c, category)
}

// GetByID 根据 ID 获取分类
func (h *CategoryHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid category ID")
		return
	}

	category, err := h.categoryService.GetByID(id)
	if err != nil {
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get category", err)
		return
	}

	response.Success(c, category)
}

// GetBySlug 根据 slug 获取分类
func (h *CategoryHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.categoryService.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalErrorWithErr(c, "Failed to get category", err)
		return
	}

	response.Success(c, category)
}

// List 获取所有分类
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categoryService.List()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list categories", err)
		return
	}

	response.Success(c, categories)
}

// AdminList 获取分类分页列表（管理员）
func (h *CategoryHandler) AdminList(c *gin.Context) {
	p := pagination.FromQuery(c)
	categories, total, err := h.categoryService.ListPaginated(repository.CategoryFilter{
		Page:     p.Page,
		PageSize: p.PageSize,
	})
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list categories", err)
		return
	}

	response.Success(c, gin.H{
		"data":       categories,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

// Update 更新分类（管理员）
func (h *CategoryHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid category ID")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	category, err := h.categoryService.Update(id, req.Name, req.Slug, req.Description, req.SortOrder)
	if err != nil {
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "Category not found")
			return
		}
		if errors.Is(err, svcerrors.ErrSlugAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalErrorWithErr(c, "Failed to update category", err)
		return
	}

	response.Success(c, category)
}

// Delete 删除分类（管理员）
func (h *CategoryHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid category ID")
		return
	}

	if err := h.categoryService.Delete(id); err != nil {
		if errors.Is(err, svcerrors.ErrCategoryNotFound) {
			response.NotFound(c, "Category not found")
			return
		}
		if errors.Is(err, svcerrors.ErrCannotDeleteCategoryWithArticles) {
			response.Error(c, response.CodeInvalidParams, "Cannot delete category with articles")
			return
		}
		response.InternalErrorWithErr(c, "Failed to delete category", err)
		return
	}

	response.Success(c, gin.H{
		"message": "Category deleted successfully",
	})
}

// BatchDelete 批量删除分类（管理员）
func (h *CategoryHandler) BatchDelete(c *gin.Context) {
	var req BatchDeleteCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "请选择要删除的分类")
		return
	}
	ids, ok := normalizeCategoryIDs(req.IDs)
	if !ok {
		response.InvalidParams(c, "分类 ID 无效")
		return
	}
	if err := h.categoryService.DeleteBatch(ids); err != nil {
		if errors.Is(err, svcerrors.ErrCannotDeleteCategoryWithArticles) {
			response.Error(c, response.CodeInvalidParams, "Cannot delete category with articles")
			return
		}
		response.InternalErrorWithErr(c, "批量删除分类失败", err)
		return
	}
	response.Success(c, gin.H{"message": "Categories deleted successfully", "deleted_count": len(ids)})
}

func normalizeCategoryIDs(ids []int64) ([]int64, bool) {
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
