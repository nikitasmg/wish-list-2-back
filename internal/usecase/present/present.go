package present

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	minioPkg "main/pkg/minio"
)

type presentUseCase struct {
	presentRepo  repo.PresentRepo
	wishlistRepo repo.WishlistRepo
	fileStorage  minioPkg.FileStorage
}

func New(presentRepo repo.PresentRepo, wishlistRepo repo.WishlistRepo, fileStorage minioPkg.FileStorage) usecase.PresentUseCase {
	return &presentUseCase{
		presentRepo:  presentRepo,
		wishlistRepo: wishlistRepo,
		fileStorage:  fileStorage,
	}
}

func (uc *presentUseCase) Create(ctx context.Context, wishlistID uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	if _, err := uc.wishlistRepo.GetByID(ctx, wishlistID); err != nil {
		return entity.Present{}, errors.New("вишлист с таким ID не существует")
	}

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Present{}, err
	}

	price, err := parsePrice(input.PriceStr)
	if err != nil {
		return entity.Present{}, err
	}

	p := entity.Present{
		ID:          uuid.New(),
		WishlistID:  wishlistID,
		Title:       input.Title,
		Description: input.Description,
		Cover:       coverURL,
		Link:        input.Link,
		Price:       price,
		Reserved:    false,
	}

	if err := uc.presentRepo.Create(ctx, p); err != nil {
		return entity.Present{}, fmt.Errorf("create present: %w", err)
	}

	if err := uc.wishlistRepo.IncrementPresentsCount(ctx, wishlistID); err != nil {
		return entity.Present{}, fmt.Errorf("increment presents count: %w", err)
	}

	return p, nil
}

func (uc *presentUseCase) GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error) {
	return uc.presentRepo.GetByID(ctx, id)
}

func (uc *presentUseCase) GetAllByWishlist(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error) {
	return uc.presentRepo.GetAllByWishlistID(ctx, wishlistID)
}

func (uc *presentUseCase) Update(ctx context.Context, id uuid.UUID, input usecase.CreatePresentInput) (entity.Present, error) {
	p, err := uc.presentRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Present{}, fmt.Errorf("present not found: %w", err)
	}

	p.Title = input.Title
	p.Description = input.Description
	p.Link = input.Link

	price, err := parsePrice(input.PriceStr)
	if err != nil {
		return entity.Present{}, err
	}
	p.Price = price

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Present{}, err
	}
	p.Cover = coverURL

	if err := uc.presentRepo.Update(ctx, p); err != nil {
		return entity.Present{}, fmt.Errorf("update present: %w", err)
	}

	return p, nil
}

func (uc *presentUseCase) Delete(ctx context.Context, wishlistID, id uuid.UUID) error {
	if err := uc.presentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete present: %w", err)
	}
	if err := uc.wishlistRepo.DecrementPresentsCount(ctx, wishlistID); err != nil {
		return fmt.Errorf("decrement presents count: %w", err)
	}
	return nil
}

func (uc *presentUseCase) Reserve(ctx context.Context, id uuid.UUID) error {
	p, err := uc.presentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("present not found: %w", err)
	}
	if p.Reserved {
		return errors.New("упс... подарок уже был забронирован, пожалуйста перезагрузите страницу")
	}
	p.Reserved = true
	return uc.presentRepo.Update(ctx, p)
}

func (uc *presentUseCase) Release(ctx context.Context, id uuid.UUID) error {
	p, err := uc.presentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("present not found: %w", err)
	}
	p.Reserved = false
	return uc.presentRepo.Update(ctx, p)
}

func (uc *presentUseCase) resolveCover(data []byte, name, url string) (string, error) {
	if len(data) > 0 {
		uploaded, err := uc.fileStorage.Upload(name, data)
		if err != nil {
			return "", fmt.Errorf("upload cover: %w", err)
		}
		return uploaded, nil
	}
	return url, nil
}

func parsePrice(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	s = strings.Replace(s, ",", ".", 1)
	s = strings.ReplaceAll(s, " ", "")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, errors.New("неверный формат цены")
	}
	return &v, nil
}
