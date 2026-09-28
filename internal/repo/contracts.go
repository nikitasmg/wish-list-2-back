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
	// ReservedCountsByUser — занятые подарки по каждому вишлисту пользователя,
	// одним запросом для карточек кабинета.
	ReservedCountsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uint, error)
	// RegisterView засчитывает просмотр, если этот гость его ещё не делал.
	RegisterView(ctx context.Context, wishlistID, guestID uuid.UUID) error
	// UpdateBlocks пишет блоки с проверкой версии. false — версия разошлась,
	// вишлист успели изменить в другом месте.
	UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block, blocksVersion int, expectedUpdatedAt time.Time) (bool, error)
}

type PresentRepo interface {
	Create(ctx context.Context, present entity.Present) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error)
	GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, present entity.Present) error
	Delete(ctx context.Context, id uuid.UUID) error
	// Reserve и Release — условные апдейты. Возвращают false, когда строка под
	// условие не подошла: подарок уже занят или бронь ставил другой гость.
	Reserve(ctx context.Context, id, guestID uuid.UUID) (bool, error)
	Release(ctx context.Context, id, guestID uuid.UUID) (bool, error)
}

// GuestDataRepo — всё, что оставляют гости: ответы, голоса, треки, записи.
//
// Один интерфейс на четыре блока, а не четыре отдельных: у них общая механика
// (привязка к блоку, дедупликация по гостю) и общий вызывающий.
type GuestDataRepo interface {
	UpsertRSVP(ctx context.Context, response entity.RSVPResponse) error
	ListRSVP(ctx context.Context, blockID string) ([]entity.RSVPResponse, error)

	UpsertPollVote(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, option int) error
	CountPollVotes(ctx context.Context, blockID string, guestID uuid.UUID) (map[int]int, *int, error)

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
