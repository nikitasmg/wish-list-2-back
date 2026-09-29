package mockrepo

import (
	"context"
	"time"

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

func (m *MockWishlistRepo) UpdateMetadata(ctx context.Context, id uuid.UUID, wishlist entity.Wishlist, expectedUpdatedAt time.Time) (entity.Wishlist, bool, error) {
	args := m.Called(ctx, id, wishlist, expectedUpdatedAt)
	saved, _ := args.Get(0).(entity.Wishlist)
	return saved, args.Bool(1), args.Error(2)
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

func (m *MockWishlistRepo) ReservedCountsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uint, error) {
	args := m.Called(ctx, userID)
	if counts, ok := args.Get(0).(map[uuid.UUID]uint); ok {
		return counts, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWishlistRepo) RegisterView(ctx context.Context, wishlistID, guestID uuid.UUID) error {
	args := m.Called(ctx, wishlistID, guestID)
	return args.Error(0)
}

func (m *MockWishlistRepo) UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block, blocksVersion int, expectedUpdatedAt time.Time) (bool, error) {
	args := m.Called(ctx, id, blocks, blocksVersion, expectedUpdatedAt)
	return args.Bool(0), args.Error(1)
}
