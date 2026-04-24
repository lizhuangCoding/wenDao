package knowledge

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// KnowledgeDocumentSourceRepository 知识文档来源数据访问接口
type KnowledgeDocumentSourceRepository interface {
	CreateBatch(sources []*model.KnowledgeDocumentSource) error
	ListByDocumentID(documentID int64) ([]*model.KnowledgeDocumentSource, error)
	DeleteByDocumentID(documentID int64) error
}

type knowledgeDocumentSourceRepository struct {
	db *gorm.DB
}

// NewKnowledgeDocumentSourceRepository 创建知识文档来源仓储实例
func NewKnowledgeDocumentSourceRepository(db *gorm.DB) KnowledgeDocumentSourceRepository {
	return &knowledgeDocumentSourceRepository{db: db}
}

func (r *knowledgeDocumentSourceRepository) CreateBatch(sources []*model.KnowledgeDocumentSource) error {
	if len(sources) == 0 {
		return nil
	}
	return r.db.Create(&sources).Error
}

func (r *knowledgeDocumentSourceRepository) ListByDocumentID(documentID int64) ([]*model.KnowledgeDocumentSource, error) {
	var sources []*model.KnowledgeDocumentSource
	err := r.db.Where("knowledge_document_id = ?", documentID).
		Order("sort_order ASC").
		Find(&sources).Error
	return sources, err
}

func (r *knowledgeDocumentSourceRepository) DeleteByDocumentID(documentID int64) error {
	return r.db.Where("knowledge_document_id = ?", documentID).Delete(&model.KnowledgeDocumentSource{}).Error
}
