package model

import "time"

// Upload 上传文件模型
type Upload struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   int64  `gorm:"not null;index:idx_user" json:"user_id"`
	Filename string `gorm:"size:255;not null" json:"filename"`
	FilePath string `gorm:"size:500;not null" json:"file_path"`
	FileSize int64  `gorm:"not null" json:"file_size"`
	MimeType string `gorm:"size:100" json:"mime_type"`
	FileType string `gorm:"size:10;default:'image'" json:"file_type"` // image/other

	CreatedAt time.Time `json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Upload) TableName() string {
	return "uploads"
}
