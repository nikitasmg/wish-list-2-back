# Security Hardening Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the wishlist backend against resource abuse and oversized input by adding Fiber body limits, per-entity count limits, string length validation, and block content validation.

**Architecture:** Changes flow through three layers: HTTP (body/file size), usecase (count limits + string/block validation), repo (new count queries). All constants live in a single `limits.go` file. Mocks are updated to satisfy the new repo interfaces.

**Tech Stack:** Go 1.22, Fiber v2, GORM, PostgreSQL, testify/mock

---

## Chunk 1: Foundation — Constants, Repo Interface, Mocks

### Task 1: Create `internal/usecase/limits.go`

**Files:**
- Create: `internal/usecase/limits.go`

- [ ] **Step 1: Create the file**

```go
package usecase

const (
	MaxWishlistsPerUser    = 20
	MaxPresentsPerWishlist = 100
	MaxBlocksPerWishlist   = 100
	MaxBulkUploadFiles     = 10
	MaxFileSize            = 10 * 1024 * 1024 // 10MB

	MaxTitleLen       = 200
	MaxDescriptionLen = 2000
	MaxURLLen         = 2048
	MaxBlockDataSize  = 10 * 1024 // 10KB per block (raw JSON bytes)
	MaxBlockTextField = 5000      // chars for text/quote/checklist content
)
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/usecase/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/usecase/limits.go
git commit -m "feat: add usecase limits constants"
```

---

### Task 2: Add `CountByUserID` and `CountByWishlistID` to repo interfaces

**Files:**
- Modify: `internal/repo/contracts.go`

- [ ] **Step 1: Add `CountByUserID` to `WishlistRepo`**

In `internal/repo/contracts.go`, add to the `WishlistRepo` interface after `DecrementPresentsCount`:

```go
CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
```

- [ ] **Step 2: Add `CountByWishlistID` to `PresentRepo`**

In the same file, add to the `PresentRepo` interface after `Delete`:

```go
CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error)
```

- [ ] **Step 3: Note — compile errors expected until Tasks 3–6 complete**

Adding methods to the interfaces will immediately break the existing concrete implementations and mocks. Do NOT run `go build ./...` here — proceed directly to Task 3 to fix the mocks, then Tasks 5–6 for the implementations. You can verify the full build compiles cleanly after Task 6.

---

### Task 3: Update `MockWishlistRepo` with `CountByUserID`

**Files:**
- Modify: `mock/repo/mock_wishlist_repo.go`

- [ ] **Step 1: Add the mock method**

Append to `mock/repo/mock_wishlist_repo.go`:

```go
func (m *MockWishlistRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}
```

---

### Task 4: Update `MockPresentRepo` with `CountByWishlistID`

**Files:**
- Modify: `mock/repo/mock_present_repo.go`

- [ ] **Step 1: Add the mock method**

Append to `mock/repo/mock_present_repo.go`:

```go
func (m *MockPresentRepo) CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	args := m.Called(ctx, wishlistID)
	return args.Get(0).(int64), args.Error(1)
}
```

- [ ] **Step 2: Verify all mocks satisfy interfaces**

```bash
go build ./mock/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Run existing tests to ensure nothing broke**

```bash
go test ./...
```

Expected: all existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/repo/contracts.go mock/repo/mock_wishlist_repo.go mock/repo/mock_present_repo.go
git commit -m "feat: add CountByUserID and CountByWishlistID to repo interfaces and mocks"
```

---

### Task 5: Implement `CountByUserID` in `wishlist_postgres.go`

**Files:**
- Modify: `internal/repo/persistent/wishlist_postgres.go`

- [ ] **Step 1: Read the file to understand structure**

Open `internal/repo/persistent/wishlist_postgres.go` and find the end of the file.

- [ ] **Step 2: Add the implementation**

Append the following method to the `wishlistRepo` struct:

```go
func (r *wishlistRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&WishlistModel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
```

---

### Task 6: Implement `CountByWishlistID` in `present_postgres.go`

**Files:**
- Modify: `internal/repo/persistent/present_postgres.go`

- [ ] **Step 1: Read the file to understand structure**

Open `internal/repo/persistent/present_postgres.go` and find the end of the file.

- [ ] **Step 2: Add the implementation**

Append the following method to the `presentRepo` struct:

