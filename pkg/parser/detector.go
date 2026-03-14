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
