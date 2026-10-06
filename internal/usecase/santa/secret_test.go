package santa

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlug_Format(t *testing.T) {
	a, err := newSlug()
	require.NoError(t, err)
	b, err := newSlug()
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^[a-zA-Z0-9]{8}$`), a)
	assert.NotEqual(t, a, b)
}

func TestNewToken_HashMatches(t *testing.T) {
	raw, hash, err := newToken()
	require.NoError(t, err)
	assert.Len(t, raw, 43) // 32 байта в base64url без паддинга
	assert.Len(t, hash, 64)
	assert.Equal(t, hash, hashToken(raw))
	assert.NotEqual(t, raw, hash)
}
