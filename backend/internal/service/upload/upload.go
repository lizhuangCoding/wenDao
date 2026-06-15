package upload

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/watermark"
	"wenDao/internal/repository"
	"wenDao/internal/svcerrors"
)

// UploadService 上传服务接口
type UploadService interface {
	UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	CleanupByFilePath(filePath string) error
	CleanupUnreferenced(now time.Time) (UploadCleanupResult, error)
}

// uploadService 上传服务实现
type uploadService struct {
	uploadRepo repository.UploadRepository
	cfg        *config.Config
}

// UploadCleanupResult 上传清理结果
type UploadCleanupResult struct {
	Candidates int
	Deleted    int
	Skipped    int
}

// NewUploadService 创建上传服务实例
func NewUploadService(uploadRepo repository.UploadRepository, cfg *config.Config) UploadService {
	return &uploadService{
		uploadRepo: uploadRepo,
		cfg:        cfg,
	}
}

func (s *uploadService) shouldCompress(contentType string) bool {
	if !s.cfg.Upload.EnableImageCompression {
		return false
	}
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
}

func resizeImageIfNeeded(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= maxWidth && height <= maxHeight {
		return img
	}

	ratioW := float64(maxWidth) / float64(width)
	ratioH := float64(maxHeight) / float64(height)
	ratio := ratioW
	if ratioH < ratio {
		ratio = ratioH
	}

	newWidth := int(float64(width) * ratio)
	newHeight := int(float64(height) * ratio)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := bounds.Min.X + x*width/newWidth
			srcY := bounds.Min.Y + y*height/newHeight
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	return dst
}

func (s *uploadService) compressImage(fileBytes []byte, contentType string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	img = resizeImageIfNeeded(img, s.cfg.Upload.MaxImageWidth, s.cfg.Upload.MaxImageHeight)

	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: s.cfg.Upload.ImageQuality}); err != nil {
			return nil, fmt.Errorf("failed to encode jpeg: %w", err)
		}
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode png: %w", err)
		}
	case "image/webp":
		return fileBytes, nil
	default:
		return fileBytes, nil
	}

	return buf.Bytes(), nil
}

// UploadImage 上传图片
func (s *uploadService) UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.uploadImage(file, header, userID, watermark.ModeNone)
}

// UploadArticleImage 上传文章图片并添加版权水印
func (s *uploadService) UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.uploadImage(file, header, userID, watermark.ModeText)
}

// UploadCoverImage 上传封面图片并添加裁剪安全的版权水印
func (s *uploadService) UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.uploadImage(file, header, userID, watermark.ModeTile)
}

func (s *uploadService) uploadImage(file multipart.File, header *multipart.FileHeader, userID int64, wmMode watermark.Mode) (*model.Upload, error) {
	if header.Size > s.cfg.Upload.MaxSize {
		return nil, fmt.Errorf("%w: maximum %d bytes", svcerrors.ErrFileSizeExceedsLimit, s.cfg.Upload.MaxSize)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	detectedContentType := http.DetectContentType(fileBytes)
	if !s.isAllowedType(detectedContentType) {
		return nil, svcerrors.ErrFileTypeNotAllowed
	}

	safeExt, ok := safeExtensionForContentType(detectedContentType)
	if !ok {
		return nil, svcerrors.ErrFileTypeNotAllowed
	}

	storedBytes := fileBytes
	if s.shouldCompress(detectedContentType) {
		compressed, err := s.compressImage(fileBytes, detectedContentType)
		if err != nil {
			return nil, err
		}
		storedBytes = compressed
	}

	if wmMode != watermark.ModeNone {
		if detectedContentType == "image/jpeg" || detectedContentType == "image/png" {
			img, _, err := image.Decode(bytes.NewReader(storedBytes))
			if err != nil {
				return nil, fmt.Errorf("failed to decode image for watermark: %w", err)
			}
			watermarked, err := watermark.Apply(img, "lizhuang", wmMode)
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			switch detectedContentType {
			case "image/jpeg":
				if err := jpeg.Encode(&buf, watermarked, &jpeg.Options{Quality: s.cfg.Upload.ImageQuality}); err != nil {
					return nil, fmt.Errorf("failed to encode jpeg watermark: %w", err)
				}
			case "image/png":
				encoder := png.Encoder{CompressionLevel: png.BestCompression}
				if err := encoder.Encode(&buf, watermarked); err != nil {
					return nil, fmt.Errorf("failed to encode png watermark: %w", err)
				}
			}
			storedBytes = buf.Bytes()
		}
	}

	hash := md5.Sum(storedBytes)
	hashStr := hex.EncodeToString(hash[:])
	timestamp := time.Now().UnixNano()
	newFilename := fmt.Sprintf("%s_%d%s", hashStr, timestamp, safeExt)

	now := time.Now()
	subPath := filepath.Join(fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()), newFilename)
	fullPath := filepath.Join(s.cfg.Upload.StoragePath, subPath)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	destFile, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer destFile.Close()

	if _, err := destFile.Write(storedBytes); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	upload := &model.Upload{
		UserID:   userID,
		Filename: header.Filename,
		FilePath: "/uploads/" + filepath.ToSlash(subPath),
		FileSize: int64(len(storedBytes)),
		MimeType: detectedContentType,
		FileType: "image",
	}

	if err := s.uploadRepo.Create(upload); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("failed to save upload record: %w", err)
	}

	return upload, nil
}

