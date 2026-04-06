package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/pkg/shortid"
)

const maxTemplatesPerUser = 50
const publicPageSize = 20

type templateUseCase struct {
	templateRepo repo.TemplateRepo
	wishlistRepo repo.WishlistRepo
}

func New(templateRepo repo.TemplateRepo, wishlistRepo repo.WishlistRepo) usecase.TemplateUseCase {
	return &templateUseCase{
		templateRepo: templateRepo,
		wishlistRepo: wishlistRepo,
	}
}

func (uc *templateUseCase) Create(ctx context.Context, userID uuid.UUID, input usecase.CreateTemplateInput) (entity.Template, error) {
	if input.Name == "" {
		return entity.Template{}, errors.New("name is required")
	}
	if len([]rune(input.Name)) > 200 {
		return entity.Template{}, errors.New("name exceeds 200 characters")
	}

	wishlist, err := uc.wishlistRepo.GetByID(ctx, input.WishlistID)
	if err != nil {
		return entity.Template{}, fmt.Errorf("wishlist not found: %w", err)
	}
	if wishlist.UserID != userID {
		return entity.Template{}, errors.New("forbidden")
	}

	count, err := uc.templateRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Template{}, fmt.Errorf("count templates: %w", err)
	}
	if count >= maxTemplatesPerUser {
		return entity.Template{}, errors.New("достигнут лимит шаблонов (50)")
	}

	// Strip block content — keep structure only
	strippedBlocks := make([]entity.Block, len(wishlist.Blocks))
	for i, b := range wishlist.Blocks {
		strippedBlocks[i] = entity.Block{
			Type:    b.Type,
			Row:     b.Row,
			Col:     b.Col,
			ColSpan: b.ColSpan,
			Data:    json.RawMessage("{}"),
		}
	}

	t := entity.Template{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     input.Name,
		Settings: wishlist.Settings,
		Blocks:   strippedBlocks,
		IsPublic: input.IsPublic,
	}

	if err := uc.templateRepo.Create(ctx, t); err != nil {
		return entity.Template{}, fmt.Errorf("create template: %w", err)
	}
	return t, nil
}

func (uc *templateUseCase) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Template, error) {
	return uc.templateRepo.GetAllByUserID(ctx, userID)
}

func (uc *templateUseCase) GetPublic(ctx context.Context, limit int, cursorStr string) ([]entity.TemplateWithAuthor, string, error) {
	if limit <= 0 || limit > publicPageSize {
		limit = publicPageSize
	}

	var cursor time.Time
	if cursorStr != "" {
		parsed, err := time.Parse(time.RFC3339Nano, cursorStr)
		if err == nil {
			cursor = parsed
		}
	}

	items, err := uc.templateRepo.GetPublic(ctx, limit, cursor)
	if err != nil {
		return nil, "", fmt.Errorf("get public templates: %w", err)
	}

	var nextCursor string
	if len(items) == limit {
		nextCursor = items[len(items)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	return items, nextCursor, nil
}

func (uc *templateUseCase) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input usecase.UpdateTemplateInput) (entity.Template, error) {
	t, err := uc.templateRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Template{}, fmt.Errorf("template not found: %w", err)
	}
	if t.UserID != userID {
		return entity.Template{}, errors.New("forbidden")
	}
	if input.Name == "" {
		return entity.Template{}, errors.New("name is required")
	}
	if len([]rune(input.Name)) > 200 {
		return entity.Template{}, errors.New("name exceeds 200 characters")
	}

	t.Name = input.Name
	t.IsPublic = input.IsPublic

	if err := uc.templateRepo.Update(ctx, t); err != nil {
		return entity.Template{}, fmt.Errorf("update template: %w", err)
	}
	return t, nil
}

func (uc *templateUseCase) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	t, err := uc.templateRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}
	if t.UserID != userID {
		return errors.New("forbidden")
	}
	return uc.templateRepo.Delete(ctx, id)
}

func (uc *templateUseCase) CreateWishlistFromTemplate(ctx context.Context, templateID uuid.UUID, userID uuid.UUID, title string) (entity.Wishlist, error) {
	if title == "" {
		return entity.Wishlist{}, errors.New("title is required")
	}

	t, err := uc.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("template not found: %w", err)
	}
	// Allow only public templates or user's own templates
	if !t.IsPublic && t.UserID != userID {
		return entity.Wishlist{}, errors.New("forbidden")
	}

	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}

	// Generate unique shortID — retry up to 5 times
	var sid string
	for i := 0; i < 5; i++ {
		s, err := shortid.Generate()
		if err != nil {
			return entity.Wishlist{}, fmt.Errorf("generate short id: %w", err)
		}
		if _, err := uc.wishlistRepo.GetByShortID(ctx, s); err != nil {
			sid = s
			break
		}
	}
	if sid == "" {
		return entity.Wishlist{}, errors.New("failed to generate unique short id")
	}

	// Copy blocks from template — they already have empty data
	blocks := make([]entity.Block, len(t.Blocks))
	copy(blocks, t.Blocks)

	w := entity.Wishlist{
		ID:       uuid.New(),
		UserID:   userID,
		ShortID:  sid,
		Title:    title,
		Settings: t.Settings,
		Blocks:   blocks,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create wishlist from template: %w", err)
	}
	return w, nil
}
