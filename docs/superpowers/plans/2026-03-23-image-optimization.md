# Image Optimization: WebP Conversion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert user-uploaded JPEG/PNG/static GIF images to WebP at 80% quality before storing in MinIO, via a decorator over the `FileStorage` interface.

**Architecture:** A new `pkg/imageconv` package handles format detection and WebP encoding. A new `optimizingStorage` decorator in `pkg/minio/optimizing.go` wraps the real MinIO client, runs conversion, then delegates to the inner storage. `internal/app/app.go` wraps the real storage with the decorator at startup — zero changes to usecases or mocks.

**Tech Stack:** `github.com/chai2010/webp` (pure Go WebP encoder), `golang.org/x/image/gif` (animated GIF frame counting), Go standard library `image`, `image/jpeg`, `image/png`, `net/http`.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `pkg/imageconv/imageconv.go` | Create | `Convert([]byte) []byte` — detect MIME, decode, encode WebP Q80, fallback to original |
| `pkg/imageconv/imageconv_test.go` | Create | Unit tests for all input types |
| `pkg/minio/optimizing.go` | Create | `optimizingStorage` decorator — wraps `FileStorage`, calls `imageconv.Convert` |
| `pkg/minio/optimizing_test.go` | Create | Unit tests using existing `MockFileStorage` |
| `pkg/minio/minio.go` | Modify | Add `detectContentType` helper; set `ContentType` in `PutObjectOptions` |
| `internal/app/app.go` | Modify | Wrap `fileStorage` with `minioPkg.NewOptimizing(fileStorage)` after creation |
| `go.mod` / `go.sum` | Modify | Add `github.com/chai2010/webp` and `golang.org/x/image` |

---

## Task 1: Add dependencies

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add webp and golang.org/x/image**

```bash
cd /Users/nvsmagin/GolandProjects/wishlist
go get github.com/chai2010/webp
go get golang.org/x/image
```

Expected: `go.mod` updated with both modules, no errors.

- [ ] **Step 2: Verify build still compiles**

```bash
go build ./...
```

Expected: success, no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add chai2010/webp and golang.org/x/image dependencies"
```

---

## Task 2: `pkg/imageconv` — image conversion (TDD)

**Files:**
- Create: `pkg/imageconv/imageconv_test.go`
- Create: `pkg/imageconv/imageconv.go`

- [ ] **Step 1: Write the failing tests**

Create `pkg/imageconv/imageconv_test.go`:

```go
package imageconv_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	xgif "golang.org/x/image/gif"

	"main/pkg/imageconv"
)

// helpers

func makeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func makeStaticGIF(t *testing.T) []byte {
	t.Helper()
	p := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}}
	frame := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	var buf bytes.Buffer
	require.NoError(t, xgif.Encode(&buf, frame, nil))
	return buf.Bytes()
}

func makeAnimatedGIF(t *testing.T) []byte {
	t.Helper()
	p := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255}}
	f1 := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	f2 := image.NewPaletted(image.Rect(0, 0, 4, 4), p)
	anim := &xgif.GIF{
		Image: []*image.Paletted{f1, f2},
		Delay: []int{10, 10},
	}
	var buf bytes.Buffer
	require.NoError(t, xgif.EncodeAll(&buf, anim))
	return buf.Bytes()
}

func isWebP(data []byte) bool {
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

// tests

func TestConvert_JPEG_to_WebP(t *testing.T) {
	result := imageconv.Convert(makeJPEG(t))
	assert.True(t, isWebP(result), "expected WebP output for JPEG input")
}

func TestConvert_PNG_to_WebP(t *testing.T) {
	result := imageconv.Convert(makePNG(t))
	assert.True(t, isWebP(result), "expected WebP output for PNG input")
}

func TestConvert_StaticGIF_to_WebP(t *testing.T) {
	result := imageconv.Convert(makeStaticGIF(t))
	assert.True(t, isWebP(result), "expected WebP output for static GIF input")
}

func TestConvert_AnimatedGIF_passthrough(t *testing.T) {
	original := makeAnimatedGIF(t)
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "animated GIF must pass through unchanged")
}

