package persistent

import (
	"encoding/json"

	"main/internal/entity"
)

// User

func toUserEntity(m UserModel) entity.User {
	return entity.User{
		ID:       m.ID,
		Username: m.Username,
		Password: m.Password,
	}
}

func toUserModel(u entity.User) UserModel {
	return UserModel{
		ID:       u.ID,
		Username: u.Username,
		Password: u.Password,
	}
}

// Wishlist

func toWishlistEntity(m WishlistModel) entity.Wishlist {
	var shortID string
	if m.ShortID != nil {
		shortID = *m.ShortID
	}

	var blocks []entity.Block
	if m.Blocks != nil {
		blocks = make([]entity.Block, 0, len(m.Blocks))
		for _, b := range m.Blocks {
			colSpan := b.ColSpan
			if colSpan < 1 {
				colSpan = 1
			}
			rowSpan := b.RowSpan
			if rowSpan < 1 {
				rowSpan = 1
			}
			blocks = append(blocks, entity.Block{
				Type:           b.Type,
				Position:       b.Position,
				MobilePosition: b.MobilePosition,
				ColSpan:        colSpan,
				RowSpan:        rowSpan,
				Data:           b.Data,
			})
		}
	}

	return entity.Wishlist{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Cover:       m.Cover,
		UserID:      m.UserID,
		Settings: entity.Settings{
			ColorScheme:          m.Settings.ColorScheme,
			ShowGiftAvailability: m.Settings.ShowGiftAvailability,
		},
		Location: entity.Location{
			Name: m.Location.Name,
			Link: m.Location.Link,
			Time: m.Location.Time,
		},
		PresentsCount: m.PresentsCount,
		ShortID:       shortID,
		Blocks:        blocks,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

func toWishlistModel(w entity.Wishlist) WishlistModel {
	var shortID *string
	if w.ShortID != "" {
		s := w.ShortID
		shortID = &s
	}

	var blocks BlocksJSON
	if w.Blocks != nil {
		blocks = make(BlocksJSON, 0, len(w.Blocks))
		for _, b := range w.Blocks {
			data := b.Data
			if data == nil {
				data = json.RawMessage("{}")
			}
			colSpanW := b.ColSpan
			if colSpanW < 1 {
				colSpanW = 1
			}
			rowSpanW := b.RowSpan
			if rowSpanW < 1 {
				rowSpanW = 1
			}
			blocks = append(blocks, blockJSON{
				Type:           b.Type,
				Position:       b.Position,
				MobilePosition: b.MobilePosition,
				ColSpan:        colSpanW,
				RowSpan:        rowSpanW,
				Data:           data,
			})
		}
	}

	return WishlistModel{
		ID:          w.ID,
		Title:       w.Title,
		Description: w.Description,
		Cover:       w.Cover,
		UserID:      w.UserID,
		Settings: SettingsJSON{
			ColorScheme:          w.Settings.ColorScheme,
			ShowGiftAvailability: w.Settings.ShowGiftAvailability,
		},
		Location: LocationJSON{
			Name: w.Location.Name,
			Link: w.Location.Link,
			Time: w.Location.Time,
		},
		PresentsCount: w.PresentsCount,
		ShortID:       shortID,
		Blocks:        blocks,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

// Present

func toPresentEntity(m PresentModel) entity.Present {
	return entity.Present{
		ID:          m.ID,
		Title:       m.Title,
		Description: m.Description,
		Reserved:    m.Reserved,
		Cover:       m.Cover,
		Link:        m.Link,
		Price:       m.Price,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		WishlistID:  m.WishlistID,
	}
}

func toPresentModel(p entity.Present) PresentModel {
	return PresentModel{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		Reserved:    p.Reserved,
		Cover:       p.Cover,
		Link:        p.Link,
		Price:       p.Price,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		WishlistID:  p.WishlistID,
	}
}
