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
