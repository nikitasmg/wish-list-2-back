package parse_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	parseUC "main/internal/usecase/parse"
	mockrepo "main/mock/repo"
)

func TestParse_PerUserLimitExceeded(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	// Per-user counter returns 21 (over limit of 20)
	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(21, nil).Once()

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, parseUC.ErrRateLimit))
	rl.AssertExpectations(t)
}

func TestParse_GlobalLimitExceeded(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	// Per-user OK (count=1), global over limit (count=201)
	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(1, nil).Once()
	rl.On("IncrementAndCheck", mock.Anything, uuid.Nil, mock.AnythingOfType("time.Time")).Return(201, nil).Once()

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.True(t, errors.Is(err, parseUC.ErrRateLimit))
}

func TestParse_RateLimitDBError(t *testing.T) {
	rl := &mockrepo.MockParseRateLimitRepo{}
	userID := uuid.New()

	rl.On("IncrementAndCheck", mock.Anything, userID, mock.AnythingOfType("time.Time")).Return(0, errors.New("db down"))

	uc := parseUC.NewParseUseCase(rl, &http.Client{})
	_, err := uc.Parse(context.Background(), userID, "https://ozon.ru/product/123")
	require.Error(t, err)
	assert.False(t, errors.Is(err, parseUC.ErrRateLimit))
}
