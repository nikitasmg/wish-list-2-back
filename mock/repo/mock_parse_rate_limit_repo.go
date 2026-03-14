package mockrepo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockParseRateLimitRepo struct {
	mock.Mock
}

func (m *MockParseRateLimitRepo) IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error) {
	args := m.Called(ctx, userID, windowStart)
	return args.Int(0), args.Error(1)
}
