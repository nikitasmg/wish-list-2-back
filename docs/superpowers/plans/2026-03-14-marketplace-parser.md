# Marketplace Link Parser Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `GET /api/v1/parse?url=...` endpoint that parses marketplace product pages (Ozon, Wildberries, Яндекс Маркет) and returns pre-filled gift data, with per-user rate limiting and metadata storage for future recommendations.

**Architecture:** OG tags are tried first (universal), then a marketplace-specific HTML scraper fills any empty fields. Rate limits (20/user/hour + 200/global/hour) are stored atomically in PostgreSQL. Product metadata (category, brand, source) is saved to `present_meta` when a present is created/updated after parsing.

**Tech Stack:** Go 1.22, Fiber v2, GORM + PostgreSQL, `golang.org/x/net/html` for HTML parsing, `testify/mock` for unit tests.

---

## Chunk 1: Foundation — entities, interfaces, GORM models, mock repos

### Task 1: Add entities for ParseResult and PresentMeta

**Files:**
- Create: `internal/entity/parse.go`
- Create: `internal/entity/present_meta.go`

- [ ] **Step 1: Create `internal/entity/parse.go`**

```go
package entity

type ParseResult struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Price       *float64 `json:"price"`
	ImageURL    string   `json:"image_url"`
	Category    string   `json:"category"`
	Brand       string   `json:"brand"`
	Source      string   `json:"source"`
}
```

- [ ] **Step 2: Create `internal/entity/present_meta.go`**

```go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type PresentMeta struct {
	PresentID   uuid.UUID
	Source      string
	OriginalURL string
	Category    string
	Brand       string
	ParsedAt    time.Time
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/entity/...
```

Expected: no output (success).

- [ ] **Step 4: Commit**

```bash
git add internal/entity/parse.go internal/entity/present_meta.go
git commit -m "feat: add ParseResult and PresentMeta entities"
```

---

### Task 2: Add repo interfaces and GORM models

**Files:**
- Modify: `internal/repo/contracts.go`
- Modify: `internal/repo/persistent/models.go`

- [ ] **Step 1: Add `ParseRateLimitRepo` and `PresentMetaRepo` to `internal/repo/contracts.go`**

Append to the file after the existing `PresentRepo` interface:

```go
type ParseRateLimitRepo interface {
	// IncrementAndCheck atomically increments the counter for userID in the
	// current hour window and returns the new count.
	// Pass uuid.Nil for the global counter.
	IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error)
}

type PresentMetaRepo interface {
	Upsert(ctx context.Context, meta entity.PresentMeta) error
}
```

Also add `"time"` to imports in that file.

- [ ] **Step 2: Add GORM models to `internal/repo/persistent/models.go`**

Append after `PresentModel`:

```go
// ParseRateLimitModel — GORM-модель для таблицы "parse_rate_limits"
type ParseRateLimitModel struct {
	UserID      uuid.UUID `gorm:"primaryKey"`
	WindowStart time.Time `gorm:"not null"`
	Count       int       `gorm:"not null;default:0"`
}

func (ParseRateLimitModel) TableName() string { return "parse_rate_limits" }

// PresentMetaModel — GORM-модель для таблицы "present_meta"
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

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/repo/...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/repo/contracts.go internal/repo/persistent/models.go
git commit -m "feat: add ParseRateLimitRepo and PresentMetaRepo interfaces and GORM models"
```

---

### Task 3: Add mock repos for ParseRateLimitRepo and PresentMetaRepo

**Files:**
- Create: `mock/repo/mock_parse_rate_limit_repo.go`
- Create: `mock/repo/mock_present_meta_repo.go`

- [ ] **Step 1: Create `mock/repo/mock_parse_rate_limit_repo.go`**

```go
package mockrepo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockParseRateLimitRepo struct {
	mock.Mock
}

func (m *MockParseRateLimitRepo) IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error) {
	args := m.Called(ctx, userID, windowStart)
	return args.Int(0), args.Error(1)
}
```

- [ ] **Step 2: Create `mock/repo/mock_present_meta_repo.go`**

```go
package mockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockPresentMetaRepo struct {
	mock.Mock
}

func (m *MockPresentMetaRepo) Upsert(ctx context.Context, meta entity.PresentMeta) error {
	args := m.Called(ctx, meta)
	return args.Error(0)
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./mock/...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add mock/repo/mock_parse_rate_limit_repo.go mock/repo/mock_present_meta_repo.go
git commit -m "feat: add mock repos for ParseRateLimitRepo and PresentMetaRepo"
```

---

### Task 4: Add ParseUseCase to usecase contracts and update CreatePresentInput

