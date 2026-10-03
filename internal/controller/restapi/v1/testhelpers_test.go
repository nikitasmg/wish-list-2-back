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

func (m *MockUserUC) GetProfile(ctx context.Context, userID uuid.UUID) (entity.User, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(entity.User), args.Error(1)
}

func (m *MockUserUC) UpdateProfile(ctx context.Context, userID uuid.UUID, input usecase.UpdateProfileInput) (entity.User, error) {
	args := m.Called(ctx, userID, input)
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

func (m *MockWishlistUC) GetByShortID(ctx context.Context, shortID string, guestID uuid.UUID) (entity.Wishlist, error) {
	args := m.Called(ctx, shortID, guestID)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) Update(ctx context.Context, userID, id uuid.UUID, input usecase.CreateWishlistInput, expectedUpdatedAt time.Time) (entity.Wishlist, error) {
	args := m.Called(ctx, userID, id, input, expectedUpdatedAt)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) UpdateBlocks(ctx context.Context, userID, id uuid.UUID, blocks []entity.Block, rows []entity.RowSettings, expectedUpdatedAt time.Time) (entity.Wishlist, error) {
	args := m.Called(ctx, userID, id, blocks, rows, expectedUpdatedAt)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockWishlistUC) Delete(ctx context.Context, userID, id uuid.UUID) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}

// MockPresentUC

type MockPresentUC struct{ mock.Mock }

func (m *MockPresentUC) Create(ctx context.Context, userID, wishlistID uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	args := m.Called(ctx, userID, wishlistID, input)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) GetByID(ctx context.Context, userID, id uuid.UUID) (entity.Present, error) {
	args := m.Called(ctx, userID, id)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) GetAllByWishlist(ctx context.Context, wishlistID, viewerID uuid.UUID) ([]entity.Present, error) {
	args := m.Called(ctx, wishlistID, viewerID)
	return args.Get(0).([]entity.Present), args.Error(1)
}

func (m *MockPresentUC) Update(ctx context.Context, userID, id uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	args := m.Called(ctx, userID, id, input)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) Delete(ctx context.Context, userID, wishlistID, id uuid.UUID) error {
	args := m.Called(ctx, userID, wishlistID, id)
	return args.Error(0)
}

func (m *MockPresentUC) Reserve(ctx context.Context, id, guestID uuid.UUID, name string) error {
	args := m.Called(ctx, id, guestID, name)
	return args.Error(0)
}

func (m *MockPresentUC) Release(ctx context.Context, id, guestID uuid.UUID) error {
	args := m.Called(ctx, id, guestID)
	return args.Error(0)
}

func (m *MockPresentUC) Reorder(ctx context.Context, userID, wishlistID uuid.UUID, ids []uuid.UUID) error {
	return m.Called(ctx, userID, wishlistID, ids).Error(0)
}

func (m *MockPresentUC) SetGifted(ctx context.Context, userID, id uuid.UUID, gifted bool) (entity.Present, error) {
	args := m.Called(ctx, userID, id, gifted)
	return args.Get(0).(entity.Present), args.Error(1)
}

