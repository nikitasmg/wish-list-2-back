package wishlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	minioPkg "main/pkg/minio"
	"main/pkg/shortid"
)

type wishlistUseCase struct {
	wishlistRepo repo.WishlistRepo
	fileStorage  minioPkg.FileStorage
}

func New(wishlistRepo repo.WishlistRepo, fileStorage minioPkg.FileStorage) usecase.WishlistUseCase {
	return &wishlistUseCase{
		wishlistRepo: wishlistRepo,
		fileStorage:  fileStorage,
	}
}

func (uc *wishlistUseCase) Create(ctx context.Context, userID uuid.UUID, input usecase.CreateWishlistInput) (entity.Wishlist, error) {
	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}
	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Wishlist{}, err
	}

	sid, err := uc.generateUniqueShortID(ctx)
	if err != nil {
		return entity.Wishlist{}, err
	}

	w := entity.Wishlist{
		ID:      uuid.New(),
		UserID:  userID,
		ShortID: sid,
		Title:   input.Title,
		Description: input.Description,
		Cover:   coverURL,
		Settings: entity.Settings{
			ColorScheme:          input.ColorScheme,
			ShowGiftAvailability: input.ShowGiftAvailability,
			PresentsLayout:       input.PresentsLayout,
		},
		Location: entity.Location{
			Name: input.LocationName,
			Link: input.LocationLink,
			Time: input.LocationTime,
		},
		PresentsCount: 0,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) CreateConstructor(ctx context.Context, userID uuid.UUID, input usecase.CreateConstructorInput) (entity.Wishlist, error) {
	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}
	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}

	if err := validateBlocks(input.Blocks); err != nil {
		return entity.Wishlist{}, err
	}

	sid, err := uc.generateUniqueShortID(ctx)
	if err != nil {
		return entity.Wishlist{}, err
	}

	w := entity.Wishlist{
		ID:      uuid.New(),
		UserID:  userID,
		ShortID: sid,
		Title:   input.Title,
		Description: input.Description,
		Cover:   input.CoverURL,
		Settings: entity.Settings{
			ColorScheme:          input.ColorScheme,
			ShowGiftAvailability: input.ShowGiftAvailability,
			PresentsLayout:       input.PresentsLayout,
		},
		Location: entity.Location{
			Name: input.LocationName,
			Link: input.LocationLink,
			Time: input.LocationTime,
		},
		Blocks:        input.Blocks,
		PresentsCount: 0,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create constructor wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	return uc.wishlistRepo.GetByID(ctx, id)
}

func (uc *wishlistUseCase) GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error) {
	return uc.wishlistRepo.GetByShortID(ctx, shortID)
}

func (uc *wishlistUseCase) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	return uc.wishlistRepo.GetAllByUserID(ctx, userID)
}