**Files:**
- Modify: `internal/usecase/contracts.go`

- [ ] **Step 1: Add `ParseUseCase` interface and update `CreatePresentInput` in `internal/usecase/contracts.go`**

Add to `CreatePresentInput` (after `CoverURL`):

```go
	// Parser metadata (optional, populated after /parse call)
	Category    string
	Brand       string
	Source      string // "ozon" | "wildberries" | "yamarket" | "other"
	OriginalURL string
```

Add new interface after `UploadUseCase`:

```go
// ParseUseCase — парсинг ссылок с маркетплейсов
type ParseUseCase interface {
	Parse(ctx context.Context, userID uuid.UUID, rawURL string) (entity.ParseResult, error)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/usecase/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/usecase/contracts.go
git commit -m "feat: add ParseUseCase interface and extend CreatePresentInput with parser metadata"
```

---

## Chunk 2: Parser package — types, detector, OG parser, scrapers

### Task 5: Create `pkg/parser/types.go` with constants and interfaces

**Files:**
- Create: `pkg/parser/types.go`

- [ ] **Step 1: Create `pkg/parser/types.go`**

```go
package parser

import "context"

const (
	SourceOzon        = "ozon"
	SourceWildberries = "wildberries"
	SourceYaMarket    = "yamarket"
	SourceOther       = "other"
)

// ParseResult holds data extracted from a product page.
type ParseResult struct {
	Title       string
	Description string
	Price       *float64
	ImageURL    string
	Category    string
	Brand       string
	Source      string
}

// MarketplaceParser parses a product URL and returns extracted data.
type MarketplaceParser interface {
	Parse(ctx context.Context, rawURL string) (ParseResult, error)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/parser/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add pkg/parser/types.go
git commit -m "feat: add pkg/parser types, constants, and MarketplaceParser interface"
```

---

### Task 6: Implement URL detector

**Files:**
- Create: `pkg/parser/detector_test.go`
- Create: `pkg/parser/detector.go`

- [ ] **Step 1: Write failing tests in `pkg/parser/detector_test.go`**

```go
package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"main/pkg/parser"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.ozon.ru/product/krossovki-123", parser.SourceOzon},
		{"https://ozon.ru/product/something", parser.SourceOzon},
		{"https://www.wildberries.ru/catalog/12345/detail.aspx", parser.SourceWildberries},
		{"https://market.yandex.ru/product/123", parser.SourceYaMarket},
		{"https://amazon.com/product/123", parser.SourceOther},
		{"https://example.com", parser.SourceOther},
		{"not-a-url", parser.SourceOther},
		{"", parser.SourceOther},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, parser.Detect(tt.url))
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./pkg/parser/... -run TestDetect -v
```

Expected: compile error — `parser.Detect` undefined.

- [ ] **Step 3: Implement `pkg/parser/detector.go`**

```go
package parser

import "net/url"

// Detect returns the canonical source constant for a given URL.
// Returns SourceOther for unrecognised or unparseable URLs.
func Detect(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return SourceOther
	}
	host := u.Hostname()
	switch {
	case hasSuffix(host, "ozon.ru"):
		return SourceOzon
	case hasSuffix(host, "wildberries.ru"):
		return SourceWildberries
	case host == "market.yandex.ru":
		return SourceYaMarket
	default:
		return SourceOther
	}
}

// hasSuffix returns true if host equals suffix or ends with "."+suffix.
func hasSuffix(host, suffix string) bool {
	return host == suffix || len(host) > len(suffix) && host[len(host)-len(suffix)-1] == '.' && host[len(host)-len(suffix):] == suffix
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./pkg/parser/... -run TestDetect -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/parser/detector.go pkg/parser/detector_test.go
git commit -m "feat: add URL source detector for marketplace parser"
```

---

### Task 7: Implement OG parser

**Files:**
- Create: `pkg/parser/og_test.go`
- Create: `pkg/parser/og.go`

The OG parser fetches page HTML and extracts Open Graph meta tags. It uses `golang.org/x/net/html` for parsing. First, check if the dependency is available:

```bash
go list -m golang.org/x/net
```

If not present, add it:

```bash
go get golang.org/x/net
```

- [ ] **Step 1: Write failing tests in `pkg/parser/og_test.go`**

