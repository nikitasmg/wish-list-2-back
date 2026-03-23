# Image Optimization: WebP Conversion Before MinIO Upload

**Date:** 2026-03-23
**Status:** Approved

## Problem

User-uploaded images are stored in MinIO as-is — no format conversion, no compression, no content-type set. This wastes storage and bandwidth.

## Goal

Convert all user-uploaded images (JPEG, PNG, static GIF) to WebP at 80% quality before storing in MinIO. Non-image files pass through unchanged.

## Current Upload Paths

Three code paths call `fileStorage.Upload()`:

1. `/upload` endpoint → `internal/usecase/upload/upload.go`
2. Wishlist cover → `internal/usecase/wishlist/wishlist.go` (`resolveCover`)
3. Present cover → `internal/usecase/present/present.go` (`resolveCover`)

All three must benefit from optimization without changing their code.

## Approach: Decorator over `FileStorage`

Wrap the real `minioStorage` with an `optimizingStorage` decorator that intercepts `Upload()`, converts the image, then delegates to the inner storage.

The `FileStorage` interface signature does not change — mocks and all usecases remain untouched.

## Components

### `pkg/imageconv/imageconv.go`

Single exported function:

```go
func Convert(data []byte) []byte
```

Logic:
1. `http.DetectContentType(data)` → MIME type
2. Switch on `image/jpeg`, `image/png`, `image/gif` → `image.Decode` → `webp.Encode` at Q80
3. Any other MIME type, or decode/encode error → return original `data` unchanged

Library: `github.com/chai2010/webp` (pure Go, no CGO, Alpine-compatible).

### `pkg/minio/optimizing.go`

```go
type optimizingStorage struct{ inner FileStorage }

func NewOptimizing(inner FileStorage) FileStorage

func (s *optimizingStorage) Upload(name string, data []byte) (string, error) {
    converted := imageconv.Convert(data)
    return s.inner.Upload(name, converted)
}

func (s *optimizingStorage) Delete(id string) error {
    return s.inner.Delete(id)
}
```

### `pkg/minio/minio.go` (minor fix)

Add content-type detection in `PutObjectOptions`:

```go
minio.PutObjectOptions{
    ContentType: http.DetectContentType(data),
}
```

This ensures WebP bytes are stored with `image/webp` content-type.

### `internal/app/app.go` (wiring)

One line after creating the raw MinIO storage:

```go
storage = minioPkg.NewOptimizing(storage)
```

## Error Handling

- Decode failure (corrupt file, unsupported format) → `Convert` returns original bytes, upload proceeds normally.
- Non-image MIME type → pass through unchanged, no error.
- MinIO upload failure → propagated as before, unaffected by this change.

## Testing

**`pkg/imageconv`** (unit tests):
- Feed valid JPEG/PNG/GIF bytes → verify output decodes as WebP and is smaller
- Feed random bytes → verify original returned, no panic

**`pkg/minio/optimizing.go`** (unit tests):
- Use existing `mock/minio/mock_file_storage.go`
- Verify `Upload` calls inner with converted data
- Verify `Delete` delegates directly

**Existing tests:** No changes required — interface unchanged, mocks still valid.

## File Changes Summary

| File | Change |
|------|--------|
| `pkg/imageconv/imageconv.go` | New — conversion logic |
| `pkg/minio/optimizing.go` | New — decorator |
| `pkg/minio/minio.go` | Add `ContentType` to `PutObjectOptions` |
| `internal/app/app.go` | Wrap storage with `NewOptimizing` |
| `go.mod` / `go.sum` | Add `github.com/chai2010/webp` |
