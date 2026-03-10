package wishlist

import (
	"context"
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
	w, err := uc.wishlistRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
	}

	w.Title = input.Title
	w.Description = input.Description
	w.Settings = entity.Settings{
		ColorScheme:          input.ColorScheme,
		ShowGiftAvailability: input.ShowGiftAvailability,
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
	if coverURL != "" {
		w.Cover = coverURL
	}

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

// validateBlocks — проверяет типы блоков
func validateBlocks(blocks []entity.Block) error {
	for i, b := range blocks {
		if !entity.ValidBlockTypes[b.Type] {
			return fmt.Errorf("block[%d]: unknown type %q", i, b.Type)
		}
		if b.ColSpan > 2 {
			return fmt.Errorf("block[%d]: colSpan %d exceeds maximum of 2", i, b.ColSpan)
		}
		if b.RowSpan > 3 {
			return fmt.Errorf("block[%d]: rowSpan %d exceeds maximum of 3", i, b.RowSpan)
		}
	}
	return nil
}