```go
package parser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/pkg/parser"
)

func ogPage(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))
}

const fullOGHTML = `<html><head>
<meta property="og:title" content="Nike Air Max">
<meta property="og:description" content="Great shoes">
<meta property="og:image" content="https://example.com/img.jpg">
<meta property="og:price:amount" content="8990">
<meta property="product:brand" content="Nike">
<meta property="product:category" content="Shoes">
</head></html>`

const emptyOGHTML = `<html><head><title>No OG</title></head></html>`

func TestOGParser_FullTags(t *testing.T) {
	srv := ogPage(fullOGHTML)
	defer srv.Close()

	p := parser.OGParser{Client: srv.Client()}
	res, err := p.Parse(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "Nike Air Max", res.Title)
	assert.Equal(t, "Great shoes", res.Description)
	assert.Equal(t, "https://example.com/img.jpg", res.ImageURL)
	require.NotNil(t, res.Price)
	assert.InDelta(t, 8990.0, *res.Price, 0.001)
	assert.Equal(t, "Nike", res.Brand)
	assert.Equal(t, "Shoes", res.Category)
}

func TestOGParser_EmptyPage(t *testing.T) {
	srv := ogPage(emptyOGHTML)
	defer srv.Close()

	p := parser.OGParser{Client: srv.Client()}
	res, err := p.Parse(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Empty(t, res.Title)
	assert.Nil(t, res.Price)
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./pkg/parser/... -run TestOGParser -v
```

Expected: compile error — `parser.OGParser` undefined.

- [ ] **Step 3: Implement `pkg/parser/og.go`**

```go
package parser

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// OGParser implements MarketplaceParser using Open Graph meta tags.
type OGParser struct {
	Client *http.Client
}

func (p OGParser) Parse(ctx context.Context, rawURL string) (ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ParseResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WishlistBot/1.0)")

	resp, err := p.Client.Do(req)
	if err != nil {
		return ParseResult{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	tags := extractMetaTags(resp.Body)
	var res ParseResult

	res.Title = tags["og:title"]
	res.Description = tags["og:description"]
	res.ImageURL = tags["og:image"]
	res.Brand = firstNonEmpty(tags["product:brand"], tags["og:brand"])
	res.Category = firstNonEmpty(tags["product:category"], tags["og:category"])

	if priceStr := firstNonEmpty(tags["og:price:amount"], tags["product:price:amount"]); priceStr != "" {
		priceStr = strings.ReplaceAll(priceStr, " ", "")
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		if v, err := strconv.ParseFloat(priceStr, 64); err == nil {
			res.Price = &v
		}
	}

	return res, nil
}

// extractMetaTags parses HTML and returns a map of meta property/name → content.
func extractMetaTags(r interface{ Read([]byte) (int, error) }) map[string]string {
	tags := make(map[string]string)
	doc, err := html.Parse(r)
	if err != nil {
		return tags
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var prop, name, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property":
					prop = a.Val
				case "name":
					name = a.Val
				case "content":
					content = a.Val
				}
			}
			if prop != "" {
				tags[prop] = content
			} else if name != "" {
				tags[name] = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return tags
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
```

- [ ] **Step 4: Add dependency if needed**

```bash
go mod tidy
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./pkg/parser/... -run TestOGParser -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/parser/og.go pkg/parser/og_test.go go.mod go.sum
git commit -m "feat: implement OG tag parser"
```

---

### Task 8: Implement marketplace HTML scrapers (Ozon, Wildberries, Яндекс Маркет)

**Files:**
- Create: `pkg/parser/ozon.go`
- Create: `pkg/parser/wildberries.go`
- Create: `pkg/parser/yamarket.go`

These scrapers are fallbacks — they only fill fields empty after OG parsing. They are not unit-tested (depend on live sites). Each follows the same structure.

- [ ] **Step 1: Create `pkg/parser/ozon.go`**

```go
package parser

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// OzonParser extracts product data from ozon.ru pages via HTML scraping.
type OzonParser struct {
	Client *http.Client
}

func (p OzonParser) Parse(ctx context.Context, rawURL string) (ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ParseResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")

	resp, err := p.Client.Do(req)
	if err != nil {
		return ParseResult{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return ParseResult{}, fmt.Errorf("parse html: %w", err)
	}

	var res ParseResult

	// Ozon renders title in <h1> with data-widget="webProductHeading"
	// and price in a span with class containing "price". These selectors
	// may break on Ozon redesigns — verify manually after updates.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				if res.Title == "" {
					res.Title = strings.TrimSpace(textContent(n))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return res, nil
}
```

- [ ] **Step 2: Create `pkg/parser/wildberries.go`**

```go
package parser

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// WildberriesParser extracts product data from wildberries.ru pages.
type WildberriesParser struct {
	Client *http.Client
}

func (p WildberriesParser) Parse(ctx context.Context, rawURL string) (ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ParseResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")

	resp, err := p.Client.Do(req)
	if err != nil {
		return ParseResult{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return ParseResult{}, fmt.Errorf("parse html: %w", err)
	}

	var res ParseResult

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			if res.Title == "" {
				res.Title = strings.TrimSpace(textContent(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return res, nil
}
```