func (uc *wishlistUseCase) Update(ctx context.Context, id uuid.UUID, input usecase.CreateWishlistInput) (entity.Wishlist, error) {
	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}

	w, err := uc.wishlistRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
	}

	w.Title = input.Title
	w.Description = input.Description
	w.Settings = entity.Settings{
		ColorScheme:          input.ColorScheme,
		ShowGiftAvailability: input.ShowGiftAvailability,
		PresentsLayout:       input.PresentsLayout,
	}
	w.Location = entity.Location{
		Name: input.LocationName,
		Link: input.LocationLink,
		Time: input.LocationTime,
	}

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Wishlist{}, err
	}
	w.Cover = coverURL

	if err := uc.wishlistRepo.Update(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("update wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block) (entity.Wishlist, error) {
	if err := validateBlocks(blocks); err != nil {
		return entity.Wishlist{}, err
	}

	w, err := uc.wishlistRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
	}

	w.Blocks = blocks

	if err := uc.wishlistRepo.Update(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("update blocks: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.wishlistRepo.Delete(ctx, id)
}

// resolveCover — возвращает URL обложки: загружает файл в MinIO или возвращает URL as-is
func (uc *wishlistUseCase) resolveCover(data []byte, name, url string) (string, error) {
	if len(data) > 0 {
		uploaded, err := uc.fileStorage.Upload(name, data)
		if err != nil {
			return "", fmt.Errorf("upload cover: %w", err)
		}
		return uploaded, nil
	}
	return url, nil
}

// generateUniqueShortID — генерирует уникальный короткий ID с повторной попыткой при коллизии
func (uc *wishlistUseCase) generateUniqueShortID(ctx context.Context) (string, error) {
	for i := 0; i < 5; i++ {
		sid, err := shortid.Generate()
		if err != nil {
			return "", fmt.Errorf("generate short id: %w", err)
		}
		_, err = uc.wishlistRepo.GetByShortID(ctx, sid)
		if err != nil {
			// не найден — значит уникальный
			return sid, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique short id after 5 attempts")
}

func validateWishlistFields(title, description, locationName, locationLink, coverURL string) error {
	if len([]rune(title)) > usecase.MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len([]rune(description)) > usecase.MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", usecase.MaxDescriptionLen)
	}
	if len([]rune(locationName)) > usecase.MaxTitleLen {
		return fmt.Errorf("location name exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len(locationLink) > usecase.MaxURLLen {
		return fmt.Errorf("location link exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	if len(coverURL) > usecase.MaxURLLen {
		return fmt.Errorf("cover URL exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	return nil
}

// validateBlocks — проверяет типы блоков
func validateBlocks(blocks []entity.Block) error {
	if len(blocks) > usecase.MaxBlocksPerWishlist {
		return fmt.Errorf("too many blocks: max %d", usecase.MaxBlocksPerWishlist)
	}
	for i, b := range blocks {
		if !entity.ValidBlockTypes[b.Type] {
			return fmt.Errorf("block[%d]: unknown type %q", i, b.Type)
		}
		if b.ColSpan > 2 {
			return fmt.Errorf("block[%d]: colSpan %d exceeds maximum of 2", i, b.ColSpan)
		}
		if b.Row < 0 {
			return fmt.Errorf("block[%d]: row must be >= 0", i)
		}
		if b.Col < 0 || b.Col > 1 {
			return fmt.Errorf("block[%d]: col must be 0 or 1", i)
		}
		if len(b.Data) > usecase.MaxBlockDataSize {
			return fmt.Errorf("block[%d]: data too large (max %d bytes)", i, usecase.MaxBlockDataSize)
		}
		if err := validateBlockData(i, b.Type, b.Data); err != nil {
			return err
		}
	}
	return nil
}

// validateBlockData checks type-specific content constraints.
// b.Data is guaranteed non-nil at this point (nil-substituted to "{}" in HTTP layer).
func validateBlockData(idx int, blockType string, data json.RawMessage) error {
	switch blockType {
	case "text", "quote":
		var d struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil // malformed but not oversized — let DB store it, frontend owns schema
		}
		if len([]rune(d.Content)) > usecase.MaxBlockTextField {
			return fmt.Errorf("block[%d]: content exceeds maximum length of %d characters", idx, usecase.MaxBlockTextField)
		}

	case "checklist":
		var d struct {
			Items []struct {
				Text string `json:"text"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Items) > 100 {
			return fmt.Errorf("block[%d]: checklist exceeds maximum of 100 items", idx)
		}
		for j, item := range d.Items {
			if len([]rune(item.Text)) > 500 {
				return fmt.Errorf("block[%d]: checklist item[%d] text exceeds 500 characters", idx, j)
			}
		}

	case "image", "text_image", "video":
		var d struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.URL) > usecase.MaxURLLen {
			return fmt.Errorf("block[%d]: url exceeds maximum length of %d", idx, usecase.MaxURLLen)
		}

	case "gallery":
		var d struct {
			Images []string `json:"images"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Images) > 50 {
			return fmt.Errorf("block[%d]: gallery exceeds maximum of 50 images", idx)
		}
		for j, u := range d.Images {
			if len(u) > usecase.MaxURLLen {
				return fmt.Errorf("block[%d]: gallery image[%d] URL exceeds maximum length", idx, j)
			}
		}
	}
	return nil
}
