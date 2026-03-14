package repo

import (
	"context"
	"time"

	"main/internal/entity"

	"github.com/google/uuid"
)

type UserRepo interface {
	Create(ctx context.Context, user entity.User) error
	GetByUsername(ctx context.Context, username string) (entity.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.User, error)
}

type WishlistRepo interface {
	Create(ctx context.Context, wishlist entity.Wishlist) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	Update(ctx context.Context, wishlist entity.Wishlist) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementPresentsCount(ctx context.Context, id uuid.UUID) error
	DecrementPresentsCount(ctx context.Context, id uuid.UUID) error
}

type PresentRepo interface {
	Create(ctx context.Context, present entity.Present) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error)
	GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, present entity.Present) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ParseRateLimitRepo interface {
	// IncrementAndCheck atomically increments the counter for userID in the
	// current hour window and returns the new count.
	// Pass uuid.Nil for the global counter.
	IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error)
}

type PresentMetaRepo interface {
	Upsert(ctx context.Context, meta entity.PresentMeta) error
}
