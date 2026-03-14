# Marketplace Link Parser — Design Spec

**Date:** 2026-03-14
**Status:** Approved

## Overview

When creating a gift (present), the user pastes a marketplace URL. The backend parses the page and returns pre-filled data (title, price, image, description, category, brand) for the form. The user reviews and edits before saving. Metadata is stored separately for future recommendation features.

Supported marketplaces: Ozon, Wildberries, Яндекс Маркет. Universal OG fallback for any other URL.

---

## API

**Endpoint:** `GET /api/v1/parse?url=<encoded-url>` (protected, JWT required)

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

**Partial result** — if some fields are missing (e.g. price not found), return 200 with available fields. Frontend fills missing fields manually.

**Error responses:**
- `400` — missing or invalid URL
- `422` — URL valid but parsing returned nothing useful
- `429` — rate limit exceeded (header: `Retry-After: 3600`)
- `504` — parsing timed out (8 second hard limit)

**Metadata save flow:** When the user creates/updates a present after parsing, the frontend additionally sends `category`, `brand`, `source` fields. These are saved to `present_meta`.

---

## Parsing Strategy

1. **OG parser (always first)** — fetches page HTML, extracts Open Graph tags: `og:title`, `og:description`, `og:image`, `og:price:amount`, plus any marketplace-specific meta tags for category/brand.
2. **HTML scraper (fallback)** — if `title` or `price` is empty after OG parsing, run a marketplace-specific HTML scraper to extract missing fields.
3. **Timeout:** 8 seconds via `context.WithTimeout` covering the entire parse attempt.
4. **Source detection:** URL host determines the parser (`ozon.ru` → Ozon, `wildberries.ru` → Wildberries, `market.yandex.ru` → Яндекс Маркет, anything else → OG-only).

---

## Rate Limiting

Stored in PostgreSQL (no Redis dependency).

| Scope | Limit |
|-------|-------|
| Per authenticated user | 20 requests / hour |
| Global (all users combined) | 200 requests / hour |

**Table `parse_rate_limits`:**
```sql
id           UUID PRIMARY KEY
user_id      UUID REFERENCES users(id) NULLABLE  -- NULL = global counter
window_start TIMESTAMP NOT NULL
count        INT NOT NULL DEFAULT 0
```

Logic per request:
1. Check/upsert row for `user_id` in current hour window → if `count >= 20`, return 429.
2. Check/upsert row for `user_id = NULL` in current hour window → if `count >= 200`, return 429.
3. Increment both counters.
4. Stale windows are overwritten on next request from the same user (no background cleanup needed).

---

## Data Models

### `present_meta` table
Stores parsed metadata linked to a present (1:1).

```sql
present_id   UUID PRIMARY KEY REFERENCES presents(id) ON DELETE CASCADE
source       VARCHAR NOT NULL  -- "ozon" | "wildberries" | "yamarket" | "other"
original_url TEXT NOT NULL
category     VARCHAR
brand        VARCHAR
parsed_at    TIMESTAMP NOT NULL
```

### `ParseResult` entity (DTO, not persisted)
```go
type ParseResult struct {
    Title       string
    Description string
    Price       *float64
    ImageURL    string
    Category    string
    Brand       string
    Source      string // "ozon" | "wildberries" | "yamarket" | "other"
}
```

### `PresentMeta` entity
```go
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

## Code Structure

Follows existing layered architecture (entity → usecase → repo → controller):

```
internal/
  entity/
    parse.go                          -- ParseResult, PresentMeta
  usecase/
    contracts.go                      -- add ParseUseCase, PresentMetaRepo interfaces
    parse/
      parse.go                        -- orchestrates: rate limit → detect → parse → return
  repo/
    contracts.go                      -- add ParseRateLimitRepo, PresentMetaRepo
    persistent/
      parse_rate_limit_postgres.go    -- GORM impl for rate limit table
      present_meta_postgres.go        -- GORM impl for present_meta table
  controller/restapi/v1/
    parse.go                          -- HTTP handler
    router.go                         -- add GET /api/v1/parse route

pkg/parser/
  detector.go       -- maps URL host → source string
  og.go             -- universal OG tag parser
  ozon.go           -- Ozon HTML scraper (fallback)
  wildberries.go    -- Wildberries HTML scraper (fallback)
  yamarket.go       -- Яндекс Маркет HTML scraper (fallback)
```

**Parser interface:**
```go
// pkg/parser
type MarketplaceParser interface {
    Parse(ctx context.Context, rawURL string) (ParseResult, error)
}
```

`ParseUseCase` holds a map of `source → MarketplaceParser`. OG parser always runs first; if result is incomplete, the marketplace-specific scraper fills gaps.

---

## Error Handling

- Parser errors (network, timeout, parse failure) are logged but do not surface as 500 — they return 422 with a user-friendly message.
- Partial results (some fields found, others not) always return 200.
- Rate limit errors return 429 with `Retry-After: 3600`.
- Context cancellation / deadline exceeded returns 504.

---

## Testing

| File | Type | What it covers |
|------|------|----------------|
| `pkg/parser/og_test.go` | Unit | OG parsing from static HTML fixtures |
| `pkg/parser/detector_test.go` | Unit | URL → source detection |
| `internal/usecase/parse/parse_test.go` | Unit | Rate limit logic via mock repo |
| `internal/controller/restapi/v1/parse_test.go` | HTTP | Handler response codes and shape |

HTML scrapers (`ozon.go`, `wildberries.go`, `yamarket.go`) are **not unit-tested** — they depend on live external sites and break on redesigns. Verified manually.

---

## Out of Scope

- Async/polling parsing job queue
- Redis-based rate limiting
- Recommendation engine (this spec only covers data collection)
- Admin UI for rate limit monitoring
- Caching parsed results
