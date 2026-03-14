# Marketplace Link Parser — Design Spec

**Date:** 2026-03-14
**Status:** Approved

## Overview

When creating a gift (present), the user pastes a marketplace URL. The backend parses the page and returns pre-filled data (title, price, image, description, category, brand) for the form. The user reviews and edits before saving. Metadata is stored separately for future recommendation features.

Supported marketplaces: Ozon, Wildberries, Яндекс Маркет. Universal OG fallback for any other URL.

---

## API

### Parse endpoint

`GET /api/v1/parse?url=<encoded-url>` (protected, JWT required)

**Success response (200):**
```json
{
  "data": {
    "title": "Кроссовки Nike Air Max",
    "description": "...",
    "price": 8990.0,
    "image_url": "https://...",
    "category": "Обувь",
    "brand": "Nike",
    "source": "ozon"
  }
}
```

**Partial result:** if some fields are missing after all parsing attempts, return 200 with available fields. Returns 422 only if `title` is empty after all parsing attempts.

**Error responses:**
- `400` — missing or invalid URL (validated in handler before calling usecase)
- `422` — URL valid but `title` could not be parsed from the page
- `429` — rate limit exceeded (`Retry-After: 3600` header)
- `504` — parsing timed out (detected via `errors.Is(err, ErrTimeout)` — see sentinel below)

**URL validation rule (handler layer):** reject if `url` query param is absent, `net/url.Parse` returns an error, host is empty, or scheme is not `http` or `https`. The usecase receives only valid, pre-checked URLs.

**Note on existing Fiber limiter:** `restapi.NewRouter` already applies a global IP-based rate limiter (`Max: 10, Expiration: 1s`). The usecase-layer limiter is additive. A client behind NAT may hit the IP limiter first and receive a different 429 response shape — this is acceptable.

### Timeout error sentinel

`internal/usecase/parse/parse.go` exports:
```go
var ErrTimeout = errors.New("parse timeout")
```

When the 8-second HTTP fetch context deadline is exceeded, the usecase wraps the error: `fmt.Errorf("%w: %w", ErrTimeout, ctx.Err())`. The handler uses `errors.Is(err, ErrTimeout)` to map to 504.

### Metadata save flow

`CreatePresentInput` gains four optional new fields populated by the frontend after a successful parse. The frontend echoes back the original URL it sent to `/parse` as `OriginalURL` — no URL transformation is expected.

```go
type CreatePresentInput struct {
    Title       string
    Description string
    Link        string
    PriceStr    string
    CoverData   []byte
    CoverName   string
    CoverURL    string
    // Parser metadata (optional, populated after /parse call)
    Category    string
    Brand       string
    Source      string // "ozon" | "wildberries" | "yamarket" | "other"
    OriginalURL string
}
```

**Handler validation for metadata fields:**
- If `Source` is non-empty, reject with 400 if it is not one of the four canonical values.
- If `Source` is non-empty and `OriginalURL` is empty, reject with 400.

`PresentUseCase.Create`: if `Source` is non-empty, call `PresentMetaRepo.Upsert` with `ParsedAt = time.Now().UTC()`. Upsert failure is logged and does not fail the request.
`PresentUseCase.Update`:
- `Source` non-empty → upsert `present_meta`.
- `Source` empty → leave existing `present_meta` row unchanged. **Never delete `present_meta` on update.**

---

## Parsing Strategy

1. **OG parser (always first)** — fetches page HTML, extracts: `og:title`, `og:description`, `og:image`, `og:price:amount`, plus marketplace-specific meta tags for category/brand.
2. **HTML scraper (merge)** — if `title` is still empty or `price` is empty and source is a known marketplace, call the marketplace-specific scraper. **Merge rule: scraper fills only empty fields; it never overwrites fields already populated by OG.** If the scraper errors but `title` is already populated, log and discard the scraper error — return 200 with available data.
3. **Timeout:** 8 seconds via `context.WithTimeout`, applied only to the outbound HTTP fetch portion. Rate-limit DB calls run before the timeout context is created.
4. **Source detection:**
   - `ozon.ru` → `"ozon"`, `wildberries.ru` → `"wildberries"`, `market.yandex.ru` → `"yamarket"`, else → `"other"`

---

## Package layout and import rules

`pkg/parser` must not import `internal/`. `ParseResult` is defined inside `pkg/parser`. The usecase maps `parser.ParseResult` → `entity.ParseResult` inline (trivial field-for-field copy, no converters file needed).

