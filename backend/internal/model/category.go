package model

import "time"

// Category 分类模型
type Category struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Slug         string `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	Description  string `gorm:"size:200" json:"description"`
	ArticleCount int    `gorm:"default:0" json:"article_count"`
	SortOrder    int    `gorm:"default:0" json:"sort_order"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}
