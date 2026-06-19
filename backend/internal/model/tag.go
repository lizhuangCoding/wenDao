package model

import "time"

// Tag 文章标签模型
type Tag struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Slug         string    `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	ArticleCount int       `gorm:"default:0" json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Tag) TableName() string {
	return "tags"
}

// ArticleTag 文章与标签的多对多关联。
type ArticleTag struct {
	ArticleID int64     `gorm:"primaryKey;index" json:"article_id"`
	TagID     int64     `gorm:"primaryKey;index" json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (ArticleTag) TableName() string {
	return "article_tags"
}