---

## `pkg/parser` exported definitions

**`types.go`:**
```go
package parser

import "context"

const (
    SourceOzon        = "ozon"
    SourceWildberries = "wildberries"
    SourceYaMarket    = "yamarket"
    SourceOther       = "other"
)

type ParseResult struct {
    Title       string
    Description string
    Price       *float64
    ImageURL    string
    Category    string
    Brand       string
    Source      string
}

type MarketplaceParser interface {
    Parse(ctx context.Context, rawURL string) (ParseResult, error)
}
```

**`detector.go`:** `func Detect(rawURL string) string`

**`og.go`:**
```go
type OGParser struct{ Client *http.Client }
func (p OGParser) Parse(ctx context.Context, rawURL string) (ParseResult, error)
```

**`ozon.go`, `wildberries.go`, `yamarket.go`:** same pattern (`OzonParser`, `WildberriesParser`, `YaMarketParser`), each with `Client *http.Client`.

**HTTP client:** `app.go` constructs one shared `*http.Client` with a realistic `User-Agent` (`"Mozilla/5.0 ..."`) and passes it to `NewParseUseCase`. The usecase uses it to instantiate all parsers. `http.DefaultClient` is not used.

---

## Rate Limiting

Stored in PostgreSQL. One row per user plus one global row keyed by `uuid.Nil`.

**Rate limit constants (hard-coded in `internal/usecase/parse/parse.go`):**
```go
const (
    perUserLimit  = 20
    globalLimit   = 200
)
```

**Table `parse_rate_limits`:**
```sql
user_id      UUID PRIMARY KEY
window_start TIMESTAMP NOT NULL
count        INT NOT NULL DEFAULT 0
```

The usecase calls `IncrementAndCheck` **twice** per request (real userID → compare against `perUserLimit`, then `uuid.Nil` → compare against `globalLimit`). Both counters increment unconditionally. Counter drift when global is saturated (~200 phantom increments/hour per user) is acceptable.

**Atomic upsert (raw SQL via GORM `Exec`):**
```sql
INSERT INTO parse_rate_limits (user_id, window_start, count)
VALUES ($1, $2, 1)
ON CONFLICT (user_id) DO UPDATE SET
  count        = CASE WHEN parse_rate_limits.window_start = $2
                      THEN parse_rate_limits.count + 1
                      ELSE 1 END,
  window_start = $2
RETURNING count
```

`$windowStart` = `time.Now().UTC().Truncate(time.Hour)`.

---

## Data Models

### `present_meta` table

```sql
present_id   UUID PRIMARY KEY REFERENCES presents(id) ON DELETE CASCADE
source       VARCHAR NOT NULL
original_url TEXT NOT NULL
category     VARCHAR
brand        VARCHAR
parsed_at    TIMESTAMP NOT NULL
```

### GORM models (added to `internal/repo/persistent/models.go`)

```go
type ParseRateLimitModel struct {
    UserID      uuid.UUID `gorm:"primaryKey"`
    WindowStart time.Time `gorm:"not null"`
    Count       int       `gorm:"not null;default:0"`
}
func (ParseRateLimitModel) TableName() string { return "parse_rate_limits" }

type PresentMetaModel struct {
    PresentID   uuid.UUID `gorm:"primaryKey"`
    Source      string    `gorm:"not null"`
    OriginalURL string    `gorm:"not null"`
    Category    string
    Brand       string
    ParsedAt    time.Time `gorm:"not null"`
}
func (PresentMetaModel) TableName() string { return "present_meta" }
```

### Entities (`internal/entity/`)

```go
// parse.go
type ParseResult struct {
    Title       string
    Description string
    Price       *float64
    ImageURL    string
    Category    string
    Brand       string
    Source      string
}

// present_meta.go
type PresentMeta struct {
    PresentID   uuid.UUID
    Source      string
    OriginalURL string
    Category    string
    Brand       string
    ParsedAt    time.Time
}
```

---

## Interfaces

### `internal/usecase/contracts.go` additions

```go
type ParseUseCase interface {
    Parse(ctx context.Context, userID uuid.UUID, rawURL string) (entity.ParseResult, error)
}
```

### `internal/repo/contracts.go` additions

```go
type ParseRateLimitRepo interface {
    IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error)
}

type PresentMetaRepo interface {
    Upsert(ctx context.Context, meta entity.PresentMeta) error
}
```

