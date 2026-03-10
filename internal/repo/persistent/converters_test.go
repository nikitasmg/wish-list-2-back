package persistent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"main/internal/entity"
)

func TestUserConverter_RoundTrip(t *testing.T) {
	u := entity.User{
		ID:       uuid.New(),
		Username: "alice",
		Password: "hashed",
	}
	got := toUserEntity(toUserModel(u))
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Username, got.Username)
	assert.Equal(t, u.Password, got.Password)
}

func TestWishlistConverter_RoundTrip_WithBlocks(t *testing.T) {
	mobilePos := 1
	w := entity.Wishlist{
		ID:          uuid.New(),
		Title:       "My Wishlist",
		Description: "desc",
		Cover:       "https://example.com/img.jpg",
		UserID:      uuid.New(),
		ShortID:     "abc-def-ghi",
		Settings: entity.Settings{
			ColorScheme:          "dark",
			ShowGiftAvailability: true,
		},
		Location: entity.Location{
			Name: "Кафе",
			Link: "https://maps.example.com",
			Time: time.Now().Truncate(time.Second),
		},
		PresentsCount: 3,
		Blocks: []entity.Block{
			{
				Type:           "text",
				Position:       0,
				MobilePosition: &mobilePos,
				ColSpan:        2,
				RowSpan:        3,
				Data:           json.RawMessage(`{"text":"hello"}`),
			},
		},
	}

	got := toWishlistEntity(toWishlistModel(w))
	assert.Equal(t, w.ID, got.ID)
	assert.Equal(t, w.Title, got.Title)
	assert.Equal(t, w.ShortID, got.ShortID)
	assert.Equal(t, w.Settings, got.Settings)
	assert.Len(t, got.Blocks, 1)
	assert.Equal(t, w.Blocks[0].Type, got.Blocks[0].Type)
	assert.Equal(t, w.Blocks[0].Position, got.Blocks[0].Position)
	assert.Equal(t, w.Blocks[0].MobilePosition, got.Blocks[0].MobilePosition)
	assert.Equal(t, w.Blocks[0].ColSpan, got.Blocks[0].ColSpan)
	assert.Equal(t, w.Blocks[0].RowSpan, got.Blocks[0].RowSpan)
}

func TestWishlistConverter_RoundTrip_NilBlocks(t *testing.T) {
	w := entity.Wishlist{
		ID:     uuid.New(),
		Title:  "Simple Wishlist",
		UserID: uuid.New(),
		Blocks: nil,
	}
	got := toWishlistEntity(toWishlistModel(w))
	assert.Empty(t, got.Blocks)
}

func TestWishlistConverter_Block_SpanDefaults(t *testing.T) {
	w := entity.Wishlist{
		ID:     uuid.New(),
		Title:  "Span Defaults Test",
		UserID: uuid.New(),
		Blocks: []entity.Block{
			{
				Type:     "text",
				Position: 0,
				ColSpan:  0,
				RowSpan:  0,
				Data:     json.RawMessage(`{"text":"test"}`),
			},
		},
	}

	got := toWishlistEntity(toWishlistModel(w))
	assert.Len(t, got.Blocks, 1)
	assert.Equal(t, 1, got.Blocks[0].ColSpan)
	assert.Equal(t, 1, got.Blocks[0].RowSpan)
}

func TestPresentConverter_RoundTrip(t *testing.T) {
	price := 1500.50
	p := entity.Present{
		ID:          uuid.New(),
		Title:       "Book",
		Description: "Go book",
		Reserved:    true,
		Cover:       "https://example.com/cover.jpg",
		Link:        "https://shop.example.com",
		Price:       &price,
		WishlistID:  uuid.New(),
	}
	got := toPresentEntity(toPresentModel(p))
	assert.Equal(t, p.ID, got.ID)
	assert.Equal(t, p.Title, got.Title)
	assert.Equal(t, p.Reserved, got.Reserved)
	assert.NotNil(t, got.Price)
	assert.InDelta(t, *p.Price, *got.Price, 0.001)
}
