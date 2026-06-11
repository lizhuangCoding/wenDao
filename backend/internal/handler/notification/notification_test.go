package notification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
)

type stubNotificationService struct {
	listUserID   int64
	listType     string
	listPage     int
	listPageSize int
	total        int64
}

func (s *stubNotificationService) Create(userID int64, notifType, title, content, linkURL string) error {
	return nil
}

func (s *stubNotificationService) ListByUser(userID int64, notifType string, page, pageSize int) ([]*model.Notification, int64, error) {
	s.listUserID = userID
	s.listType = notifType
	s.listPage = page
	s.listPageSize = pageSize
	return []*model.Notification{}, s.total, nil
}

func (s *stubNotificationService) GetUnreadCount(userID int64) (int64, error) {
	return 0, nil
}

func (s *stubNotificationService) MarkRead(userID, notificationID int64) error {
	return nil
}

func (s *stubNotificationService) MarkAllRead(userID int64) error {
	return nil
}

func (s *stubNotificationService) BroadcastToAllUsers(title, content, linkURL string, getUserIDs func() ([]int64, error)) error {
	return nil
}

func TestNotificationHandlerList_ReturnsFrontendPaginationFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	notifSvc := &stubNotificationService{total: 31}
	h := NewNotificationHandler(notifSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/notifications?page=2&pageSize=15", nil)
	c.Set("user_id", int64(7))

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if notifSvc.listUserID != 7 || notifSvc.listPage != 2 || notifSvc.listPageSize != 15 {
		t.Fatalf("expected user 7 page 2 pageSize 15, got user %d page %d pageSize %d", notifSvc.listUserID, notifSvc.listPage, notifSvc.listPageSize)
	}
	if notifSvc.listType != "" {
		t.Fatalf("expected empty type filter, got %q", notifSvc.listType)
	}

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Page       int `json:"page"`
			PageSize   int `json:"pageSize"`
			TotalPages int `json:"totalPages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Data.Page != 2 || payload.Data.PageSize != 15 || payload.Data.TotalPages != 3 {
		t.Fatalf("expected page=2 pageSize=15 totalPages=3, got page=%d pageSize=%d totalPages=%d", payload.Data.Page, payload.Data.PageSize, payload.Data.TotalPages)
	}
}

func TestNotificationHandlerList_ForwardsTypeFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	notifSvc := &stubNotificationService{total: 1}
	h := NewNotificationHandler(notifSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/notifications?type=comment_like", nil)
	c.Set("user_id", int64(7))

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if notifSvc.listType != "comment_like" {
		t.Fatalf("expected type filter comment_like, got %q", notifSvc.listType)
	}
}
