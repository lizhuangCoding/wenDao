package setting

import (
	"encoding/json"
	"sort"
	"strings"

	"wenDao/internal/repository"
)

const (
	settingKeySortByPopularity = "sort_by_popularity"
	settingKeySlogan           = "slogan"
	settingKeyContactLinks     = "contact_links"
)

type ContactLink struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	URL       string `json:"url,omitempty"`
	Enabled   bool   `json:"enabled"`
	SortOrder int    `json:"sort_order"`
}

type SettingService interface {
	GetSortByPopularity() bool
	SetSortByPopularity(enabled bool) error
	GetSlogan() string
	SetSlogan(slogan string) error
	GetContactLinks() ([]ContactLink, bool)
	SetContactLinks(links []ContactLink) error
}

type settingService struct {
	repo repository.SettingRepository
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) GetSortByPopularity() bool {
	setting, err := s.repo.Get(settingKeySortByPopularity)
	if err != nil {
		// 如果没找到或报错，默认返回 false（时间排序）
		return false
	}
	return setting.Value == "true"
}

func (s *settingService) SetSortByPopularity(enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return s.repo.Set(settingKeySortByPopularity, val)
}

func (s *settingService) GetSlogan() string {
	setting, err := s.repo.Get(settingKeySlogan)
	if err != nil {
		return ""
	}
	return setting.Value
}

func (s *settingService) SetSlogan(slogan string) error {
	return s.repo.Set(settingKeySlogan, slogan)
}

func (s *settingService) GetContactLinks() ([]ContactLink, bool) {
	setting, err := s.repo.Get(settingKeyContactLinks)
	if err != nil {
		return nil, false
	}

	var links []ContactLink
	if err := json.Unmarshal([]byte(setting.Value), &links); err != nil {
		return nil, false
	}

	sort.SliceStable(links, func(i, j int) bool {
		if links[i].SortOrder == links[j].SortOrder {
			return i < j
		}
		return links[i].SortOrder < links[j].SortOrder
	})

	return links, true
}

func (s *settingService) SetContactLinks(links []ContactLink) error {
	normalized := make([]ContactLink, 0, len(links))
	for _, link := range links {
		link.Type = strings.TrimSpace(link.Type)
		link.Label = strings.TrimSpace(link.Label)
		link.Value = strings.TrimSpace(link.Value)
		link.URL = strings.TrimSpace(link.URL)
		if link.Type == "" || link.Label == "" || link.Value == "" {
			continue
		}
		normalized = append(normalized, link)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].SortOrder == normalized[j].SortOrder {
			return i < j
		}
		return normalized[i].SortOrder < normalized[j].SortOrder
	})

	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}

	return s.repo.Set(settingKeyContactLinks, string(payload))
}
