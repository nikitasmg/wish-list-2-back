package present_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
)

// «Дарит Аня» видят гости, но не владелец: «Маша не узнает, кто что дарит».
func TestGetAllByWishlist_ReserverNameOnlyForGuests(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("GetAllByWishlistID", mock.Anything, wid).Return([]entity.Present{
		{ID: uuid.New(), Reserved: true, ReservedByName: "Аня"},
	}, nil)

	asGuest, err := uc.GetAllByWishlist(context.Background(), wid, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, "Аня", asGuest[0].ReservedByName)

	asOwner, err := uc.GetAllByWishlist(context.Background(), wid, owner)
	require.NoError(t, err)
	assert.Empty(t, asOwner[0].ReservedByName)
}

func TestReserve_PassesTrimmedName(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	uc := newPresentUC(pr, &mockrepo.MockWishlistRepo{}, &mockminio.MockFileStorage{})

	id, guest := uuid.New(), uuid.New()
	pr.On("Reserve", mock.Anything, id, guest, "Аня").Return(true, nil)

	require.NoError(t, uc.Reserve(context.Background(), id, guest, "  Аня  "))
	pr.AssertExpectations(t)
}

func TestReserve_RejectsLongName(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	uc := newPresentUC(pr, &mockrepo.MockWishlistRepo{}, &mockminio.MockFileStorage{})

	err := uc.Reserve(context.Background(), uuid.New(), uuid.New(), strings.Repeat("я", 61))
	require.Error(t, err)
	pr.AssertNotCalled(t, "Reserve", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Главная мечта одна: отметка нового подарка снимает её с остальных.
func TestCreate_MainDreamIsUnique(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("CountByWishlistID", mock.Anything, wid).Return(int64(4), nil)
	pr.On("Create", mock.Anything, mock.MatchedBy(func(p entity.Present) bool {
		return p.IsMain && p.SortOrder == 4
	})).Return(nil)
	pr.On("ClearMain", mock.Anything, wid, mock.AnythingOfType("uuid.UUID")).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{Title: "Кофемашина", IsMain: true})
	require.NoError(t, err)
	assert.True(t, p.IsMain)
	assert.Equal(t, 4, p.SortOrder, "новый подарок — в конец списка")
	pr.AssertCalled(t, "ClearMain", mock.Anything, wid, p.ID)
}

func TestCreate_NotMainDoesNotTouchOthers(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("CountByWishlistID", mock.Anything, wid).Return(int64(0), nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{Title: "Лампа"})
	require.NoError(t, err)
	pr.AssertNotCalled(t, "ClearMain", mock.Anything, mock.Anything, mock.Anything)
}

func TestReorder(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Reorder", mock.Anything, wid, ids).Return(nil)

	require.NoError(t, uc.Reorder(context.Background(), owner, wid, ids))

	err := uc.Reorder(context.Background(), uuid.New(), wid, ids)
	require.ErrorIs(t, err, usecase.ErrForbidden)
	pr.AssertNumberOfCalls(t, "Reorder", 1)
}

func TestSetGifted(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid, id := uuid.New(), uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, WishlistID: wid}, nil)
	pr.On("SetGifted", mock.Anything, id, true).Return(nil)

	p, err := uc.SetGifted(context.Background(), owner, id, true)
	require.NoError(t, err)
	assert.True(t, p.Gifted)

	_, err = uc.SetGifted(context.Background(), uuid.New(), id, true)
	require.ErrorIs(t, err, usecase.ErrForbidden)
	pr.AssertNumberOfCalls(t, "SetGifted", 1)
}

func TestUpdate_MainDreamIsUnique(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	uc := newPresentUC(pr, wr, &mockminio.MockFileStorage{})

	wid, id := uuid.New(), uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, WishlistID: wid, SortOrder: 7, Gifted: true}, nil)
	pr.On("ClearMain", mock.Anything, wid, id).Return(nil)
	pr.On("Update", mock.Anything, mock.MatchedBy(func(p entity.Present) bool {
		return p.IsMain && p.SortOrder == 7 && p.Gifted
	})).Return(nil)

	_, err := uc.Update(context.Background(), owner, id, usecase.CreatePresentInput{Title: "Т", IsMain: true})
	require.NoError(t, err)
	pr.AssertExpectations(t)
}

func TestDescriptionLimitIs1000(t *testing.T) {
	assert.Equal(t, 1000, entity.MaxPresentDescriptionLen)
}
