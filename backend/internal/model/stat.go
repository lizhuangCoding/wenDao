package model

import "time"

// DailyStat 每日流量统计
type DailyStat struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Date         string    `gorm:"size:10;not null;uniqueIndex" json:"date"` // 格式: 2024-04-07
	PV           int64     `gorm:"default:0" json:"pv"`
	UV           int64     `gorm:"default:0" json:"uv"`
	CommentCount int64     `gorm:"default:0" json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ArticleStat 文章访问统计（冗余表，方便查询趋势）
type ArticleStat struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ArticleID int64     `gorm:"not null;index:idx_article_date" json:"article_id"`
	Date      string    `gorm:"size:10;not null;index:idx_article_date" json:"date"`
	PV        int64     `gorm:"default:0" json:"pv"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (DailyStat) TableName() string {
	return "daily_stats"
}

func (ArticleStat) TableName() string {
	return "article_stats"
}
