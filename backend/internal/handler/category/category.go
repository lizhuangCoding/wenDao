package category

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
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

// Create 创建分类（管理员）
func (h *CategoryHandler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	category, err := h.categoryService.Create(req.Name, req.Slug, req.Description, req.SortOrder)
	if err != nil {
		if err.Error() == "slug already exists" {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalError(c, "Failed to create category")
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
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalError(c, "Failed to get category")
		return
	}

	response.Success(c, category)
}

// GetBySlug 根据 slug 获取分类
func (h *CategoryHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.categoryService.GetBySlug(slug)
	if err != nil {
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		response.InternalError(c, "Failed to get category")
		return
	}

	response.Success(c, category)
}

// List 获取所有分类
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categoryService.List()
	if err != nil {
		response.InternalError(c, "Failed to list categories")
		return
	}

	response.Success(c, categories)
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
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		if err.Error() == "slug already exists" {
			response.Error(c, response.CodeInvalidParams, "Slug already exists")
			return
		}
		response.InternalError(c, "Failed to update category")
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
		if err.Error() == "category not found" {
			response.NotFound(c, "Category not found")
			return
		}
		if err.Error() == "cannot delete category with articles" {
			response.Error(c, response.CodeInvalidParams, "Cannot delete category with articles")
			return
		}
		response.InternalError(c, "Failed to delete category")
		return
	}

	response.Success(c, gin.H{
		"message": "Category deleted successfully",
	})
}