func TestConvert_RandomBytes_passthrough(t *testing.T) {
	original := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "non-image bytes must pass through unchanged")
}

func TestConvert_CorruptJPEG_passthrough(t *testing.T) {
	// valid JPEG header, garbage body
	original := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 20)...)
	result := imageconv.Convert(original)
	assert.Equal(t, original, result, "corrupt JPEG must pass through unchanged")
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./pkg/imageconv/... -v
```

Expected: compile error — package `imageconv` does not exist yet.

- [ ] **Step 3: Implement `pkg/imageconv/imageconv.go`**

Create `pkg/imageconv/imageconv.go`:

```go
package imageconv

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/chai2010/webp"
	xgif "golang.org/x/image/gif"
)

// Convert converts JPEG, PNG, and static GIF to WebP at 80% quality.
// Animated GIFs and non-image data are returned unchanged.
// Conversion errors also result in the original data being returned.
func Convert(data []byte) []byte {
	mime := http.DetectContentType(data)

	switch mime {
	case "image/gif":
		return convertGIF(data)
	case "image/jpeg", "image/png":
		return convertToWebP(data)
	default:
		return data
	}
}

func convertToWebP(data []byte) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Quality: 80}); err != nil {
		return data
	}
	return buf.Bytes()
}

func convertGIF(data []byte) []byte {
	anim, err := xgif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return data
	}
	// animated GIF — pass through to preserve all frames
	if len(anim.Image) > 1 {
		return data
	}
	// static GIF — convert single frame to WebP
	var buf bytes.Buffer
	if err := webp.Encode(&buf, anim.Image[0], &webp.Options{Quality: 80}); err != nil {
		return data
	}
	return buf.Bytes()
}
```

- [ ] **Step 4: Run tests and confirm they pass**

```bash
go test ./pkg/imageconv/... -v
```

Expected: all 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/imageconv/
git commit -m "feat: add imageconv package — WebP conversion at Q80 with animated GIF passthrough"
```

---

## Task 3: `pkg/minio/minio.go` — content-type fix

**Files:**
- Modify: `pkg/minio/minio.go`

The `Upload` method currently passes `minio.PutObjectOptions{}` with no `ContentType`. After the decorator converts bytes to WebP, the content-type must be set correctly. Go's `http.DetectContentType` does not recognize WebP, so a manual magic-byte check is needed.

- [ ] **Step 1: Add `detectContentType` helper and update `PutObjectOptions`**

In `pkg/minio/minio.go`, add the import `"net/http"` and add the helper before `Upload`:

```go
// detectContentType identifies the MIME type of data.
// Falls back to http.DetectContentType for non-WebP content.
func detectContentType(data []byte) string {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(data)
}
```

Then replace the `PutObject` call (currently line 84):

```go
// Before:
_, err := s.mc.PutObject(ctx, s.bucketName, objectID, reader, int64(len(data)), minio.PutObjectOptions{})

// After:
_, err := s.mc.PutObject(ctx, s.bucketName, objectID, reader, int64(len(data)), minio.PutObjectOptions{
    ContentType: detectContentType(data),
})
```

- [ ] **Step 2: Build to confirm no compile errors**

```bash
go build ./pkg/minio/...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add pkg/minio/minio.go
git commit -m "fix: set ContentType in MinIO PutObject with WebP magic-byte detection"
```

---

## Task 4: `pkg/minio/optimizing.go` — decorator (TDD)

**Files:**
- Create: `pkg/minio/optimizing_test.go`
- Create: `pkg/minio/optimizing.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/minio/optimizing_test.go`:

