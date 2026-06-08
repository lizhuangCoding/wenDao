package stat

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"
	"wenDao/internal/model"
	"wenDao/internal/repository"

	"github.com/redis/go-redis/v9"
)

type dailyStatRepository interface {
	GetDailyStats(days int) ([]model.DailyStat, error)
	GetDailyStatsByRange(startDate, endDate string) ([]model.DailyStat, error)
	GetArticleStats(articleID int64, days int) ([]model.ArticleStat, error)
	IncrementDailyStat(date string, pvDelta, uvDelta, commentDelta int64) error
}

type statCounterDelta struct {
	PV           int64
	UV           int64
	CommentCount int64
}

type statCounterStore interface {
	IncrementPV(ctx context.Context, date string) error
	AddUV(ctx context.Context, date string, visitor string) (bool, error)
	IncrementComment(ctx context.Context, date string) error
	DrainDate(ctx context.Context, date string) (statCounterDelta, error)
	RestoreDate(ctx context.Context, date string, delta statCounterDelta) error
}

type StatService struct {
	statRepo dailyStatRepository
	counter  statCounterStore
}

func NewStatService(statRepo *repository.StatRepository, rdb *redis.Client) *StatService {
	return newStatServiceForTest(statRepo, newRedisStatCounterStore(rdb))
}

func newStatServiceForTest(statRepo dailyStatRepository, counter statCounterStore) *StatService {
	return &StatService{statRepo: statRepo, counter: counter}
}

type StatData struct {
	Labels []string `json:"labels"`
	PV     []int64  `json:"pv"`
	UV     []int64  `json:"uv"`
}

type ArticleStatData struct {
	ArticleID int64    `json:"article_id"`
	Title     string   `json:"title"`
	TotalPV   int64    `json:"total_pv"`
	Labels    []string `json:"labels"`
	PV        []int64  `json:"pv"`
}

type DashboardStats struct {
	TotalPV       int64    `json:"total_pv"`
	TotalUV       int64    `json:"total_uv"`
	TotalComments int64    `json:"total_comments"`
	DailyStat     StatData `json:"daily_stat"`
}

// GetDashboardStats 获取后台统计数据（按天数）
func (s *StatService) GetDashboardStats(days int) (*DashboardStats, error) {
	if days <= 0 {
		days = 7
	}

	dailyStats, err := s.statRepo.GetDailyStats(days)
	if err != nil {
		return nil, err
	}

	end := time.Now()
	start := end.AddDate(0, 0, -(days - 1))

	return buildDashboardStats(dailyStats, start, end), nil
}

// GetDashboardStatsByRange 获取后台统计数据（按日期范围）
func (s *StatService) GetDashboardStatsByRange(startDate, endDate string) (*DashboardStats, error) {
	start, startErr := time.ParseInLocation("2006-01-02", startDate, time.Local)
	end, endErr := time.ParseInLocation("2006-01-02", endDate, time.Local)
	if startErr != nil || endErr != nil || start.After(end) {
		return nil, fmt.Errorf("invalid date range")
	}

	dailyStats, err := s.statRepo.GetDailyStatsByRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	return buildDashboardStats(dailyStats, start, end), nil
}

func buildDashboardStats(dailyStats []model.DailyStat, start, end time.Time) *DashboardStats {
	byDate := make(map[string]model.DailyStat, len(dailyStats))
	for _, ds := range dailyStats {
		byDate[ds.Date] = ds
	}

	stat := &DashboardStats{
		DailyStat: StatData{
			Labels: make([]string, 0),
			PV:     make([]int64, 0),
			UV:     make([]int64, 0),
		},
	}

	startDate := dateOnly(start)
	endDate := dateOnly(end)
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		ds := byDate[date]
		stat.TotalPV += ds.PV
		stat.TotalUV += ds.UV
		stat.TotalComments += ds.CommentCount
		stat.DailyStat.Labels = append(stat.DailyStat.Labels, date)
		stat.DailyStat.PV = append(stat.DailyStat.PV, ds.PV)
		stat.DailyStat.UV = append(stat.DailyStat.UV, ds.UV)
	}

	return stat
}

