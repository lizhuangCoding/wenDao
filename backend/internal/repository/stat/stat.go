package stat

import (
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
	var stat model.DailyStat
	result := r.db.Where("date = ?", date).First(&stat)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		stat = model.DailyStat{Date: date}
		if isPV {
			stat.PV = 1
		} else {
			stat.UV = 1
		}
		return r.db.Create(&stat).Error
	}

	// 更新现有记录
	updates := map[string]interface{}{}
	if isPV {
		updates["pv"] = gorm.Expr("pv + 1")
	} else {
		updates["uv"] = gorm.Expr("uv + 1")
	}
	return r.db.Model(&stat).Updates(updates).Error
}

// IncrCommentCount 增加评论数
func (r *StatRepository) IncrCommentCount(date string) error {
	var stat model.DailyStat
	result := r.db.Where("date = ?", date).First(&stat)

	if result.Error == gorm.ErrRecordNotFound {
		stat = model.DailyStat{Date: date, CommentCount: 1}
		return r.db.Create(&stat).Error
	}

	return r.db.Model(&stat).Update("comment_count", gorm.Expr("comment_count + 1")).Error
}