func (m *MockPresentUC) Join(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPresentUC) Leave(ctx context.Context, id uuid.UUID) error {
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

func (m *MockWishlistUC) CreateFromSystemTemplate(ctx context.Context, userID uuid.UUID, input usecase.CreateFromSystemTemplateInput) (entity.Wishlist, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

// MockGuestDataUC

type MockGuestDataUC struct{ mock.Mock }

func (m *MockGuestDataUC) SubmitRSVP(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input usecase.RSVPInput) (entity.RSVPResponse, error) {
	args := m.Called(ctx, wishlistID, blockID, guestID, input)
	return args.Get(0).(entity.RSVPResponse), args.Error(1)
}

func (m *MockGuestDataUC) MyRSVP(ctx context.Context, blockID string, guestID uuid.UUID) (*entity.RSVPResponse, error) {
	args := m.Called(ctx, blockID, guestID)
	v, _ := args.Get(0).(*entity.RSVPResponse)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) OwnerRSVPSummary(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) (entity.RSVPSummary, error) {
	args := m.Called(ctx, userID, wishlistID, blockID)
	return args.Get(0).(entity.RSVPSummary), args.Error(1)
}

func (m *MockGuestDataUC) Vote(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, option int) (entity.PollResults, error) {
	args := m.Called(ctx, wishlistID, blockID, guestID, option)
	return args.Get(0).(entity.PollResults), args.Error(1)
}

func (m *MockGuestDataUC) PollResults(ctx context.Context, blockID string, guestID uuid.UUID) (entity.PollResults, error) {
	args := m.Called(ctx, blockID, guestID)
	return args.Get(0).(entity.PollResults), args.Error(1)
}

func (m *MockGuestDataUC) SuggestTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) ([]entity.PlaylistTrack, error) {
	args := m.Called(ctx, wishlistID, blockID, guestID, title)
	v, _ := args.Get(0).([]entity.PlaylistTrack)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) Tracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	args := m.Called(ctx, blockID, guestID)
	v, _ := args.Get(0).([]entity.PlaylistTrack)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	args := m.Called(ctx, trackID, guestID)
	v, _ := args.Get(0).([]entity.PlaylistTrack)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) AddGuestbookEntry(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input usecase.GuestbookInput) (entity.GuestbookEntry, error) {
	args := m.Called(ctx, wishlistID, blockID, guestID, input)
	return args.Get(0).(entity.GuestbookEntry), args.Error(1)
}

func (m *MockGuestDataUC) Guestbook(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.GuestbookEntry, error) {
	args := m.Called(ctx, blockID, guestID)
	v, _ := args.Get(0).([]entity.GuestbookEntry)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) OwnerGuestbook(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) ([]entity.GuestbookEntry, error) {
	args := m.Called(ctx, userID, wishlistID, blockID)
	v, _ := args.Get(0).([]entity.GuestbookEntry)
	return v, args.Error(1)
}

func (m *MockGuestDataUC) OwnerSetGuestbookHidden(ctx context.Context, userID, entryID uuid.UUID, hidden bool) error {
	args := m.Called(ctx, userID, entryID, hidden)
	return args.Error(0)
}

// MockTemplateUC

type MockTemplateUC struct{ mock.Mock }

func (m *MockTemplateUC) Create(ctx context.Context, userID uuid.UUID, input usecase.CreateTemplateInput) (entity.Template, error) {
	args := m.Called(ctx, userID, input)
	return args.Get(0).(entity.Template), args.Error(1)
}

func (m *MockTemplateUC) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Template, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]entity.Template), args.Error(1)
}

func (m *MockTemplateUC) GetPublic(ctx context.Context, limit, page int, userID *uuid.UUID) ([]entity.TemplateWithAuthor, bool, error) {
	args := m.Called(ctx, limit, page, userID)
	return args.Get(0).([]entity.TemplateWithAuthor), args.Bool(1), args.Error(2)
}

func (m *MockTemplateUC) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input usecase.UpdateTemplateInput) (entity.Template, error) {
	args := m.Called(ctx, id, userID, input)
	return args.Get(0).(entity.Template), args.Error(1)
}

func (m *MockTemplateUC) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTemplateUC) CreateWishlistFromTemplate(ctx context.Context, templateID uuid.UUID, userID uuid.UUID, title string) (entity.Wishlist, error) {
	args := m.Called(ctx, templateID, userID, title)
	return args.Get(0).(entity.Wishlist), args.Error(1)
}

func (m *MockTemplateUC) Like(ctx context.Context, userID, templateID uuid.UUID) (usecase.LikeResult, error) {
	args := m.Called(ctx, userID, templateID)
	return args.Get(0).(usecase.LikeResult), args.Error(1)
}

func (m *MockTemplateUC) Unlike(ctx context.Context, userID, templateID uuid.UUID) (usecase.LikeResult, error) {
	args := m.Called(ctx, userID, templateID)
	return args.Get(0).(usecase.LikeResult), args.Error(1)
}
