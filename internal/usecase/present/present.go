package present

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/pkg/imagefile"
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

	if err := validateDescription(input.Description); err != nil {
		return entity.Present{}, err
	}

	links, err := normalizeLinks(input.Links)
	if err != nil {
		return entity.Present{}, err
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
		Links:       links,
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

	if err := validateDescription(input.Description); err != nil {
		return entity.Present{}, err
	}

	links, err := normalizeLinks(input.Links)
	if err != nil {
		return entity.Present{}, err
	}

	p.Title = input.Title
	p.Description = input.Description
	p.Links = links

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

func (uc *presentUseCase) Reserve(ctx context.Context, id, guestID uuid.UUID) error {
	reserved, err := uc.presentRepo.Reserve(ctx, id, guestID)
	if err != nil {
		return fmt.Errorf("reserve present: %w", err)
	}
	if reserved {
		return nil
	}

	// Условный апдейт не различает «нет такого подарка» и «уже заняли»,
	// поэтому причину выясняем отдельным чтением — только когда не повезло.
	if _, err := uc.presentRepo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("present not found: %w", err)
	}
	return errors.New("упс... подарок уже был забронирован, пожалуйста перезагрузите страницу")
}

func (uc *presentUseCase) Release(ctx context.Context, id, guestID uuid.UUID) error {
	released, err := uc.presentRepo.Release(ctx, id, guestID)
	if err != nil {
		return fmt.Errorf("release present: %w", err)
	}
	if released {
		return nil
	}

	if _, err := uc.presentRepo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("present not found: %w", err)
	}
	return errors.New("снять бронь может только тот, кто её поставил")
}

// resolveCover — параметр назван coverURL, а не url: пакет net/url теперь
// импортирован, и одноимённая переменная его бы перекрыла.
func (uc *presentUseCase) resolveCover(data []byte, name, coverURL string) (string, error) {
	if len(data) > 0 {
		if err := imagefile.Validate(data); err != nil {
			return "", err
		}
		uploaded, err := uc.fileStorage.Upload(name, data)
		if err != nil {
			return "", fmt.Errorf("upload cover: %w", err)
		}
		return uploaded, nil
	}
	return coverURL, nil
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

// validateDescription — лимит описания подарка. Считаем символы, а не байты:
// описания пишут по-русски, и в 500 байт помещается вдвое меньше букв, чем
// показывает счётчик в форме.
func validateDescription(description string) error {
	if utf8.RuneCountInString(description) > entity.MaxPresentDescriptionLen {
		return fmt.Errorf("описание длиннее %d символов", entity.MaxPresentDescriptionLen)
	}
	return nil
}

// normalizeLinks чистит список магазинов: выбрасывает пустые строки, режет по
// лимиту и проверяет, что это ссылки с http(s).
//
// Схема важна: без неё "ozon.ru" превратится в относительную ссылку и уведёт
// гостя на несуществующую страницу самого вишлиста.
func normalizeLinks(links []string) ([]string, error) {
	cleaned := make([]string, 0, len(links))
	for _, raw := range links {
		link := strings.TrimSpace(raw)
		if link == "" {
			continue
		}

		parsed, err := url.Parse(link)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, fmt.Errorf("ссылка %q должна начинаться с http:// или https://", link)
		}
		cleaned = append(cleaned, link)
	}

	if len(cleaned) > entity.MaxPresentLinks {
		return nil, fmt.Errorf("не больше %d ссылок на подарок", entity.MaxPresentLinks)
	}
	if len(cleaned) == 0 {
		return nil, nil
	}
	return cleaned, nil
}
