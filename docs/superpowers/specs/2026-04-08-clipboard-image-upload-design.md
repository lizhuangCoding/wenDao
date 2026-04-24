# Clipboard Image Upload and Backend Compression Design

## Overview
Improve the article editor image workflow in two ways:
1. support uploading clipboard images directly when the user pastes into the Markdown content editor,
2. compress uploaded static images on the backend before storing them to reduce storage usage.

The goal is to keep the authoring flow smooth while making stored image assets more space-efficient.

## Goal
Authors should be able to paste images directly into the article content editor without clicking the upload button, and uploaded images should be stored in a compressed form whenever safe to do so.

## Current Problems
### 1. Content image upload only supports file picker
The current editor only uploads content images through the explicit “插入图片” file input in `frontend/src/views/admin/articles/ArticleEditor.tsx`.

### 2. Uploaded images are stored as-is
The current backend upload service in `backend/internal/service/upload.go` reads the raw file bytes and writes them directly to disk. This means image storage grows quickly, especially for screenshots and large pasted images.

### 3. Upload configuration has no compression controls
`backend/config/config.go` currently exposes max size, allowed types, and storage path, but has no image optimization settings.

## Design Summary
We will keep the current upload API contract and extend behavior around it:
- the frontend editor will detect pasted images in the clipboard and send them through the same upload API already used by the file input,
- the backend upload service will compress static image formats before saving them,
- the existing button-based upload flow remains unchanged.

This keeps the implementation simple and minimizes surface-area changes.

## Files Affected
### Frontend
- Modify: `frontend/src/views/admin/articles/ArticleEditor.tsx`

### Backend
- Modify: `backend/internal/service/upload.go`
- Modify: `backend/config/config.go`
- Modify: `backend/config/config.yaml`

### Existing API/handler likely unchanged or minimally changed
- `frontend/src/api/upload.ts`
- `backend/internal/handler/upload.go`

## Frontend Design
### Clipboard paste upload
Add image-paste handling to the Markdown content textarea in `frontend/src/views/admin/articles/ArticleEditor.tsx`.

Behavior:
1. Listen for `onPaste` on the content textarea.
2. Inspect `event.clipboardData.items`.
3. If an item is an image file:
   - prevent the default paste behavior,
   - extract the image as a `File`,
   - call the existing `handleImageUpload(file, 'content')`.
4. Reuse the existing logic that inserts Markdown image syntax at the cursor position after upload succeeds.

This means pasted images and file-picked images share the same insertion path.

### UX details
- Show a toast such as “检测到图片，正在上传...” when paste upload starts.
- On success, insert the Markdown image link exactly as current file uploads do.
- On failure, show the same error toast path used by existing image uploads.
- Only the content editor should respond to pasted images. Title, summary, and other fields should not intercept clipboard image paste.

## Backend Compression Design
### Compression ownership
Compression should happen entirely on the backend.

Reasons:
- keeps frontend simple,
- centralizes image policy,
- avoids duplicating compression logic across browsers,
- makes future tuning easier.

### Compression scope
Apply compression only to static image formats:
- JPEG
- PNG
- WebP (only if the current Go image stack supports this cleanly)

Do not compress GIF in the first version.

Reason:
- animated GIF handling is more fragile,
- first version should not risk breaking animations.

### Upload pipeline changes
In `backend/internal/service/upload.go`, change the current flow from:
- read bytes,
- save original bytes directly,

to:
- validate file size and MIME type,
- read file bytes,
- if MIME type is compressible:
  - decode image,
  - optionally resize if configured limits are exceeded,
  - re-encode with compression settings,
  - save compressed bytes,
- if MIME type is GIF:
  - keep current original bytes,
- store compressed-size metadata in DB.

## Compression Policy
### JPEG
Re-encode as JPEG using configurable quality.

Recommended default:
- quality = 80

This provides a practical balance between image quality and file size reduction.

### PNG
Re-encode as PNG.

Reason:
- preserves alpha transparency,
- avoids converting screenshots or transparent UI assets into JPEG.

This may not shrink as aggressively as JPEG compression, but it is safer for correctness.

### WebP
If the current backend dependency stack already supports decoding and encoding WebP cleanly, keep the format and re-encode.
If not, the first version may treat WebP as pass-through.

We should not introduce a large extra dependency purely for format conversion unless necessary.

### GIF
Pass through unchanged in version 1.

## Optional Resizing
For pasted screenshots, compression alone may not be enough if image dimensions are extremely large. The design supports adding dimension-based resizing.

Recommended first-version policy:
- add config fields for `MaxImageWidth` and `MaxImageHeight`,
- if either dimension is exceeded, resize proportionally before encoding.

This is optional but valuable for screenshots copied from 4K displays.

## Configuration Changes
Extend `UploadConfig` in `backend/config/config.go` with image optimization fields.

Recommended additions:
- `EnableImageCompression bool`
- `ImageQuality int`
- `MaxImageWidth int`
- `MaxImageHeight int`

Example YAML values in `backend/config/config.yaml`:

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

If resizing is not implemented in version 1, the width/height config can still be added now for future compatibility.

## Database Behavior
The upload record should continue to store:
- original filename,
- final URL/path,
- MIME type,
- file type,
- file size.

Important change:
- `FileSize` should reflect the final stored file size after compression, not the original multipart header size.

This makes admin and storage data accurate.

## Error Handling
### Frontend
- If pasted clipboard data has no image item, do nothing special.
- If image upload fails, show the existing upload failure toast and leave editor content unchanged.

### Backend
Return upload failure if:
- image decode fails,
- compression or re-encoding fails,
- filesystem write fails,
- DB record creation fails.

If DB insert fails after file write, keep the existing cleanup behavior and remove the written file.

### Format handling
If a format is allowed but not compressible in the current implementation:
- fall back to original-byte storage,
- do not reject the upload solely because compression is unavailable.

## Why This Design
This design is intentionally narrow and practical:
- no new upload endpoint,
- no separate clipboard API,
- no frontend compression,
- no speculative object storage redesign.

It solves the user-facing friction immediately while keeping the implementation localized to the editor and upload service.

## Testing Plan
### Frontend verification
1. Paste an image into the content editor.
2. Confirm upload begins automatically.
3. Confirm Markdown image syntax is inserted at the cursor position.
4. Confirm the old file-picker upload still works.

### Backend verification
1. Upload a JPEG and verify stored file size is smaller than the original in common cases.
2. Upload a PNG with transparency and confirm visual correctness.
3. Upload a GIF and confirm it still uploads successfully.
4. Verify upload record `size` matches the final stored file size.
5. Verify invalid file types are still rejected.
6. Verify oversized files are still rejected before processing.

## Acceptance Criteria
The design is successful when:
- users can paste images directly into the article content editor,
- pasted images are uploaded through the existing upload flow,
- Markdown image links are inserted automatically after upload,
- uploaded static images are stored in a compressed form when supported,
- GIF uploads still work safely,
- the existing click-to-upload behavior remains functional,
- stored file size metadata reflects the compressed output size.
