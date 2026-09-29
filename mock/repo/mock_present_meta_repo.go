package mockrepo

import (
	"context"

	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockPresentMetaRepo struct {
	mock.Mock
}

func (m *MockPresentMetaRepo) Upsert(ctx context.Context, meta entity.PresentMeta) error {
	args := m.Called(ctx, meta)
	return args.Error(0)
}
