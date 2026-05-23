package upload

import (
	"time"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

// UploadRepository 上传文件数据访问接口
type UploadRepository interface {
	Create(upload *model.Upload) error
	GetByID(id int64) (*model.Upload, error)
	DeleteByFilePath(filePath string) error
	DeleteByID(id int64) error
	ListUnreferencedBefore(cutoff time.Time, limit int) ([]*model.Upload, error)
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

// DeleteByID 根据 ID 删除上传记录
func (r *uploadRepository) DeleteByID(id int64) error {
	return r.db.Delete(&model.Upload{}, id).Error
}

// ListUnreferencedBefore 查询早于 cutoff 且没有被文章或用户头像引用的上传记录。
func (r *uploadRepository) ListUnreferencedBefore(cutoff time.Time, limit int) ([]*model.Upload, error) {
	if limit <= 0 {
		limit = 200
	}

	var uploads []*model.Upload
	err := r.db.
		Table("uploads AS uploads").
		Where("uploads.created_at < ?", cutoff).
		Where("uploads.file_path LIKE ?", "/uploads/%").
		Where(`NOT EXISTS (
			SELECT 1 FROM articles
			WHERE articles.cover_image = uploads.file_path
				OR articles.content LIKE CONCAT('%', uploads.file_path, '%')
				OR articles.content_html LIKE CONCAT('%', uploads.file_path, '%')
		)`).
		Where(`NOT EXISTS (
			SELECT 1 FROM users
			WHERE users.avatar_url = uploads.file_path
		)`).
		Order("uploads.created_at ASC").
		Limit(limit).
		Find(&uploads).Error
	if err != nil {
		return nil, err
	}

	return uploads, nil
}