```go
func (r *presentRepo) CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PresentModel{}).Where("wishlist_id = ?", wishlistID).Count(&count).Error
	return count, err
}
```

- [ ] **Step 3: Verify full build**

```bash
go build ./...
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add internal/repo/persistent/wishlist_postgres.go internal/repo/persistent/present_postgres.go
git commit -m "feat: implement CountByUserID and CountByWishlistID in postgres repos"
```

---

## Chunk 2: HTTP Layer — Body Limit and File Size Checks

### Task 7: Set Fiber `BodyLimit` in `app.go`

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Update `fiber.New()` call**

In `internal/app/app.go`, replace:

```go
app := fiber.New()
```

with:

```go
app := fiber.New(fiber.Config{
	BodyLimit: 15 * 1024 * 1024, // 15MB — headroom for multipart overhead
})
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: set Fiber BodyLimit to 15MB"
```

---

### Task 8: File size check in `upload.go` (single + bulk)

**Files:**
- Modify: `internal/controller/restapi/v1/upload.go`

- [ ] **Step 1: Add import for limits package**

The constants live in the `usecase` package (`internal/usecase/limits.go`). Add the import `"main/internal/usecase"` to `upload.go` if not already present.

- [ ] **Step 2: Add size check in `upload()` handler**

In the `upload()` handler, after `data, err := io.ReadAll(f)` and its error check, add:

```go
if len(data) > usecase.MaxFileSize {
	return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Error("файл слишком большой: максимум 10MB"))
}
```

- [ ] **Step 3: Add file count check in `bulkUpload()` handler**

In `bulkUpload()`, after the `fileHeaders` nil/empty check, add:

```go
if len(fileHeaders) > usecase.MaxBulkUploadFiles {
	return c.Status(fiber.StatusBadRequest).JSON(response.Error("слишком много файлов: максимум 10"))
}
```

- [ ] **Step 4: Add per-file size check inside the bulk loop**

In the `bulkUpload()` loop, after `data, err := io.ReadAll(f)` and its error check, before `inputs = append(...)`, add:

```go
if len(data) > usecase.MaxFileSize {
	return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Error("файл слишком большой: максимум 10MB"))
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./internal/controller/...
```

Expected: exit 0.

- [ ] **Step 6: Commit**

```bash
git add internal/controller/restapi/v1/upload.go
git commit -m "feat: add file size and count limits in upload handlers"
```

---

### Task 9: File size check in `present.go` handler

**Files:**
- Modify: `internal/controller/restapi/v1/present.go`

- [ ] **Step 1: Add import**

Add `"main/internal/usecase"` import to `present.go` if not present.

- [ ] **Step 2: Add size check in `parsePresentInput()`**

In `parsePresentInput()`, after `data, err := io.ReadAll(f)` and its error check, add:

```go
if len(data) > usecase.MaxFileSize {
	return input, errors.New("файл слишком большой: максимум 10MB")
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./internal/controller/...
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add internal/controller/restapi/v1/present.go
git commit -m "feat: add file size limit in present handler"
```

---

### Task 10: File size check + nil-substitution in `wishlist.go` handler

**Files:**
- Modify: `internal/controller/restapi/v1/wishlist.go`

- [ ] **Step 1: Add import**

Add `"main/internal/usecase"` import to `wishlist.go` if not present. Ensure `"encoding/json"` is imported too (needed for nil-substitution).

- [ ] **Step 2: Add size check in `parseWishlistInput()`**

In `parseWishlistInput()`, after `data, err := io.ReadAll(f)` and its error check, add:

```go
if len(data) > usecase.MaxFileSize {
	return input, errors.New("файл слишком большой: максимум 10MB")
}
```

- [ ] **Step 3: Add nil-substitution in `updateBlocks()` handler**

In the `updateBlocks()` handler, after the `c.BodyParser(&blocks)` call and its error check, add:

```go
for i := range blocks {
	if blocks[i].Data == nil {
		blocks[i].Data = json.RawMessage("{}")
	}
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./internal/controller/...
```

Expected: exit 0.

- [ ] **Step 5: Commit**

```bash
git add internal/controller/restapi/v1/wishlist.go
git commit -m "feat: add file size limit and block Data nil-substitution in wishlist handler"
```

---

## Chunk 3: Usecase Layer — Count Limits and String Validation

### Task 11: Wishlist count limit and string validation

