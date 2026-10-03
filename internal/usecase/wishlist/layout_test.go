package wishlist_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
)

// Ряды v3: до трёх колонок, пропорции и прочие настройки ряда едут рядом с
// блоками одним запросом.

func TestUpdateBlocks_ThreeColumnRow(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	uc := newWishlistUC(wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	blocks := []entity.Block{
		{ID: "a", Type: "text", Row: 0, Col: 0, ColSpan: 1},
		{ID: "b", Type: "text", Row: 0, Col: 1, ColSpan: 1},
		{ID: "c", Type: "text", Row: 0, Col: 2, ColSpan: 1},
	}
	rows := []entity.RowSettings{{Columns: 3, Ratio: "1:1:1", Height: "equal", Gap: "m"}}

	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	wr.On("UpdateBlocks", mock.Anything, wid, blocks, rows, entity.BlocksVersionCurrent, time.Time{}).Return(true, nil)

	_, err := uc.UpdateBlocks(context.Background(), owner, wid, blocks, rows, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 3, entity.BlocksVersionCurrent)
	wr.AssertExpectations(t)
}

func TestUpdateBlocks_LayoutValidation(t *testing.T) {
	cases := []struct {
		name    string
		blocks  []entity.Block
		rows    []entity.RowSettings
		wantErr string
	}{
		{
			name:    "без настроек ряда — две колонки, как в v2",
			blocks:  []entity.Block{{Type: "text", Row: 0, Col: 2, ColSpan: 1}},
			wantErr: "выходит за ряд",
		},
		{
			name:    "блок шире ряда",
			blocks:  []entity.Block{{Type: "text", Row: 0, Col: 1, ColSpan: 2}},
			rows:    []entity.RowSettings{{Columns: 2}},
			wantErr: "выходит за ряд",
		},
		{
			name: "блоки наезжают друг на друга",
			blocks: []entity.Block{
				{Type: "text", Row: 0, Col: 0, ColSpan: 2},
				{Type: "text", Row: 0, Col: 1, ColSpan: 1},
			},
			rows:    []entity.RowSettings{{Columns: 3}},
			wantErr: "занята",
		},
		{
			name:    "пропорция не для этого числа колонок",
			blocks:  []entity.Block{{Type: "text", Row: 0, Col: 0, ColSpan: 1}},
			rows:    []entity.RowSettings{{Columns: 2, Ratio: "1:1:1"}},
			wantErr: "ratio",
		},
		{
			name:    "четыре колонки",
			blocks:  []entity.Block{{Type: "text", Row: 0}},
			rows:    []entity.RowSettings{{Columns: 4}},
			wantErr: "columns",
		},
		{
			name:    "неизвестный отступ",
			blocks:  []entity.Block{{Type: "text", Row: 0}},
			rows:    []entity.RowSettings{{Columns: 1, Gap: "xxl"}},
			wantErr: "gap",
		},
		{
			name:    "неизвестная ширина блока",
			blocks:  []entity.Block{{Type: "text", Row: 0, Width: "huge"}},
			wantErr: "width",
		},
		{
			name:    "лишние настройки рядов",
			blocks:  []entity.Block{{Type: "text", Row: 0}},
			rows:    []entity.RowSettings{{Columns: 1}, {Columns: 1}, {Columns: 1}},
			wantErr: "rows",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
			wid := uuid.New()
			wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

			_, err := uc.UpdateBlocks(context.Background(), owner, wid, tc.blocks, tc.rows, time.Time{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			wr.AssertNotCalled(t, "UpdateBlocks", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// Оформление: шрифт заголовков, узор фона и «живость» — перечисления, иначе в
// CSS-класс уедет произвольная строка.
func TestLook_Validation(t *testing.T) {
	cases := []struct {
		name    string
		look    entity.Look
		wantErr string
	}{
		{name: "пусто — значения по умолчанию", look: entity.Look{}},
		{name: "всё задано", look: entity.Look{HeadingFont: "poster", Pattern: "stars", MainDreamLarge: true, ConfettiOnReserve: true, LiveTimer: true}},
		{name: "чужой шрифт", look: entity.Look{HeadingFont: "comic-sans"}, wantErr: "headingFont"},
		{name: "чужой узор", look: entity.Look{Pattern: "url(evil)"}, wantErr: "pattern"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
			wid := uuid.New()
			wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
			wr.On("UpdateMetadata", mock.Anything, wid, mock.Anything, mock.Anything).
				Return(entity.Wishlist{ID: wid, UserID: owner}, true, nil).Maybe()

			_, err := uc.Update(context.Background(), owner, wid,
				usecase.CreateWishlistInput{Title: "Т", Look: tc.look}, time.Time{})

			if tc.wantErr == "" {
				require.NoError(t, err)
				saved := wr.Calls[len(wr.Calls)-1].Arguments.Get(2).(entity.Wishlist)
				assert.Equal(t, tc.look, saved.Settings.Look, "оформление доходит до базы")
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// Секрет до даты в трёх режимах: «замок и таймер», «только замок», «ничего».
func TestGetByShortID_SecretModes(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	uc := newWishlistUC(wr, &mockminio.MockFileStorage{})

	wid := uuid.New()
	future := time.Now().Add(48 * time.Hour)
	data := json.RawMessage(`{"html":"сюрприз"}`)

	wr.On("GetByShortID", mock.Anything, "s").Return(entity.Wishlist{
		ID: wid,
		Blocks: []entity.Block{
			{ID: "timer", Type: "text", Row: 0, RevealAt: &future, SecretMode: "timer", SecretText: "Скоро", Data: data},
			{ID: "lock", Type: "text", Row: 1, RevealAt: &future, SecretMode: "lock", SecretText: "Секрет", Data: data},
			{ID: "hidden", Type: "text", Row: 2, RevealAt: &future, SecretMode: "hidden", Data: data},
			{ID: "legacy", Type: "text", Row: 3, RevealAt: &future, Data: data},
		},
	}, nil)

	w, err := uc.GetByShortID(context.Background(), "s", uuid.Nil)
	require.NoError(t, err)

	ids := make([]string, 0, len(w.Blocks))
	for _, b := range w.Blocks {
		ids = append(ids, b.ID)
	}
	assert.Equal(t, []string{"timer", "lock", "legacy"}, ids, "режим «ничего» прячет блок целиком")

	timer, lock, legacy := w.Blocks[0], w.Blocks[1], w.Blocks[2]
	assert.NotNil(t, timer.RevealAt)
	assert.Equal(t, "Скоро", timer.SecretText, "текст на замке нужен гостю")
	assert.JSONEq(t, `{}`, string(timer.Data))

	assert.Nil(t, lock.RevealAt, "«только замок» не выдаёт дату")
	assert.Equal(t, "lock", lock.SecretMode)
	assert.Equal(t, "Секрет", lock.SecretText)
	assert.JSONEq(t, `{}`, string(lock.Data))

	assert.NotNil(t, legacy.RevealAt, "старые секреты без режима — таймер")
}

func TestUpdateBlocks_SecretModeValidation(t *testing.T) {
	wr := &mockrepo.MockWishlistRepo{}
	uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.UpdateBlocks(context.Background(), owner, wid,
		[]entity.Block{{Type: "text", SecretMode: "fog"}}, nil, time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secretMode")

	long := make([]rune, 200)
	for i := range long {
		long[i] = 'я'
	}
	_, err = uc.UpdateBlocks(context.Background(), owner, wid,
		[]entity.Block{{Type: "text", SecretText: string(long)}}, nil, time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secretText")
}
