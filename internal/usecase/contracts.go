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
	Look                 entity.Look
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
	Look                 entity.Look
	Blocks               []entity.Block
	Rows                 []entity.RowSettings
}

// CreateFromSystemTemplateInput — создание вишлиста по готовому шаблону.
type CreateFromSystemTemplateInput struct {
	TemplateID string
	Title      string
	// Name — имя виновника праздника, подставляется вместо {name} в текстах.
	Name      string
	EventDate *time.Time
	// Age — крупная цифра на обложке «цифрой»; 0 — оставить из шаблона.
	Age int
	// Page — ответы опросника с телефона. Есть — страница собирается из
	// ответов, а тексты-примеры шаблона в вишлист не попадают. Нет — блоки
	// шаблона копируются как есть (экран создания на компьютере).
	Page *TemplatePage
}

// TemplatePage — что человек выбрал и написал в опроснике. JSON-теги здесь,
// а не в контроллере: структура приходит из запроса без преобразований.
type TemplatePage struct {
	// Blocks — отмеченные ключи каталога: about, place, program, dress,
	// contact, likes, stop, sizes, rsvp, playlist, guestbook. Обложка и
	// подарки есть всегда.
	Blocks  []string            `json:"blocks"`
	About   string              `json:"about"`
	Place   TemplatePlace       `json:"place"`
	Program []TemplateTimeEntry `json:"program"`
	Dress   TemplateDress       `json:"dress"`
	Contact TemplateContact     `json:"contact"`
	Likes   []string            `json:"likes"`
	Stop    []string            `json:"stop"`
	Sizes   TemplateSizes       `json:"sizes"`
}

type TemplatePlace struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Note    string `json:"note"`
}

type TemplateTimeEntry struct {
	T string `json:"t"`
	V string `json:"v"`
}

type TemplateDress struct {
	Colors []TemplateColor `json:"colors"`
	Note   string          `json:"note"`
}

type TemplateColor struct {
	Hex  string `json:"hex"`
	Name string `json:"name"`
}

type TemplateContact struct {
	Name string `json:"name"`
	// Way — телеграм (@ник или t.me/…) или телефон: в опроснике это одно поле.
	Way string `json:"way"`
}

type TemplateSizes struct {
	Clothes string `json:"clothes"`
	Shoes   string `json:"shoes"`
	Height  string `json:"height"`
	Ring    string `json:"ring"`
}

