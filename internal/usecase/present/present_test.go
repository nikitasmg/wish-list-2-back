package present_test

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
	presentUC "main/internal/usecase/present"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
)

func newPresentUC(pr *mockrepo.MockPresentRepo, wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.PresentUseCase {
	mr := &mockrepo.MockPresentMetaRepo{}
	return presentUC.New(pr, wr, fs, mr)
}

func TestParsePrice_Empty(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title:    "Gift",
		PriceStr: "",
	})
	require.NoError(t, err)
	assert.Nil(t, p.Price)
}

func TestParsePrice_CommaSpaces(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title:    "Gift",
		PriceStr: "1 500,50",
	})
	require.NoError(t, err)
	require.NotNil(t, p.Price)
	assert.InDelta(t, 1500.50, *p.Price, 0.001)
}

func TestParsePrice_Invalid(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title:    "Gift",
		PriceStr: "abc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "формат")
}

func TestReserve_AlreadyReserved(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id := uuid.New()
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, Reserved: true}, nil)

	err := uc.Reserve(context.Background(), id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "уже был забронирован")
}

func TestReserve_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id := uuid.New()
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, Reserved: false}, nil)
	pr.On("Update", mock.Anything, mock.MatchedBy(func(p entity.Present) bool {
		return p.Reserved == true
	})).Return(nil)

	err := uc.Reserve(context.Background(), id)
	require.NoError(t, err)
	pr.AssertExpectations(t)
}

func TestRelease_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id := uuid.New()
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, Reserved: true}, nil)
	pr.On("Update", mock.Anything, mock.MatchedBy(func(p entity.Present) bool {
		return p.Reserved == false
	})).Return(nil)

	err := uc.Release(context.Background(), id)
	require.NoError(t, err)
	pr.AssertExpectations(t)
}

func TestCreate_WishlistNotFound(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{}, errors.New("not found"))

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{Title: "Gift"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не существует")
}

func TestCreate_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{Title: "Gift"})
	require.NoError(t, err)
	pr.AssertCalled(t, "Create", mock.Anything, mock.Anything)
	wr.AssertCalled(t, "IncrementPresentsCount", mock.Anything, wid)
}

func TestDelete_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id := uuid.New()
	wid := uuid.New()
	pr.On("Delete", mock.Anything, id).Return(nil)
	wr.On("DecrementPresentsCount", mock.Anything, wid).Return(nil)

	err := uc.Delete(context.Background(), wid, id)
	require.NoError(t, err)
	pr.AssertCalled(t, "Delete", mock.Anything, id)
	wr.AssertCalled(t, "DecrementPresentsCount", mock.Anything, wid)
}

func TestCreate_WithSource_SavesMeta(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	mr := &mockrepo.MockPresentMetaRepo{}
	uc := presentUC.New(pr, wr, fs, mr)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)
	mr.On("Upsert", mock.Anything, mock.MatchedBy(func(m entity.PresentMeta) bool {
		return m.Source == "ozon" && m.OriginalURL == "https://ozon.ru/product/1"
	})).Return(nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{
		Title:       "Gift",
		Source:      "ozon",
		OriginalURL: "https://ozon.ru/product/1",
	})
	require.NoError(t, err)
	mr.AssertExpectations(t)
}

func TestCreate_WithoutSource_SkipsMeta(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	mr := &mockrepo.MockPresentMetaRepo{}
	uc := presentUC.New(pr, wr, fs, mr)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), wid, usecase.CreatePresentInput{Title: "Gift"})
	require.NoError(t, err)
	mr.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
}
