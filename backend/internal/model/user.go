package model

import (
	"time"
)

const (
	AvatarSourceDefault = "default"
	AvatarSourceGitHub  = "github"
	AvatarSourceCustom  = "custom"
)

// User 用户模型 (普通用户：只能评论、聊天)
type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	Username string `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Email    string `gorm:"size:100;not null;uniqueIndex" json:"email"`
	// PasswordHash 密码哈希（OAuth用户为空）
	PasswordHash *string `gorm:"size:255" json:"-"`

	// 角色控制
	Role string `gorm:"size:10;not null;default:'user'" json:"role"`

	// OAuth 相关
	OAuthProvider *string `gorm:"size:20" json:"oauth_provider,omitempty"`
	OAuthID       *string `gorm:"size:100" json:"oauth_id,omitempty"`
	AvatarURL     *string `gorm:"size:500" json:"avatar_url,omitempty"`
	AvatarSource  string  `gorm:"size:20;not null;default:'default'" json:"avatar_source"`

	// 扩展字段
	EmailVerified bool   `gorm:"default:false" json:"email_verified"`
	Status        string `gorm:"size:10;not null;default:'active'" json:"status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// IsActive 判断账号是否激活
func (u *User) IsActive() bool {
	return u.Status == "active"
}

// IsAdmin 判断是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}
