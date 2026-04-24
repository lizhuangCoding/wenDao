package model

import "time"

const (
	ArticleSourceTypeManual            = "manual"
	ArticleSourceTypeKnowledgeDocument = "knowledge_document"
)

// Article 文章模型
type Article struct {
	ID          int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string `gorm:"size:200;not null" json:"title"`
	Slug        string `gorm:"size:200;not null;uniqueIndex" json:"slug"`
	Summary     string `gorm:"type:text" json:"summary"`
	Content     string `gorm:"type:longtext;not null" json:"content"`
	ContentHTML string `gorm:"type:longtext" json:"content_html"` // 渲染后的 HTML 缓存

	CategoryID int64 `gorm:"not null;index:idx_category" json:"category_id"`
	AuthorID   int64 `gorm:"not null" json:"author_id"`

	CoverImage    *string `gorm:"size:500" json:"cover_image,omitempty"`
	Status        string  `gorm:"size:10;not null;default:'draft';index:idx_status_published" json:"status"` // draft/published
	AIIndexStatus string  `gorm:"size:20;not null;default:'pending'" json:"ai_index_status"`                 // pending/success/failed
	SourceType    string  `gorm:"size:32;not null;default:'manual';index:idx_article_source" json:"source_type"`
	SourceID      *int64  `gorm:"index:idx_article_source" json:"source_id,omitempty"`

	ViewCount    int     `gorm:"default:0" json:"view_count"`
	CommentCount int     `gorm:"default:0" json:"comment_count"`
	LikeCount    int     `gorm:"default:0" json:"like_count"`
	IsTop        bool    `gorm:"default:false" json:"is_top"`
	Popularity   float64 `gorm:"default:0;index:idx_popularity" json:"popularity"` // 活跃度分数

	PublishedAt *time.Time `gorm:"index:idx_status_published" json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联（不存储在数据库）
	Category *Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Author   *User     `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

// TableName 指定表名
func (Article) TableName() string {
	return "articles"
}

// IsPublished 判断是否已发布
func (a *Article) IsPublished() bool {
	return a.Status == "published"
}

// IsDraft 判断是否为草稿
func (a *Article) IsDraft() bool {
	return a.Status == "draft"
}
