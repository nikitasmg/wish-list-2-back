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
