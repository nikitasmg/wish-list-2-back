package persistent

import (
	"encoding/json"

	"main/internal/entity"
)

// User

func toUserEntity(m UserModel) entity.User {
	return entity.User{
		ID:          m.ID,
		Username:    m.Username,
		Password:    m.Password,
		DisplayName: m.DisplayName,
		Avatar:      m.Avatar,
	}
}

func toUserModel(u entity.User) UserModel {
	return UserModel{
		ID:          u.ID,
		Username:    u.Username,
		Password:    u.Password,
		DisplayName: u.DisplayName,
		Avatar:      u.Avatar,
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
			blocks = append(blocks, entity.Block{
				ID:         b.ID,
				Type:       b.Type,
				Row:        b.Row,
				Col:        b.Col,
				ColSpan:    colSpan,
				View:       b.View,
				Caption:    b.Caption,
				Title:      b.Title,
				Hidden:     b.Hidden,
				RevealAt:   b.RevealAt,
				SecretMode: b.SecretMode,
				SecretText: b.SecretText,
				Width:      b.Width,
				Data:       b.Data,
			})
		}
	}

	var customScheme *entity.CustomScheme
	if m.Settings.CustomScheme != nil {
		customScheme = &entity.CustomScheme{
			Base:   m.Settings.CustomScheme.Base,
			Accent: m.Settings.CustomScheme.Accent,
		}
	}

	// Строки, созданные до появления версии, читаются как формат v1.
	blocksVersion := m.BlocksVersion
	if blocksVersion < entity.BlocksVersionLegacy {
		blocksVersion = entity.BlocksVersionLegacy
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
			PresentsLayout:       m.Settings.PresentsLayout,
			CustomScheme:         customScheme,
			Look: entity.Look{
				HeadingFont:       m.Settings.HeadingFont,
				Pattern:           m.Settings.Pattern,
				MainDreamLarge:    m.Settings.MainDreamLarge,
				ConfettiOnReserve: m.Settings.ConfettiOnReserve,
				LiveTimer:         m.Settings.LiveTimer,
			},
		},
		Location: entity.Location{
			Name: m.Location.Name,
			Link: m.Location.Link,
			Time: m.Location.Time,
		},
		PresentsCount: m.PresentsCount,
		ShortID:       shortID,
		Blocks:        blocks,
		Rows:          []entity.RowSettings(m.Rows),
		BlocksVersion: blocksVersion,
		EventDate:     m.EventDate,
		Occasion:      m.Occasion,
		TemplateName:  m.TemplateName,
		ViewsCount:    m.ViewsCount,
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

	blocks := toBlocksJSON(w.Blocks)

	var customScheme *CustomSchemeJSON
	if w.Settings.CustomScheme != nil {
		customScheme = &CustomSchemeJSON{
			Base:   w.Settings.CustomScheme.Base,
			Accent: w.Settings.CustomScheme.Accent,
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
			PresentsLayout:       w.Settings.PresentsLayout,
			CustomScheme:         customScheme,
			HeadingFont:          w.Settings.HeadingFont,
			Pattern:              w.Settings.Pattern,
			MainDreamLarge:       w.Settings.MainDreamLarge,
			ConfettiOnReserve:    w.Settings.ConfettiOnReserve,
			LiveTimer:            w.Settings.LiveTimer,
		},
		Location: LocationJSON{
			Name: w.Location.Name,
			Link: w.Location.Link,
			Time: w.Location.Time,
		},
		PresentsCount: w.PresentsCount,
		ShortID:       shortID,
		Blocks:        blocks,
		Rows:          RowsJSON(w.Rows),
		BlocksVersion: w.BlocksVersion,
		EventDate:     w.EventDate,
		Occasion:      w.Occasion,
		TemplateName:  w.TemplateName,
		ViewsCount:    w.ViewsCount,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

// Present

func toPresentEntity(m PresentModel) entity.Present {
	var reservedBy string
	if m.ReservedByGuest != nil {
		reservedBy = *m.ReservedByGuest
	}

	// Подарки, созданные до появления массива, читаются через старое поле.
	links := []string(m.Links)
	if len(links) == 0 && m.Link != "" {
		links = []string{m.Link}
	}
	var firstLink string
	if len(links) > 0 {
		firstLink = links[0]
	}

	return entity.Present{
		ID:                m.ID,
		Title:             m.Title,
		Description:       m.Description,
		Reserved:          m.Reserved,
		ReservedByGuest:   reservedBy,
		Cover:             m.Cover,
		Links:             links,
		Link:              firstLink,
		Price:             m.Price,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		WishlistID:        m.WishlistID,
		Type:              m.Type,
		ParticipantsCount: m.ParticipantsCount,
		Images:            []string(m.Images),
		ReservedByName:    m.ReservedByName,
		IsMain:            m.IsMain,
		SortOrder:         m.SortOrder,
		Gifted:            m.Gifted,
	}
}

func toPresentModel(p entity.Present) PresentModel {
	var reservedBy *string
	if p.ReservedByGuest != "" {
		s := p.ReservedByGuest
		reservedBy = &s
	}

	// Link остаётся синхронной с первым элементом, пока фронт не переедет.
	var firstLink string
	if len(p.Links) > 0 {
		firstLink = p.Links[0]
	}

	return PresentModel{
		ID:                p.ID,
		Title:             p.Title,
		Description:       p.Description,
		Reserved:          p.Reserved,
		ReservedByGuest:   reservedBy,
		Cover:             p.Cover,
		Links:             LinksJSON(p.Links),
		Link:              firstLink,
		Price:             p.Price,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		WishlistID:        p.WishlistID,
		Type:              p.Type,
		ParticipantsCount: p.ParticipantsCount,
		Images:            StringSliceJSON(p.Images),
		ReservedByName:    p.ReservedByName,
		IsMain:            p.IsMain,
		SortOrder:         p.SortOrder,
		Gifted:            p.Gifted,
	}
}

// Template

func toTemplateEntity(m TemplateModel) entity.Template {
	w := toWishlistEntity(WishlistModel{Blocks: m.Blocks, Rows: m.Rows, Settings: m.Settings})
	return entity.Template{ID: m.ID, UserID: m.UserID, Name: m.Name, Settings: w.Settings, Blocks: w.Blocks, Rows: w.Rows, IsPublic: m.IsPublic, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func toTemplateModel(t entity.Template) TemplateModel {
	w := toWishlistModel(entity.Wishlist{Blocks: t.Blocks, Rows: t.Rows, Settings: t.Settings})
	return TemplateModel{ID: t.ID, UserID: t.UserID, Name: t.Name, Settings: w.Settings, Blocks: w.Blocks, Rows: w.Rows, IsPublic: t.IsPublic, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

// toBlocksJSON — блоки в вид для jsonb. Вынесено из toWishlistModel, потому что
// UpdateBlocks пишет одну колонку и целая модель ему не нужна.
func toBlocksJSON(blocks []entity.Block) BlocksJSON {
	if blocks == nil {
		return nil
	}

	out := make(BlocksJSON, 0, len(blocks))
	for _, b := range blocks {
		data := b.Data
		if data == nil {
			data = json.RawMessage("{}")
		}
		colSpan := b.ColSpan
		if colSpan < 1 {
			colSpan = 1
		}
		out = append(out, blockJSON{
			ID:         b.ID,
			Type:       b.Type,
			Row:        b.Row,
			Col:        b.Col,
			ColSpan:    colSpan,
			View:       b.View,
			Caption:    b.Caption,
			Title:      b.Title,
			Hidden:     b.Hidden,
			RevealAt:   b.RevealAt,
			SecretMode: b.SecretMode,
			SecretText: b.SecretText,
			Width:      b.Width,
			Data:       data,
		})
	}
	return out
}
