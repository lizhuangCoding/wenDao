package knowledge

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// KnowledgeDocumentFilter 知识文档筛选条件
type KnowledgeDocumentFilter struct {
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

// KnowledgeDocumentRepository 知识文档数据访问接口
type KnowledgeDocumentRepository interface {
	Create(doc *model.KnowledgeDocument) error
	GetByID(id int64) (*model.KnowledgeDocument, error)
	List(filter KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error)
	Update(doc *model.KnowledgeDocument) error
	Delete(id int64) error
}

type knowledgeDocumentRepository struct {
	db *gorm.DB
}

// NewKnowledgeDocumentRepository 创建知识文档仓储实例
func NewKnowledgeDocumentRepository(db *gorm.DB) KnowledgeDocumentRepository {
	return &knowledgeDocumentRepository{db: db}
}

func (r *knowledgeDocumentRepository) Create(doc *model.KnowledgeDocument) error {
	return r.db.Create(doc).Error
}

func (r *knowledgeDocumentRepository) GetByID(id int64) (*model.KnowledgeDocument, error) {
	var doc model.KnowledgeDocument
	if err := r.db.Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *knowledgeDocumentRepository) List(filter KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error) {
	var docs []*model.KnowledgeDocument
	var total int64

	db := r.db.Model(&model.KnowledgeDocument{})
	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		db = db.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", keyword, keyword, keyword)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query := db.Order("created_at DESC")
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}
	if err := query.Find(&docs).Error; err != nil {
		return nil, 0, err
	}
	return docs, total, nil
}

func (r *knowledgeDocumentRepository) Update(doc *model.KnowledgeDocument) error {
	return r.db.Save(doc).Error
}

func (r *knowledgeDocumentRepository) Delete(id int64) error {
	return r.db.Delete(&model.KnowledgeDocument{}, id).Error
}
