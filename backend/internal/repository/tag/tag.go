package tag

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// TagFilter 标签筛选条件
type TagFilter struct {
	Page     int
	PageSize int
}

// TagRepository 标签数据访问接口
type TagRepository interface {
	Create(tag *model.Tag) error
	GetByID(id int64) (*model.Tag, error)
	GetBySlug(slug string) (*model.Tag, error)
	List() ([]*model.Tag, error)
	ListPaginated(filter TagFilter) ([]*model.Tag, int64, error)
	Update(tag *model.Tag) error
	Delete(id int64) error
	IncrementArticleCount(id int64) error
	DecrementArticleCount(id int64) error
	SetArticleTags(articleID int64, tagIDs []int64) error
	GetArticleTags(articleID int64) ([]*model.Tag, error)
}

type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签数据访问实例
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(tag *model.Tag) error {
	return r.db.Create(tag).Error
}

func (r *tagRepository) GetByID(id int64) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) GetBySlug(slug string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *tagRepository) List() ([]*model.Tag, error) {
	var tags []*model.Tag
	err := r.db.Order("article_count DESC, name ASC, created_at DESC").Find(&tags).Error
	return tags, err
}

func (r *tagRepository) ListPaginated(filter TagFilter) ([]*model.Tag, int64, error) {
	var tags []*model.Tag
	var total int64

	query := r.db.Model(&model.Tag{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	err := query.Order("article_count DESC, name ASC, created_at DESC").Find(&tags).Error
	return tags, total, err
}

func (r *tagRepository) Update(tag *model.Tag) error {
	return r.db.Save(tag).Error
}

func (r *tagRepository) Delete(id int64) error {
	return r.db.Delete(&model.Tag{}, id).Error
}

func (r *tagRepository) IncrementArticleCount(id int64) error {
	return r.db.Model(&model.Tag{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("article_count + ?", 1)).Error
}

func (r *tagRepository) DecrementArticleCount(id int64) error {
	return r.db.Model(&model.Tag{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("CASE WHEN article_count > ? THEN article_count - ? ELSE 0 END", 0, 1)).Error
}

func (r *tagRepository) SetArticleTags(articleID int64, tagIDs []int64) error {
	normalized := normalizeTagIDs(tagIDs)

	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(normalized) > 0 {
			var count int64
			if err := tx.Model(&model.Tag{}).Where("id IN ?", normalized).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(normalized)) {
				return gorm.ErrRecordNotFound
			}
		}

		var existing []model.ArticleTag
		if err := tx.Where("article_id = ?", articleID).Find(&existing).Error; err != nil {
			return err
		}

		current := make(map[int64]struct{}, len(existing))
		next := make(map[int64]struct{}, len(normalized))
		for _, item := range existing {
			current[item.TagID] = struct{}{}
		}
		for _, id := range normalized {
			next[id] = struct{}{}
		}

		for tagID := range current {
			if _, keep := next[tagID]; keep {
				continue
			}
			if err := tx.Where("article_id = ? AND tag_id = ?", articleID, tagID).Delete(&model.ArticleTag{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Tag{}).Where("id = ?", tagID).
				UpdateColumn("article_count", gorm.Expr("CASE WHEN article_count > ? THEN article_count - ? ELSE 0 END", 0, 1)).Error; err != nil {
				return err
			}
		}

		for _, tagID := range normalized {
			if _, exists := current[tagID]; exists {
				continue
			}
			if err := tx.Create(&model.ArticleTag{ArticleID: articleID, TagID: tagID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Tag{}).Where("id = ?", tagID).
				UpdateColumn("article_count", gorm.Expr("article_count + ?", 1)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *tagRepository) GetArticleTags(articleID int64) ([]*model.Tag, error) {
	var tags []*model.Tag
	err := r.db.Table("tags").
		Select("tags.*").
		Joins("JOIN article_tags ON article_tags.tag_id = tags.id").
		Where("article_tags.article_id = ?", articleID).
		Order("tags.name ASC").
		Find(&tags).Error
	return tags, err
}

func normalizeTagIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized
}
