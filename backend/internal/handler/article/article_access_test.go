package article

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
)

type stubArticleService struct {
	articleByID      *model.Article
	articleBySlug    *model.Article
	incrViewCountIDs []int64
	listPage         int
	listPageSize     int
	batchIDs         []int64
}

func (s *stubArticleService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	return nil, nil
}
func (s *stubArticleService) GetByID(id int64) (*model.Article, error) { return s.articleByID, nil }
func (s *stubArticleService) GetBySlug(slug string) (*model.Article, error) {
	return s.articleBySlug, nil
}
func (s *stubArticleService) List(status string, categoryID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
	s.listPage = page
	s.listPageSize = pageSize
	return nil, 0, nil
}
func (s *stubArticleService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	return nil, nil
}
func (s *stubArticleService) Delete(id int64) error { return nil }
func (s *stubArticleService) DeleteBatch(ids []int64) error {
	s.batchIDs = ids
	return nil
}
func (s *stubArticleService) Publish(id int64) error                                  { return nil }
func (s *stubArticleService) Draft(id int64) error                                    { return nil }
func (s *stubArticleService) AutoSave(id int64, title, content, summary string) error { return nil }
func (s *stubArticleService) IncrViewCount(id int64) error {
	s.incrViewCountIDs = append(s.incrViewCountIDs, id)
	return nil
}
func (s *stubArticleService) LikeArticle(id int64) error                 { return nil }
func (s *stubArticleService) UnlikeArticle(id int64) error               { return nil }
func (s *stubArticleService) ToggleTop(id int64) (*model.Article, error) { return nil, nil }
func (s *stubArticleService) UpdatePopularityScores() error              { return nil }

type stubSettingService struct{}

func (s *stubSettingService) GetSortByPopularity() bool              { return false }
func (s *stubSettingService) SetSortByPopularity(enabled bool) error { return nil }

func TestArticleHandlerGetByID_HidesDraftFromPublicRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		articleByID: &model.Article{ID: 18, Status: "draft", Title: "secret"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles/18", nil)
	c.Params = gin.Params{{Key: "id", Value: "18"}}

	h.GetByID(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for public draft access, got %d", w.Code)
	}
	if len(articleSvc.incrViewCountIDs) != 0 {
		t.Fatalf("expected no view count increments for hidden draft, got %v", articleSvc.incrViewCountIDs)
	}
}

func TestArticleHandlerGetByID_AllowsAdminToReadDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{
		articleByID: &model.Article{ID: 18, Status: "draft", Title: "secret"},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/articles/18", nil)
	c.Params = gin.Params{{Key: "id", Value: "18"}}
	c.Set("user_role", "admin")

	h.GetByID(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for admin draft access, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body["code"].(float64) != 0 {
		t.Fatalf("expected success response, got %#v", body)
	}
	if len(articleSvc.incrViewCountIDs) != 0 {
		t.Fatalf("expected admin article fetch to avoid public view increments, got %v", articleSvc.incrViewCountIDs)
	}
}

func TestArticleHandlerList_AcceptsCamelCasePageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles?page=3&pageSize=9", nil)

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if articleSvc.listPage != 3 || articleSvc.listPageSize != 9 {
		t.Fatalf("expected page 3 pageSize 9, got page %d pageSize %d", articleSvc.listPage, articleSvc.listPageSize)
	}
}

func TestArticleHandlerBatchDelete_DeletesSelectedArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	articleSvc := &stubArticleService{}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/articles/batch-delete", bytes.NewBufferString(`{"ids":[4,5,5,6]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	expected := []int64{4, 5, 6}
	if len(articleSvc.batchIDs) != len(expected) {
		t.Fatalf("expected ids %v, got %v", expected, articleSvc.batchIDs)
	}
	for i := range expected {
		if articleSvc.batchIDs[i] != expected[i] {
			t.Fatalf("expected ids %v, got %v", expected, articleSvc.batchIDs)
		}
	}

	var payload struct {
		Data struct {
			DeletedCount int `json:"deleted_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Data.DeletedCount != 3 {
		t.Fatalf("expected deleted_count 3, got %d", payload.Data.DeletedCount)
	}
}
