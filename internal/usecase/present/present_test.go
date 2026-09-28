package present_test

import (
	"context"
	"errors"
	"strings"
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

// owner — владелец вишлистов во всех тестах пакета: подарки правит только он.
var owner = uuid.New()

func newPresentUC(pr *mockrepo.MockPresentRepo, wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.PresentUseCase {
	return presentUC.New(pr, wr, fs)
}

func TestParsePrice_Empty(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
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
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
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
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
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

	id, guest := uuid.New(), uuid.New()
	// Условный апдейт не нашёл свободной строки — подарок успели занять.
	pr.On("Reserve", mock.Anything, id, guest).Return(false, nil)
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, Reserved: true}, nil)

	err := uc.Reserve(context.Background(), id, guest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "уже был забронирован")
}

func TestReserve_NotFound(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id, guest := uuid.New(), uuid.New()
	pr.On("Reserve", mock.Anything, id, guest).Return(false, nil)
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{}, errors.New("no rows"))

	err := uc.Reserve(context.Background(), id, guest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReserve_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id, guest := uuid.New(), uuid.New()
	pr.On("Reserve", mock.Anything, id, guest).Return(true, nil)

	err := uc.Reserve(context.Background(), id, guest)
	require.NoError(t, err)
	pr.AssertExpectations(t)
	pr.AssertNotCalled(t, "GetByID", mock.Anything, id)
}

func TestRelease_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id, guest := uuid.New(), uuid.New()
	pr.On("Release", mock.Anything, id, guest).Return(true, nil)

	err := uc.Release(context.Background(), id, guest)
	require.NoError(t, err)
	pr.AssertExpectations(t)
}

// Главная дыра, которую закрывает привязка брони к гостю: раньше release был
// публичным и снимал любую бронь по одному лишь UUID подарка.
func TestRelease_RejectsForeignBooking(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	id, stranger := uuid.New(), uuid.New()
	pr.On("Release", mock.Anything, id, stranger).Return(false, nil)
	pr.On("GetByID", mock.Anything, id).Return(entity.Present{ID: id, Reserved: true}, nil)

	err := uc.Release(context.Background(), id, stranger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "только тот, кто её поставил")
}

func TestCreate_WishlistNotFound(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{}, errors.New("not found"))

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{Title: "Gift"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не существует")
}

func TestCreate_Success(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{Title: "Gift"})
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
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Delete", mock.Anything, id).Return(nil)
	wr.On("DecrementPresentsCount", mock.Anything, wid).Return(nil)

	err := uc.Delete(context.Background(), owner, wid, id)
	require.NoError(t, err)
	pr.AssertCalled(t, "Delete", mock.Anything, id)
	wr.AssertCalled(t, "DecrementPresentsCount", mock.Anything, wid)
}

// Главная проверка авторизации: JWT говорит, кто пришёл, но не чей вишлист он
// открыл. Без неё чужие подарки правились бы по одному UUID из публичной ссылки.
func TestMutations_RejectForeignWishlist(t *testing.T) {
	stranger := uuid.New()
	wid, pid := uuid.New(), uuid.New()

	newUC := func() (*mockrepo.MockPresentRepo, *mockrepo.MockWishlistRepo, usecase.PresentUseCase) {
		pr := &mockrepo.MockPresentRepo{}
		wr := &mockrepo.MockWishlistRepo{}
		fs := &mockminio.MockFileStorage{}
		wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
		return pr, wr, newPresentUC(pr, wr, fs)
	}

	t.Run("create", func(t *testing.T) {
		pr, _, uc := newUC()
		_, err := uc.Create(context.Background(), stranger, wid, usecase.CreatePresentInput{Title: "Gift"})
		require.ErrorIs(t, err, usecase.ErrForbidden)
		pr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("update", func(t *testing.T) {
		pr, _, uc := newUC()
		pr.On("GetByID", mock.Anything, pid).Return(entity.Present{ID: pid, WishlistID: wid}, nil)
		_, err := uc.Update(context.Background(), stranger, pid, usecase.CreatePresentInput{Title: "Gift"})
		require.ErrorIs(t, err, usecase.ErrForbidden)
		pr.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("delete", func(t *testing.T) {
		pr, _, uc := newUC()
		err := uc.Delete(context.Background(), stranger, wid, pid)
		require.ErrorIs(t, err, usecase.ErrForbidden)
		pr.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})

	t.Run("getByID", func(t *testing.T) {
		pr, _, uc := newUC()
		pr.On("GetByID", mock.Anything, pid).Return(entity.Present{ID: pid, WishlistID: wid}, nil)
		_, err := uc.GetByID(context.Background(), stranger, pid)
		require.ErrorIs(t, err, usecase.ErrForbidden)
	})
}

func TestCreate_RejectsTooLongDescription(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
		Title:       "Gift",
		Description: strings.Repeat("я", entity.MaxPresentDescriptionLen+1),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	pr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Лимит считается в символах: 500 кириллических букв — это 1000 байт, и
// побайтовая проверка отрезала бы ровно половину допустимого описания.
func TestCreate_AllowsExactlyMaxCyrillicDescription(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
		Title:       "Gift",
		Description: strings.Repeat("я", entity.MaxPresentDescriptionLen),
	})

	require.NoError(t, err)
}

func TestCreate_RejectsNonImageCover(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
		Title:     "Gift",
		CoverData: []byte("%PDF-1.7 это не картинка"),
		CoverName: "doc.pdf",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "формат не поддерживается")
	fs.AssertNotCalled(t, "Upload", mock.Anything, mock.Anything)
}

func TestCreate_NormalizesLinks(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	pr.On("Create", mock.Anything, mock.Anything).Return(nil)
	wr.On("IncrementPresentsCount", mock.Anything, wid).Return(nil)

	p, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
		Title: "Лампа-гриб",
		Links: []string{" https://ozon.ru/p/1 ", "", "https://market.yandex.ru/p/2"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"https://ozon.ru/p/1", "https://market.yandex.ru/p/2"}, p.Links)
}

// Ссылка без схемы стала бы относительной и увела гостя на несуществующую
// страницу самого вишлиста вместо магазина.
func TestCreate_RejectsSchemelessLink(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{
		Title: "Лампа-гриб",
		Links: []string{"ozon.ru/p/1"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "http")
	pr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreate_RejectsTooManyLinks(t *testing.T) {
	pr := &mockrepo.MockPresentRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newPresentUC(pr, wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	links := make([]string, entity.MaxPresentLinks+1)
	for i := range links {
		links[i] = "https://shop.example/p"
	}

	_, err := uc.Create(context.Background(), owner, wid, usecase.CreatePresentInput{Title: "Gift", Links: links})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "не больше")
}
