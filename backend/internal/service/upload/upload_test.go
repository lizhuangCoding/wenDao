package upload

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"wenDao/config"
	"wenDao/internal/model"
)

type stubUploadRepository struct {
	created       []*model.Upload
	unreferenced  []*model.Upload
	deletedIDs    []int64
	listCutoff    time.Time
	listBatchSize int
}

func (r *stubUploadRepository) Create(upload *model.Upload) error {
	r.created = append(r.created, upload)
	return nil
}

func (r *stubUploadRepository) GetByID(id int64) (*model.Upload, error) {
	return nil, nil
}

func (r *stubUploadRepository) DeleteByFilePath(filePath string) error {
	return nil
}

func (r *stubUploadRepository) ListUnreferencedBefore(cutoff time.Time, limit int) ([]*model.Upload, error) {
	r.listCutoff = cutoff
	r.listBatchSize = limit
	return r.unreferenced, nil
}

func (r *stubUploadRepository) DeleteByID(id int64) error {
	r.deletedIDs = append(r.deletedIDs, id)
	return nil
}

func TestUploadServiceUploadImage_RejectsSpoofedContentType(t *testing.T) {
	storageDir := t.TempDir()
	svc := NewUploadService(&stubUploadRepository{}, &config.Config{
		Upload: config.UploadConfig{
			MaxSize:      1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			StoragePath:  storageDir,
		},
	})

	file := multipartFileFromBytes([]byte("not really an image"))
	header := newMultipartHeader("avatar.png", "image/png", int64(len([]byte("not really an image"))))

	upload, err := svc.UploadImage(file, header, 1)
	if err == nil {
		t.Fatalf("expected spoofed file to be rejected, got upload %+v", upload)
	}
	if err.Error() != "file type not allowed" {
		t.Fatalf("expected file type not allowed, got %v", err)
	}
}

func TestUploadServiceUploadImage_UsesDetectedImageExtensionForStoredPath(t *testing.T) {
	storageDir := t.TempDir()
	repo := &stubUploadRepository{}
	svc := NewUploadService(repo, &config.Config{
		Upload: config.UploadConfig{
			MaxSize:      1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			StoragePath:  storageDir,
		},
	})

	pngBytes := buildPNGImageBytes(t)
	file := multipartFileFromBytes(pngBytes)
	header := newMultipartHeader("avatar.php", "image/png", int64(len(pngBytes)))

	upload, err := svc.UploadImage(file, header, 2)
	if err != nil {
		t.Fatalf("expected upload to succeed, got %v", err)
	}

	if !strings.HasSuffix(upload.FilePath, ".png") {
		t.Fatalf("expected stored file path to end with .png, got %q", upload.FilePath)
	}
	if strings.HasSuffix(upload.FilePath, ".php") {
		t.Fatalf("expected dangerous original extension to be ignored, got %q", upload.FilePath)
	}
	if upload.MimeType != "image/png" {
		t.Fatalf("expected mime type image/png, got %q", upload.MimeType)
	}

	storedRelativePath := strings.TrimPrefix(upload.FilePath, "/uploads/")
	if _, err := os.Stat(filepath.Join(storageDir, filepath.FromSlash(storedRelativePath))); err != nil {
		t.Fatalf("expected stored file to exist: %v", err)
	}
}

func TestUploadServiceUploadArticleImage_AddsLizhuangWatermark(t *testing.T) {
	storageDir := t.TempDir()
	svc := NewUploadService(&stubUploadRepository{}, &config.Config{
		Upload: config.UploadConfig{
			MaxSize:      1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			StoragePath:  storageDir,
		},
	})

	pngBytes := buildSolidPNGImageBytes(t, 240, 120, color.RGBA{R: 20, G: 80, B: 140, A: 255})
	file := multipartFileFromBytes(pngBytes)
	header := newMultipartHeader("cover.png", "image/png", int64(len(pngBytes)))

	upload, err := svc.UploadArticleImage(file, header, 3)
	if err != nil {
		t.Fatalf("expected article image upload to succeed, got %v", err)
	}

	stored := readStoredImage(t, storageDir, upload.FilePath)
	if unchangedPixelCount(stored, color.RGBA{R: 20, G: 80, B: 140, A: 255}) == stored.Bounds().Dx()*stored.Bounds().Dy() {
		t.Fatal("expected stored article image to include a visible watermark")
	}
}

func TestUploadServiceUploadCoverImage_AddsWatermarkNearCenter(t *testing.T) {
	storageDir := t.TempDir()
	svc := NewUploadService(&stubUploadRepository{}, &config.Config{
		Upload: config.UploadConfig{
			MaxSize:      1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			StoragePath:  storageDir,
		},
	})

	bg := color.RGBA{R: 20, G: 80, B: 140, A: 255}
	pngBytes := buildSolidPNGImageBytes(t, 360, 180, bg)
	file := multipartFileFromBytes(pngBytes)
	header := newMultipartHeader("cover.png", "image/png", int64(len(pngBytes)))

	upload, err := svc.UploadCoverImage(file, header, 3)
	if err != nil {
		t.Fatalf("expected cover image upload to succeed, got %v", err)
	}

	stored := readStoredImage(t, storageDir, upload.FilePath)
	centerChanges := changedPixelCountInRect(stored, bg, image.Rect(120, 60, 240, 120))
	if centerChanges == 0 {
		t.Fatal("expected cover image watermark to be visible near the center crop-safe area")
	}
}

