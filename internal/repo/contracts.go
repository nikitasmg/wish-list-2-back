package repo

import (
	"context"
	"errors"
	"time"

	"main/internal/entity"

	"github.com/google/uuid"
)

type UserRepo interface {
	Create(ctx context.Context, user entity.User) error
	GetByUsername(ctx context.Context, username string) (entity.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.User, error)
	Update(ctx context.Context, user entity.User) error
}

type WishlistRepo interface {
	Create(ctx context.Context, wishlist entity.Wishlist) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	// UpdateMetadata пишет только настройки и возвращает сохранённую версию.
	// Полного Save модели здесь нет намеренно: он затирал бы блоки снимком,
	// прочитанным до правки.
	UpdateMetadata(ctx context.Context, id uuid.UUID, wishlist entity.Wishlist, expectedUpdatedAt time.Time) (entity.Wishlist, bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementPresentsCount(ctx context.Context, id uuid.UUID) error
	DecrementPresentsCount(ctx context.Context, id uuid.UUID) error
	// ReservedCountsByUser — занятые подарки по каждому вишлисту пользователя,
	// одним запросом для карточек кабинета.
	ReservedCountsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uint, error)
	// RegisterView засчитывает просмотр, если этот гость его ещё не делал.
	RegisterView(ctx context.Context, wishlistID, guestID uuid.UUID) error
	// UpdateBlocks пишет блоки с проверкой версии. false — версия разошлась,
	// вишлист успели изменить в другом месте.
	// Ряды пишутся тем же запросом: их настройки адресуются индексом ряда и
	// без блоков смысла не имеют.
	UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block, rows []entity.RowSettings, blocksVersion int, expectedUpdatedAt time.Time) (bool, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type PresentRepo interface {
	Create(ctx context.Context, present entity.Present) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error)
	GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, present entity.Present) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error)
	// Reserve и Release — условные апдейты. Возвращают false, когда строка под
	// условие не подошла: подарок уже занят или бронь ставил другой гость.
	Reserve(ctx context.Context, id, guestID uuid.UUID, name string) (bool, error)
	Release(ctx context.Context, id, guestID uuid.UUID) (bool, error)
	// ClearMain снимает «главную мечту» со всех подарков вишлиста, кроме exceptID.
	ClearMain(ctx context.Context, wishlistID, exceptID uuid.UUID) error
	// Reorder проставляет sort_order по порядку ids в пределах вишлиста.
	Reorder(ctx context.Context, wishlistID uuid.UUID, ids []uuid.UUID) error
	SetGifted(ctx context.Context, id uuid.UUID, gifted bool) error
}

// GuestDataRepo — всё, что оставляют гости: ответы, голоса, треки, записи.
//
// Один интерфейс на четыре блока, а не четыре отдельных: у них общая механика
// (привязка к блоку, дедупликация по гостю) и общий вызывающий.
type GuestDataRepo interface {
	UpsertRSVP(ctx context.Context, response entity.RSVPResponse) error
	ListRSVP(ctx context.Context, blockID string) ([]entity.RSVPResponse, error)

	// ReplacePollChoices заменяет выбор гостя целиком: переголосование не
	// копит старые голоса.
	ReplacePollChoices(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, optionIDs []string) error
	CountPollChoices(ctx context.Context, blockID string, guestID uuid.UUID) (map[string]int, []string, error)
	CreatePollOption(ctx context.Context, option entity.PollGuestOption) error
	CountPollOptionsByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	ListPollOptions(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.PollOption, error)
	PollOptionWishlist(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error)
	SetPollOptionHidden(ctx context.Context, optionID uuid.UUID, hidden bool) error

	CountTracksByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	CreateTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) (uuid.UUID, error)
	ListTracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error)
	TrackBlockID(ctx context.Context, trackID uuid.UUID) (uuid.UUID, string, error)
	ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) (bool, error)

	CountGuestbookByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	CreateGuestbookEntry(ctx context.Context, entry entity.GuestbookEntry, wishlistID uuid.UUID, blockID string, guestID uuid.UUID) error
	ListGuestbook(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.GuestbookEntry, error)
	GuestbookEntryWishlist(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error)
	SetGuestbookHidden(ctx context.Context, entryID uuid.UUID, hidden bool) error
}

type PresentMetaRepo interface {
	Upsert(ctx context.Context, meta entity.PresentMeta) error
}

type TemplateRepo interface {
	Create(ctx context.Context, template entity.Template) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Template, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Template, error)
	GetPublic(ctx context.Context, limit, offset int, userID uuid.UUID) ([]entity.TemplateWithAuthor, error)
	Update(ctx context.Context, template entity.Template) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	Like(ctx context.Context, userID, templateID uuid.UUID) (int, error)
	Unlike(ctx context.Context, userID, templateID uuid.UUID) (int, error)
}

// ErrNotFound — записи нет. Использование отличает по нему 404 от сбоя базы,
// не зная про gorm. Пока его возвращает только SantaRepo.
var ErrNotFound = errors.New("not found")

// ErrStatusMismatch — комната не в том статусе, которого ждал вызов.
var ErrStatusMismatch = errors.New("status mismatch")

type SantaRepo interface {
	CreateRoom(ctx context.Context, room entity.SantaRoom) error
	GetRoomByID(ctx context.Context, id uuid.UUID) (entity.SantaRoom, error)
	GetRoomBySlug(ctx context.Context, slug string) (entity.SantaRoom, error)
	// ListRoomsByUser — комнаты, где пользователь владелец или участник, новые сверху.
	ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SantaRoom, error)
	UpdateRoom(ctx context.Context, room entity.SantaRoom) error
	// DeleteRoom удаляет комнату вместе с участниками и парами.
	DeleteRoom(ctx context.Context, id uuid.UUID) error

	CreateParticipant(ctx context.Context, p entity.SantaParticipant) error
	GetParticipant(ctx context.Context, id uuid.UUID) (entity.SantaParticipant, error)
	// GetParticipantByToken ищет только внутри комнаты: токен из другой
	// комнаты здесь «не найден».
	GetParticipantByToken(ctx context.Context, roomID uuid.UUID, tokenHash string) (entity.SantaParticipant, error)
	GetParticipantByUser(ctx context.Context, roomID, userID uuid.UUID) (entity.SantaParticipant, error)
	// ListParticipants — в порядке вступления.
	ListParticipants(ctx context.Context, roomID uuid.UUID) ([]entity.SantaParticipant, error)
	CountParticipants(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]int, error)
	UpdateParticipant(ctx context.Context, p entity.SantaParticipant) error
	DeleteParticipant(ctx context.Context, id uuid.UUID) error

	GetAssignment(ctx context.Context, roomID, giverID uuid.UUID) (entity.SantaAssignment, error)
	// Draw в одной транзакции: блокирует комнату, проверяет статус expected
	// (иначе ErrStatusMismatch), стирает старые пары, отдаёт build id
	// участников в порядке вступления, пишет пары и ставит status=drawn.
	// Ошибка build откатывает всё и возвращается как есть.
	Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func(ids []uuid.UUID) ([]entity.SantaAssignment, error)) error
}
