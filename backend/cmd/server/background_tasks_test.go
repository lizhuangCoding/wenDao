package main

import (
	"mime/multipart"
	"testing"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/service"
)

type immediateCleanupUploadService struct {
	called chan time.Time
}

func (s *immediateCleanupUploadService) UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) CleanupByFilePath(filePath string) error {
	return nil
}

func (s *immediateCleanupUploadService) CleanupUnreferenced(now time.Time) (service.UploadCleanupResult, error) {
	s.called <- now
	return service.UploadCleanupResult{}, nil
}

func TestStartUploadCleanupScheduler_RunsCleanupImmediately(t *testing.T) {
	uploadService := &immediateCleanupUploadService{called: make(chan time.Time, 1)}
	stop := startUploadCleanupScheduler(&config.Config{
		Upload: config.UploadConfig{
			CleanupEnabled:       true,
			CleanupIntervalHours: 24,
		},
	}, zap.NewNop(), uploadService)
	defer stop()

	select {
	case <-uploadService.called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected upload cleanup to run immediately after scheduler starts")
	}
}

type immediateArticleSchedulerService struct {
	published chan int64
}

func (s *immediateArticleSchedulerService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetByID(id int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) List(status string, categoryID, tagID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (s *immediateArticleSchedulerService) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) SetTags(id int64, tagIDs []int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) Delete(id int64) error         { return nil }
func (s *immediateArticleSchedulerService) DeleteBatch(ids []int64) error { return nil }
func (s *immediateArticleSchedulerService) Publish(id int64) error        { return nil }
func (s *immediateArticleSchedulerService) Draft(id int64) error          { return nil }
func (s *immediateArticleSchedulerService) AutoSave(id int64, title, content, summary string) error {
	return nil
}
func (s *immediateArticleSchedulerService) IncrViewCount(id int64) error { return nil }
func (s *immediateArticleSchedulerService) LikeArticle(id int64) error   { return nil }
func (s *immediateArticleSchedulerService) UnlikeArticle(id int64) error { return nil }
func (s *immediateArticleSchedulerService) LikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UnlikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) FavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UnfavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetArticleInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) ListArticlesByInteraction(userID int64, interactionType string, page, pageSize int) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (s *immediateArticleSchedulerService) ToggleTop(id int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UpdatePopularityScores() error { return nil }
func (s *immediateArticleSchedulerService) GetAllPublished() ([]*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetDueScheduledArticles() ([]*model.Article, error) {
	scheduledAt := time.Now().Add(-time.Minute)
	return []*model.Article{{ID: 77, Title: "due", ScheduledPublishAt: &scheduledAt}}, nil
}
func (s *immediateArticleSchedulerService) PublishScheduled(articleID int64) error {
	s.published <- articleID
	return nil
}
func (s *immediateArticleSchedulerService) SetScheduledPublishAt(articleID int64, scheduledAt *time.Time) error {
	return nil
}

func TestStartArticleScheduler_RunsImmediately(t *testing.T) {
	articleService := &immediateArticleSchedulerService{published: make(chan int64, 1)}
	stop := startArticleScheduler(zap.NewNop(), articleService)
	defer stop()

	select {
	case articleID := <-articleService.published:
		if articleID != 77 {
			t.Fatalf("expected article 77 to publish, got %d", articleID)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected article scheduler to process due articles immediately after start")
	}
}
