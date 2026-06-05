package stat

import (
	"time"

	"wenDao/internal/model"

	"gorm.io/gorm"
)

// StatRepository 统计仓库
type StatRepository struct {
	db *gorm.DB
}

// NewStatRepository 创建统计仓库
func NewStatRepository(db *gorm.DB) *StatRepository {
	return &StatRepository{db: db}
}

// GetDailyStats 获取每日流量统计
func (r *StatRepository) GetDailyStats(days int) ([]model.DailyStat, error) {
	var stats []model.DailyStat
	err := r.db.Where("date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)", days).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

// GetDailyStatsByRange 按日期范围获取流量统计
func (r *StatRepository) GetDailyStatsByRange(startDate, endDate string) ([]model.DailyStat, error) {
	var stats []model.DailyStat
	err := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

// GetArticleStats 获取文章访问统计
func (r *StatRepository) GetArticleStats(articleID int64, days int) ([]model.ArticleStat, error) {
	var stats []model.ArticleStat
	err := r.db.Where("article_id = ? AND date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)", articleID, days).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

// GetArticleStatsByDateRange 按日期范围获取文章统计
func (r *StatRepository) GetArticleStatsByDateRange(articleID int64, startDate, endDate string) ([]model.ArticleStat, error) {
	var stats []model.ArticleStat
	err := r.db.Where("article_id = ? AND date >= ? AND date <= ?", articleID, startDate, endDate).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

// GetAllArticleStats 获取所有文章在指定日期范围内的统计
func (r *StatRepository) GetAllArticleStats(days int) ([]model.ArticleStat, error) {
	var stats []model.ArticleStat
	err := r.db.Where("date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)", days).
		Order("article_id ASC, date ASC").
		Find(&stats).Error
	return stats, err
}

// GetDailyStat 获取指定日期的统计
func (r *StatRepository) GetDailyStat(date string) (*model.DailyStat, error) {
	var stat model.DailyStat
	err := r.db.Where("date = ?", date).First(&stat).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stat, err
}

// CreateOrUpdateDailyStat 创建或更新每日统计
func (r *StatRepository) CreateOrUpdateDailyStat(date string, isPV bool) error {
	if isPV {
		return r.IncrementDailyStat(date, 1, 0, 0)
	}
	return r.IncrementDailyStat(date, 0, 1, 0)
}

// IncrementDailyStat 使用单条 upsert 原子累加统计值，避免先查再写。
func (r *StatRepository) IncrementDailyStat(date string, pvDelta, uvDelta, commentDelta int64) error {
	if pvDelta == 0 && uvDelta == 0 && commentDelta == 0 {
		return nil
	}
	now := time.Now()
	return r.db.Exec(
		`INSERT INTO daily_stats (date, pv, uv, comment_count, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
pv = pv + VALUES(pv),
uv = uv + VALUES(uv),
comment_count = comment_count + VALUES(comment_count),
updated_at = VALUES(updated_at)`,
		date, pvDelta, uvDelta, commentDelta, now, now,
	).Error
}

// IncrCommentCount 增加评论数
func (r *StatRepository) IncrCommentCount(date string) error {
	return r.IncrementDailyStat(date, 0, 0, 1)
}
