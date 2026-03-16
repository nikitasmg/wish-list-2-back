package wishlist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	wishlistUC "main/internal/usecase/wishlist"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
)

func newWishlistUC(wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.WishlistUseCase {
	return wishlistUC.New(wr, fs)
}

func TestValidateBlocks_UnknownType(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "Test",
		Blocks: []entity.Block{
			{Type: "unknown_type", Position: 0},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block[0]")
	assert.Contains(t, err.Error(), "unknown_type")
}

func TestValidateBlocks_Valid(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	blocks := []entity.Block{
		{Type: "text", Position: 0},
		{Type: "image", Position: 1},
		{Type: "date", Position: 2},
		{Type: "location", Position: 3},
		{Type: "color_scheme", Position: 4},
		{Type: "timing", Position: 5},
		{Type: "text_image", Position: 6},
	}

	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title:  "Test",
		Blocks: blocks,
	})
	require.NoError(t, err)
}

func TestCreate_FileUpload(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)
	fs.On("Upload", "cover.jpg", []byte("imgdata")).Return("https://minio/cover.jpg", nil)

	w, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title:     "My Wishlist",
		CoverData: []byte("imgdata"),
		CoverName: "cover.jpg",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://minio/cover.jpg", w.Cover)
	fs.AssertCalled(t, "Upload", "cover.jpg", []byte("imgdata"))
}

func TestCreate_URLCover(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	w, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title:    "My Wishlist",
		CoverURL: "https://example.com/img.jpg",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/img.jpg", w.Cover)
	fs.AssertNotCalled(t, "Upload", mock.Anything, mock.Anything)
}

func TestGenerateUniqueShortID_Collision(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)
	// First call: short ID is taken (collision)
	wr.On("GetByShortID", mock.Anything, mock.Anything).
		Return(entity.Wishlist{ID: uuid.New()}, nil).Once()
	// Second call: short ID is free
	wr.On("GetByShortID", mock.Anything, mock.Anything).
		Return(entity.Wishlist{}, errors.New("not found")).Once()
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title: "My Wishlist",
	})
	require.NoError(t, err)
	wr.AssertNumberOfCalls(t, "GetByShortID", 2)
}

func TestCreate_WishlistLimitExceeded(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(20), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{Title: "X"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "лимит вишлистов")
}

func TestCreateConstructor_WishlistLimitExceeded(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(20), nil)

	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{Title: "X"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "лимит вишлистов")
}

func TestCreate_TitleTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title: string(make([]byte, 201)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title")
}

func TestCreate_DescriptionTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	_, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title:       "OK",
		Description: string(make([]byte, 2001)),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description")
}

func TestUpdateBlocks_Success(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	wr.On("Update", mock.Anything, mock.MatchedBy(func(w entity.Wishlist) bool {
		return len(w.Blocks) == 1 && w.Blocks[0].Type == "text"
	})).Return(nil)

	blocks := []entity.Block{{Type: "text", Position: 0}}
	w, err := uc.UpdateBlocks(context.Background(), wid, blocks)
	require.NoError(t, err)
	assert.Len(t, w.Blocks, 1)
	wr.AssertExpectations(t)
}
