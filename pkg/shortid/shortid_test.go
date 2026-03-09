package shortid_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/pkg/shortid"
)

func TestGenerate_Format(t *testing.T) {
	id, err := shortid.Generate()
	require.NoError(t, err)
	assert.Len(t, id, 11)
	assert.Equal(t, '-', rune(id[3]))
	assert.Equal(t, '-', rune(id[7]))
}

func TestGenerate_Alphabet(t *testing.T) {
	id, err := shortid.Generate()
	require.NoError(t, err)
	re := regexp.MustCompile(`^[a-z0-9]{3}-[a-z0-9]{3}-[a-z0-9]{3}$`)
	assert.True(t, re.MatchString(id), "id %q does not match pattern", id)
}

func TestGenerate_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id, err := shortid.Generate()
		require.NoError(t, err)
		assert.False(t, seen[id], "duplicate id: %s", id)
		seen[id] = true
	}
}
