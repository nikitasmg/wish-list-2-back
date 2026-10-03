package template

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

type templateStub struct {
	repo.TemplateRepo
	value entity.Template
}

func (r *templateStub) Create(_ context.Context, value entity.Template) error {
	r.value = value
	return nil
}
func (r *templateStub) GetByID(context.Context, uuid.UUID) (entity.Template, error) {
	return r.value, nil
}
func (r *templateStub) CountByUserID(context.Context, uuid.UUID) (int64, error) { return 0, nil }

type wishlistStub struct {
	repo.WishlistRepo
	source  entity.Wishlist
	created []entity.Wishlist
}

func (r *wishlistStub) GetByID(context.Context, uuid.UUID) (entity.Wishlist, error) {
	return r.source, nil
}
func (r *wishlistStub) GetByShortID(context.Context, string) (entity.Wishlist, error) {
	return entity.Wishlist{}, errors.New("not found")
}
func (r *wishlistStub) CountByUserID(context.Context, uuid.UUID) (int64, error) { return 0, nil }
func (r *wishlistStub) Create(_ context.Context, value entity.Wishlist) error {
	r.created = append(r.created, value)
	return nil
}

func TestTemplateRoundTripPreservesContentAndIsolatesInstances(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	reveal := time.Now().Add(time.Hour)
	wishlists := &wishlistStub{source: entity.Wishlist{
		ID: uuid.New(), UserID: owner,
		Settings: entity.Settings{ColorScheme: "custom", ShowGiftAvailability: true, PresentsLayout: "grid2", CustomScheme: &entity.CustomScheme{Base: "dark", Accent: "#123456"}},
	}}
	for _, kind := range []string{"rsvp", "poll", "playlist", "guestbook"} {
		wishlists.source.Blocks = append(wishlists.source.Blocks, entity.Block{ID: uuid.NewString(), Type: kind, Row: 1, Col: 1, ColSpan: 1, View: "view", Caption: "caption", Title: "title", Hidden: true, RevealAt: &reveal, Data: json.RawMessage(`{"value":"preserved"}`)})
	}
	templates := &templateStub{}
	uc := New(templates, wishlists)
	saved, err := uc.Create(ctx, owner, usecase.CreateTemplateInput{WishlistID: wishlists.source.ID, Name: "sample", IsPublic: true})
	require.NoError(t, err)
	require.Equal(t, wishlists.source.Settings, saved.Settings)
	seen := map[string]bool{}
	for _, block := range wishlists.source.Blocks {
		seen[block.ID] = true
	}
	for _, block := range saved.Blocks {
		require.False(t, seen[block.ID])
		seen[block.ID] = true
	}
	for i := 0; i < 2; i++ {
		created, err := uc.CreateWishlistFromTemplate(ctx, saved.ID, uuid.New(), "copy")
		require.NoError(t, err)
		require.Equal(t, entity.BlocksVersionCurrent, created.BlocksVersion)
		require.Equal(t, saved.Settings, created.Settings)
		for j, block := range created.Blocks {
			_, err := uuid.Parse(block.ID)
			require.NoError(t, err)
			require.False(t, seen[block.ID])
			seen[block.ID] = true
			expected := wishlists.source.Blocks[j]
			expected.ID = block.ID
			require.Equal(t, expected, block)
		}
	}
	require.Len(t, wishlists.created, 2)
	wishlists.created[0].Blocks[0].Data[0] = '['
	*wishlists.created[0].Blocks[0].RevealAt = time.Time{}
	wishlists.created[0].Settings.CustomScheme.Accent = "changed"
	require.Equal(t, saved.Blocks, templates.value.Blocks)
	require.Equal(t, saved.Blocks[0].Data, wishlists.created[1].Blocks[0].Data)
	require.Equal(t, saved.Blocks[0].RevealAt, wishlists.created[1].Blocks[0].RevealAt)
	require.Equal(t, saved.Settings, wishlists.created[1].Settings)
}

// Шаблон хранит раскладку рядов и оформление: без них вишлист по шаблону
// разъехался бы обратно в две колонки и потерял шрифт.
func TestTemplateCopiesRowsAndLook(t *testing.T) {
	ctx := context.Background()
	owner := uuid.New()
	rows := []entity.RowSettings{{Columns: 3, Ratio: "1:1:1"}}
	wishlists := &wishlistStub{source: entity.Wishlist{
		ID: uuid.New(), UserID: owner,
		Settings: entity.Settings{ColorScheme: "space", Look: entity.Look{HeadingFont: "poster", Pattern: "stars"}},
		Blocks:   []entity.Block{{ID: "a", Type: "text", Row: 0, Col: 0, ColSpan: 1}},
		Rows:     rows,
	}}
	uc := New(&templateStub{}, wishlists)

	saved, err := uc.Create(ctx, owner, usecase.CreateTemplateInput{WishlistID: wishlists.source.ID, Name: "t", IsPublic: true})
	require.NoError(t, err)
	require.Equal(t, rows, saved.Rows)
	require.Equal(t, "poster", saved.Settings.HeadingFont)

	saved.Rows[0].Columns = 1 // копия, а не ссылка на массив вишлиста
	require.Equal(t, 3, wishlists.source.Rows[0].Columns)

	created, err := uc.CreateWishlistFromTemplate(ctx, saved.ID, uuid.New(), "copy")
	require.NoError(t, err)
	require.Len(t, created.Rows, 1)
	require.Equal(t, "stars", created.Settings.Pattern)
}
