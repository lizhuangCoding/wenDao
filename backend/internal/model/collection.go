package model

import "time"

// Collection represents an editorial series of articles with an explicit reading order.
type Collection struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Slug         string `gorm:"size:100;not null;uniqueIndex" json:"slug"`
	Description  string `gorm:"size:500" json:"description"`
	ArticleCount int    `gorm:"default:0" json:"article_count"`
	SortOrder    int    `gorm:"default:0" json:"sort_order"`
	Status       string `gorm:"size:20;not null;default:'active';index" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Collection) TableName() string {
	return "collections"
}

// ArticleCollection stores the ordered placement of an article inside a collection.
type ArticleCollection struct {
	ID           int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	CollectionID int64 `gorm:"not null;index:idx_article_collection_collection_position,priority:1;uniqueIndex:idx_collection_article" json:"collection_id"`
	ArticleID    int64 `gorm:"not null;index;uniqueIndex:idx_collection_article" json:"article_id"`
	Position     int   `gorm:"not null;default:0;index:idx_article_collection_collection_position,priority:2" json:"position"`
	IsPrimary    bool  `gorm:"not null;default:true;index" json:"is_primary"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Collection *Collection `gorm:"foreignKey:CollectionID" json:"collection,omitempty"`
	Article    *Article    `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

func (ArticleCollection) TableName() string {
	return "article_collections"
}

type ArticleCollectionMembership struct {
	CollectionID int64  `json:"collection_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Position     int    `json:"position"`
}

type CollectionNavigationArticle struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type ArticleCollectionNavigation struct {
	CollectionID   int64                        `json:"collection_id"`
	CollectionName string                       `json:"collection_name"`
	CollectionSlug string                       `json:"collection_slug"`
	Position       int                          `json:"position"`
	Total          int64                        `json:"total"`
	Previous       *CollectionNavigationArticle `json:"previous,omitempty"`
	Next           *CollectionNavigationArticle `json:"next,omitempty"`
}