- [ ] **Step 3: Create `pkg/parser/yamarket.go`**

```go
package parser

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// YaMarketParser extracts product data from market.yandex.ru pages.
type YaMarketParser struct {
	Client *http.Client
}

func (p YaMarketParser) Parse(ctx context.Context, rawURL string) (ParseResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ParseResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")

	resp, err := p.Client.Do(req)
	if err != nil {
		return ParseResult{}, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return ParseResult{}, fmt.Errorf("parse html: %w", err)
	}

	var res ParseResult

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			if res.Title == "" {
				res.Title = strings.TrimSpace(textContent(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return res, nil
}
```

- [ ] **Step 4: Add shared `textContent` helper — append to `og.go`**

Add this function at the bottom of `pkg/parser/og.go` (shared by all scrapers):

```go
// textContent returns the concatenated text content of an HTML node.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./pkg/parser/...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add pkg/parser/ozon.go pkg/parser/wildberries.go pkg/parser/yamarket.go pkg/parser/og.go
git commit -m "feat: add marketplace HTML scrapers (Ozon, Wildberries, YaMarket)"
```

---

## Chunk 3: Repo implementations

### Task 9: Implement `ParseRateLimitRepo`

**Files:**
- Create: `internal/repo/persistent/parse_rate_limit_postgres.go`

- [ ] **Step 1: Create `internal/repo/persistent/parse_rate_limit_postgres.go`**

```go
package persistent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"main/internal/repo"
)

type parseRateLimitRepo struct {
	db *gorm.DB
}

func NewParseRateLimitRepo(db *gorm.DB) repo.ParseRateLimitRepo {
	return &parseRateLimitRepo{db: db}
}

func (r *parseRateLimitRepo) IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error) {
	var count int
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO parse_rate_limits (user_id, window_start, count)
		VALUES (?, ?, 1)
		ON CONFLICT (user_id) DO UPDATE SET
		  count        = CASE WHEN parse_rate_limits.window_start = ?
		                      THEN parse_rate_limits.count + 1
		                      ELSE 1 END,
		  window_start = ?
		RETURNING count
	`, userID, windowStart, windowStart, windowStart).Scan(&count).Error
	return count, err
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/repo/persistent/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/repo/persistent/parse_rate_limit_postgres.go
git commit -m "feat: implement ParseRateLimitRepo with atomic PostgreSQL upsert"
```

---

### Task 10: Implement `PresentMetaRepo`

**Files:**
- Create: `internal/repo/persistent/present_meta_postgres.go`

- [ ] **Step 1: Create `internal/repo/persistent/present_meta_postgres.go`**

```go
package persistent

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"main/internal/entity"
	"main/internal/repo"
)

type presentMetaRepo struct {
	db *gorm.DB
}

func NewPresentMetaRepo(db *gorm.DB) repo.PresentMetaRepo {
	return &presentMetaRepo{db: db}
}

func (r *presentMetaRepo) Upsert(ctx context.Context, meta entity.PresentMeta) error {
	model := PresentMetaModel{
		PresentID:   meta.PresentID,
		Source:      meta.Source,
		OriginalURL: meta.OriginalURL,
		Category:    meta.Category,
		Brand:       meta.Brand,
		ParsedAt:    meta.ParsedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "present_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"source", "original_url", "category", "brand", "parsed_at"}),
	}).Create(&model).Error
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/repo/persistent/...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/repo/persistent/present_meta_postgres.go
git commit -m "feat: implement PresentMetaRepo with GORM upsert"
```

---

## Chunk 4: ParseUseCase

### Task 11: Implement ParseUseCase with rate limiting and parser orchestration

**Files:**
- Create: `internal/usecase/parse/parse_test.go`
- Create: `internal/usecase/parse/parse.go`

- [ ] **Step 1: Write failing tests in `internal/usecase/parse/parse_test.go`**

```go
package parse_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	parseUC "main/internal/usecase/parse"
	mockrepo "main/mock/repo"
)

func TestParse_PerUserLimitExceeded(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	// Per-user counter returns 21 (over limit of 20)
	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(21, nil).Once()

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, parseUC.ErrRateLimit))
	rl.AssertExpectations(t)
}

func TestParse_GlobalLimitExceeded(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	// Per-user OK (count=1), global over limit (count=201)
	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(1, nil).Once()
	rl.On("IncrementAndCheck", mock.Anything, uuid.Nil, mock.AnythingOfType("time.Time")).Return(201, nil).Once()

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, parseUC.ErrRateLimit))
}

