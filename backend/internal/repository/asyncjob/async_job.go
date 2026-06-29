package asyncjob

import (
	"time"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

const staleRunningTimeout = 5 * time.Minute

type AsyncJobRepository interface {
	Enqueue(job *model.AsyncJob) error
	ListRunnable(now time.Time, limit int) ([]*model.AsyncJob, error)
	Claim(id int64, now time.Time) (bool, error)
	MarkSucceeded(id int64, finishedAt time.Time) error
	MarkFailed(id int64, runAfter time.Time, lastError string) error
}

type asyncJobRepository struct {
	db *gorm.DB
}

func NewAsyncJobRepository(db *gorm.DB) AsyncJobRepository {
	return &asyncJobRepository{db: db}
}

func (r *asyncJobRepository) Enqueue(job *model.AsyncJob) error {
	return r.db.Create(job).Error
}

func (r *asyncJobRepository) ListRunnable(now time.Time, limit int) ([]*model.AsyncJob, error) {
	if limit <= 0 {
		limit = 20
	}

	staleBefore := now.Add(-staleRunningTimeout)
	var jobs []*model.AsyncJob
	err := r.db.
		Where(
			"(status IN ? AND run_after <= ? AND attempts < max_attempts) OR (status = ? AND locked_at IS NOT NULL AND locked_at <= ? AND attempts < max_attempts)",
			[]string{model.AsyncJobStatusPending, model.AsyncJobStatusFailed},
			now,
			model.AsyncJobStatusRunning,
			staleBefore,
		).
		Order("run_after ASC, id ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

func (r *asyncJobRepository) Claim(id int64, now time.Time) (bool, error) {
	result := r.db.Model(&model.AsyncJob{}).
		Where("id = ? AND attempts < max_attempts", id).
		Where(
			"(status IN ?) OR (status = ? AND locked_at IS NOT NULL AND locked_at <= ?)",
			[]string{model.AsyncJobStatusPending, model.AsyncJobStatusFailed},
			model.AsyncJobStatusRunning,
			now.Add(-staleRunningTimeout),
		).
		Updates(map[string]any{
			"status":    model.AsyncJobStatusRunning,
			"attempts":  gorm.Expr("attempts + 1"),
			"locked_at": now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *asyncJobRepository) MarkSucceeded(id int64, finishedAt time.Time) error {
	return r.db.Model(&model.AsyncJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     model.AsyncJobStatusSucceeded,
			"locked_at":  nil,
			"last_error": "",
			"run_after":  finishedAt,
		}).Error
}

func (r *asyncJobRepository) MarkFailed(id int64, runAfter time.Time, lastError string) error {
	return r.db.Model(&model.AsyncJob{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     model.AsyncJobStatusFailed,
			"locked_at":  nil,
			"last_error": lastError,
			"run_after":  runAfter,
		}).Error
}
