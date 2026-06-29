package model

import "time"

const (
	AsyncJobStatusPending   = "pending"
	AsyncJobStatusRunning   = "running"
	AsyncJobStatusSucceeded = "succeeded"
	AsyncJobStatusFailed    = "failed"
)

type AsyncJob struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	JobType     string     `gorm:"size:100;not null;index:idx_async_jobs_dispatch,priority:1" json:"job_type"`
	Status      string     `gorm:"size:20;not null;index:idx_async_jobs_dispatch,priority:2" json:"status"`
	Payload     []byte     `gorm:"type:json;not null" json:"payload"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int        `gorm:"not null;default:3" json:"max_attempts"`
	RunAfter    time.Time  `gorm:"not null;index:idx_async_jobs_dispatch,priority:3" json:"run_after"`
	LockedAt    *time.Time `json:"locked_at"`
	LastError   string     `gorm:"type:text" json:"last_error"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (AsyncJob) TableName() string {
	return "async_jobs"
}
