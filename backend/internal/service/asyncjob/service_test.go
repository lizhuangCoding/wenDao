package asyncjob

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"wenDao/internal/model"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
)

type serviceAsyncJobRepoStub struct {
	jobs            []*model.AsyncJob
	claimed         []int64
	succeeded       []int64
	failed          []int64
	failedRunAfter  map[int64]time.Time
	failedLastError map[int64]string
}

func (r *serviceAsyncJobRepoStub) Enqueue(job *model.AsyncJob) error {
	r.jobs = append(r.jobs, job)
	return nil
}

func (r *serviceAsyncJobRepoStub) ListRunnable(now time.Time, limit int) ([]*model.AsyncJob, error) {
	if limit > 0 && len(r.jobs) > limit {
		return append([]*model.AsyncJob(nil), r.jobs[:limit]...), nil
	}
	return append([]*model.AsyncJob(nil), r.jobs...), nil
}

func (r *serviceAsyncJobRepoStub) Claim(id int64, now time.Time) (bool, error) {
	r.claimed = append(r.claimed, id)
	for _, job := range r.jobs {
		if job.ID == id {
			job.Attempts++
			return true, nil
		}
	}
	return false, nil
}

func (r *serviceAsyncJobRepoStub) MarkSucceeded(id int64, finishedAt time.Time) error {
	r.succeeded = append(r.succeeded, id)
	return nil
}

func (r *serviceAsyncJobRepoStub) MarkFailed(id int64, runAfter time.Time, lastError string) error {
	if r.failedRunAfter == nil {
		r.failedRunAfter = make(map[int64]time.Time)
	}
	if r.failedLastError == nil {
		r.failedLastError = make(map[int64]string)
	}
	r.failed = append(r.failed, id)
	r.failedRunAfter[id] = runAfter
	r.failedLastError[id] = lastError
	return nil
}

var _ asyncjobrepo.AsyncJobRepository = (*serviceAsyncJobRepoStub)(nil)

func TestServiceProcessPendingMarksSucceededWhenHandlerPasses(t *testing.T) {
	repo := &serviceAsyncJobRepoStub{
		jobs: []*model.AsyncJob{
			{ID: 11, JobType: JobTypeArticleVectorize, Payload: []byte(`{"article_id":7}`), MaxAttempts: 3},
		},
	}

	handled := 0
	svc := NewService(
		repo,
		zap.NewNop(),
		WithClock(func() time.Time { return time.Unix(1700000000, 0) }),
		WithJobHandler(JobTypeArticleVectorize, func(ctx context.Context, job *model.AsyncJob) error {
			handled++
			if job.ID != 11 {
				t.Fatalf("expected job 11, got %d", job.ID)
			}
			return nil
		}),
	)

	if err := svc.ProcessPending(context.Background(), 10); err != nil {
		t.Fatalf("expected processing to succeed, got %v", err)
	}

	if handled != 1 {
		t.Fatalf("expected one handled job, got %d", handled)
	}
	if len(repo.succeeded) != 1 || repo.succeeded[0] != 11 {
		t.Fatalf("expected job 11 to be marked succeeded, got %#v", repo.succeeded)
	}
	if len(repo.failed) != 0 {
		t.Fatalf("expected no failed jobs, got %#v", repo.failed)
	}
}

func TestServiceProcessPendingMarksFailedWithRetryDelayWhenHandlerFails(t *testing.T) {
	now := time.Unix(1700000000, 0)
	repo := &serviceAsyncJobRepoStub{
		jobs: []*model.AsyncJob{
			{ID: 22, JobType: JobTypeNotificationCreate, Payload: []byte(`{}`), Attempts: 0, MaxAttempts: 3},
		},
	}

	svc := NewService(
		repo,
		zap.NewNop(),
		WithClock(func() time.Time { return now }),
		WithJobHandler(JobTypeNotificationCreate, func(ctx context.Context, job *model.AsyncJob) error {
			return errors.New("temporary failure")
		}),
	)

	if err := svc.ProcessPending(context.Background(), 10); err != nil {
		t.Fatalf("expected processing loop not to fail hard, got %v", err)
	}

	if len(repo.failed) != 1 || repo.failed[0] != 22 {
		t.Fatalf("expected job 22 to be marked failed, got %#v", repo.failed)
	}
	if repo.failedLastError[22] == "" {
		t.Fatalf("expected failure reason to be recorded")
	}
	if !repo.failedRunAfter[22].After(now) {
		t.Fatalf("expected retry run_after after %v, got %v", now, repo.failedRunAfter[22])
	}
}
