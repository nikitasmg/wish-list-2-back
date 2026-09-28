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

func newWishlistUC(wr *mockrepo.MockWishlistRepo, fs *mockminio.MockFileStorage) usecase.WishlistUseCase {
	return wishlistUC.New(wr, fs)
}

func TestValidateBlocks_UnknownType(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	userID := uuid.New()
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
	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	blocks := []entity.Block{
		{Type: "cover", Position: 0, View: "number"},
		{Type: "text", Position: 1},
		{Type: "media", Position: 2, View: "row"},
		{Type: "list", Position: 3, View: "tags"},
		{Type: "location", Position: 4},
		{Type: "color_scheme", Position: 5},
		{Type: "timing", Position: 6},
		{Type: "wishlist", Position: 7, View: "cards"},
		{Type: "rsvp", Position: 8},
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
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	blocks := []entity.Block{{Type: "text", Position: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, time.Time{}).
		Return(true, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, Blocks: blocks}, nil)

	w, err := uc.UpdateBlocks(context.Background(), wid, blocks, time.Time{})
	require.NoError(t, err)
	assert.Len(t, w.Blocks, 1)
	wr.AssertExpectations(t)
}

// Автосохранение из двух вкладок: та, что держит устаревшую версию, должна
// получить конфликт, а не молча затереть чужую правку.
func TestUpdateBlocks_Conflict(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	stale := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	current := entity.Wishlist{ID: wid, Blocks: []entity.Block{{Type: "quote"}}}

	blocks := []entity.Block{{Type: "text", Position: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, stale).
		Return(false, nil)
	wr.On("GetByID", mock.Anything, wid).Return(current, nil)

	w, err := uc.UpdateBlocks(context.Background(), wid, blocks, stale)

	require.ErrorIs(t, err, usecase.ErrBlocksConflict)
	assert.Equal(t, "quote", w.Blocks[0].Type, "вместе с ошибкой отдаётся актуальная версия")
}

// Вишлист формата v1 должен оставаться сохраняемым: иначе человек откроет
// старый список, поправит один заголовок и не сможет нажать «Сохранить».
func TestUpdateBlocks_AllowsLegacyTypes(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid := uuid.New()
	blocks := []entity.Block{{Type: "agenda", Position: 0}}
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, entity.BlocksVersionCurrent, time.Time{}).
		Return(true, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, Blocks: blocks}, nil)

	_, err := uc.UpdateBlocks(context.Background(), wid, blocks, time.Time{})

	require.NoError(t, err)
}

// А вот собрать новый вишлист из блоков v1 уже нельзя.
func TestCreateConstructor_RejectsLegacyTypes(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	_, err := uc.CreateConstructor(context.Background(), uuid.New(), usecase.CreateConstructorInput{
		Title:  "Test",
		Blocks: []entity.Block{{Type: "checklist", Position: 0}},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "list и media")
	wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestGetByShortID_HidesHiddenAndSecretBlocks(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wid, guest := uuid.New(), uuid.New()
	future := time.Now().Add(48 * time.Hour)
	past := time.Now().Add(-48 * time.Hour)

	wr.On("GetByShortID", mock.Anything, "abc-def-ghi").Return(entity.Wishlist{
		ID: wid,
		Blocks: []entity.Block{
			{Type: "text", Position: 0, Data: json.RawMessage(`{"html":"виден"}`)},
			{Type: "text", Position: 1, Hidden: true, Data: json.RawMessage(`{"html":"скрыт"}`)},
			{Type: "poll", Position: 2, Title: "Мальчик или девочка?", RevealAt: &future,
				Data: json.RawMessage(`{"answer":"девочка"}`)},
			{Type: "media", Position: 3, RevealAt: &past, Data: json.RawMessage(`{"url":"уже можно"}`)},
		},
	}, nil)
	wr.On("RegisterView", mock.Anything, wid, guest).Return(nil)

	w, err := uc.GetByShortID(context.Background(), "abc-def-ghi", guest)
	require.NoError(t, err)

	require.Len(t, w.Blocks, 3, "скрытый блок не отдаётся вовсе")

	assert.JSONEq(t, `{"html":"виден"}`, string(w.Blocks[0].Data))

	secret := w.Blocks[1]
	assert.Equal(t, "poll", secret.Type, "тип нужен, чтобы нарисовать таймер")
	assert.JSONEq(t, `{}`, string(secret.Data), "содержимое секрета не должно попадать в ответ")
	assert.Empty(t, secret.Title, "заголовок тоже раскрывает секрет")
	assert.NotNil(t, secret.RevealAt)

	assert.JSONEq(t, `{"url":"уже можно"}`, string(w.Blocks[2].Data), "срок вышел — блок раскрыт")
}

func TestGetAllByUser_FillsReservedCount(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
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

func TestCreateFromTemplate_CopiesBlocksAndScheme(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	eventDate := time.Date(2027, 6, 6, 15, 0, 0, 0, time.UTC)
	w, err := uc.CreateFromTemplate(context.Background(), uuid.New(), usecase.CreateFromTemplateInput{
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
func TestCreateFromTemplate_DoesNotMutateTemplate(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := uc.CreateFromTemplate(context.Background(), uuid.New(), usecase.CreateFromTemplateInput{
		TemplateID: "boy", Title: "Первый",
	})
	require.NoError(t, err)

	second, err := uc.CreateFromTemplate(context.Background(), uuid.New(), usecase.CreateFromTemplateInput{
		TemplateID: "boy", Title: "Второй",
	})
	require.NoError(t, err)

	for _, b := range second.Blocks {
		if b.Type == "cover" {
			assert.Equal(t, "Второй", b.Title, "шаблон не должен запоминать чужое название")
		}
	}
}

func TestCreateFromTemplate_UsesSampleTitleWhenEmpty(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	wr.On("GetByShortID", mock.Anything, mock.Anything).Return(entity.Wishlist{}, errors.New("not found"))
	wr.On("Create", mock.Anything, mock.Anything).Return(nil)

	w, err := uc.CreateFromTemplate(context.Background(), uuid.New(), usecase.CreateFromTemplateInput{
		TemplateID: "jubilee",
	})

	require.NoError(t, err)
	assert.Equal(t, "Сергей Петрович", w.Title)
}

func TestCreateFromTemplate_UnknownTemplate(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	fs := &mockminio.MockFileStorage{}
	uc := newWishlistUC(wr, fs)

	_, err := uc.CreateFromTemplate(context.Background(), uuid.New(), usecase.CreateFromTemplateInput{
		TemplateID: "нет-такого",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "не найден")
	wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
