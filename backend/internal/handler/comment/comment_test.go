package comment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
)

type stubCommentService struct {
	listPage     int
	listPageSize int
	total        int64
}

func (s *stubCommentService) Create(articleID, userID int64, content string, parentID, replyToUserID *int64) (*model.Comment, error) {
	return nil, nil
}

func (s *stubCommentService) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	return nil, nil
}

func (s *stubCommentService) ListAll(page, pageSize int) ([]*model.Comment, int64, error) {
	s.listPage = page
	s.listPageSize = pageSize
	return []*model.Comment{}, s.total, nil
}

func (s *stubCommentService) Delete(id, userID int64, isAdmin bool) error {
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
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/comments?page=2&pageSize=15", nil)

	h.AdminList(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if commentSvc.listPage != 2 || commentSvc.listPageSize != 15 {
		t.Fatalf("expected page 2 pageSize 15, got page %d pageSize %d", commentSvc.listPage, commentSvc.listPageSize)
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
