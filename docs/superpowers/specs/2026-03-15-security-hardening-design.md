# Security Hardening Design

**Date:** 2026-03-15
**Project:** wishlist backend (Go/Fiber)
**Scope:** DDoS mitigation, resource limits, input validation, file size increase

---

## Goals

1. Prevent uncontrolled database growth from authenticated users
2. Protect against large-body and large-file attacks
3. Add per-type content validation for wishlist constructor blocks
4. Increase max upload size from ~4MB to 10MB

## Out of Scope

- IP-based DDoS mitigation at application level (handled by Nginx/infrastructure)
- Role-based limits (e.g. premium users get higher limits) — deferred
- Full strict schema validation per block type (deferred, requires stable frontend schemas)

---

## Architecture

Changes are split across three layers, each responsible for what it controls best:

- **HTTP layer** — physical request limits (body size, file size, file count)
- **Usecase layer** — business rule limits (counts, string lengths, block content)
- **Repo layer** — new count query methods

---

## Constants (`internal/usecase/limits.go`)

All limits in one file for easy future adjustment:

```go
const (
    MaxWishlistsPerUser    = 20
    MaxPresentsPerWishlist = 100
    MaxBlocksPerWishlist   = 100
    MaxBulkUploadFiles     = 10
    MaxFileSize            = 10 * 1024 * 1024 // 10MB

    MaxTitleLen       = 200
    MaxDescriptionLen = 2000
    MaxURLLen         = 2048
    MaxBlockDataSize  = 10 * 1024 // 10KB per block
    MaxBlockTextField = 5000      // characters for text/quote/checklist content
)
```

---

## Section 1: HTTP Layer

### Fiber BodyLimit (`internal/app/app.go`)

```go
app := fiber.New(fiber.Config{
    BodyLimit: 15 * 1024 * 1024, // 15MB — headroom for multipart overhead
})
```

### File Size Validation (upload handlers + present/wishlist handlers)

In `upload.go`, `present.go`, `wishlist.go` — after reading file bytes, before passing to usecase:

```go
if len(data) > MaxFileSize {
    return c.Status(fiber.StatusRequestEntityTooLarge).JSON(response.Error("file too large: max 10MB"))
}
```

### Bulk Upload File Count (`upload.go`)

```go
if len(fileHeaders) > MaxBulkUploadFiles {
    return c.Status(fiber.StatusBadRequest).JSON(response.Error("too many files: max 10"))
}
```

---

## Section 2: Usecase Layer — Count Limits

### Wishlists per User

In `wishlistUseCase.Create()` and `CreateConstructor()`:

```go
count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
if err != nil {
    return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
}
if count >= MaxWishlistsPerUser {
    return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
}
```

### Presents per Wishlist

In `presentUseCase.Create()`:

```go
count, err := uc.presentRepo.CountByWishlistID(ctx, wishlistID)
if err != nil {
    return entity.Present{}, fmt.Errorf("count presents: %w", err)
}
if count >= MaxPresentsPerWishlist {
    return entity.Present{}, errors.New("достигнут лимит подарков (100)")
}
```

### Blocks per Wishlist

In `validateBlocks()`:

```go
if len(blocks) > MaxBlocksPerWishlist {
    return fmt.Errorf("too many blocks: max %d", MaxBlocksPerWishlist)
}
```

---

## Section 3: Usecase Layer — String & Block Validation

### String Field Lengths

New `validateWishlistInput()` and `validatePresentInput()` functions in respective usecases:

| Field | Limit |
|-------|-------|
| Title | 200 chars |
| Description | 2000 chars |
| LocationName | 200 chars |
| LocationLink | 2048 chars |
| CoverURL | 2048 chars |
| Present.Link | 2048 chars |

### Block Data Validation (`validateBlocks()`)

Added to existing block validation loop:

1. **Size check** — `len(block.Data) > MaxBlockDataSize` → reject
2. **Type-specific content checks** — unmarshal only the known text field:
   - `text`, `quote` — field `content` ≤ 5000 chars
   - `checklist` — field `items` array, each item's text ≤ 500 chars, max 100 items
   - `image`, `text_image` — field `url` ≤ 2048 chars
   - `video` — field `url` ≤ 2048 chars
   - `gallery` — field `images` array of URLs, each ≤ 2048 chars, max 50 images
   - All other types — size check only

---

## Section 4: Repo Layer — New Count Methods

### `WishlistRepo` interface (`internal/repo/contracts.go`)

```go
CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
```

### `PresentRepo` interface (`internal/repo/contracts.go`)

```go
CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error)
```

### Implementations

`wishlist_postgres.go`:
```go
func (r *wishlistRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&WishlistModel{}).Where("user_id = ?", userID).Count(&count).Error
    return count, err
}
```

`present_postgres.go`:
```go
func (r *presentRepo) CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&PresentModel{}).Where("wishlist_id = ?", wishlistID).Count(&count).Error
    return count, err
}
```

---

## Files Changed

| File | Change |
|------|--------|
| `internal/app/app.go` | Add `BodyLimit: 15MB` to `fiber.New()` |
| `internal/usecase/limits.go` | **New** — all constants |
| `internal/usecase/wishlist/wishlist.go` | Count limit + string validation |
| `internal/usecase/present/present.go` | Count limit + string validation |
| `internal/repo/contracts.go` | 2 new interface methods |
| `internal/repo/persistent/wishlist_postgres.go` | `CountByUserID` impl |
| `internal/repo/persistent/present_postgres.go` | `CountByWishlistID` impl |
| `internal/controller/restapi/v1/upload.go` | File count + size checks |
| `internal/controller/restapi/v1/wishlist.go` | File size check |
| `internal/controller/restapi/v1/present.go` | File size check |
| `mock/repo/mock_wishlist_repo.go` | Add mock for `CountByUserID` |
| `mock/repo/mock_present_repo.go` | Add mock for `CountByWishlistID` |

---

## Error Responses

All limit violations return `400 Bad Request` (or `413 Request Entity Too Large` for oversized files) with a descriptive Russian-language message consistent with existing error messages in the codebase.

---

## Testing

- Unit tests for `validateBlocks()` covering: block count limit, data size limit, per-type field validation
- Unit tests for wishlist/present count limits (mock repo)
- No new integration tests required — existing pattern is sufficient
