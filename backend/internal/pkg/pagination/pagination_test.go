package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFromQueryAcceptsSnakeAndCamelPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		target   string
		page     int
		pageSize int
	}{
		{name: "snake case", target: "/items?page=3&page_size=12", page: 3, pageSize: 12},
		{name: "camel case", target: "/items?page=4&pageSize=9", page: 4, pageSize: 9},
		{name: "fallbacks", target: "/items?page=-1&pageSize=0", page: 1, pageSize: 20},
		{name: "max page size", target: "/items?page=2&page_size=500", page: 2, pageSize: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.target, nil)

			got := FromQuery(c)

			if got.Page != tt.page || got.PageSize != tt.pageSize {
				t.Fatalf("expected page=%d pageSize=%d, got page=%d pageSize=%d", tt.page, tt.pageSize, got.Page, got.PageSize)
			}
		})
	}
}

func TestTotalPagesReturnsAtLeastOnePage(t *testing.T) {
	if got := TotalPages(0, 20); got != 1 {
		t.Fatalf("expected empty result totalPages 1, got %d", got)
	}
	if got := TotalPages(31, 15); got != 3 {
		t.Fatalf("expected totalPages 3, got %d", got)
	}
}