func TestParse_RateLimitDBError(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, errors.New("db down"))

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.False(t, errors.Is(err, parseUC.ErrRateLimit))
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./internal/usecase/parse/... -v
```

Expected: compile error — `parseUC.NewParseUseCase` undefined.

- [ ] **Step 3: Implement `internal/usecase/parse/parse.go`**

```go
package parse

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/pkg/parser"
)

var (
	ErrRateLimit = errors.New("rate limit exceeded")
	ErrTimeout   = errors.New("parse timeout")
)

const (
	perUserLimit = 20
	globalLimit  = 200
	parseTimeout = 8 * time.Second
)

// ParseUseCaseImpl is exported for testing (type assertion in tests).
type ParseUseCaseImpl struct {
	rateLimitRepo repo.ParseRateLimitRepo
	ogParser      parser.MarketplaceParser
	scrapers      map[string]parser.MarketplaceParser
}

func NewParseUseCase(rateLimitRepo repo.ParseRateLimitRepo, httpClient *http.Client) usecase.ParseUseCase {
	return &ParseUseCaseImpl{
		rateLimitRepo: rateLimitRepo,
		ogParser:      parser.OGParser{Client: httpClient},
		scrapers: map[string]parser.MarketplaceParser{
			parser.SourceOzon:        parser.OzonParser{Client: httpClient},
			parser.SourceWildberries: parser.WildberriesParser{Client: httpClient},
			parser.SourceYaMarket:    parser.YaMarketParser{Client: httpClient},
		},
	}
}

func (uc *ParseUseCaseImpl) Parse(ctx context.Context, userID uuid.UUID, rawURL string) (entity.ParseResult, error) {
	// Rate limit checks run before the parse timeout context.
	window := time.Now().UTC().Truncate(time.Hour)

	userCount, err := uc.rateLimitRepo.IncrementAndCheck(ctx, userID, window)
	if err != nil {
		return entity.ParseResult{}, fmt.Errorf("rate limit check: %w", err)
	}
	if userCount > perUserLimit {
		return entity.ParseResult{}, ErrRateLimit
	}

	globalCount, err := uc.rateLimitRepo.IncrementAndCheck(ctx, uuid.Nil, window)
	if err != nil {
		return entity.ParseResult{}, fmt.Errorf("global rate limit check: %w", err)
	}
	if globalCount > globalLimit {
		return entity.ParseResult{}, ErrRateLimit
	}

	// Parse with 8-second timeout.
	fetchCtx, cancel := context.WithTimeout(ctx, parseTimeout)
	defer cancel()

	source := parser.Detect(rawURL)

	// Step 1: OG parse (always first).
	ogResult, err := uc.ogParser.Parse(fetchCtx, rawURL)
	if err != nil {
		if isTimeout(err) {
			return entity.ParseResult{}, fmt.Errorf("%w: %w", ErrTimeout, err)
		}
		log.Printf("og parse error for %s: %v", rawURL, err)
	}

	result := ogResult
	result.Source = source

	// Step 2: Scraper fallback if title or price is empty and source is known.
	needsScraper := (result.Title == "" || result.Price == nil) && source != parser.SourceOther
	if needsScraper {
		if scraper, ok := uc.scrapers[source]; ok {
			scraperResult, err := scraper.Parse(fetchCtx, rawURL)
			if err != nil {
				if isTimeout(err) {
					return entity.ParseResult{}, fmt.Errorf("%w: %w", ErrTimeout, err)
				}
				// If title is already set from OG, discard scraper error.
				if result.Title == "" {
					log.Printf("scraper error for %s: %v", rawURL, err)
				}
			} else {
				merge(&result, scraperResult)
			}
		}
	}

	return entity.ParseResult{
		Title:       result.Title,
		Description: result.Description,
		Price:       result.Price,
		ImageURL:    result.ImageURL,
		Category:    result.Category,
		Brand:       result.Brand,
		Source:      result.Source,
	}, nil
}

// merge fills empty fields in dst from src. Never overwrites non-empty fields.
func merge(dst *parser.ParseResult, src parser.ParseResult) {
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Description == "" {
		dst.Description = src.Description
	}
	if dst.Price == nil {
		dst.Price = src.Price
	}
	if dst.ImageURL == "" {
		dst.ImageURL = src.ImageURL
	}
	if dst.Category == "" {
		dst.Category = src.Category
	}
	if dst.Brand == "" {
		dst.Brand = src.Brand
	}
}