**Files:**
- Modify: `internal/usecase/wishlist/wishlist.go`

- [ ] **Step 1: Write the failing test for wishlist count limit**

In `internal/usecase/wishlist/wishlist_test.go`, add:

```go
func TestCreate_WishlistLimitExceeded(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(20), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{Title: "X"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "лимит вишлистов")
}

func TestCreateConstructor_WishlistLimitExceeded(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(20), nil)

	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{Title: "X"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "лимит вишлистов")
}

func TestCreate_TitleTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title: string(make([]byte, 201)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestCreate_DescriptionTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title:       "OK",
		Description: string(make([]byte, 2001)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/usecase/wishlist/... -run "TestCreate_WishlistLimitExceeded|TestCreateConstructor_WishlistLimitExceeded|TestCreate_TitleTooLong|TestCreate_DescriptionTooLong" -v
```

Expected: FAIL (CountByUserID not called yet / validation not implemented).

- [ ] **Step 3: Add `validateWishlistFields()` helper to `wishlist.go`**

Add this function near the bottom of `internal/usecase/wishlist/wishlist.go`:

```go
func validateWishlistFields(title, description, locationName, locationLink, coverURL string) error {
	if len([]rune(title)) > usecase.MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len([]rune(description)) > usecase.MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", usecase.MaxDescriptionLen)
	}
	if len([]rune(locationName)) > usecase.MaxTitleLen {
		return fmt.Errorf("location name exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len(locationLink) > usecase.MaxURLLen {
		return fmt.Errorf("location link exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	if len(coverURL) > usecase.MaxURLLen {
		return fmt.Errorf("cover URL exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	return nil
}
```

Note: string length is measured in runes (Unicode characters) for text fields, in bytes for URLs (URLs are ASCII).

- [ ] **Step 4: Add count check and field validation to `Create()`**

At the start of `Create()`, before `resolveCover`, add:

```go
count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
if err != nil {
	return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
}
if count >= usecase.MaxWishlistsPerUser {
	return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
}
if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
	return entity.Wishlist{}, err
}
```

Add `"errors"` import if not present.

- [ ] **Step 5: Add count check and field validation to `CreateConstructor()`**

At the start of `CreateConstructor()`, before `validateBlocks`, add:

```go
count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
if err != nil {
	return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
}
if count >= usecase.MaxWishlistsPerUser {
	return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
}
if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
	return entity.Wishlist{}, err
}
```

- [ ] **Step 6: Add field validation to `Update()`**

At the start of `Update()`, before `wishlistRepo.GetByID`, add:

```go
if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
	return entity.Wishlist{}, err
}
```

- [ ] **Step 7: Fix existing tests that call `Create()` or `CreateConstructor()` — they now need `CountByUserID` mock**

Five existing tests in `wishlist_test.go` call `uc.Create()` or `uc.CreateConstructor()` but don't register `CountByUserID`. Add `wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)` **before the first `wr.On(...)` line** in each:

- `TestCreate_FileUpload` — add before `wr.On("GetByShortID", ...)`
- `TestCreate_URLCover` — add before `wr.On("GetByShortID", ...)`
- `TestValidateBlocks_Valid` — add before `wr.On("GetByShortID", ...)`
- `TestGenerateUniqueShortID_Collision` — add before the first `wr.On("GetByShortID", ...)`
- `TestValidateBlocks_UnknownType` — add before the `uc.CreateConstructor(...)` call (this test has zero mock setup). The mock must return `int64(0)` so execution proceeds past the count check and reaches `validateBlocks` where the expected error occurs.

- [ ] **Step 8: Run the new tests**

```bash
go test ./internal/usecase/wishlist/... -v
```

Expected: all tests pass including the new ones.

- [ ] **Step 9: Commit**

```bash
git add internal/usecase/wishlist/wishlist.go internal/usecase/wishlist/wishlist_test.go
git commit -m "feat: add wishlist count limit and field length validation"
```

---

### Task 12: Present count limit and string validation

**Files:**
- Modify: `internal/usecase/present/present.go`
- Modify: `internal/usecase/present/present_test.go`

- [ ] **Step 1: Write the failing tests**

In `internal/usecase/present/present_test.go`, add:

