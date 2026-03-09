package v1_test

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
	"main/internal/usecase"
)

const testSecret = "test-jwt-secret"

func makeTestToken(userID uuid.UUID) string {
	claims := jwt.MapClaims{
		"id":  userID.String(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testSecret))
	return signed
}

// MockUserUC

type MockUserUC struct{ mock.Mock }

func (m *MockUserUC) Register(ctx context.Context, username, password string) (usecase.AuthResult, error) {
	args := m.Called(ctx, username, password)
	return args.Get(0).(usecase.AuthResult), args.Error(1)
}

func (m *MockUserUC) Login(ctx context.Context, username, password string) (usecase.AuthResult, error) {
	args := m.Called(ctx, username, password)
	return args.Get(0).(usecase.AuthResult), args.Error(1)
}

func (m *MockUserUC) AuthenticateTelegram(ctx context.Context, input usecase.TelegramAuthInput) (usecase.AuthResult, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(usecase.AuthResult), args.Error(1)
}

func (m *MockUserUC) GetMe(ctx context.Context, userID uuid.UUID) (entity.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(entity.User), args.Error(1)
}

// MockWishlistUC

type MockWishlistUC struct{ mock.Mock }

func (m *MockWishlistUC) Create(ctx context.Context, userID uuid.UUID, input usecase.CreateWishlistInput) (entity.Wishlist, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) CreateConstructor(ctx context.Context, userID uuid.UUID, input usecase.CreateConstructorInput) (entity.Wishlist, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error) {
	args := m.Called(ctx, shortID)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) Update(ctx context.Context, id uuid.UUID, input usecase.CreateWishlistInput) (entity.Wishlist, error) {
	args := m.Called(ctx, id, input)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block) (entity.Wishlist, error) {
	args := m.Called(ctx, id, blocks)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockPresentUC

type MockPresentUC struct{ mock.Mock }

func (m *MockPresentUC) Create(ctx context.Context, wishlistID uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	args := m.Called(ctx, wishlistID, input)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) GetAllByWishlist(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error) {
	args := m.Called(ctx, wishlistID)
	return args.Get(0).([]entity.Present), args.Error(1)
}

func (m *MockPresentUC) Update(ctx context.Context, id uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	args := m.Called(ctx, id, input)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) Delete(ctx context.Context, wishlistID, id uuid.UUID) error {
	args := m.Called(ctx, wishlistID, id)
	return args.Error(0)
}

func (m *MockPresentUC) Reserve(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPresentUC) Release(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockUploadUC

type MockUploadUC struct{ mock.Mock }

func (m *MockUploadUC) Upload(ctx context.Context, name string, data []byte) (usecase.UploadResult, error) {
	args := m.Called(ctx, name, data)
	return args.Get(0).(usecase.UploadResult), args.Error(1)
}

func (m *MockUploadUC) BulkUpload(ctx context.Context, files []usecase.FileInput) ([]usecase.BulkUploadResult, error) {
	args := m.Called(ctx, files)
	return args.Get(0).([]usecase.BulkUploadResult), args.Error(1)
}