// isTimeout returns true only for deadline exceeded (8s HTTP fetch timeout).
// context.Canceled (client disconnect) is not treated as a timeout.
func isTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/usecase/parse/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/parse/parse.go internal/usecase/parse/parse_test.go
git commit -m "feat: implement ParseUseCase with rate limiting and OG+scraper orchestration"
```

---

## Chunk 5: Update PresentUseCase to save present_meta

### Task 12: Add MockPresentMetaRepo to mock/repo and update present.New

**Files:**
- Modify: `internal/usecase/present/present.go`
- Modify: `internal/usecase/present/present_test.go`

- [ ] **Step 1: Update `present.New` to accept `metaRepo`**

In `internal/usecase/present/present.go`:

1. Add `metaRepo repo.PresentMetaRepo` field to `presentUseCase` struct.
2. Update `New` signature:

```go
func New(presentRepo repo.PresentRepo, wishlistRepo repo.WishlistRepo, fileStorage minioPkg.FileStorage, metaRepo repo.PresentMetaRepo) usecase.PresentUseCase {
	return &presentUseCase{
		presentRepo:  presentRepo,
		wishlistRepo: wishlistRepo,
		fileStorage:  fileStorage,
		metaRepo:     metaRepo,
	}
}
```

3. In `Create`, after `uc.wishlistRepo.IncrementPresentsCount(...)`, add:

```go
	if input.Source != "" {
		meta := entity.PresentMeta{
			PresentID:   p.ID,
			Source:      input.Source,
			OriginalURL: input.OriginalURL,
			Category:    input.Category,
			Brand:       input.Brand,
			ParsedAt:    time.Now().UTC(),
		}
		if err := uc.metaRepo.Upsert(ctx, meta); err != nil {
			log.Printf("present_meta upsert failed for present %s: %v", p.ID, err)
		}
	}
```

Also add `"log"` and `"time"` to imports.

4. In `Update`, after `uc.presentRepo.Update(ctx, p)`, add:

```go
	if input.Source != "" {
		meta := entity.PresentMeta{
			PresentID:   p.ID,
			Source:      input.Source,
			OriginalURL: input.OriginalURL,
			Category:    input.Category,
			Brand:       input.Brand,
			ParsedAt:    time.Now().UTC(),
		}
		if err := uc.metaRepo.Upsert(ctx, meta); err != nil {
			log.Printf("present_meta upsert failed for present %s: %v", p.ID, err)
		}
	}
```

- [ ] **Step 2: Run existing tests to see which break**

```bash
go test ./internal/usecase/present/... -v
```

Expected: compile errors because `newPresentUC` in tests calls `present.New` with 3 args.

- [ ] **Step 3: Update `present_test.go` to pass `metaRepo`**

In `internal/usecase/present/present_test.go`, update `newPresentUC`:

```go
func newPresentUC(pr *mockrepo.MockPresentRepo, wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.PresentUseCase {
	mr := &mockrepo.MockPresentMetaRepo{}
	return presentUC.New(pr, wr, fs, mr)
}
```

Also add `"main/mock/repo" as mockrepo` if not already imported (it already is).

- [ ] **Step 4: Add test for meta upsert on Create**

Append to `present_test.go`:

```go
func TestCreate_WithSource_SavesMeta(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	mr := &mockrepo.MockPresentMetaRepo{}
	uc := presentUC.New(pr, wr, fs, mr)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)
	mr.On("Upsert", mock.Anything, mock.MatchedBy(func(m entity.PresentMeta) bool {
		return m.Source == "ozon" && m.OriginalURL == "https://ozon.ru/product/1"
	})).Return(nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title:       "Gift",
		Source:      "ozon",
		OriginalURL: "https://ozon.ru/product/1",
	})
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

func TestCreate_WithoutSource_SkipsMeta(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	mr := &mockrepo.MockPresentMetaRepo{}
	uc := presentUC.New(pr, wr, fs, mr)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{Title: "Gift"})
	require.NoError(t, err)
	mr.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
}
```

- [ ] **Step 5: Run all present tests to verify they pass**

```bash
go test ./internal/usecase/present/... -v
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/present/present.go internal/usecase/present/present_test.go
git commit -m "feat: save present_meta on create/update when Source is set"
```

---

## Chunk 6: HTTP layer — handler, router, test helpers

### Task 13: Implement parse HTTP handler and add route

**Files:**
- Create: `internal/controller/restapi/v1/parse_test.go`
- Create: `internal/controller/restapi/v1/parse.go`
- Modify: `internal/controller/restapi/v1/router.go`
- Modify: `internal/controller/restapi/router.go`
- Modify: `internal/controller/restapi/v1/testhelpers_test.go`
- Modify: `internal/controller/restapi/v1/present.go` (add source validation)

- [ ] **Step 1: Add `MockParseUC` to `testhelpers_test.go`**

Append at the end of `internal/controller/restapi/v1/testhelpers_test.go`:

```go
// MockParseUC