```go
func TestCreate_PresentLimitExceeded(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("CountByWishlistID", mock.Anything, wid).Return(int64(100), nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{Title: "Gift"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "лимит подарков")
}

func TestCreate_TitleTooLong(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("CountByWishlistID", mock.Anything, wid).Return(int64(0), nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title: string(make([]byte, 201)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestUpdate_TitleTooLong(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	// Validation runs before GetByID, so no mock setup needed.
	id := uuid.New()
	_, err := uc.Update(context.Background(), id, usecase.CreatePresentInput{
		Title: string(make([]byte, 201)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/usecase/present/... -run "TestCreate_PresentLimitExceeded|TestCreate_TitleTooLong|TestUpdate_TitleTooLong" -v
```

Expected: FAIL.

- [ ] **Step 3: Add `validatePresentFields()` to `present.go`**

Append to `internal/usecase/present/present.go`:

```go
func validatePresentFields(title, description, link, coverURL string) error {
	if len([]rune(title)) > usecase.MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len([]rune(description)) > usecase.MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", usecase.MaxDescriptionLen)
	}
	if len(link) > usecase.MaxURLLen {
		return fmt.Errorf("link exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	if len(coverURL) > usecase.MaxURLLen {
		return fmt.Errorf("cover URL exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	return nil
}
```

- [ ] **Step 4: Add count check and validation to `Create()`**

After the existing wishlist existence check (`if _, err := uc.wishlistRepo.GetByID(...)`), add:

```go
count, err := uc.presentRepo.CountByWishlistID(ctx, wishlistID)
if err != nil {
	return entity.Present{}, fmt.Errorf("count presents: %w", err)
}
if count >= usecase.MaxPresentsPerWishlist {
	return entity.Present{}, errors.New("достигнут лимит подарков (100)")
}
if err := validatePresentFields(input.Title, input.Description, input.Link, input.CoverURL); err != nil {
	return entity.Present{}, err
}
```

- [ ] **Step 5: Add validation to `Update()`**

At the start of `Update()`, before `presentRepo.GetByID`, add:

```go
if err := validatePresentFields(input.Title, input.Description, input.Link, input.CoverURL); err != nil {
	return entity.Present{}, err
}
```

- [ ] **Step 5b: Fix existing present tests that call `Create()` successfully — they now need `CountByWishlistID` mock**

After Step 4 inserts the count check in `Create()`, these six existing tests will panic because `CountByWishlistID` is unregistered. Add `pr.On("CountByWishlistID", mock.Anything, wid).Return(int64(0), nil)` **after the `wr.On("GetByID", ...)` line** in each:

- `TestParsePrice_Empty`
- `TestParsePrice_CommaSpaces`
- `TestParsePrice_Invalid` — has no `pr.On("Create")`, add after `wr.On("GetByID", ...)`
- `TestCreate_Success`
- `TestCreate_WithSource_SavesMeta`
- `TestCreate_WithoutSource_SkipsMeta`

- [ ] **Step 6: Run all present tests**

```bash
go test ./internal/usecase/present/... -v
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/usecase/present/present.go internal/usecase/present/present_test.go
git commit -m "feat: add present count limit and field length validation"
```

---

## Chunk 4: Block Validation

### Task 13: Block count limit and content validation

**Files:**
- Modify: `internal/usecase/wishlist/wishlist.go`
- Modify: `internal/usecase/wishlist/wishlist_test.go`

- [ ] **Step 1: Write failing tests for block count and data size**

In `internal/usecase/wishlist/wishlist_test.go`, add:

```go
func TestValidateBlocks_TooManyBlocks(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	blocks := make([]entity.Block, 101)
	for i := range blocks {
		blocks[i] = entity.Block{Type: "text"}
	}
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title:  "X",
		Blocks: blocks,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many blocks")
}

func TestValidateBlocks_DataTooLarge(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	bigData := `{"content":"` + string(make([]byte, 11*1024)) + `"}`
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "text", Data: json.RawMessage(bigData)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data too large")
}

func TestValidateBlocks_TextContentTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	longContent := string(make([]rune, 5001))
	contentJSON, _ := json.Marshal(map[string]string{"content": longContent})
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "text", Data: json.RawMessage(contentJSON)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestValidateBlocks_VideoURLTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	longURL := "https://" + string(make([]byte, 2048))
	urlJSON, _ := json.Marshal(map[string]string{"url": longURL})
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "video", Data: json.RawMessage(urlJSON)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url")
}
```

