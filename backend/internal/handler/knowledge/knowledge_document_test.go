package knowledge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

type stubKnowledgeDocumentService struct {
	approved *model.KnowledgeDocument
	rejected *model.KnowledgeDocument
	listed   []*model.KnowledgeDocument
	total    int64
	doc      *model.KnowledgeDocument
	sources  []*model.KnowledgeDocumentSource
	deleted  int64
	batchIDs []int64
	filter   repository.KnowledgeDocumentFilter
}

func (s *stubKnowledgeDocumentService) CreateResearchDraft(input service.CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error) {
	return nil, nil
}
func (s *stubKnowledgeDocumentService) Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	return s.approved, nil
}
func (s *stubKnowledgeDocumentService) Reject(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	return s.rejected, nil
}
func (s *stubKnowledgeDocumentService) GetByID(id int64) (*model.KnowledgeDocument, []*model.KnowledgeDocumentSource, error) {
	return s.doc, s.sources, nil
}
func (s *stubKnowledgeDocumentService) List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error) {
	s.filter = filter
	return s.listed, s.total, nil
}
func (s *stubKnowledgeDocumentService) Delete(id int64) error {
	s.deleted = id
	return nil
}
func (s *stubKnowledgeDocumentService) DeleteBatch(ids []int64) error {
	s.batchIDs = append([]int64(nil), ids...)
	return nil
}

func TestKnowledgeDocumentHandlerApprove_ReturnsApprovedDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubKnowledgeDocumentService{approved: &model.KnowledgeDocument{ID: 5, Title: "调研结果", Status: model.KnowledgeDocumentStatusApproved}}
	h := NewKnowledgeDocumentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/knowledge-documents/5/approve", strings.NewReader(`{"review_note":"通过"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "5"}}
	c.Set("user_id", int64(1))

	h.Approve(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestKnowledgeDocumentHandlerDelete_DeletesDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubKnowledgeDocumentService{}
	h := NewKnowledgeDocumentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/admin/knowledge-documents/5", nil)
	c.Params = gin.Params{{Key: "id", Value: "5"}}

	h.Delete(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if svc.deleted != 5 {
		t.Fatalf("expected delete 5, got %d", svc.deleted)
	}
}

func TestKnowledgeDocumentHandlerList_AcceptsCamelCasePageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubKnowledgeDocumentService{}
	h := NewKnowledgeDocumentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/knowledge-documents?page=2&pageSize=10", nil)

	h.List(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if svc.filter.Page != 2 || svc.filter.PageSize != 10 {
		t.Fatalf("expected page 2 pageSize 10, got page %d pageSize %d", svc.filter.Page, svc.filter.PageSize)
	}
}

func TestKnowledgeDocumentHandlerBatchDelete_DeletesSelectedDocuments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubKnowledgeDocumentService{}
	h := NewKnowledgeDocumentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/knowledge-documents/batch-delete", strings.NewReader(`{"ids":[7,8,9]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.BatchDelete(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if len(svc.batchIDs) != 3 || svc.batchIDs[0] != 7 || svc.batchIDs[2] != 9 {
		t.Fatalf("expected batch delete ids [7 8 9], got %#v", svc.batchIDs)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	if data["deleted_count"].(float64) != 3 {
		t.Fatalf("expected deleted_count 3, got %#v", data)
	}
}
