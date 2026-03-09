package usecase

import (
	"context"
	"time"

	"main/internal/entity"

	"github.com/google/uuid"
)

// AuthResult — результат аутентификации
type AuthResult struct {
	Token string
	User  entity.User
}

// CreateWishlistInput — входные данные для создания/обновления вишлиста
type CreateWishlistInput struct {
	Title       string
	Description string
	CoverData   []byte
	CoverName   string
	ColorScheme string
	ShowGiftAvailability bool
	LocationName string
	LocationLink string
	LocationTime time.Time
}

// CreatePresentInput — входные данные для создания/обновления подарка
type CreatePresentInput struct {
	Title       string
	Description string
	Link        string
	PriceStr    string
	CoverData   []byte
	CoverName   string
}

// TelegramAuthInput — входные данные для Telegram-авторизации
type TelegramAuthInput struct {
	ID        int64
	FirstName string
	LastName  string
	PhotoURL  string
	Username  string
	AuthDate  int64
	Hash      string
}

// UserUseCase — бизнес-логика пользователей
type UserUseCase interface {
	Register(ctx context.Context, username, password string) (AuthResult, error)
	Login(ctx context.Context, username, password string) (AuthResult, error)
	AuthenticateTelegram(ctx context.Context, input TelegramAuthInput) (AuthResult, error)
	GetMe(ctx context.Context, userID uuid.UUID) (entity.User, error)
}

// WishlistUseCase — бизнес-логика вишлистов
type WishlistUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, input CreateWishlistInput) (entity.Wishlist, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	Update(ctx context.Context, id uuid.UUID, input CreateWishlistInput) (entity.Wishlist, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PresentUseCase — бизнес-логика подарков
type PresentUseCase interface {
	Create(ctx context.Context, wishlistID uuid.UUID, input CreatePresentInput) (entity.Present, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error)
	GetAllByWishlist(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, id uuid.UUID, input CreatePresentInput) (entity.Present, error)
	Delete(ctx context.Context, wishlistID, id uuid.UUID) error
	Reserve(ctx context.Context, id uuid.UUID) error
	Release(ctx context.Context, id uuid.UUID) error
}