The test file must import `"encoding/json"` at the top.

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/usecase/wishlist/... -run "TestValidateBlocks_TooManyBlocks|TestValidateBlocks_DataTooLarge|TestValidateBlocks_TextContentTooLong|TestValidateBlocks_VideoURLTooLong" -v
```

Expected: FAIL.

- [ ] **Step 3: Expand `validateBlocks()` in `wishlist.go`**

Replace the existing `validateBlocks()` function with:

```go
func validateBlocks(blocks []entity.Block) error {
	if len(blocks) > usecase.MaxBlocksPerWishlist {
		return fmt.Errorf("too many blocks: max %d", usecase.MaxBlocksPerWishlist)
	}
	for i, b := range blocks {
		if !entity.ValidBlockTypes[b.Type] {
			return fmt.Errorf("block[%d]: unknown type %q", i, b.Type)
		}
		if b.ColSpan > 2 {
			return fmt.Errorf("block[%d]: colSpan %d exceeds maximum of 2", i, b.ColSpan)
		}
		if b.RowSpan > 3 {
			return fmt.Errorf("block[%d]: rowSpan %d exceeds maximum of 3", i, b.RowSpan)
		}
		if len(b.Data) > usecase.MaxBlockDataSize {
			return fmt.Errorf("block[%d]: data too large (max %d bytes)", i, usecase.MaxBlockDataSize)
		}
		if err := validateBlockData(i, b.Type, b.Data); err != nil {
			return err
		}
	}
	return nil
}

// validateBlockData checks type-specific content constraints.
// b.Data is guaranteed non-nil at this point (nil-substituted to "{}" in HTTP layer).
func validateBlockData(idx int, blockType string, data json.RawMessage) error {
	switch blockType {
	case "text", "quote":
		var d struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil // malformed but not oversized — let DB store it, frontend owns schema
		}
		if len([]rune(d.Content)) > usecase.MaxBlockTextField {
			return fmt.Errorf("block[%d]: content exceeds maximum length of %d characters", idx, usecase.MaxBlockTextField)
		}

	case "checklist":
		var d struct {
			Items []struct {
				Text string `json:"text"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Items) > 100 {
			return fmt.Errorf("block[%d]: checklist exceeds maximum of 100 items", idx)
		}
		for j, item := range d.Items {
			if len([]rune(item.Text)) > 500 {
				return fmt.Errorf("block[%d]: checklist item[%d] text exceeds 500 characters", idx, j)
			}
		}

	case "image", "text_image", "video":
		var d struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.URL) > usecase.MaxURLLen {
			return fmt.Errorf("block[%d]: url exceeds maximum length of %d", idx, usecase.MaxURLLen)
		}

	case "gallery":
		var d struct {
			Images []string `json:"images"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Images) > 50 {
			return fmt.Errorf("block[%d]: gallery exceeds maximum of 50 images", idx)
		}
		for j, u := range d.Images {
			if len(u) > usecase.MaxURLLen {
				return fmt.Errorf("block[%d]: gallery image[%d] URL exceeds maximum length", idx, j)
			}
		}
	}
	return nil
}
```

Add `"encoding/json"` import to `wishlist.go`.

- [ ] **Step 4: Run all wishlist tests**

```bash
go test ./internal/usecase/wishlist/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/wishlist/wishlist.go internal/usecase/wishlist/wishlist_test.go
git commit -m "feat: add block count limit and per-type content validation"
```

---

### Task 14: Final verification

- [ ] **Step 1: Full build**

```bash
go build ./...
```

Expected: exit 0, no errors.

- [ ] **Step 2: Full test suite**

```bash
go test ./... -v 2>&1 | tail -20
```

Expected: `ok` for all packages, no failures.

- [ ] **Step 3: Verify limits constants are used throughout**

```bash
grep -r "MaxFileSize\|MaxWishlistsPerUser\|MaxPresentsPerWishlist\|MaxBlocksPerWishlist\|MaxTitleLen" --include="*.go" .
```

Expected: references in `upload.go`, `wishlist.go` (handler), `present.go` (handler), `wishlist.go` (usecase), `present.go` (usecase).

- [ ] **Step 4: Final commit if any stray changes**

```bash
git status
```

If clean, nothing to do. Otherwise commit remaining files.
