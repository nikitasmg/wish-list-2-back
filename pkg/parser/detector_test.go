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
