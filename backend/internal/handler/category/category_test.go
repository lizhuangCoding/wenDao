package category

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type stubCategoryService struct {
	filter repository.CategoryFilter
	total  int64
	ids    []int64
}

func (s *stubCategoryService) Create(name, slug, description string, sortOrder int) (*model.Category, error) {
	return nil, nil
}

func (s *stubCategoryService) GetByID(id int64) (*model.Category, error) {
	return nil, nil
}

func (s *stubCategoryService) GetBySlug(slug string) (*model.Category, error) {
	return nil, nil
}

func (s *stubCategoryService) List() ([]*model.Category, error) {
	return nil, nil
}

func (s *stubCategoryService) ListPaginated(filter repository.CategoryFilter) ([]*model.Category, int64, error) {
	s.filter = filter
	return []*model.Category{}, s.total, nil
}

func (s *stubCategoryService) Update(id int64, name, slug, description string, sortOrder int) (*model.Category, error) {
	return nil, nil
}

func (s *stubCategoryService) Delete(id int64) error {
	return nil
}

func (s *stubCategoryService) DeleteBatch(ids []int64) error {
	s.ids = ids
	return nil
}

func TestCategoryHandlerAdminList_ReturnsPaginatedCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubCategoryService{total: 23}
	h := NewCategoryHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/categories?page=2&pageSize=10", nil)

	h.AdminList(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if svc.filter.Page != 2 || svc.filter.PageSize != 10 {
		t.Fatalf("expected page 2 pageSize 10, got page %d pageSize %d", svc.filter.Page, svc.filter.PageSize)
	}

	var payload struct {
		Data struct {
			TotalPages int `json:"totalPages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Data.TotalPages != 3 {
		t.Fatalf("expected totalPages 3, got %d", payload.Data.TotalPages)
	}
}

func TestCategoryHandlerBatchDelete_DeletesSelectedCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubCategoryService{}
	h := NewCategoryHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/categories/batch-delete", bytes.NewBufferString(`{"ids":[1,2,2,3]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	expected := []int64{1, 2, 3}
	if len(svc.ids) != len(expected) {
		t.Fatalf("expected ids %v, got %v", expected, svc.ids)
	}
	for i := range expected {
		if svc.ids[i] != expected[i] {
			t.Fatalf("expected ids %v, got %v", expected, svc.ids)
		}
	}
}