---

## Code Structure

```
internal/
  entity/parse.go, present_meta.go
  usecase/
    contracts.go
    parse/parse.go     -- ErrTimeout sentinel + orchestration
  repo/
    contracts.go
    persistent/
      models.go
      parse_rate_limit_postgres.go
      present_meta_postgres.go
  controller/restapi/v1/
    parse.go
    router.go
  controller/restapi/router.go

pkg/parser/
  types.go / detector.go / og.go / ozon.go / wildberries.go / yamarket.go
```

### Constructor signatures

**`parse.NewParseUseCase`:**
```go
func NewParseUseCase(rateLimitRepo repo.ParseRateLimitRepo, httpClient *http.Client) usecase.ParseUseCase
```
Internally builds `OGParser{Client: httpClient}` and marketplace-specific parsers.

**`present.New` (updated):**
```go
func New(presentRepo repo.PresentRepo, wishlistRepo repo.WishlistRepo, fileStorage minioPkg.FileStorage, metaRepo repo.PresentMetaRepo) usecase.PresentUseCase
```

### `PresentMetaRepo.Upsert` implementation pattern

Uses GORM `clause.OnConflict` (same approach as other upserts in the project):
```go
db.Clauses(clause.OnConflict{
    Columns:   []clause.Column{{Name: "present_id"}},
    DoUpdates: clause.AssignmentColumns([]string{"source", "original_url", "category", "brand", "parsed_at"}),
}).Create(&model)
```

### Updated `v1.NewRouter` signature

```go
func NewRouter(
    router fiber.Router,
    jwtSecret string,
    cookieDomain string,
    secureCookie bool,
    userUC usecase.UserUseCase,
    wishlistUC usecase.WishlistUseCase,
    presentUC usecase.PresentUseCase,
    uploadUC usecase.UploadUseCase,
    parseUC usecase.ParseUseCase,   // added last
) {
```

`restapi.NewRouter` signature gains the same `parseUC usecase.ParseUseCase` parameter (last) and forwards it to `v1.NewRouter`.

### `app.go` wiring additions

1. Add `&ParseRateLimitModel{}`, `&PresentMetaModel{}` to `AutoMigrate`.
2. Build shared HTTP client with custom User-Agent.
3. `rateLimitRepo := persistent.NewParseRateLimitRepo(db)`
4. `metaRepo := persistent.NewPresentMetaRepo(db)`
5. `parseUC := parse.NewParseUseCase(rateLimitRepo, httpClient)`
6. Update `present.New(presentRepo, wishlistRepo, fileStorage, metaRepo)`.
7. Pass `parseUC` to `restapi.NewRouter`.

### Handler: userID extraction

The parse handler extracts `userID` using the existing `getUserID(c)` helper in `internal/controller/restapi/v1/helpers.go`.

### Test helper updates

Add `MockParseUC` (implementing `usecase.ParseUseCase`) to `testhelpers_test.go`. Update all `v1.NewRouter` call sites in `internal/controller/restapi/v1/*_test.go` to pass `MockParseUC` as the new final argument.

---

## Error Handling

- OG succeeds (title populated), scraper errors: log, discard, return 200 with partial data.
- Both OG and scraper fail to produce `title`: return 422.
- `errors.Is(err, ErrTimeout)` → 504.
- Rate limit exceeded → 429 with `Retry-After: 3600`.
- `present_meta` write failure: logged, does not fail the request.

---

## Testing

| File | Type | Cases |
|------|------|-------|
| `pkg/parser/og_test.go` | Unit | Parses OG fields from static HTML; empty result for page with no OG tags |
| `pkg/parser/detector_test.go` | Unit | Correct source for ozon/wb/yamarket/other URLs |
| `internal/usecase/parse/parse_test.go` | Unit | Per-user limit hit → ErrRateLimit; global limit hit → ErrRateLimit; both clear → result; DB error → error |
| `internal/controller/restapi/v1/parse_test.go` | HTTP | Missing URL → 400; bad scheme → 400; success → 200; partial result → 200; title empty → 422; rate limit → 429 |

HTML scrapers not unit-tested — verified manually.

---

## Out of Scope

- Async/polling parsing job queue
- Redis-based rate limiting
- Recommendation engine (data collection only)
- Caching parsed results
- Re-uploading marketplace images to MinIO (image_url stored as-is; anti-hotlinking CDNs may break images over time — known limitation)
