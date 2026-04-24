# Clipboard Image Upload and Backend Compression Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let authors paste clipboard images directly into the article Markdown editor and store uploaded images in a compressed form on the backend.

**Architecture:** Reuse the existing upload API and editor insertion flow instead of inventing a new endpoint. The frontend editor will detect pasted images and call the same upload helper already used by the file picker, while the backend upload service will decode, optionally resize, compress, and save supported static images before writing the upload record to MySQL.

**Tech Stack:** React, TypeScript, Go, Gin, multipart upload, Go image codecs, existing upload repository/service pipeline

---

## File Structure

**Modify:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx` — add clipboard paste image detection and reuse current insertion flow
- `/Users/lizhuang/go/src/wenDao/backend/config/config.go` — add upload compression config fields
- `/Users/lizhuang/go/src/wenDao/backend/config/config.yaml` — add default upload compression config values
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/upload.go` — compress/re-encode supported images before saving and store final compressed size

**Likely unchanged:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/api/upload.ts`
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/upload.go`

---

### Task 1: Add upload compression config

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/config/config.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/config/config.yaml`

- [ ] **Step 1: Extend `UploadConfig` with compression fields**

In `/Users/lizhuang/go/src/wenDao/backend/config/config.go`, update `UploadConfig`:

```go
type UploadConfig struct {
	MaxSize                int64    `mapstructure:"max_size"`
	AllowedTypes           []string `mapstructure:"allowed_types"`
	StoragePath            string   `mapstructure:"storage_path"`
	EnableImageCompression bool     `mapstructure:"enable_image_compression"`
	ImageQuality           int      `mapstructure:"image_quality"`
	MaxImageWidth          int      `mapstructure:"max_image_width"`
	MaxImageHeight         int      `mapstructure:"max_image_height"`
}
```

- [ ] **Step 2: Add defaults to YAML**

In `/Users/lizhuang/go/src/wenDao/backend/config/config.yaml`, replace the `upload:` section with:

```yaml
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
  storage_path: "./uploads"
  enable_image_compression: true
  image_quality: 80
  max_image_width: 2560
  max_image_height: 2560
```

- [ ] **Step 3: Clamp invalid config after unmarshal**

In `/Users/lizhuang/go/src/wenDao/backend/config/config.go`, after the existing AI defaulting block, add safe defaults:

```go
	if cfg.Upload.ImageQuality <= 0 || cfg.Upload.ImageQuality > 100 {
		cfg.Upload.ImageQuality = 80
	}
	if cfg.Upload.MaxImageWidth <= 0 {
		cfg.Upload.MaxImageWidth = 2560
	}
	if cfg.Upload.MaxImageHeight <= 0 {
		cfg.Upload.MaxImageHeight = 2560
	}
```

- [ ] **Step 4: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: PASS.

---

### Task 2: Add backend image compression pipeline

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/upload.go`

- [ ] **Step 1: Add required imports for image processing**

Update the imports in `/Users/lizhuang/go/src/wenDao/backend/internal/service/upload.go` to include:

```go
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
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/repository"
)
```

- [ ] **Step 2: Add helpers for resize and compression eligibility**

In `/Users/lizhuang/go/src/wenDao/backend/internal/service/upload.go`, add these helpers before `UploadImage`:

```go
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
```

- [ ] **Step 3: Compress bytes before hashing and saving**

In `UploadImage`, replace the current “read bytes -> hash -> save raw bytes” flow with this logic:

```go
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	storedBytes := fileBytes
	if s.shouldCompress(contentType) {
		compressed, err := s.compressImage(fileBytes, contentType)
		if err != nil {
			return nil, err
		}
		storedBytes = compressed
	}

	hash := md5.Sum(storedBytes)
	hashStr := hex.EncodeToString(hash[:])
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().UnixNano()
	newFilename := fmt.Sprintf("%s_%d%s", hashStr, timestamp, ext)
```

And replace the write and DB size lines with:

