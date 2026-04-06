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

// CreateWishlistInput — входные данные для создания/обновления простого вишлиста
type CreateWishlistInput struct {
	Title                string
	Description          string
	CoverData            []byte
	CoverName            string
	CoverURL             string // URL картинки as-is (альтернатива CoverData)
	ColorScheme          string
	ShowGiftAvailability bool
	PresentsLayout       string
	LocationName         string
	LocationLink         string
	LocationTime         time.Time
}

// CreateConstructorInput — входные данные для создания/обновления вишлиста-конструктора
type CreateConstructorInput struct {
	Title                string
	Description          string
	CoverURL             string
	ColorScheme          string
	ShowGiftAvailability bool
	PresentsLayout       string
	LocationName         string
	LocationLink         string
	LocationTime         time.Time
	Blocks               []entity.Block
}

// CreatePresentInput — входные данные для создания/обновления подарка
type CreatePresentInput struct {
	Title       string
	Description string
	Link        string
	PriceStr    string
	CoverData   []byte
	CoverName   string
	CoverURL    string // URL картинки as-is (альтернатива CoverData)
	// Parser metadata (optional, populated after /parse call)
	Category    string
	Brand       string
	Source      string // "ozon" | "wildberries" | "yamarket" | "other"
	OriginalURL string
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

// UploadResult — результат загрузки файла
type UploadResult struct {
	URL string
}

// BulkUploadResult — результат массовой загрузки
type BulkUploadResult struct {
	Index int
	URL   string
}

// UpdateProfileInput — данные для обновления профиля
type UpdateProfileInput struct {
	DisplayName *string // nil = не менять
	Avatar      *string // nil = не менять
}

// UserUseCase — бизнес-логика пользователей
type UserUseCase interface {
	Register(ctx context.Context, username, password string) (AuthResult, error)
	Login(ctx context.Context, username, password string) (AuthResult, error)
	AuthenticateTelegram(ctx context.Context, input TelegramAuthInput) (AuthResult, error)
	GetMe(ctx context.Context, userID uuid.UUID) (entity.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (entity.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (entity.User, error)
}

// WishlistUseCase — бизнес-логика вишлистов
type WishlistUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, input CreateWishlistInput) (entity.Wishlist, error)
	CreateConstructor(ctx context.Context, userID uuid.UUID, input CreateConstructorInput) (entity.Wishlist, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	Update(ctx context.Context, id uuid.UUID, input CreateWishlistInput) (entity.Wishlist, error)
	UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block) (entity.Wishlist, error)
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

// UploadUseCase — загрузка файлов
type UploadUseCase interface {
	Upload(ctx context.Context, name string, data []byte) (UploadResult, error)
	BulkUpload(ctx context.Context, files []FileInput) ([]BulkUploadResult, error)
}

// ParseUseCase — парсинг ссылок с маркетплейсов
type ParseUseCase interface {
	Parse(ctx context.Context, userID uuid.UUID, rawURL string) (entity.ParseResult, error)
}

// FileInput — входной файл для загрузки
type FileInput struct {
	Index int
	Name  string
	Data  []byte
}
