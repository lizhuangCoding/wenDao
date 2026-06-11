package site

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/service/setting"
)

type stubSiteSettingService struct {
	contactLinks   []setting.ContactLink
	contactLinksOK bool
	savedLinks     []setting.ContactLink
}

func (s *stubSiteSettingService) GetSortByPopularity() bool              { return false }
func (s *stubSiteSettingService) SetSortByPopularity(enabled bool) error { return nil }
func (s *stubSiteSettingService) GetSlogan() string                      { return "" }
func (s *stubSiteSettingService) SetSlogan(slogan string) error          { return nil }
func (s *stubSiteSettingService) GetContactLinks() ([]setting.ContactLink, bool) {
	return s.contactLinks, s.contactLinksOK
}
func (s *stubSiteSettingService) SetContactLinks(links []setting.ContactLink) error {
	s.savedLinks = append([]setting.ContactLink(nil), links...)
	return nil
}

func TestSiteHandlerGetContactLinks_FallsBackToDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSiteHandler(&config.Config{}, nil, &stubSiteSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/contact-links", nil)

	h.GetContactLinks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body struct {
		Data struct {
			ContactLinks []setting.ContactLink `json:"contact_links"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Data.ContactLinks) != 2 {
		t.Fatalf("expected default links, got %#v", body.Data.ContactLinks)
	}
	if body.Data.ContactLinks[0].Value != "3174285493@qq.com" {
		t.Fatalf("expected QQ mailbox fallback, got %#v", body.Data.ContactLinks[0])
	}
}

func TestSiteHandlerSetContactLinks_StoresIncomingList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubSiteSettingService{}
	h := NewSiteHandler(&config.Config{}, nil, svc)

	body, err := json.Marshal(gin.H{
		"contact_links": []setting.ContactLink{
			{Type: "wechat", Label: "微信", Value: "wendao", Enabled: true, SortOrder: 2},
			{Type: "email", Label: "QQ 邮箱", Value: "3174285493@qq.com", Enabled: true, SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatalf("failed to build request body: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/admin/settings/contact-links", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.SetContactLinks(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}
	if len(svc.savedLinks) != 2 {
		t.Fatalf("expected two saved links, got %#v", svc.savedLinks)
	}
	if svc.savedLinks[0].Type != "wechat" || svc.savedLinks[1].Type != "email" {
		t.Fatalf("expected handler to forward request order, got %#v", svc.savedLinks)
	}
}
