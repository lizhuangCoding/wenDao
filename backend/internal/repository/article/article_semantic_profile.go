package article

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"wenDao/internal/model"
)

type ArticleSemanticProfileRepository interface {
	Upsert(profile *model.ArticleSemanticProfile) error
	DeleteByArticleID(articleID int64) error
	ListByArticleIDs(articleIDs []int64) (map[int64]*model.ArticleSemanticProfile, error)
}

type articleSemanticProfileRepository struct {
	db *gorm.DB
}

func NewArticleSemanticProfileRepository(db *gorm.DB) ArticleSemanticProfileRepository {
	return &articleSemanticProfileRepository{db: db}
}

func (r *articleSemanticProfileRepository) Upsert(profile *model.ArticleSemanticProfile) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"embedding_json",
			"content_hash",
			"map_x",
			"map_y",
			"map_z",
			"neighbor_json",
			"updated_at",
		}),
	}).Create(profile).Error
}

func (r *articleSemanticProfileRepository) DeleteByArticleID(articleID int64) error {
	return r.db.Where("article_id = ?", articleID).Delete(&model.ArticleSemanticProfile{}).Error
}

func (r *articleSemanticProfileRepository) ListByArticleIDs(articleIDs []int64) (map[int64]*model.ArticleSemanticProfile, error) {
	profilesByArticleID := make(map[int64]*model.ArticleSemanticProfile, len(articleIDs))
	if len(articleIDs) == 0 {
		return profilesByArticleID, nil
	}

	var profiles []*model.ArticleSemanticProfile
	if err := r.db.Where("article_id IN ?", articleIDs).Find(&profiles).Error; err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		profilesByArticleID[profile.ArticleID] = profile
	}
	return profilesByArticleID, nil
}

