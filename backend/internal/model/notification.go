package model

import "time"

// Notification 站内通知
type Notification struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index:idx_user_unread,priority:1" json:"user_id"`
	Type      string    `gorm:"size:30;not null;index" json:"type"` // comment_reply, admin_broadcast, system_notice
	Title     string    `gorm:"size:200;not null" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	LinkURL   string    `gorm:"size:500" json:"link_url"`
	IsRead    bool      `gorm:"default:false;index:idx_user_unread,priority:2" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}

const (
	NotificationTypeCommentReply   = "comment_reply"
	NotificationTypeAdminBroadcast = "admin_broadcast"
	NotificationTypeSystemNotice   = "system_notice"
)
