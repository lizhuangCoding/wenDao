package repository

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ArticleFilter 文章筛选条件
type ArticleFilter struct {
	Status           string // draft/published/空=全部
	CategoryID       int64  // 分类 ID，0=全部
	Keyword          string // 搜索关键字
	SortByPopularity bool   // 是否按活跃度排序
	Page             int    // 页码，从 1 开始
	PageSize         int    // 每页数量
}

// ArticleRepository 文章数据访问接口
type ArticleRepository interface {
	Create(article *model.Article) error
	GetByID(id int64) (*model.Article, error)
	GetBySlug(slug string) (*model.Article, error)
	GetBySource(sourceType string, sourceID int64) (*model.Article, error)
	List(filter ArticleFilter) ([]*model.Article, int64, error)
	Update(article *model.Article) error
	Delete(id int64) error
	UpdateSlug(id int64, slug string) error
	UpdateAIIndexStatus(id int64, status string) error
	IncrementViewCount(id int64) error
	IncrementCommentCount(id int64) error
	DecrementCommentCount(id int64) error
	IncrementLikeCount(id int64) error
	DecrementLikeCount(id int64) error
	UpdateTop(id int64, isTop bool) error
	UpdatePopularity(id int64, popularity float64) error
	GetAllPublished() ([]*model.Article, error)
}

// articleRepository 文章数据访问实现
type articleRepository struct {
	db *gorm.DB
}

// NewArticleRepository 创建文章数据访问实例
func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

// Create 创建文章
func (r *articleRepository) Create(article *model.Article) error {
	return r.db.Create(article).Error
}

// GetByID 根据 ID 查询文章（预加载分类和作者）
func (r *articleRepository) GetByID(id int64) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Category").Preload("Author").Where("id = ?", id).First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// GetBySlug 根据 slug 查询文章（预加载分类和作者）
func (r *articleRepository) GetBySlug(slug string) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Category").Preload("Author").Where("slug = ?", slug).First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// GetBySource 根据来源查询文章
func (r *articleRepository) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	var article model.Article
	err := r.db.Preload("Category").Preload("Author").
		Where("source_type = ? AND source_id = ?", sourceType, sourceID).
		First(&article).Error
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// List 获取文章列表（支持分页和筛选）
func (r *articleRepository) List(filter ArticleFilter) ([]*model.Article, int64, error) {
	var articles []*model.Article
	var total int64

	db := r.db.Model(&model.Article{})

	if filter.Status != "" {
		db = db.Where("status = ?", filter.Status)
	}
	if filter.CategoryID > 0 {
		db = db.Where("category_id = ?", filter.CategoryID)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		db = db.Where("title LIKE ? OR summary LIKE ? OR content LIKE ?", keyword, keyword, keyword)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := db
	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query = query.Offset(offset).Limit(filter.PageSize)
	}

	// 排序逻辑：置顶 > (活跃度/发布时间)
	orderStr := "is_top DESC"
	if filter.SortByPopularity {
		orderStr += ", popularity DESC, published_at DESC"
	} else {
		orderStr += ", published_at DESC, created_at DESC"
	}

	err := query.Preload("Category").Preload("Author").
		Order(orderStr).
		Find(&articles).Error

	return articles, total, err
}

// Update 更新文章
func (r *articleRepository) Update(article *model.Article) error {
	return r.db.Save(article).Error
}

// Delete 删除文章
func (r *articleRepository) Delete(id int64) error {
	return r.db.Delete(&model.Article{}, id).Error
}

// UpdateSlug 更新文章的 slug
func (r *articleRepository) UpdateSlug(id int64, slug string) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Update("slug", slug).Error
}

// UpdateAIIndexStatus 更新 AI 索引状态
func (r *articleRepository) UpdateAIIndexStatus(id int64, status string) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Update("ai_index_status", status).Error
}

// IncrementViewCount 增加浏览次数
func (r *articleRepository) IncrementViewCount(id int64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// IncrementCommentCount 增加评论数
func (r *articleRepository) IncrementCommentCount(id int64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error
}

// DecrementCommentCount 减少评论数
func (r *articleRepository) DecrementCommentCount(id int64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).Error
}

// IncrementLikeCount 增加点赞数
func (r *articleRepository) IncrementLikeCount(id int64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLikeCount 减少点赞数
func (r *articleRepository) DecrementLikeCount(id int64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).Error
}

// UpdateTop 更新置顶状态
func (r *articleRepository) UpdateTop(id int64, isTop bool) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Update("is_top", isTop).Error
}

// UpdatePopularity 更新活跃度分数
func (r *articleRepository) UpdatePopularity(id int64, popularity float64) error {
	return r.db.Model(&model.Article{}).Where("id = ?", id).Update("popularity", popularity).Error
}

// GetAllPublished 获取所有已发布的文章（用于批量计算分数）
func (r *articleRepository) GetAllPublished() ([]*model.Article, error) {
	var articles []*model.Article
	err := r.db.Where("status = ?", "published").Find(&articles).Error
	return articles, err
}
