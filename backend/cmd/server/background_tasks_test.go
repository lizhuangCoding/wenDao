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