// CreatePresentInput — входные данные для создания/обновления подарка
type CreatePresentInput struct {
	Link        string
	Title       string
	Description string
	PriceStr    string
	CoverData   []byte
	CoverName   string
	CoverURL    string // URL картинки as-is (альтернатива CoverData)
	// Parser metadata (optional, populated after /parse call)
	Category    string
	Brand       string
	Source      string // "ozon" | "wildberries" | "yamarket" | "other"
	OriginalURL string
	Type        string   // "single" | "group" | "multi"; пусто => "single"
	Images      []string // галерея для multi
	Links       []string // несколько ссылок для multi
	IsMain      bool     // главная мечта
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
	// CreateFromSystemTemplate копирует блоки шаблона вместе с текстами-подсказками:
	// пользователь заменит их в конструкторе.
	CreateFromSystemTemplate(ctx context.Context, userID uuid.UUID, input CreateFromSystemTemplateInput) (entity.Wishlist, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	// GetByShortID — публичная страница: скрытые блоки вырезаются, нераскрытые
	// секреты отдаются без содержимого, просмотр засчитывается гостю.
	GetByShortID(ctx context.Context, shortID string, guestID uuid.UUID) (entity.Wishlist, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	// Update меняет только настройки и, как UpdateBlocks, проверяет версию:
	// иначе сохранение настроек затирало бы блоки, сохранённые в соседней
	// вкладке. Возвращает вишлист в том виде, в каком он лёг в базу.
	Update(ctx context.Context, userID, id uuid.UUID, input CreateWishlistInput, expectedUpdatedAt time.Time) (entity.Wishlist, error)
	// UpdateBlocks: expectedUpdatedAt — версия, которую держит клиент. Нулевое
	// время отключает проверку (старые клиенты без заголовка If-Match).
	// Расхождение версий возвращает ErrVersionConflict.
	UpdateBlocks(ctx context.Context, userID, id uuid.UUID, blocks []entity.Block, rows []entity.RowSettings, expectedUpdatedAt time.Time) (entity.Wishlist, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

// ErrVersionConflict — вишлист изменили в другом месте, пока клиент держал свою
// версию. Обработчик отдаёт 409 и актуальный вишлист, чтобы было что показать.
var ErrVersionConflict = errors.New("вишлист изменили в другой вкладке")

// ErrForbidden — вишлист принадлежит другому пользователю.
//
// JWT сам по себе этого не ловит: он говорит, кто пришёл, но не чей вишлист
// открыт. Без явной проверки любой залогиненный человек правил бы чужие
// страницы, зная только UUID из публичной ссылки.
var ErrForbidden = errors.New("это чужой вишлист")

// ErrClosed — приём ответов или голосов закрыт датой из настроек блока.
var ErrClosed = errors.New("приём ответов закрыт")

// PresentUseCase — бизнес-логика подарков
type PresentUseCase interface {
	Create(ctx context.Context, userID, wishlistID uuid.UUID, input CreatePresentInput) (entity.Present, error)
	GetByID(ctx context.Context, userID, id uuid.UUID) (entity.Present, error)
	// GetAllByWishlist: viewerID — залогиненный пользователь или uuid.Nil.
	// Владельцу не отдаётся имя забронировавшего гостя.
	GetAllByWishlist(ctx context.Context, wishlistID, viewerID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, userID, id uuid.UUID, input CreatePresentInput) (entity.Present, error)
	Delete(ctx context.Context, userID, wishlistID, id uuid.UUID) error
	// Reserve и Release принимают гостя из куки: бронь ставится от его имени,
	// и снять её может только он.
	// name — подпись гостя; пусто — анонимно.
	Reserve(ctx context.Context, id, guestID uuid.UUID, name string) error
	Release(ctx context.Context, id, guestID uuid.UUID) error
	// Reorder задаёт порядок подарков: ids — все подарки вишлиста сверху вниз.
	Reorder(ctx context.Context, userID, wishlistID uuid.UUID, ids []uuid.UUID) error
	// SetGifted — владелец отмечает, что подарок уже подарен.
	SetGifted(ctx context.Context, userID, id uuid.UUID, gifted bool) (entity.Present, error)
	Join(ctx context.Context, id uuid.UUID) error
	Leave(ctx context.Context, id uuid.UUID) error
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
	Answers  map[string]string
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

	// RSVPGuests — «Кто идёт», если организатор включил его в блоке.
	RSVPGuests(ctx context.Context, wishlistID uuid.UUID, blockID string) (entity.RSVPGuests, error)

	Vote(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, optionIDs []string) (entity.PollResults, error)
	// PollResults: viewerID — залогиненный пользователь или uuid.Nil; владелец
	// видит результаты всегда.
	PollResults(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID, viewerID uuid.UUID) (entity.PollResults, error)
	AddPollOption(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, text string) (entity.PollResults, error)
	OwnerSetPollOptionHidden(ctx context.Context, userID, optionID uuid.UUID, hidden bool) error

	SuggestTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) ([]entity.PlaylistTrack, error)
	Tracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error)
	ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) ([]entity.PlaylistTrack, error)

	AddGuestbookEntry(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input GuestbookInput) (entity.GuestbookEntry, error)
	Guestbook(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.GuestbookEntry, error)
	OwnerGuestbook(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) ([]entity.GuestbookEntry, error)
	OwnerSetGuestbookHidden(ctx context.Context, userID, entryID uuid.UUID, hidden bool) error
}

// CreateTemplateInput — data for creating a template from a wishlist
type CreateTemplateInput struct {
	WishlistID uuid.UUID
	Name       string
	IsPublic   bool
}

// UpdateTemplateInput — data for updating a template
type UpdateTemplateInput struct {
	Name     string
	IsPublic bool
}

// LikeResult — returned by Like/Unlike operations
type LikeResult struct {
	LikesCount int  `json:"likesCount"`
	LikedByMe  bool `json:"likedByMe"`
}

// TemplateUseCase — business logic for templates
type TemplateUseCase interface {
	Create(ctx context.Context, userID uuid.UUID, input CreateTemplateInput) (entity.Template, error)
	GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Template, error)
	GetPublic(ctx context.Context, limit, page int, userID *uuid.UUID) ([]entity.TemplateWithAuthor, bool, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, input UpdateTemplateInput) (entity.Template, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	CreateWishlistFromTemplate(ctx context.Context, templateID uuid.UUID, userID uuid.UUID, title string) (entity.Wishlist, error)
	Like(ctx context.Context, userID, templateID uuid.UUID) (LikeResult, error)
	Unlike(ctx context.Context, userID, templateID uuid.UUID) (LikeResult, error)
}