type MockParseUC struct{ mock.Mock }

func (m *MockParseUC) Parse(ctx context.Context, userID uuid.UUID, rawURL string) (entity.ParseResult, error) {
	args := m.Called(ctx, userID, rawURL)
	return args.Get(0).(entity.ParseResult), args.Error(1)
}
```

- [ ] **Step 2: Update `v1.NewRouter` signature**

In `internal/controller/restapi/v1/router.go`, add `parseUC usecase.ParseUseCase` as the last parameter:

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
	parseUC usecase.ParseUseCase,
) {
```

Also add in the function body:
```go
parseH := newParseHandler(parseUC)
```

And add to protected routes:
```go
protected.Get("/parse", parseH.parse)
```

- [ ] **Step 3: Update `restapi.NewRouter` to accept and forward `parseUC`**

In `internal/controller/restapi/router.go`, add `parseUC usecase.ParseUseCase` as last parameter and update the `v1.NewRouter` call to pass it.

- [ ] **Step 4: Write failing tests in `internal/controller/restapi/v1/parse_test.go`**

```go
package v1_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	v1 "main/internal/controller/restapi/v1"
	parseUC "main/internal/usecase/parse"
)


func setupParseAppWithUC(pu *MockParseUC) *fiber.App {
	app := fiber.New()
	v1.NewRouter(app, testSecret, "localhost", false,
		&MockUserUC{}, &MockWishlistUC{}, &MockPresentUC{}, &MockUploadUC{}, pu)
	return app
}

func doParseRequest(app *fiber.App, url string, userID uuid.UUID) *http.Response {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parse?url="+url, nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(userID))
	resp, _ := app.Test(req)
	return resp
}

func TestParse_MissingURL(t *testing.T) {
	app := setupParseAppWithUC(&MockParseUC{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parse", nil)
	req.Header.Set("Authorization", "Bearer "+makeTestToken(uuid.New()))
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestParse_BadScheme(t *testing.T) {
	app := setupParseAppWithUC(&MockParseUC{})
	resp := doParseRequest(app, "ftp://ozon.ru/product/1", uuid.New())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestParse_Success(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	price := 8990.0
	pu.On("Parse", mock.Anything, userID, "https://ozon.ru/product/1").Return(entity.ParseResult{
		Title:    "Nike Air Max",
		Price:    &price,
		Source:   "ozon",
		ImageURL: "https://example.com/img.jpg",
	}, nil)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Nike Air Max", data["title"])
}

func TestParse_TitleEmpty_Returns422(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, "https://example.com").Return(entity.ParseResult{}, nil)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://example.com", userID)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestParse_RateLimit_Returns429(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, mock.Anything).Return(entity.ParseResult{}, parseUC.ErrRateLimit)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Equal(t, "3600", resp.Header.Get("Retry-After"))
}

func TestParse_Timeout_Returns504(t *testing.T) {
	pu := &MockParseUC{}
	userID := uuid.New()
	pu.On("Parse", mock.Anything, userID, mock.Anything).Return(entity.ParseResult{}, parseUC.ErrTimeout)

	app := setupParseAppWithUC(pu)
	resp := doParseRequest(app, "https://ozon.ru/product/1", userID)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
}
```

- [ ] **Step 5: Run tests to see they fail**

```bash
go test ./internal/controller/restapi/v1/... -run TestParse -v
```

Expected: compile errors — `newParseHandler` undefined.

- [ ] **Step 6: Create `internal/controller/restapi/v1/parse.go`**

```go
package v1

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"main/internal/controller/restapi/v1/response"
	"main/internal/usecase"
	parseUC "main/internal/usecase/parse"
)

type parseHandler struct {
	uc usecase.ParseUseCase
}

func newParseHandler(uc usecase.ParseUseCase) *parseHandler {
	return &parseHandler{uc: uc}
}

func (h *parseHandler) parse(c *fiber.Ctx) error {
	rawURL := c.Query("url")
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("url is required"))
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("invalid url: must be http or https"))
	}

	userID, err := getUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(response.Error(err.Error()))
	}

	result, err := h.uc.Parse(c.Context(), userID, rawURL)
	if err != nil {
		switch {
		case errors.Is(err, parseUC.ErrRateLimit):
			c.Set("Retry-After", "3600")
			return c.Status(fiber.StatusTooManyRequests).JSON(response.Error("rate limit exceeded"))
		case errors.Is(err, parseUC.ErrTimeout):
			return c.Status(fiber.StatusGatewayTimeout).JSON(response.Error("parse timeout"))
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(response.Error(err.Error()))
		}
	}

	if result.Title == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("could not parse title from page"))
	}

	return c.JSON(response.Data(result))
}
```

