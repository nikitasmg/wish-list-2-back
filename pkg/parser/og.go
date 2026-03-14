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
