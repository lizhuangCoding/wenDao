package setting

import (
	"encoding/json"
	"errors"
	"testing"

	"wenDao/internal/model"
)

type fakeSettingRepo struct {
	values     map[string]string
	getErr     error
	setErr     error
	lastSetKey string
	lastSetVal string
}

func (r *fakeSettingRepo) Get(key string) (*model.Setting, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.values == nil {
		return nil, errors.New("not found")
	}
	value, ok := r.values[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return &model.Setting{Key: key, Value: value}, nil
}

func (r *fakeSettingRepo) Set(key, value string) error {
	if r.setErr != nil {
		return r.setErr
	}
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	r.lastSetKey = key
	r.lastSetVal = value
	return nil
}

func TestSettingServiceSetContactLinks_NormalizesAndSorts(t *testing.T) {
	repo := &fakeSettingRepo{}
	svc := NewSettingService(repo)

	err := svc.SetContactLinks([]ContactLink{
		{Type: "github", Label: "GitHub", Value: "lizhuangCoding", SortOrder: 3, Enabled: true},
		{Type: " ", Label: "Skip", Value: "   ", SortOrder: 1, Enabled: true},
		{Type: "email", Label: "QQ 邮箱", Value: "3174285493@qq.com", SortOrder: 2, Enabled: true},
	})
	if err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}

	if repo.lastSetKey != settingKeyContactLinks {
		t.Fatalf("expected setting key %q, got %q", settingKeyContactLinks, repo.lastSetKey)
	}

	var stored []ContactLink
	if err := json.Unmarshal([]byte(repo.lastSetVal), &stored); err != nil {
		t.Fatalf("failed to decode stored payload: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored links, got %d (%v)", len(stored), stored)
	}
	if stored[0].Type != "email" || stored[0].SortOrder != 2 {
		t.Fatalf("expected email link first, got %#v", stored[0])
	}
	if stored[1].Type != "github" || stored[1].SortOrder != 3 {
		t.Fatalf("expected github link second, got %#v", stored[1])
	}
}

func TestSettingServiceGetContactLinks_SortsStoredLinks(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{}}
	svc := NewSettingService(repo)

	raw := []ContactLink{
		{Type: "github", Label: "GitHub", Value: "lizhuangCoding", SortOrder: 20, Enabled: true},
		{Type: "email", Label: "QQ 邮箱", Value: "3174285493@qq.com", SortOrder: 10, Enabled: true},
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("failed to build payload: %v", err)
	}
	repo.values[settingKeyContactLinks] = string(payload)

	links, ok := svc.GetContactLinks()
	if !ok {
		t.Fatalf("expected stored links to be returned")
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Type != "email" || links[0].SortOrder != 10 {
		t.Fatalf("expected email link first, got %#v", links[0])
	}
	if links[1].Type != "github" || links[1].SortOrder != 20 {
		t.Fatalf("expected github link second, got %#v", links[1])
	}
}

func TestSettingServiceGetContactLinks_ReturnsFalseOnBadPayload(t *testing.T) {
	repo := &fakeSettingRepo{values: map[string]string{
		settingKeyContactLinks: "{bad json",
	}}
	svc := NewSettingService(repo)

	links, ok := svc.GetContactLinks()
	if ok {
		t.Fatalf("expected bad payload to be rejected, got %#v", links)
	}
}
