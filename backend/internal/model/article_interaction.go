package model

import "time"

const (
	ArticleInteractionTypeLike     = "like"
	ArticleInteractionTypeFavorite = "favorite"
)

// ArticleInteraction 记录用户对文章的点赞、收藏等互动。
type ArticleInteraction struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          int64     `gorm:"not null;uniqueIndex:idx_article_interaction_unique;index:idx_article_interaction_user_type,priority:1" json:"user_id"`
	ArticleID       int64     `gorm:"not null;uniqueIndex:idx_article_interaction_unique;index:idx_article_interaction_article_type,priority:1" json:"article_id"`
	InteractionType string    `gorm:"size:20;not null;uniqueIndex:idx_article_interaction_unique;index:idx_article_interaction_user_type,priority:2;index:idx_article_interaction_article_type,priority:2" json:"interaction_type"`
	CreatedAt       time.Time `gorm:"index:idx_article_interaction_user_type,priority:3" json:"created_at"`

	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article *Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

func (ArticleInteraction) TableName() string {
	return "article_interactions"
}

type ArticleInteractionState struct {
	Liked     bool `json:"liked"`
	Favorited bool `json:"favorited"`
}
