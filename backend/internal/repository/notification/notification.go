package notification

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// NotificationRepository 通知数据访问接口
type NotificationRepository interface {
	Create(notification *model.Notification) error
	ListByUser(userID int64, page, pageSize int) ([]*model.Notification, int64, error)
	GetUnreadCount(userID int64) (int64, error)
	MarkRead(userID, notificationID int64) error
	MarkAllRead(userID int64) error
	CreateBatch(notifications []*model.Notification) error
}

type notificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(notification *model.Notification) error {
	return r.db.Create(notification).Error
}

func (r *notificationRepository) ListByUser(userID int64, page, pageSize int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64

	query := r.db.Model(&model.Notification{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&notifications).Error
	return notifications, total, err
}

func (r *notificationRepository) GetUnreadCount(userID int64) (int64, error) {
	var count int64
	err := r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) MarkRead(userID, notificationID int64) error {
	return r.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

func (r *notificationRepository) MarkAllRead(userID int64) error {
	return r.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *notificationRepository) CreateBatch(notifications []*model.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return r.db.CreateInBatches(notifications, 200).Error
}
