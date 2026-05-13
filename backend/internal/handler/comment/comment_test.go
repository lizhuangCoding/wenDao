package comment

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

type stubCommentService struct {
	listPage     int
	listPageSize int
	listStatus   string
	listKeyword  string
	total        int64
	batchIDs     []int64
}

func (s *stubCommentService) Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error) {
	return nil, nil
}

func (s *stubCommentService) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	return nil, nil
}

func (s *stubCommentService) ListAll(filter repository.CommentFilter) ([]*model.Comment, int64, error) {
	s.listPage = filter.Page
	s.listPageSize = filter.PageSize
	s.listStatus = filter.Status
	s.listKeyword = filter.Keyword
	return []*model.Comment{}, s.total, nil
}

func (s *stubCommentService) Delete(id, userID int64, isAdmin bool) error {
	return nil
}

func (s *stubCommentService) DeleteBatch(ids []int64, userID int64, isAdmin bool) error {
	s.batchIDs = ids
	return nil
}

func (s *stubCommentService) Restore(id int64) error {
	return nil
}

func TestCommentHandlerAdminList_ReturnsTotalPagesAndAcceptsCamelCasePageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	commentSvc := &stubCommentService{total: 31}
	h := NewCommentHandler(commentSvc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/comments?page=2&pageSize=15&status=deleted&keyword=hello", nil)

	h.AdminList(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if commentSvc.listPage != 2 || commentSvc.listPageSize != 15 {
		t.Fatalf("expected page 2 pageSize 15, got page %d pageSize %d", commentSvc.listPage, commentSvc.listPageSize)
	}
	if commentSvc.listStatus != "deleted" || commentSvc.listKeyword != "hello" {
		t.Fatalf("expected status deleted keyword hello, got status %q keyword %q", commentSvc.listStatus, commentSvc.listKeyword)
	}

	var payload struct {
		Code int `json:"code"`
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

func TestCommentHandlerBatchDelete_DeletesSelectedComments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	commentSvc := &stubCommentService{}
	h := NewCommentHandler(commentSvc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/comments/batch-delete", bytes.NewBufferString(`{"ids":[7,8,8,9]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", int64(1))
	c.Set("user_role", "admin")

	h.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	expected := []int64{7, 8, 9}
	if len(commentSvc.batchIDs) != len(expected) {
		t.Fatalf("expected ids %v, got %v", expected, commentSvc.batchIDs)
	}
	for i := range expected {
		if commentSvc.batchIDs[i] != expected[i] {
			t.Fatalf("expected ids %v, got %v", expected, commentSvc.batchIDs)
		}
	}
}
