package stat

import (
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
