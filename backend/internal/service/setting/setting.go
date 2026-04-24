package setting

import (
	"wenDao/internal/repository"
)

type SettingService interface {
	GetSortByPopularity() bool
	SetSortByPopularity(enabled bool) error
}

type settingService struct {
	repo repository.SettingRepository
}

func NewSettingService(repo repository.SettingRepository) SettingService {
	return &settingService{repo: repo}
}

func (s *settingService) GetSortByPopularity() bool {
	setting, err := s.repo.Get("sort_by_popularity")
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
	return s.repo.Set("sort_by_popularity", val)
}