func dateOnly(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// GetArticleStats 获取单个文章的访问统计
func (s *StatService) GetArticleStats(articleID int64, days int) ([]model.ArticleStat, error) {
	return s.statRepo.GetArticleStats(articleID, days)
}

// RecordPV 记录页面浏览
func (s *StatService) RecordPV() error {
	date := time.Now().Format("2006-01-02")
	if s.counter != nil {
		if err := s.counter.IncrementPV(context.Background(), date); err == nil {
			return nil
		} else {
			log.Printf("[Stat] RecordPV: counter error: %v", err)
		}
	}
	return s.incrementDailyStat(date, 1, 0, 0)
}

// RecordUV 记录独立访客（基于IP地址，Redis去重）
func (s *StatService) RecordUV(ip string) error {
	date := time.Now().Format("2006-01-02")
	visitor := normalizeVisitorID(ip)
	if s.counter != nil {
		if _, err := s.counter.AddUV(context.Background(), date, visitor); err == nil {
			return nil
		} else {
			log.Printf("[Stat] RecordUV: counter error: %v", err)
		}
	}
	return s.incrementDailyStat(date, 0, 1, 0)
}

// RecordCommentCount 记录评论数
func (s *StatService) RecordCommentCount() error {
	date := time.Now().Format("2006-01-02")
	if s.counter != nil {
		if err := s.counter.IncrementComment(context.Background(), date); err == nil {
			return nil
		} else {
			log.Printf("[Stat] RecordCommentCount: counter error: %v", err)
		}
	}
	return s.incrementDailyStat(date, 0, 0, 1)
}

func (s *StatService) FlushDailyStatCounters(date string) error {
	if s.counter == nil {
		return nil
	}
	delta, err := s.counter.DrainDate(context.Background(), date)
	if err != nil {
		return err
	}
	if delta.PV == 0 && delta.UV == 0 && delta.CommentCount == 0 {
		return nil
	}
	if err := s.incrementDailyStat(date, delta.PV, delta.UV, delta.CommentCount); err != nil {
		if restoreErr := s.counter.RestoreDate(context.Background(), date, delta); restoreErr != nil {
			log.Printf("[Stat] FlushDailyStatCounters: restore error: %v", restoreErr)
		}
		return err
	}
	return nil
}

func (s *StatService) FlushRecentDailyStatCounters() error {
	now := time.Now()
	dates := []string{
		now.Format("2006-01-02"),
		now.AddDate(0, 0, -1).Format("2006-01-02"),
	}
	var firstErr error
	for _, date := range dates {
		if err := s.FlushDailyStatCounters(date); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *StatService) flushRecentCountersQuietly() {
	if s == nil {
		return
	}
	if err := s.FlushRecentDailyStatCounters(); err != nil {
		log.Printf("[Stat] flush recent counters error: %v", err)
	}
}

func (s *StatService) incrementDailyStat(date string, pvDelta, uvDelta, commentDelta int64) error {
	if s.statRepo == nil {
		return fmt.Errorf("stat repository is not configured")
	}
	return s.statRepo.IncrementDailyStat(date, pvDelta, uvDelta, commentDelta)
}

func normalizeVisitorID(ip string) string {
	if ip == "" || ip == "unknown" {
		return "local"
	}
	if ip == "127.0.0.1" || ip == "::1" {
		return "localhost"
	}
	return ip
}

const statCounterTTL = 48 * time.Hour

type redisStatCounterStore struct {
	rdb *redis.Client
}

func newRedisStatCounterStore(rdb *redis.Client) statCounterStore {
	if rdb == nil {
		return nil
	}
	return &redisStatCounterStore{rdb: rdb}
}

func (s *redisStatCounterStore) IncrementPV(ctx context.Context, date string) error {
	key := statCounterKey(date, "pv")
	if err := s.rdb.Incr(ctx, key).Err(); err != nil {
		return err
	}
	_ = s.rdb.Expire(ctx, key, statCounterTTL).Err()
	return nil
}

func (s *redisStatCounterStore) AddUV(ctx context.Context, date string, visitor string) (bool, error) {
	setKey := statCounterKey(date, "uv_visitors")
	added, err := s.rdb.SAdd(ctx, setKey, visitor).Result()
	if err != nil {
		return false, err
	}
	_ = s.rdb.Expire(ctx, setKey, statCounterTTL).Err()
	if added == 0 {
		return false, nil
	}

	countKey := statCounterKey(date, "uv")
	if err := s.rdb.Incr(ctx, countKey).Err(); err != nil {
		return true, err
	}
	_ = s.rdb.Expire(ctx, countKey, statCounterTTL).Err()
	return true, nil
}

func (s *redisStatCounterStore) IncrementComment(ctx context.Context, date string) error {
	key := statCounterKey(date, "comments")
	if err := s.rdb.Incr(ctx, key).Err(); err != nil {
		return err
	}
	_ = s.rdb.Expire(ctx, key, statCounterTTL).Err()
	return nil
}

func (s *redisStatCounterStore) DrainDate(ctx context.Context, date string) (statCounterDelta, error) {
	pv, err := s.drainCounter(ctx, statCounterKey(date, "pv"))
	if err != nil {
		return statCounterDelta{}, err
	}
	uv, err := s.drainCounter(ctx, statCounterKey(date, "uv"))
	if err != nil {
		return statCounterDelta{}, err
	}
	comments, err := s.drainCounter(ctx, statCounterKey(date, "comments"))
	if err != nil {
		return statCounterDelta{}, err
	}
	return statCounterDelta{PV: pv, UV: uv, CommentCount: comments}, nil
}

func (s *redisStatCounterStore) RestoreDate(ctx context.Context, date string, delta statCounterDelta) error {
	var firstErr error
	restore := func(key string, value int64) {
		if value == 0 {
			return
		}
		if err := s.rdb.IncrBy(ctx, key, value).Err(); err != nil && firstErr == nil {
			firstErr = err
			return
		}
		_ = s.rdb.Expire(ctx, key, statCounterTTL).Err()
	}

	restore(statCounterKey(date, "pv"), delta.PV)
	restore(statCounterKey(date, "uv"), delta.UV)
	restore(statCounterKey(date, "comments"), delta.CommentCount)
	return firstErr
}

func (s *redisStatCounterStore) drainCounter(ctx context.Context, key string) (int64, error) {
	value, err := s.rdb.GetSet(ctx, key, "0").Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	_ = s.rdb.Expire(ctx, key, statCounterTTL).Err()
	if value == "" {
		return 0, nil
	}
	count, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func statCounterKey(date string, name string) string {
	return fmt.Sprintf("stats:daily:%s:%s", date, name)
}
