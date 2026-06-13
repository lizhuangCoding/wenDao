package collection

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

type CollectionFilter struct {
	Page     int
	PageSize int
}

type CollectionRepository interface {
	Create(collection *model.Collection) error
	GetByID(id int64) (*model.Collection, error)
	GetBySlug(slug string) (*model.Collection, error)
	List() ([]*model.Collection, error)
	ListPaginated(filter CollectionFilter) ([]*model.Collection, int64, error)
	Update(collection *model.Collection) error
	Delete(id int64) error
	SetPrimaryArticlePlacement(articleID int64, collectionID *int64, position int) error
	GetPrimaryArticlePlacement(articleID int64) (*model.ArticleCollection, error)
	GetArticleNavigation(articleID int64) (*model.ArticleCollectionNavigation, error)
	DeleteArticlePlacements(articleID int64) error
}

type collectionRepository struct {
	db *gorm.DB
}

func NewCollectionRepository(db *gorm.DB) CollectionRepository {
	return &collectionRepository{db: db}
}

func (r *collectionRepository) Create(collection *model.Collection) error {
	return r.db.Create(collection).Error
}

func (r *collectionRepository) GetByID(id int64) (*model.Collection, error) {
	var collection model.Collection
	err := r.db.Where("id = ?", id).First(&collection).Error
	if err != nil {
		return nil, err
	}
	return &collection, nil
}

func (r *collectionRepository) GetBySlug(slug string) (*model.Collection, error) {
	var collection model.Collection
	err := r.db.Where("slug = ?", slug).First(&collection).Error
	if err != nil {
		return nil, err
	}
	return &collection, nil
}

func (r *collectionRepository) List() ([]*model.Collection, error) {
	var collections []*model.Collection
	err := r.db.Where("status = ?", "active").Order("sort_order ASC, created_at DESC").Find(&collections).Error
	return collections, err
}

func (r *collectionRepository) ListPaginated(filter CollectionFilter) ([]*model.Collection, int64, error) {
	var collections []*model.Collection
	var total int64
	query := r.db.Model(&model.Collection{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Page > 0 && filter.PageSize > 0 {
		query = query.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}
	err := query.Order("sort_order ASC, created_at DESC").Find(&collections).Error
	return collections, total, err
}

func (r *collectionRepository) Update(collection *model.Collection) error {
	return r.db.Save(collection).Error
}

func (r *collectionRepository) Delete(id int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("collection_id = ?", id).Delete(&model.ArticleCollection{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Collection{}, id).Error
	})
}

func (r *collectionRepository) SetPrimaryArticlePlacement(articleID int64, collectionID *int64, position int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var oldPlacements []model.ArticleCollection
		if err := tx.Where("article_id = ?", articleID).Find(&oldPlacements).Error; err != nil {
			return err
		}
		if err := tx.Where("article_id = ?", articleID).Delete(&model.ArticleCollection{}).Error; err != nil {
			return err
		}
		for _, placement := range oldPlacements {
			if err := tx.Model(&model.Collection{}).Where("id = ?", placement.CollectionID).
				UpdateColumn("article_count", gorm.Expr("CASE WHEN article_count > ? THEN article_count - ? ELSE 0 END", 0, 1)).Error; err != nil {
				return err
			}
		}

		if collectionID == nil || *collectionID <= 0 {
			return nil
		}

		placement := &model.ArticleCollection{
			CollectionID: *collectionID,
			ArticleID:    articleID,
			Position:     position,
			IsPrimary:    true,
		}
		if err := tx.Create(placement).Error; err != nil {
			return err
		}
		return tx.Model(&model.Collection{}).Where("id = ?", *collectionID).
			UpdateColumn("article_count", gorm.Expr("article_count + ?", 1)).Error
	})
}

func (r *collectionRepository) GetPrimaryArticlePlacement(articleID int64) (*model.ArticleCollection, error) {
	var placement model.ArticleCollection
	err := r.db.Preload("Collection").
		Where("article_id = ? AND is_primary = ?", articleID, true).
		Order("updated_at DESC").
		First(&placement).Error
	if err != nil {
		return nil, err
	}
	return &placement, nil
}

func (r *collectionRepository) GetArticleNavigation(articleID int64) (*model.ArticleCollectionNavigation, error) {
	placement, err := r.GetPrimaryArticlePlacement(articleID)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := r.db.Model(&model.ArticleCollection{}).
		Joins("JOIN articles ON articles.id = article_collections.article_id").
		Where("article_collections.collection_id = ? AND articles.status = ?", placement.CollectionID, "published").
		Count(&total).Error; err != nil {
		return nil, err
	}

	navigation := &model.ArticleCollectionNavigation{
		CollectionID:   placement.CollectionID,
		CollectionName: placement.Collection.Name,
		CollectionSlug: placement.Collection.Slug,
		Position:       placement.Position,
		Total:          total,
	}

	previous, err := r.adjacentPublishedArticle(placement, true)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	navigation.Previous = previous

	next, err := r.adjacentPublishedArticle(placement, false)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	navigation.Next = next

	return navigation, nil
}

func (r *collectionRepository) adjacentPublishedArticle(placement *model.ArticleCollection, previous bool) (*model.CollectionNavigationArticle, error) {
	var article model.Article
	query := r.db.Model(&model.Article{}).
		Select("articles.id", "articles.title", "articles.slug").
		Joins("JOIN article_collections ON article_collections.article_id = articles.id").
		Where("article_collections.collection_id = ? AND articles.status = ?", placement.CollectionID, "published")
	if previous {
		query = query.Where(
			"(article_collections.position < ? OR (article_collections.position = ? AND articles.id < ?))",
			placement.Position,
			placement.Position,
			placement.ArticleID,
		).Order("article_collections.position DESC, articles.id DESC")
	} else {
		query = query.Where(
			"(article_collections.position > ? OR (article_collections.position = ? AND articles.id > ?))",
			placement.Position,
			placement.Position,
			placement.ArticleID,
		).Order("article_collections.position ASC, articles.id ASC")
	}
	if err := query.First(&article).Error; err != nil {
		return nil, err
	}
	return &model.CollectionNavigationArticle{
		ID:    article.ID,
		Title: article.Title,
		Slug:  article.Slug,
	}, nil
}

func (r *collectionRepository) DeleteArticlePlacements(articleID int64) error {
	return r.SetPrimaryArticlePlacement(articleID, nil, 0)
}