```go
	if _, err := destFile.Write(storedBytes); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	upload := &model.Upload{
		UserID:   userID,
		Filename: header.Filename,
		FilePath: "/uploads/" + filepath.ToSlash(subPath),
		FileSize: int64(len(storedBytes)),
		MimeType: contentType,
		FileType: "image",
	}
```

- [ ] **Step 4: Keep GIF pass-through behavior**

Do not change GIF behavior. With the helper above, GIF is not compressed and continues to store original bytes.

- [ ] **Step 5: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: PASS.

---

### Task 3: Add clipboard image paste upload in the article editor

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`

- [ ] **Step 1: Add a paste handler for clipboard images**

In `/Users/lizhuang/go/src/wenDao/frontend/src/views/admin/articles/ArticleEditor.tsx`, add this function near `handleImageUpload`:

```tsx
  const handleContentPaste = async (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (const item of Array.from(items)) {
      if (item.kind === 'file' && item.type.startsWith('image/')) {
        const file = item.getAsFile();
        if (!file) return;

        e.preventDefault();
        showToast('检测到图片，正在上传...', 'info');
        await handleImageUpload(file, 'content');
        return;
      }
    }
  };
```

- [ ] **Step 2: Wire the paste handler to the content textarea**

In the Markdown content textarea, add:

```tsx
onPaste={handleContentPaste}
```

So the textarea becomes:

```tsx
<textarea
  ref={contentInputRef}
  className="input w-full h-[500px] font-mono py-2 text-sm leading-relaxed"
  value={formData.content}
  onChange={(e) => setFormData({ ...formData, content: e.target.value })}
  onPaste={handleContentPaste}
  placeholder="使用 Markdown 编写内容..."
/>
```

- [ ] **Step 3: Keep button-based upload unchanged**

Do not remove the existing file input upload behavior for cover and content images.

- [ ] **Step 4: Run frontend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

---

### Task 4: Verify integrated behavior

**Files:**
- Modify: none unless fixes are needed

- [ ] **Step 1: Start backend**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go run ./cmd/server
```

Expected: upload service starts without config or image codec errors.

- [ ] **Step 2: Start frontend**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run dev
```

Expected: article editor opens normally.

- [ ] **Step 3: Verify paste upload in content editor**

Manual test:
1. Copy an image to the clipboard.
2. Focus the Markdown content textarea.
3. Press paste.

Expected:
- toast appears: `检测到图片，正在上传...`
- upload succeeds
- a Markdown image line is inserted at the current cursor position

- [ ] **Step 4: Verify file picker uploads still work**

Manual test:
- upload a cover image using the existing picker
- upload a content image using the “插入图片” picker

Expected: both still succeed unchanged.

- [ ] **Step 5: Verify backend compression result**

Manual test for JPEG/PNG:
- upload a reasonably large image
- inspect saved file under `/Users/lizhuang/go/src/wenDao/backend/uploads/...`
- compare final saved file size with the original local file size

Expected:
- saved JPEG is usually smaller than original
- PNG still displays correctly
- upload record size reflects the saved compressed size

- [ ] **Step 6: Verify GIF pass-through**

Manual test:
- upload a GIF

Expected:
- upload succeeds
- file remains usable
- no decode/compression failure occurs

- [ ] **Step 7: Re-run builds if any verification fixes were required**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

---

## Self-Review

### Spec coverage
- Clipboard paste upload: Task 3
- Backend compression: Task 2
- Config support for compression: Task 1
- GIF pass-through: Task 2 and Task 4
- Preserved click-upload behavior: Task 3 and Task 4
- File size metadata reflects stored size: Task 2
- Manual verification: Task 4

No gaps found.

### Placeholder scan
- No TODO/TBD placeholders remain.
- Every file path is explicit.
- Every command is concrete.
- All introduced helpers are defined in the same task where they are first used.

### Type consistency
- Upload config field names are consistent between Go and YAML.
- `handleContentPaste` reuses existing `handleImageUpload(file, 'content')` signature.
- `FileSize` consistently switches to compressed/stored byte length.
