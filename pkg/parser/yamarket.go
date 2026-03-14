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
