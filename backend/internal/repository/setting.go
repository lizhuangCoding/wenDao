package repository

import (
	"gorm.io/gorm"
	"wenDao/internal/model"
)

type SettingRepository interface {
	Get(key string) (*model.Setting, error)
	Set(key, value string) error
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) Get(key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) Set(key, value string) error {
	var setting model.Setting
	err := r.db.Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.db.Create(&model.Setting{Key: key, Value: value}).Error
		}
		return err
	}
	return r.db.Model(&setting).Update("value", value).Error
}
