package wishlist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	minioPkg "main/pkg/minio"
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
	var coverURL string
	if len(input.CoverData) > 0 {
		url, err := uc.fileStorage.Upload(input.CoverName, input.CoverData)
		if err != nil {
			return entity.Wishlist{}, fmt.Errorf("upload cover: %w", err)
		}
		coverURL = url
	}

	w := entity.Wishlist{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		Cover:       coverURL,
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

func (uc *wishlistUseCase) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	return uc.wishlistRepo.GetByID(ctx, id)
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

	if len(input.CoverData) > 0 {
		url, err := uc.fileStorage.Upload(input.CoverName, input.CoverData)
		if err != nil {
			return entity.Wishlist{}, fmt.Errorf("upload cover: %w", err)
		}
		w.Cover = url
	}

	if err := uc.wishlistRepo.Update(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("update wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.wishlistRepo.Delete(ctx, id)
}