// CleanupByFilePath 删除上传记录和本地文件
func (s *uploadService) CleanupByFilePath(filePath string) error {
	fullPath, ok := managedUploadDiskPath(s.cfg.Upload.StoragePath, filePath)
	if !ok {
		return fmt.Errorf("unsafe upload file path: %s", filePath)
	}

	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := s.uploadRepo.DeleteByFilePath(filePath); err != nil {
		return err
	}

	return nil
}

// CleanupUnreferenced 删除数据库已确认未引用、且超过保留期的上传文件。
func (s *uploadService) CleanupUnreferenced(now time.Time) (UploadCleanupResult, error) {
	var result UploadCleanupResult
	if s == nil || s.uploadRepo == nil || s.cfg == nil {
		return result, nil
	}

	retentionDays := s.cfg.Upload.CleanupRetentionDays
	if retentionDays <= 0 {
		retentionDays = 2
	}
	batchSize := s.cfg.Upload.CleanupBatchSize
	if batchSize <= 0 {
		batchSize = 200
	}

	cutoff := now.Add(-time.Duration(retentionDays) * 24 * time.Hour)
	candidates, err := s.uploadRepo.ListUnreferencedBefore(cutoff, batchSize)
	if err != nil {
		return result, err
	}

	for _, upload := range candidates {
		result.Candidates++
		if upload == nil || upload.ID <= 0 {
			result.Skipped++
			continue
		}
		fullPath, ok := managedUploadDiskPath(s.cfg.Upload.StoragePath, upload.FilePath)
		if !ok {
			result.Skipped++
			continue
		}
		if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("failed to remove upload file %q: %w", upload.FilePath, err)
		}
		if err := s.uploadRepo.DeleteByID(upload.ID); err != nil {
			return result, fmt.Errorf("failed to delete upload record %d: %w", upload.ID, err)
		}
		result.Deleted++
	}

	return result, nil
}

func managedUploadDiskPath(storagePath string, filePath string) (string, bool) {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" {
		return "", false
	}

	uploadPath := strings.TrimSpace(filePath)
	relativePath := strings.TrimPrefix(uploadPath, "/uploads/")
	if relativePath == uploadPath {
		relativePath = strings.TrimPrefix(uploadPath, "uploads/")
	}
	if relativePath == "" || relativePath == uploadPath || strings.Contains(relativePath, "\\") {
		return "", false
	}

	segments := strings.Split(relativePath, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", false
		}
	}

	cleanRelativePath := filepath.Clean(filepath.FromSlash(relativePath))
	if cleanRelativePath == "." || filepath.IsAbs(cleanRelativePath) {
		return "", false
	}

	storageAbs, err := filepath.Abs(storagePath)
	if err != nil {
		return "", false
	}
	fullPath, err := filepath.Abs(filepath.Join(storageAbs, cleanRelativePath))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(storageAbs, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}

	return fullPath, true
}

func safeExtensionForContentType(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

// isAllowedType 检查文件类型是否允许
func (s *uploadService) isAllowedType(contentType string) bool {
	for _, allowedType := range s.cfg.Upload.AllowedTypes {
		if strings.EqualFold(contentType, allowedType) {
			return true
		}
	}
	return false
}
