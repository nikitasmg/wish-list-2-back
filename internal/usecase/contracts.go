package usecase

import (
	"context"
	"errors"
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
	CustomScheme         *entity.CustomScheme
	EventDate            *time.Time
	Occasion             string
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
	CustomScheme         *entity.CustomScheme
	EventDate            *time.Time
	Occasion             string
	Blocks               []entity.Block
}

// CreateFromTemplateInput — создание вишлиста по готовому шаблону.
type CreateFromTemplateInput struct {
	TemplateID string
	Title      string
	EventDate  *time.Time
}

// CreatePresentInput — входные данные для создания/обновления подарка
type CreatePresentInput struct {
	Title       string
	Description string
	Links       []string
	PriceStr    string
	CoverData   []byte
	CoverName   string
	CoverURL    string // URL картинки as-is (альтернатива CoverData)
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
	CreateConstructor(ctx context.Context, userID uuid.UUID, input CreateConstructorInput) (entity.Wishlist, error)
	// CreateFromTemplate копирует блоки шаблона вместе с текстами-подсказками:
	// пользователь заменит их в конструкторе.
	CreateFromTemplate(ctx context.Context, userID uuid.UUID, input CreateFromTemplateInput) (entity.Wishlist, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	// GetByShortID — публичная страница: скрытые блоки вырезаются, нераскрытые
	// секреты отдаются без содержимого, просмотр засчитывается гостю.
	GetByShortID(ctx context.Context, shortID string, guestID uuid.UUID) (entity.Wishlist, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	Update(ctx context.Context, userID, id uuid.UUID, input CreateWishlistInput) (entity.Wishlist, error)
	// UpdateBlocks: expectedUpdatedAt — версия, которую держит клиент. Нулевое
	// время отключает проверку (старые клиенты без заголовка If-Match).
	// Расхождение версий возвращает ErrBlocksConflict.
	UpdateBlocks(ctx context.Context, userID, id uuid.UUID, blocks []entity.Block, expectedUpdatedAt time.Time) (entity.Wishlist, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

// ErrBlocksConflict — вишлист изменили в другом месте, пока клиент держал свою
// версию. Обработчик отдаёт 409 и актуальный вишлист, чтобы было что показать.
var ErrBlocksConflict = errors.New("вишлист изменили в другой вкладке")

// ErrForbidden — вишлист принадлежит другому пользователю.
//
// JWT сам по себе этого не ловит: он говорит, кто пришёл, но не чей вишлист
// открыт. Без явной проверки любой залогиненный человек правил бы чужие
// страницы, зная только UUID из публичной ссылки.
var ErrForbidden = errors.New("это чужой вишлист")

// PresentUseCase — бизнес-логика подарков
type PresentUseCase interface {
	Create(ctx context.Context, userID, wishlistID uuid.UUID, input CreatePresentInput) (entity.Present, error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (entity.Present, error)
	GetAllByWishlist(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, userID, id uuid.UUID, input CreatePresentInput) (entity.Present, error)
	Delete(ctx context.Context, userID, wishlistID, id uuid.UUID) error
	// Reserve и Release принимают гостя из куки: бронь ставится от его имени,
	// и снять её может только он.
	Reserve(ctx context.Context, id, guestID uuid.UUID) error
	Release(ctx context.Context, id, guestID uuid.UUID) error
}

// UploadUseCase — загрузка файлов
type UploadUseCase interface {
	Upload(ctx context.Context, name string, data []byte) (UploadResult, error)
	BulkUpload(ctx context.Context, files []FileInput) ([]BulkUploadResult, error)
}

// FileInput — входной файл для загрузки
type FileInput struct {
	Index int
	Name  string
	Data  []byte
}

// RSVPInput — ответ гостя на приглашение.
type RSVPInput struct {
	Name     string
	Going    bool
	PlusOne  int
	Kids     int
	Menu     string
	Transfer bool
	Comment  string
}

// GuestbookInput — запись в гостевой книге.
type GuestbookInput struct {
	Name     string
	Text     string
	PhotoURL string
}

// GuestDataUseCase — данные, которые оставляют гости публичной страницы.
//
// Все гостевые методы принимают guestID из куки: он решает, чей ответ
// обновить, чей голос снять и какую запись пометить как свою. Методы со
// словом Owner требуют userID и проверяют, что вишлист принадлежит ему.
type GuestDataUseCase interface {
	SubmitRSVP(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input RSVPInput) (entity.RSVPResponse, error)
	MyRSVP(ctx context.Context, blockID string, guestID uuid.UUID) (*entity.RSVPResponse, error)
	OwnerRSVPSummary(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) (entity.RSVPSummary, error)

	Vote(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, option int) (entity.PollResults, error)
	PollResults(ctx context.Context, blockID string, guestID uuid.UUID) (entity.PollResults, error)

	SuggestTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) ([]entity.PlaylistTrack, error)
	Tracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error)
	ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) ([]entity.PlaylistTrack, error)

	AddGuestbookEntry(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input GuestbookInput) (entity.GuestbookEntry, error)
	Guestbook(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.GuestbookEntry, error)
	OwnerGuestbook(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) ([]entity.GuestbookEntry, error)
	OwnerSetGuestbookHidden(ctx context.Context, userID, entryID uuid.UUID, hidden bool) error
}
