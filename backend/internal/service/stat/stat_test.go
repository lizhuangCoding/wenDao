package stat

import (
	"context"
	"testing"
	"time"

	"wenDao/internal/model"
)

func TestBuildDashboardStats_FillsMissingDatesWithZeroes(t *testing.T) {
	start := time.Date(2026, 4, 14, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 4, 16, 0, 0, 0, 0, time.Local)
	stats := []model.DailyStat{
		{Date: "2026-04-14", PV: 10, UV: 2, CommentCount: 1},
		{Date: "2026-04-16", PV: 20, UV: 4, CommentCount: 3},
	}

	dashboard := buildDashboardStats(stats, start, end)

	wantLabels := []string{"2026-04-14", "2026-04-15", "2026-04-16"}
	for i, want := range wantLabels {
		if dashboard.DailyStat.Labels[i] != want {
			t.Fatalf("label[%d] = %q, want %q", i, dashboard.DailyStat.Labels[i], want)
		}
	}
	wantPV := []int64{10, 0, 20}
	wantUV := []int64{2, 0, 4}
	for i := range wantPV {
		if dashboard.DailyStat.PV[i] != wantPV[i] {
			t.Fatalf("pv[%d] = %d, want %d", i, dashboard.DailyStat.PV[i], wantPV[i])
		}
		if dashboard.DailyStat.UV[i] != wantUV[i] {
			t.Fatalf("uv[%d] = %d, want %d", i, dashboard.DailyStat.UV[i], wantUV[i])
		}
	}
	if dashboard.TotalPV != 30 || dashboard.TotalUV != 6 || dashboard.TotalComments != 4 {
		t.Fatalf("unexpected totals: %+v", dashboard)
	}
}

type statRepoIncrement struct {
	Date         string
	PV           int64
	UV           int64
	CommentCount int64
}

type stubDailyStatRepository struct {
	increments []statRepoIncrement
	stats      []model.DailyStat
}

func (r *stubDailyStatRepository) GetDailyStats(days int) ([]model.DailyStat, error) {
	return r.stats, nil
}

func (r *stubDailyStatRepository) GetDailyStatsByRange(startDate, endDate string) ([]model.DailyStat, error) {
	return r.stats, nil
}

func (r *stubDailyStatRepository) GetArticleStats(articleID int64, days int) ([]model.ArticleStat, error) {
	return nil, nil
}

func (r *stubDailyStatRepository) IncrementDailyStat(date string, pvDelta, uvDelta, commentDelta int64) error {
	r.increments = append(r.increments, statRepoIncrement{Date: date, PV: pvDelta, UV: uvDelta, CommentCount: commentDelta})
	return nil
}

type stubStatCounterStore struct {
	pvDates       []string
	uvInputs      []string
	commentDates  []string
	drainDates    []string
	drained       statCounterDelta
	drainedByDate map[string]statCounterDelta
}

func (s *stubStatCounterStore) IncrementPV(ctx context.Context, date string) error {
	s.pvDates = append(s.pvDates, date)
	return nil
}

func (s *stubStatCounterStore) AddUV(ctx context.Context, date string, visitor string) (bool, error) {
	s.uvInputs = append(s.uvInputs, date+"|"+visitor)
	return true, nil
}

func (s *stubStatCounterStore) IncrementComment(ctx context.Context, date string) error {
	s.commentDates = append(s.commentDates, date)
	return nil
}

func (s *stubStatCounterStore) DrainDate(ctx context.Context, date string) (statCounterDelta, error) {
	s.drainDates = append(s.drainDates, date)
	if s.drainedByDate != nil {
		return s.drainedByDate[date], nil
	}
	return s.drained, nil
}

func (s *stubStatCounterStore) RestoreDate(ctx context.Context, date string, delta statCounterDelta) error {
	return nil
}

func TestStatServiceRecordMethods_UseCounterStoreInsteadOfDirectDatabaseWrites(t *testing.T) {
	repo := &stubDailyStatRepository{}
	counter := &stubStatCounterStore{}
	svc := newStatServiceForTest(repo, counter)

	if err := svc.RecordPV(); err != nil {
		t.Fatalf("expected RecordPV success, got %v", err)
	}
	if err := svc.RecordUV("127.0.0.1"); err != nil {
		t.Fatalf("expected RecordUV success, got %v", err)
	}
	if err := svc.RecordCommentCount(); err != nil {
		t.Fatalf("expected RecordCommentCount success, got %v", err)
	}

	if len(repo.increments) != 0 {
		t.Fatalf("expected record methods to avoid direct database writes, got %#v", repo.increments)
	}
	if len(counter.pvDates) != 1 || len(counter.uvInputs) != 1 || len(counter.commentDates) != 1 {
		t.Fatalf("expected counter store to receive all increments, got pv=%#v uv=%#v comments=%#v", counter.pvDates, counter.uvInputs, counter.commentDates)
	}
}

func TestStatServiceFlushDailyStatCounters_PersistsAggregatedDeltaOnce(t *testing.T) {
	repo := &stubDailyStatRepository{}
	counter := &stubStatCounterStore{drained: statCounterDelta{PV: 12, UV: 3, CommentCount: 2}}
	svc := newStatServiceForTest(repo, counter)

	if err := svc.FlushDailyStatCounters("2026-06-05"); err != nil {
		t.Fatalf("expected flush success, got %v", err)
	}

	if len(repo.increments) != 1 {
		t.Fatalf("expected one aggregated database increment, got %#v", repo.increments)
	}
	got := repo.increments[0]
	if got.Date != "2026-06-05" || got.PV != 12 || got.UV != 3 || got.CommentCount != 2 {
		t.Fatalf("unexpected increment payload: %#v", got)
	}
}

func TestStatServiceFlushRecentDailyStatCounters_FlushesTodayAndYesterday(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	repo := &stubDailyStatRepository{}
	counter := &stubStatCounterStore{drainedByDate: map[string]statCounterDelta{
		today:     {PV: 5},
		yesterday: {PV: 2, UV: 1},
	}}
	svc := newStatServiceForTest(repo, counter)

	if err := svc.FlushRecentDailyStatCounters(); err != nil {
		t.Fatalf("expected recent flush success, got %v", err)
	}

	if len(counter.drainDates) != 2 || counter.drainDates[0] != today || counter.drainDates[1] != yesterday {
		t.Fatalf("expected flush to drain today and yesterday, got %#v", counter.drainDates)
	}
	if len(repo.increments) != 2 {
		t.Fatalf("expected two aggregated increments, got %#v", repo.increments)
	}
	if repo.increments[0].Date != today || repo.increments[0].PV != 5 {
		t.Fatalf("unexpected today increment: %#v", repo.increments[0])
	}
	if repo.increments[1].Date != yesterday || repo.increments[1].PV != 2 || repo.increments[1].UV != 1 {
		t.Fatalf("unexpected yesterday increment: %#v", repo.increments[1])
	}
}

func TestStatServiceGetDashboardStats_DoesNotFlushCountersOnReadPath(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	repo := &stubDailyStatRepository{
		stats: []model.DailyStat{
			{Date: today, PV: 11, UV: 3, CommentCount: 2},
		},
	}
	counter := &stubStatCounterStore{drainedByDate: map[string]statCounterDelta{today: {PV: 9}}}
	svc := newStatServiceForTest(repo, counter)

	if _, err := svc.GetDashboardStats(7); err != nil {
		t.Fatalf("expected dashboard stats success, got %v", err)
	}
	if len(counter.drainDates) != 0 {
		t.Fatalf("expected dashboard read to avoid flushing counters, got %v", counter.drainDates)
	}
}