```go
package minio_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	minioPkg "main/pkg/minio"
	mockminio "main/mock/minio"
)

func makeJPEGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func isWebP(data []byte) bool {
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

func TestOptimizingStorage_Upload_ConvertsJPEGToWebP(t *testing.T) {
	inner := new(mockminio.MockFileStorage)
	inner.On("Upload", mock.AnythingOfType("string"), mock.MatchedBy(isWebP)).
		Return("http://files.example.com/img", nil)

	storage := minioPkg.NewOptimizing(inner)

	url, err := storage.Upload("photo.jpg", makeJPEGBytes(t))

	require.NoError(t, err)
	assert.Equal(t, "http://files.example.com/img", url)
	inner.AssertExpectations(t)
}

func TestOptimizingStorage_Upload_PassthroughNonImage(t *testing.T) {
	original := []byte("not an image")

	inner := new(mockminio.MockFileStorage)
	inner.On("Upload", "file.bin", original).Return("http://files.example.com/file", nil)

	storage := minioPkg.NewOptimizing(inner)

	url, err := storage.Upload("file.bin", original)

	require.NoError(t, err)
	assert.Equal(t, "http://files.example.com/file", url)
	inner.AssertExpectations(t)
}

func TestOptimizingStorage_Delete_Delegates(t *testing.T) {
	inner := new(mockminio.MockFileStorage)
	inner.On("Delete", "some-object-id").Return(nil)

	storage := minioPkg.NewOptimizing(inner)

	err := storage.Delete("some-object-id")

	require.NoError(t, err)
	inner.AssertExpectations(t)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./pkg/minio/... -run TestOptimizing -v
```

Expected: compile error — `NewOptimizing` undefined.

- [ ] **Step 3: Implement `pkg/minio/optimizing.go`**

Create `pkg/minio/optimizing.go`:

```go
package minio

import "main/pkg/imageconv"

type optimizingStorage struct {
	inner FileStorage
}

// NewOptimizing returns a FileStorage decorator that converts images to WebP
// before delegating to the inner storage.
func NewOptimizing(inner FileStorage) FileStorage {
	return &optimizingStorage{inner: inner}
}

func (s *optimizingStorage) Upload(name string, data []byte) (string, error) {
	return s.inner.Upload(name, imageconv.Convert(data))
}

func (s *optimizingStorage) Delete(objectID string) error {
	return s.inner.Delete(objectID)
}
```

- [ ] **Step 4: Run all minio tests to confirm they pass**

```bash
go test ./pkg/minio/... -v
```

Expected: all tests PASS (decorator tests + any existing tests).

- [ ] **Step 5: Commit**

```bash
git add pkg/minio/optimizing.go pkg/minio/optimizing_test.go
git commit -m "feat: add optimizingStorage decorator — converts images to WebP before MinIO upload"
```

---

## Task 5: Wire decorator in `internal/app/app.go`

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Wrap `fileStorage` with `NewOptimizing`**

In `internal/app/app.go`, find the MinIO block (currently lines 46–49):

```go
// MinIO
fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.MinioPublicURL)
if err != nil {
    log.Fatalf("minio: %v", err)
}
```

Add one line directly after:

```go
fileStorage = minioPkg.NewOptimizing(fileStorage)
```

The block should look like:

```go
// MinIO
fileStorage, err := minioPkg.New(cfg.Minio, cfg.App.MinioPublicURL)
if err != nil {
    log.Fatalf("minio: %v", err)
}
fileStorage = minioPkg.NewOptimizing(fileStorage)
```

- [ ] **Step 2: Build the full project**

```bash
go build ./...
```

Expected: success, no errors.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS. No existing tests should break (interface unchanged).

- [ ] **Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: wire image optimization decorator in app startup"
```

---

## Task 6: Smoke test (manual)

- [ ] **Step 1: Start services**

```bash
docker-compose up -d postgres minio server
```

- [ ] **Step 2: Upload a JPEG via the API and verify WebP response**

Use any JWT token from a logged-in user:

```bash
curl -s -X POST http://localhost:3000/api/v1/upload \
  -H "Cookie: token=<your-jwt>" \
  -F "file=@/path/to/test.jpg" | jq .
```

Expected response:
```json
{ "url": "https://files.prosto-namekni.ru/wishlist/<uuid>" }
```

- [ ] **Step 3: Verify the stored file is WebP**

Fetch the returned URL and check content-type:

```bash
curl -sI <returned-url> | grep -i content-type
```

Expected: `Content-Type: image/webp`

- [ ] **Step 4: Verify animated GIF passes through unchanged**

```bash
curl -s -X POST http://localhost:3000/api/v1/upload \
  -H "Cookie: token=<your-jwt>" \
  -F "file=@/path/to/animated.gif" | jq .

curl -sI <returned-url> | grep -i content-type
```

Expected: `Content-Type: image/gif`
