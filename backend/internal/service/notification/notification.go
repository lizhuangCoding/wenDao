package notification

import (
	"fmt"

	"wenDao/internal/model"
	notifrepo "wenDao/internal/repository/notification"
)

// NotificationService 通知服务接口
type NotificationService interface {
	Create(userID int64, notifType, title, content, linkURL string) error
	ListByUser(userID int64, notifType string, page, pageSize int) ([]*model.Notification, int64, error)
	GetUnreadCount(userID int64) (int64, error)
	MarkRead(userID, notificationID int64) error
	MarkAllRead(userID int64) error
	BroadcastToAllUsers(title, content, linkURL string, getUserIDs func() ([]int64, error)) error
}

type notificationService struct {
	repo notifrepo.NotificationRepository
}

func NewNotificationService(repo notifrepo.NotificationRepository) NotificationService {
	return &notificationService{repo: repo}
}

func (s *notificationService) Create(userID int64, notifType, title, content, linkURL string) error {
	notif := &model.Notification{
		UserID:  userID,
		Type:    notifType,
		Title:   title,
		Content: content,
		LinkURL: linkURL,
		IsRead:  false,
	}
	if err := s.repo.Create(notif); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (s *notificationService) ListByUser(userID int64, notifType string, page, pageSize int) ([]*model.Notification, int64, error) {
	return s.repo.ListByUser(userID, notifType, page, pageSize)
}

func (s *notificationService) GetUnreadCount(userID int64) (int64, error) {
	return s.repo.GetUnreadCount(userID)
}

func (s *notificationService) MarkRead(userID, notificationID int64) error {
	return s.repo.MarkRead(userID, notificationID)
}

func (s *notificationService) MarkAllRead(userID int64) error {
	return s.repo.MarkAllRead(userID)
}

func (s *notificationService) BroadcastToAllUsers(title, content, linkURL string, getUserIDs func() ([]int64, error)) error {
	userIDs, err := getUserIDs()
	if err != nil {
		return fmt.Errorf("failed to get user IDs for broadcast: %w", err)
	}

	notifications := make([]*model.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		notifications = append(notifications, &model.Notification{
			UserID:  uid,
			Type:    model.NotificationTypeAdminBroadcast,
			Title:   title,
			Content: content,
			LinkURL: linkURL,
			IsRead:  false,
		})
	}

	return s.repo.CreateBatch(notifications)
}
