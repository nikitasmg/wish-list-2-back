package wishlist_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	wishlistUC "main/internal/usecase/wishlist"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
	"main/pkg/imagefile/imagefiletest"
)

// owner — владелец вишлистов в тестах пакета.
var owner = uuid.New()

func newWishlistUC(wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.WishlistUseCase {
	return wishlistUC.New(wr, fs)
}

func TestValidateBlocks_UnknownType(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "Test",
		Blocks: []entity.Block{
			{Type: "unknown_type", Row: 0},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "block[0]")
	assert.Contains(t, err.Error(), "unknown_type")
}

func TestValidateBlocks_Valid(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	blocks := []entity.Block{
		{Type: "cover", Row: 0, View: "number"},
		{Type: "text", Row: 1},
		{Type: "media", Row: 2, View: "row"},
		{Type: "list", Row: 3, View: "tags"},
		{Type: "location", Row: 4},
		{Type: "color_scheme", Row: 5},
		{Type: "timing", Row: 6},
		{Type: "wishlist", Row: 7, View: "cards"},
		{Type: "rsvp", Row: 8},
	}

	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title:  "Test",
		Blocks: blocks,
	})
	require.NoError(t, err)
}

func TestCreate_FileUpload(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	cover := imagefiletest.PNG(t)
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)
	fs.On("Upload", "cover.png", cover).Return("https://minio/cover.png", nil)

	w, err := uc.Create(context.Background(), userID, usecase.CreateWishlistInput{
		Title:     "My Wishlist",
		CoverData: cover,
		CoverName: "cover.png",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://minio/cover.png", w.Cover)
	fs.AssertCalled(t, "Upload", "cover.png", cover)
}

// Обложка вишлиста проходит ту же проверку, что и загрузка через /upload:
// иначе валидацию можно обойти, подложив файл в поле обложки.
func TestCreate_RejectsNonImageCover(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))

	_, err := uc.Create(context.Background(), uuid.New(), usecase.CreateWishlistInput{
		Title:     "My Wishlist",
		CoverData: []byte("%PDF-1.7 не картинка"),
		CoverName: "doc.pdf",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "формат не поддерживается")
	fs.AssertNotCalled(t, "Upload", mock.Anything, mock.Anything)
	wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreate_URLCover(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
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
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
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

func TestUpdateBlocks_Success(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	blocks := []entity.Block{{Type: "text", Row: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, time.Time{}).
		Return(true, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner, Blocks: blocks}, nil)

	w, err := uc.UpdateBlocks(context.Background(), owner, wid, blocks, time.Time{})
	require.NoError(t, err)
	assert.Len(t, w.Blocks, 1)
	wr.AssertExpectations(t)
}

// Автосохранение из двух вкладок: та, что держит устаревшую версию, должна
// получить конфликт, а не молча затереть чужую правку.
func TestUpdateBlocks_Conflict(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	stale := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	current := entity.Wishlist{ID: wid, UserID: owner, Blocks: []entity.Block{{Type: "quote"}}}

	blocks := []entity.Block{{Type: "text", Row: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, stale).
		Return(false, nil)
	wr.On("GetByID", mock.Anything, wid).Return(current, nil)

	w, err := uc.UpdateBlocks(context.Background(), owner, wid, blocks, stale)

	require.ErrorIs(t, err, usecase.ErrVersionConflict)
	assert.Equal(t, "quote", w.Blocks[0].Type, "вместе с ошибкой отдаётся актуальная версия")
}

// Вишлист формата v1 должен оставаться сохраняемым: иначе человек откроет
// старый список, поправит один заголовок и не сможет нажать «Сохранить».
func TestUpdateBlocks_AllowsLegacyTypes(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	blocks := []entity.Block{{Type: "agenda", Row: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, time.Time{}).
		Return(true, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner, Blocks: blocks}, nil)

	_, err := uc.UpdateBlocks(context.Background(), owner, wid, blocks, time.Time{})

	require.NoError(t, err)
}

// А вот собрать новый вишлист из блоков v1 уже нельзя.
func TestCreateConstructor_RejectsLegacyTypes(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	_, err := uc.CreateConstructor(context.Background(), uuid.New(), usecase.CreateConstructorInput{
		Title:  "Test",
		Blocks: []entity.Block{{Type: "checklist", Row: 0}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list и media")
	wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestGetByShortID_HidesHiddenAndSecretBlocks(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid, guest := uuid.New(), uuid.New()
	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(-48 * time.Hour)

	wr.On("GetByShortID", mock.Anything, "abc-def-ghi").Return(entity.Wishlist{
		ID: wid,
		Blocks: []entity.Block{
			{Type: "text", Row: 0, Data: json.RawMessage(`{"html":"виден"}`)},
			{Type: "text", Row: 1, Hidden: true, Data: json.RawMessage(`{"html":"скрыт"}`)},
			{ID: "secret", Type: "poll", Row: 2, Col: 1, ColSpan: 1, View: "cards", Caption: "секрет", Title: "Мальчик или девочка?", RevealAt: &future,
				Data: json.RawMessage(`{"answer":"девочка"}`)},
			{Type: "media", Row: 3, RevealAt: &past, Data: json.RawMessage(`{"url":"уже можно"}`)},
		},
	}, nil)
	wr.On("RegisterView", mock.Anything, wid, guest).Return(nil)

	w, err := uc.GetByShortID(context.Background(), "abc-def-ghi", guest)
	require.NoError(t, err)

	require.Len(t, w.Blocks, 3, "скрытый блок не отдаётся вовсе")

	assert.JSONEq(t, `{"html":"виден"}`, string(w.Blocks[0].Data))

	secret := w.Blocks[1]
	assert.Equal(t, "secret", secret.ID)
	assert.Equal(t, 2, secret.Row)
	assert.Equal(t, 1, secret.Col)
	assert.Equal(t, 1, secret.ColSpan)
	assert.Empty(t, secret.View)
	assert.Empty(t, secret.Caption)
	assert.Equal(t, "poll", secret.Type, "тип нужен, чтобы нарисовать таймер")
	assert.JSONEq(t, `{}`, string(secret.Data), "содержимое секрета не должно попадать в ответ")
	assert.Empty(t, secret.Title, "заголовок тоже раскрывает секрет")
	assert.NotNil(t, secret.RevealAt)

	assert.JSONEq(t, `{"url":"уже можно"}`, string(w.Blocks[2].Data), "срок вышел — блок раскрыт")
}

func TestGetAllByUser_FillsReservedCount(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	first, second := uuid.New(), uuid.New()
	wr.On("GetAllByUserID", mock.Anything, userID).Return([]entity.Wishlist{
		{ID: first, PresentsCount: 8},
		{ID: second, PresentsCount: 6},
	}, nil)
	wr.On("ReservedCountsByUser", mock.Anything, userID).
		Return(map[uuid.UUID]uint{first: 4}, nil)

	wishlists, err := uc.GetAllByUser(context.Background(), userID)
	require.NoError(t, err)

	assert.Equal(t, uint(4), wishlists[0].ReservedCount)
	assert.Equal(t, uint(0), wishlists[1].ReservedCount, "без броней — ноль, а не пропуск")
}

func TestCustomScheme_Validation(t *testing.T) {
	cases := []struct {
		name        string
		colorScheme string
		scheme      *entity.CustomScheme
		wantErr     string
	}{
		{name: "валидная", colorScheme: "custom", scheme: &entity.CustomScheme{Base: "dark", Accent: "#FF8A65"}},
		{name: "без своей схемы", colorScheme: "space", scheme: nil},
		{name: "чужая база", colorScheme: "custom", scheme: &entity.CustomScheme{Base: "neon", Accent: "#FF8A65"}, wantErr: "dark или light"},
		{name: "акцент не цвет", colorScheme: "custom", scheme: &entity.CustomScheme{Base: "light", Accent: "red; }"}, wantErr: "#RRGGBB"},
		{name: "схема задана не для custom", colorScheme: "space", scheme: &entity.CustomScheme{Base: "dark", Accent: "#FF8A65"}, wantErr: "только при colorScheme"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
			fs := &mockminio.MockFileStorage{}
			uc := newWishlistUC(wr, fs)
			wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
			wr.On("Create", mock.Anything, mock.Anything).Return(nil)

			_, err := uc.Create(context.Background(), uuid.New(), usecase.CreateWishlistInput{
				Title:        "Test",
				ColorScheme:  tc.colorScheme,
				CustomScheme: tc.scheme,
			})

			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestCreateFromSystemTemplate_CopiesBlocksAndScheme(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	eventDate := time.Date(2027, 6, 6, 15, 0, 0, 0, time.UTC)
	w, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), usecase.CreateFromSystemTemplateInput{
		TemplateID: "wedding",
		Title:      "Маша и Петя",
		EventDate:  &eventDate,
	})

	require.NoError(t, err)
	assert.Equal(t, "Маша и Петя", w.Title)
	assert.Equal(t, "linen", w.Settings.ColorScheme)
	assert.Equal(t, "Свадьба", w.Occasion)
	assert.Equal(t, entity.BlocksVersionCurrent, w.BlocksVersion)
	assert.NotEmpty(t, w.Blocks)
	require.NotNil(t, w.EventDate)
	assert.Equal(t, eventDate, *w.EventDate)

	// Обложка должна называться так же, как вишлист, иначе на странице
	// останется имя из шаблона.
	for _, b := range w.Blocks {
		if b.Type == "cover" {
			assert.Equal(t, "Маша и Петя", b.Title)
		}
	}
}

// Блоки шаблона общие для всех, поэтому вишлист должен получать копию.
func TestCreateFromSystemTemplate_DoesNotMutateTemplate(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), usecase.CreateFromSystemTemplateInput{
		TemplateID: "boy", Title: "Первый",
	})
	require.NoError(t, err)

	second, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), usecase.CreateFromSystemTemplateInput{
		TemplateID: "boy", Title: "Второй",
	})
	require.NoError(t, err)

	for _, b := range second.Blocks {
		if b.Type == "cover" {
			assert.Equal(t, "Второй", b.Title, "шаблон не должен запоминать чужое название")
		}
	}
}

func TestCreateFromSystemTemplate_UsesSampleTitleWhenEmpty(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	w, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), usecase.CreateFromSystemTemplateInput{
		TemplateID: "jubilee",
	})

	require.NoError(t, err)
	assert.Equal(t, "Сергей Петрович", w.Title)
}

func TestCreateFromSystemTemplate_UnknownTemplate(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	_, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), usecase.CreateFromSystemTemplateInput{
		TemplateID: "нет-такого",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "не найден")
	wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Без проверки владельца любой залогиненный человек правил бы и удалял чужие
// вишлисты, зная только UUID из публичной ссылки.
func TestMutations_RejectForeignWishlist(t *testing.T) {
	stranger := uuid.New()
	wid := uuid.New()

	newUC := func() (*mockrepo.MockWishlistRepo, usecase.WishlistUseCase) {
		wr := &mockrepo.MockWishlistRepo{}
		wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
		fs := &mockminio.MockFileStorage{}
		wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
		return wr, newWishlistUC(wr, fs)
	}

	t.Run("update", func(t *testing.T) {
		wr, uc := newUC()
		_, err := uc.Update(context.Background(), stranger, wid, usecase.CreateWishlistInput{Title: "Чужой"}, time.Time{})
		require.ErrorIs(t, err, usecase.ErrForbidden)
		wr.AssertNotCalled(t, "UpdateMetadata", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("updateBlocks", func(t *testing.T) {
		wr, uc := newUC()
		_, err := uc.UpdateBlocks(context.Background(), stranger, wid, []entity.Block{{Type: "text"}}, time.Time{})
		require.ErrorIs(t, err, usecase.ErrForbidden)
		wr.AssertNotCalled(t, "UpdateBlocks", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("delete", func(t *testing.T) {
		wr, uc := newUC()
		err := uc.Delete(context.Background(), stranger, wid)
		require.ErrorIs(t, err, usecase.ErrForbidden)
		wr.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}

// Ответы гостей, голоса и треки привязаны к блоку, поэтому у каждого блока
// должен быть стабильный id — позиция меняется при первой же перестановке.
func TestCreateConstructor_AssignsBlockIDs(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	w, err := uc.CreateConstructor(context.Background(), owner, usecase.CreateConstructorInput{
		Title: "Test",
		Blocks: []entity.Block{
			{Type: "poll", Row: 0},
			{Type: "text", Row: 1, ID: "уже-есть"},
		},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, w.Blocks[0].ID, "новому блоку выдаётся id")
	assert.Equal(t, "уже-есть", w.Blocks[1].ID, "чужой id не перетирается")
}

// Наружу должна уходить версия, которую вернула база, а не та, что лежала в
// памяти до записи: со старым updatedAt клиент унёс бы в следующий запрос
// версию, которой уже нет, и проверка конфликтов перестала бы работать.
func TestUpdate_ReturnsSavedVersion(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	was := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	became := time.Date(2026, 9, 29, 15, 30, 0, 0, time.UTC)

	wr.On("GetByID", mock.Anything, wid).
		Return(entity.Wishlist{ID: wid, UserID: owner, Title: "Было", UpdatedAt: was}, nil)
	wr.On("UpdateMetadata", mock.Anything, wid, mock.Anything, was).
		Return(entity.Wishlist{ID: wid, UserID: owner, Title: "Стало", UpdatedAt: became}, true, nil)

	saved, err := uc.Update(context.Background(), owner, wid,
		usecase.CreateWishlistInput{Title: "Стало"}, was)

	require.NoError(t, err)
	assert.Equal(t, "Стало", saved.Title)
	assert.Equal(t, became, saved.UpdatedAt, "отдаём версию из базы, а не из памяти")
}

func TestUpdate_Conflict(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	stale := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	current := entity.Wishlist{ID: wid, UserID: owner, Title: "Соседняя вкладка"}

	wr.On("GetByID", mock.Anything, wid).Return(current, nil)
	wr.On("UpdateMetadata", mock.Anything, wid, mock.Anything, stale).
		Return(entity.Wishlist{}, false, nil)

	w, err := uc.Update(context.Background(), owner, wid,
		usecase.CreateWishlistInput{Title: "Затирание"}, stale)

	require.ErrorIs(t, err, usecase.ErrVersionConflict)
	assert.Equal(t, "Соседняя вкладка", w.Title, "вместе с ошибкой отдаётся актуальная версия")
}

// Настройки пишутся отдельным запросом, который не знает о блоках: у usecase
// нет способа их затереть, потому что он их и не передаёт.
func TestUpdate_DoesNotSendBlocks(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{
		ID: wid, UserID: owner,
		Blocks: []entity.Block{{ID: "b1", Type: "text"}},
	}, nil)
	wr.On("UpdateMetadata", mock.Anything, wid, mock.Anything, mock.Anything).
		Return(entity.Wishlist{ID: wid, UserID: owner}, true, nil)

	_, err := uc.Update(context.Background(), owner, wid,
		usecase.CreateWishlistInput{Title: "Стало"}, time.Time{})

	require.NoError(t, err)
	wr.AssertNotCalled(t, "UpdateBlocks", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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

func TestCreateFromSystemTemplate_Limits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count int64
		title string
		want  string
	}{
		{name: "wishlist count", count: 20, title: "Test", want: "лимит вишлистов"},
		{name: "title length", title: string(make([]byte, 201)), want: "title"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			wr.On("CountByUserID", mock.Anything, owner).Return(tc.count, nil)
			uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
			_, err := uc.CreateFromSystemTemplate(context.Background(), owner, usecase.CreateFromSystemTemplateInput{
				TemplateID: "wedding", Title: tc.title,
			})
			require.ErrorContains(t, err, tc.want)
			wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			wr.AssertExpectations(t)
		})
	}
}

func TestCreateConstructor_Coordinates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		block entity.Block
		want  string
	}{
		{name: "negative row", block: entity.Block{Type: "text", Row: -1}, want: "row"},
		{name: "negative column", block: entity.Block{Type: "text", Col: -1}, want: "col"},
		{name: "column outside grid", block: entity.Block{Type: "text", Col: 2}, want: "col"},
		{name: "too wide", block: entity.Block{Type: "text", ColSpan: 3}, want: "colSpan"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			wr.On("CountByUserID", mock.Anything, owner).Return(int64(0), nil)
			uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
			_, err := uc.CreateConstructor(context.Background(), owner, usecase.CreateConstructorInput{
				Title: "Test", Blocks: []entity.Block{tc.block},
			})
			require.ErrorContains(t, err, tc.want)
			wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

// Main's legacy blocks remain valid when saving an existing wishlist.
func TestUpdateBlocks_LegacyCoordinates(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
	wid := uuid.New()
	blocks := []entity.Block{
		{Type: "text", Row: 0, Col: 0, ColSpan: 1},
		{Type: "image", Row: 0, Col: 1, ColSpan: 1},
		{Type: "date", Row: 1, Col: 0, ColSpan: 1},
		{Type: "location", Row: 1, Col: 1, ColSpan: 1},
		{Type: "color_scheme", Row: 2, Col: 0, ColSpan: 1},
		{Type: "timing", Row: 2, Col: 1, ColSpan: 1},
		{Type: "text_image", Row: 3, Col: 0, ColSpan: 1},
	}
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner, Blocks: blocks}, nil)
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, time.Time{}).Return(true, nil)
	w, err := uc.UpdateBlocks(context.Background(), owner, wid, blocks, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, blocks, w.Blocks)
	wr.AssertExpectations(t)
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

func TestValidateBlocks_TooManyBlocks(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	blocks := make([]entity.Block, 101)
	for i := range blocks {
		blocks[i] = entity.Block{Type: "text"}
	}
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title:  "X",
		Blocks: blocks,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many blocks")
}

func TestValidateBlocks_DataTooLarge(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	bigData := `{"content":"` + string(make([]byte, 11*1024)) + `"}`
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "text", Data: json.RawMessage(bigData)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data too large")
}

func TestValidateBlocks_TextContentTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	// Use ASCII chars so JSON stays under MaxBlockDataSize (10KB) but content > 5000 runes
	buf := make([]byte, 5001)
	for i := range buf {
		buf[i] = 'a'
	}
	longContent := string(buf)
	contentJSON, _ := json.Marshal(map[string]string{"content": longContent})
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "text", Data: json.RawMessage(contentJSON)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestValidateBlocks_VideoURLTooLong(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
	wr.On("CountByUserID", mock.Anything, userID).Return(int64(0), nil)

	// Build a URL that exceeds MaxURLLen (2048) but keeps JSON under MaxBlockDataSize (10KB)
	urlBuf := make([]byte, 2049)
	urlBuf[0] = 'h'
	for i := 1; i < len(urlBuf); i++ {
		urlBuf[i] = 'a'
	}
	longURL := string(urlBuf)
	urlJSON, _ := json.Marshal(map[string]string{"url": longURL})
	_, err := uc.CreateConstructor(context.Background(), userID, usecase.CreateConstructorInput{
		Title: "X",
		Blocks: []entity.Block{
			{Type: "video", Data: json.RawMessage(urlJSON)},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url")
}
