package model

import "time"

// Comment 评论模型
type Comment struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ArticleID int64  `gorm:"not null;index:idx_article" json:"article_id"`
	UserID    int64  `gorm:"not null" json:"user_id"`
	ParentID  *int64 `gorm:"index:idx_parent" json:"parent_id,omitempty"` // NULL=评论文章，非NULL=回复评论

	Content string `gorm:"type:text;not null" json:"content"`

	// 扩展字段（预留多级评论）
	RootID        *int64 `gorm:"index" json:"root_id,omitempty"`
	ReplyToUserID *int64 `json:"reply_to_user_id,omitempty"`

	Status    string    `gorm:"size:10;not null;default:'normal'" json:"status"` // normal/deleted
	CreatedAt time.Time `gorm:"index:idx_article" json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ReplyToUser *User      `gorm:"foreignKey:ReplyToUserID" json:"reply_to_user,omitempty"`
	Article     *Article   `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	Parent      *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies     []*Comment `gorm:"-" json:"replies,omitempty"` // 树形结构支持
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// IsTopLevel 判断是否为一级评论（直接评论文章）
func (c *Comment) IsTopLevel() bool {
	return c.ParentID == nil
}

// IsDeleted 判断是否已删除
func (c *Comment) IsDeleted() bool {
	return c.Status == "deleted"
}
