package stat

import (
	"context"
	"fmt"
	"log"
	"time"
	"wenDao/internal/model"
	"wenDao/internal/repository"

	"github.com/redis/go-redis/v9"
)

type StatService struct {
	statRepo *repository.StatRepository
	rdb      *redis.Client
}

func NewStatService(statRepo *repository.StatRepository, rdb *redis.Client) *StatService {
	return &StatService{statRepo: statRepo, rdb: rdb}
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
	dailyStats, err := s.statRepo.GetDailyStats(days)
	if err != nil {
		return nil, err
	}

	if days <= 0 {
		days = 7
	}
	end := time.Now()
	start := end.AddDate(0, 0, -(days - 1))

	return buildDashboardStats(dailyStats, start, end), nil
}

// GetDashboardStatsByRange 获取后台统计数据（按日期范围）
func (s *StatService) GetDashboardStatsByRange(startDate, endDate string) (*DashboardStats, error) {
	dailyStats, err := s.statRepo.GetDailyStatsByRange(startDate, endDate)
	if err != nil {
		return nil, err
	}

	start, startErr := time.ParseInLocation("2006-01-02", startDate, time.Local)
	end, endErr := time.ParseInLocation("2006-01-02", endDate, time.Local)
	if startErr != nil || endErr != nil || start.After(end) {
		return nil, fmt.Errorf("invalid date range")
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
	return s.statRepo.CreateOrUpdateDailyStat(date, true)
}

// RecordUV 记录独立访客（基于IP地址，Redis去重）
func (s *StatService) RecordUV(ip string) error {
	// 对于本地/内网访问，使用固定标识符"local"来记录
	// 这样本地测试时也能统计到UV
	if ip == "" || ip == "unknown" {
		ip = "local"
	} else if ip == "127.0.0.1" || ip == "::1" {
		// 本地访问使用固定key，便于统计
		ip = "localhost"
	}

	date := time.Now().Format("2006-01-02")
	ctx := context.Background()

	// 使用 Redis Set 存储今日访问的IP，key: "uv:2024-04-07"
	key := fmt.Sprintf("uv:%s", date)

	// SAdd 返回添加的元素数量，如果 ip 已存在则返回 0
	added, err := s.rdb.SAdd(ctx, key, ip).Result()
	if err != nil {
		log.Printf("[Stat] RecordUV: SAdd error: %v", err)
		return err
	}

	// 只有新IP才计入UV
	if added > 0 {
		// 设置过期时间为 48 小时，确保数据在统计完成后能被自动清理
		s.rdb.Expire(ctx, key, 48*time.Hour)

		err := s.statRepo.CreateOrUpdateDailyStat(date, false)
		if err != nil {
			log.Printf("[Stat] RecordUV: CreateOrUpdateDailyStat error: %v", err)
		}
		return err
	}
	return nil
}

// RecordCommentCount 记录评论数
func (s *StatService) RecordCommentCount() error {
	date := time.Now().Format("2006-01-02")
	return s.statRepo.IncrCommentCount(date)
}
