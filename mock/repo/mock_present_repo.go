package mockrepo

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockPresentRepo struct {
	mock.Mock
}

func (m *MockPresentRepo) Create(ctx context.Context, present entity.Present) error {
	args := m.Called(ctx, present)
	return args.Error(0)
}

func (m *MockPresentRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentRepo) GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error) {
	args := m.Called(ctx, wishlistID)
	return args.Get(0).([]entity.Present), args.Error(1)
}

func (m *MockPresentRepo) Update(ctx context.Context, present entity.Present) error {
	args := m.Called(ctx, present)
	return args.Error(0)
}

func (m *MockPresentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
