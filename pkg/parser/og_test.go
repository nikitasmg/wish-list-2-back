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