func TestUploadServiceUploadImage_DoesNotWatermarkAvatarUploads(t *testing.T) {
	storageDir := t.TempDir()
	svc := NewUploadService(&stubUploadRepository{}, &config.Config{
		Upload: config.UploadConfig{
			MaxSize:      1024 * 1024,
			AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
			StoragePath:  storageDir,
		},
	})

	bg := color.RGBA{R: 20, G: 80, B: 140, A: 255}
	pngBytes := buildSolidPNGImageBytes(t, 240, 120, bg)
	file := multipartFileFromBytes(pngBytes)
	header := newMultipartHeader("avatar.png", "image/png", int64(len(pngBytes)))

	upload, err := svc.UploadImage(file, header, 4)
	if err != nil {
		t.Fatalf("expected avatar image upload to succeed, got %v", err)
	}

	stored := readStoredImage(t, storageDir, upload.FilePath)
	if unchangedPixelCount(stored, bg) != stored.Bounds().Dx()*stored.Bounds().Dy() {
		t.Fatal("expected regular image upload to preserve pixels without watermark")
	}
}

func TestUploadServiceCleanupUnreferenced_DeletesOnlySafeUnreferencedUploads(t *testing.T) {
	tempDir := t.TempDir()
	storageDir := filepath.Join(tempDir, "uploads")
	if err := os.MkdirAll(filepath.Join(storageDir, "2026", "05"), 0o755); err != nil {
		t.Fatalf("failed to create storage dir: %v", err)
	}

	managedPath := filepath.Join(storageDir, "2026", "05", "unused.png")
	if err := os.WriteFile(managedPath, []byte("unused"), 0o644); err != nil {
		t.Fatalf("failed to write managed upload: %v", err)
	}
	outsidePath := filepath.Join(tempDir, "outside.png")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}

	repo := &stubUploadRepository{
		unreferenced: []*model.Upload{
			{ID: 11, FilePath: "/uploads/2026/05/unused.png"},
			{ID: 12, FilePath: "/uploads/../outside.png"},
		},
	}
	svc := NewUploadService(repo, &config.Config{
		Upload: config.UploadConfig{
			StoragePath:          storageDir,
			CleanupRetentionDays: 2,
			CleanupBatchSize:     100,
		},
	})

	result, err := svc.CleanupUnreferenced(time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected cleanup to succeed, got %v", err)
	}

	if _, err := os.Stat(managedPath); !os.IsNotExist(err) {
		t.Fatalf("expected safe unreferenced upload to be removed, stat error: %v", err)
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("expected unsafe path outside storage to be left untouched, got %v", err)
	}
	if !reflect.DeepEqual(repo.deletedIDs, []int64{11}) {
		t.Fatalf("expected only safe upload record to be deleted, got IDs %#v", repo.deletedIDs)
	}
	if result.Candidates != 2 || result.Deleted != 1 || result.Skipped != 1 {
		t.Fatalf("unexpected cleanup result: %+v", result)
	}
}

func TestUploadServiceCleanupUnreferenced_UsesTwoDayDefaultRetention(t *testing.T) {
	repo := &stubUploadRepository{}
	svc := NewUploadService(repo, &config.Config{
		Upload: config.UploadConfig{
			StoragePath: t.TempDir(),
		},
	})
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)

	_, err := svc.CleanupUnreferenced(now)
	if err != nil {
		t.Fatalf("expected cleanup to succeed, got %v", err)
	}

	expectedCutoff := now.Add(-48 * time.Hour)
	if !repo.listCutoff.Equal(expectedCutoff) {
		t.Fatalf("expected cutoff %s, got %s", expectedCutoff, repo.listCutoff)
	}
	if repo.listBatchSize != 200 {
		t.Fatalf("expected default batch size 200, got %d", repo.listBatchSize)
	}
}

func buildPNGImageBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to build png bytes: %v", err)
	}
	return buf.Bytes()
}

func buildSolidPNGImageBytes(t *testing.T, width, height int, bg color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, bg)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to build png bytes: %v", err)
	}
	return buf.Bytes()
}

func readStoredImage(t *testing.T, storageDir, filePath string) image.Image {
	t.Helper()
	storedRelativePath := strings.TrimPrefix(filePath, "/uploads/")
	data, err := os.ReadFile(filepath.Join(storageDir, filepath.FromSlash(storedRelativePath)))
	if err != nil {
		t.Fatalf("failed to read stored image: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode stored image: %v", err)
	}
	return img
}

func unchangedPixelCount(img image.Image, expected color.RGBA) int {
	count := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if uint8(r>>8) == expected.R && uint8(g>>8) == expected.G && uint8(b>>8) == expected.B && uint8(a>>8) == expected.A {
				count++
			}
		}
	}
	return count
}

func changedPixelCountInRect(img image.Image, expected color.RGBA, rect image.Rectangle) int {
	count := 0
	bounds := rect.Intersect(img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if uint8(r>>8) != expected.R || uint8(g>>8) != expected.G || uint8(b>>8) != expected.B || uint8(a>>8) != expected.A {
				count++
			}
		}
	}
	return count
}

func multipartFileFromBytes(data []byte) multipart.File {
	return &memoryMultipartFile{Reader: bytes.NewReader(data)}
}

func newMultipartHeader(filename, contentType string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Size:     size,
		Header: textproto.MIMEHeader{
			"Content-Type": []string{contentType},
		},
	}
}

type memoryMultipartFile struct {
	*bytes.Reader
}

func (f *memoryMultipartFile) Close() error {
	return nil
}
