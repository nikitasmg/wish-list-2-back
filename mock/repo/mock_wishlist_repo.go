package mockrepo

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockWishlistRepo struct {
	mock.Mock
}

func (m *MockWishlistRepo) Create(ctx context.Context, wishlist entity.Wishlist) error {
	args := m.Called(ctx, wishlist)
	return args.Error(0)
}

func (m *MockWishlistRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistRepo) GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error) {
	args := m.Called(ctx, shortID)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistRepo) GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.Wishlist), args.Error(1)
}

func (m *MockWishlistRepo) Update(ctx context.Context, wishlist entity.Wishlist) error {
	args := m.Called(ctx, wishlist)
	return args.Error(0)
}

func (m *MockWishlistRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWishlistRepo) IncrementPresentsCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockWishlistRepo) DecrementPresentsCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
