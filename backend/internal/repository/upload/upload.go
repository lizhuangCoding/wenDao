package upload

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// UploadRepository 上传文件数据访问接口
type UploadRepository interface {
	Create(upload *model.Upload) error
	GetByID(id int64) (*model.Upload, error)
	DeleteByFilePath(filePath string) error
}

// uploadRepository 上传文件数据访问实现
type uploadRepository struct {
	db *gorm.DB
}

// NewUploadRepository 创建上传文件数据访问实例
func NewUploadRepository(db *gorm.DB) UploadRepository {
	return &uploadRepository{db: db}
}

// Create 创建上传记录
func (r *uploadRepository) Create(upload *model.Upload) error {
	return r.db.Create(upload).Error
}

// GetByID 根据 ID 查询上传记录
func (r *uploadRepository) GetByID(id int64) (*model.Upload, error) {
	var upload model.Upload
	err := r.db.Where("id = ?", id).First(&upload).Error
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

// DeleteByFilePath 根据文件路径删除上传记录
func (r *uploadRepository) DeleteByFilePath(filePath string) error {
	return r.db.Where("file_path = ?", filePath).Delete(&model.Upload{}).Error
}