- [ ] **Step 7: Run parse tests to verify they pass**

```bash
go test ./internal/controller/restapi/v1/... -run TestParse -v
```

Expected: all PASS.

- [ ] **Step 8: Update all existing test setup functions to pass `&MockParseUC{}`**

In `internal/controller/restapi/v1/present_test.go`, `user_test.go`, `wishlist_test.go` — find every call to `v1.NewRouter` and add `&MockParseUC{}` as the last argument.

Check which setup helpers exist:

```bash
grep -n "NewRouter" internal/controller/restapi/v1/*_test.go
```

Update each call site.

- [ ] **Step 9: Run all v1 tests to verify nothing is broken**

```bash
go test ./internal/controller/restapi/v1/... -v
```

Expected: all PASS.

- [ ] **Step 10: Add source validation to present handler**

In `internal/controller/restapi/v1/present.go`, add a helper and call it in `parsePresentInput`:

```go
var validSources = map[string]bool{
	"ozon": true, "wildberries": true, "yamarket": true, "other": true,
}

func validatePresentMeta(source, originalURL string) error {
	if source == "" {
		return nil
	}
	if !validSources[source] {
		return errors.New("invalid source: must be ozon, wildberries, yamarket, or other")
	}
	if originalURL == "" {
		return errors.New("original_url is required when source is set")
	}
	return nil
}
```

In `parsePresentInput`, after reading form values, add:

```go
	source := c.FormValue("source")
	originalURL := c.FormValue("original_url")
	if err := validatePresentMeta(source, originalURL); err != nil {
		return input, err
	}
	input.Source = source
	input.OriginalURL = originalURL
	input.Category = c.FormValue("category")
	input.Brand = c.FormValue("brand")
```

Also add `"errors"` to imports in `present.go` if not already there.

- [ ] **Step 11: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

- [ ] **Step 12: Commit**

```bash
git add internal/controller/restapi/v1/parse.go \
        internal/controller/restapi/v1/parse_test.go \
        internal/controller/restapi/v1/router.go \
        internal/controller/restapi/v1/testhelpers_test.go \
        internal/controller/restapi/v1/present.go \
        internal/controller/restapi/router.go
git commit -m "feat: add parse HTTP handler, route, and source validation for presents"
```

---

## Chunk 7: App wiring

### Task 14: Wire everything in app.go

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Update `internal/app/app.go`**

Add imports:
```go
parseUC "main/internal/usecase/parse"
```

Update `AutoMigrate` to include new models:
```go
if err := db.AutoMigrate(
    &persistent.UserModel{},
    &persistent.WishlistModel{},
    &persistent.PresentModel{},
    &persistent.ParseRateLimitModel{},
    &persistent.PresentMetaModel{},
); err != nil {
```

After `presentRepo := persistent.NewPresentRepo(db)`, add:
```go
rateLimitRepo := persistent.NewParseRateLimitRepo(db)
metaRepo := persistent.NewPresentMetaRepo(db)
```

Build shared HTTP client (add after `pwHasher := hasher.New()`):
```go
httpClient := &http.Client{
    Timeout: 15 * time.Second,
}
```

Add `"net/http"` and `"time"` to imports.

Update use case construction:
```go
presentUseCase := presentUC.New(presentRepo, wishlistRepo, fileStorage, metaRepo)
parseUseCase := parseUC.NewParseUseCase(rateLimitRepo, httpClient)
```

Update `restapi.NewRouter` call:
```go
restapi.NewRouter(app, cfg, userUseCase, wishlistUseCase, presentUseCase, uploadUseCase, parseUseCase)
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: wire ParseUseCase, PresentMetaRepo, and ParseRateLimitRepo in app.go"
```

---

### Task 15: Manual smoke test

- [ ] **Step 1: Start the server**

```bash
docker-compose up -d postgres minio
go run main.go
```

- [ ] **Step 2: Get a JWT token**

```bash
curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpass"}' | jq .
```

Copy the token from the response cookie or body.

- [ ] **Step 3: Call the parse endpoint with an Ozon URL**

```bash
curl -s "http://localhost:3000/api/v1/parse?url=https://www.ozon.ru/product/naushniki-123" \
  -H "Authorization: Bearer <YOUR_TOKEN>" | jq .
```

Expected: 200 with at least `title` populated, or 422 if OG tags are not available.

- [ ] **Step 4: Verify rate limiting**

Run the parse request 21 times. The 21st should return 429 with `Retry-After: 3600`.

- [ ] **Step 5: Final commit if any fixups were needed**

```bash
git add -A
git commit -m "fix: smoke test fixups"
```
