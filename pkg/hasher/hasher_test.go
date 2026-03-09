package hasher_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/pkg/hasher"
)

func TestHash_NotEqualToInput(t *testing.T) {
	h := hasher.New()
	hashed, err := h.Hash("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", hashed)
}

func TestCompare_CorrectPassword(t *testing.T) {
	h := hasher.New()
	hashed, err := h.Hash("mypassword")
	require.NoError(t, err)

	err = h.Compare(hashed, "mypassword")
	assert.NoError(t, err)
}

func TestCompare_WrongPassword(t *testing.T) {
	h := hasher.New()
	hashed, err := h.Hash("mypassword")
	require.NoError(t, err)

	err = h.Compare(hashed, "wrongpassword")
	assert.Error(t, err)
}
