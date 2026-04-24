package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
	"wenDao/internal/repository"
)

// UploadService 上传服务接口
type UploadService interface {
	UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
	CleanupByFilePath(filePath string) error
}

// uploadService 上传服务实现
type uploadService struct {
	uploadRepo repository.UploadRepository
	cfg        *config.Config
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
	return s.uploadImage(file, header, userID, watermarkNone)
}

// UploadArticleImage 上传文章图片并添加版权水印
func (s *uploadService) UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.uploadImage(file, header, userID, watermarkCorner)
}

// UploadCoverImage 上传封面图片并添加裁剪安全的版权水印
func (s *uploadService) UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.uploadImage(file, header, userID, watermarkCenter)
}

type watermarkMode int

const (
	watermarkNone watermarkMode = iota
	watermarkCorner
	watermarkCenter
)

func (s *uploadService) uploadImage(file multipart.File, header *multipart.FileHeader, userID int64, watermark watermarkMode) (*model.Upload, error) {
	if header.Size > s.cfg.Upload.MaxSize {
		return nil, fmt.Errorf("file size exceeds limit: %d bytes", s.cfg.Upload.MaxSize)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	detectedContentType := http.DetectContentType(fileBytes)
	if !s.isAllowedType(detectedContentType) {
		return nil, errors.New("file type not allowed")
	}

	safeExt, ok := safeExtensionForContentType(detectedContentType)
	if !ok {
		return nil, errors.New("file type not allowed")
	}

	storedBytes := fileBytes
	if s.shouldCompress(detectedContentType) {
		compressed, err := s.compressImage(fileBytes, detectedContentType)
		if err != nil {
			return nil, err
		}
		storedBytes = compressed
	}

	if watermark != watermarkNone {
		watermarked, err := s.applyWatermark(storedBytes, detectedContentType, watermark)
		if err != nil {
			return nil, err
		}
		storedBytes = watermarked
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

func (s *uploadService) applyWatermark(fileBytes []byte, contentType string, mode watermarkMode) ([]byte, error) {
	if contentType != "image/jpeg" && contentType != "image/png" {
		return fileBytes, nil
	}

	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image for watermark: %w", err)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	drawTextWatermark(rgba, "lizhuang", mode)

	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg":
		if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: s.cfg.Upload.ImageQuality}); err != nil {
			return nil, fmt.Errorf("failed to encode jpeg watermark: %w", err)
		}
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(&buf, rgba); err != nil {
			return nil, fmt.Errorf("failed to encode png watermark: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func drawTextWatermark(img *image.RGBA, text string, mode watermarkMode) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 24 || height < 16 {
		return
	}

	scale := width / 160
	if height/80 < scale {
		scale = height / 80
	}
	if scale < 2 {
		scale = 2
	}
	if scale > 6 {
		scale = 6
	}

	charWidth := 5 * scale
	gap := scale
	textWidth := len(text)*charWidth + (len(text)-1)*gap
	textHeight := 7 * scale
	margin := 6 * scale
	x := width - textWidth - margin
	y := height - textHeight - margin
	if mode == watermarkCenter {
		x = (width - textWidth) / 2
		y = (height - textHeight) / 2
	}
	if x < margin {
		x = margin
	}
	if y < margin {
		y = margin
	}

	drawBitmapText(img, text, x+scale, y+scale, scale, color.RGBA{A: 120})
	drawBitmapText(img, text, x, y, scale, color.RGBA{R: 255, G: 255, B: 255, A: 190})
}

func drawBitmapText(img *image.RGBA, text string, x, y, scale int, c color.RGBA) {
	cursor := x
	for _, ch := range strings.ToLower(text) {
		glyph, ok := watermarkGlyphs[ch]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for row, pattern := range glyph {
			for col, pixel := range pattern {
				if pixel != '1' {
					continue
				}
				fillRectAlpha(img, cursor+col*scale, y+row*scale, scale, scale, c)
			}
		}
		cursor += 6 * scale
	}
}

func fillRectAlpha(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	bounds := img.Bounds()
	for py := y; py < y+height; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for px := x; px < x+width; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			blendPixel(img, px, py, c)
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, overlay color.RGBA) {
	base := img.RGBAAt(x, y)
	alpha := uint32(overlay.A)
	inv := 255 - alpha
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((uint32(overlay.R)*alpha + uint32(base.R)*inv) / 255),
		G: uint8((uint32(overlay.G)*alpha + uint32(base.G)*inv) / 255),
		B: uint8((uint32(overlay.B)*alpha + uint32(base.B)*inv) / 255),
		A: base.A,
	})
}

var watermarkGlyphs = map[rune][]string{
	'a': {
		"01110",
		"10001",
		"10001",
		"11111",
		"10001",
		"10001",
		"10001",
	},
	'g': {
		"01111",
		"10000",
		"10000",
		"10111",
		"10001",
		"10001",
		"01110",
	},
	'h': {
		"10001",
		"10001",
		"10001",
		"11111",
		"10001",
		"10001",
		"10001",
	},
	'i': {
		"11111",
		"00100",
		"00100",
		"00100",
		"00100",
		"00100",
		"11111",
	},
	'l': {
		"10000",
		"10000",
		"10000",
		"10000",
		"10000",
		"10000",
		"11111",
	},
	'n': {
		"10001",
		"11001",
		"10101",
		"10011",
		"10001",
		"10001",
		"10001",
	},
	'u': {
		"10001",
		"10001",
		"10001",
		"10001",
		"10001",
		"10001",
		"01110",
	},
	'z': {
		"11111",
		"00001",
		"00010",
		"00100",
		"01000",
		"10000",
		"11111",
	},
}

// CleanupByFilePath 删除上传记录和本地文件
func (s *uploadService) CleanupByFilePath(filePath string) error {
	if err := s.uploadRepo.DeleteByFilePath(filePath); err != nil {
		return err
	}

	trimmedPath := strings.TrimPrefix(filePath, "/uploads/")
	if trimmedPath == filePath {
		trimmedPath = strings.TrimPrefix(filePath, "uploads/")
	}
	if trimmedPath == "" {
		return nil
	}

	fullPath := filepath.Join(s.cfg.Upload.StoragePath, filepath.FromSlash(trimmedPath))
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
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
