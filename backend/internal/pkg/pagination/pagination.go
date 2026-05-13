package pagination

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Pagination struct {
	Page     int
	PageSize int
}

func FromQuery(c *gin.Context) Pagination {
	page := parsePositiveInt(c.Query("page"), DefaultPage)
	pageSize := c.Query("page_size")
	if pageSize == "" {
		pageSize = c.Query("pageSize")
	}
	return Pagination{
		Page:     page,
		PageSize: normalizePageSize(parsePositiveInt(pageSize, DefaultPageSize)),
	}
}

func TotalPages(total int64, pageSize int) int {
	pageSize = normalizePageSize(pageSize)
	if total <= 0 {
		return 1
	}
	return int(math.Ceil(float64(total) / float64(pageSize)))
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
		return DefaultPageSize
	}
	if pageSize > MaxPageSize {
		return MaxPageSize
	}
	return pageSize
}
