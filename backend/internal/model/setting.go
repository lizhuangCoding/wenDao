package model

import "time"

// Setting 网站设置模型
type Setting struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Key       string    `gorm:"size:100;not null;uniqueIndex" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}
