package persistent

import (
	"time"

	"github.com/google/uuid"
)

// Модели для данных, которые оставляют гости.
//
// Уникальные составные индексы здесь — не оптимизация, а сама механика
// дедупликации: «один ответ на гостя» и «один голос в руки» держатся базой,
// а не проверкой в коде, которую обходят два одновременных запроса.

// RSVPResponseModel — ответ гостя. Один на пару «блок + гость»: повторная
// отправка обновляет свой ответ, а не создаёт второй.
type RSVPResponseModel struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	WishlistID uuid.UUID `gorm:"index;not null"`
	BlockID    string    `gorm:"uniqueIndex:idx_rsvp_block_guest;not null"`
	GuestID    uuid.UUID `gorm:"uniqueIndex:idx_rsvp_block_guest;not null"`

	Name     string
	Going    bool
	PlusOne  int
	Kids     int
	Menu     string
	Transfer bool
	Comment  string

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (RSVPResponseModel) TableName() string { return "rsvp_responses" }

// PollVoteModel — один голос в руки на блок голосования.
type PollVoteModel struct {
	WishlistID  uuid.UUID `gorm:"index;not null"`
	BlockID     string    `gorm:"primaryKey"`
	GuestID     uuid.UUID `gorm:"primaryKey"`
	OptionIndex int
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (PollVoteModel) TableName() string { return "poll_votes" }

// PlaylistTrackModel — трек, предложенный гостем.
type PlaylistTrackModel struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	WishlistID uuid.UUID `gorm:"index;not null"`
	BlockID    string    `gorm:"index;not null"`
	GuestID    uuid.UUID `gorm:"index;not null"`
	Title      string    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (PlaylistTrackModel) TableName() string { return "playlist_tracks" }

// PlaylistVoteModel — один голос в руки на трек.
type PlaylistVoteModel struct {
	TrackID   uuid.UUID `gorm:"primaryKey"`
	GuestID   uuid.UUID `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (PlaylistVoteModel) TableName() string { return "playlist_votes" }

// GuestbookEntryModel — запись в гостевой книге. Владелец может её скрыть,
// но не переписать: чужие слова остаются чужими.
type GuestbookEntryModel struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	WishlistID uuid.UUID `gorm:"index;not null"`
	BlockID    string    `gorm:"index;not null"`
	GuestID    uuid.UUID `gorm:"index;not null"`
	Name       string
	Text       string
	PhotoURL   string
	Hidden     bool
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (GuestbookEntryModel) TableName() string { return "guestbook_entries" }
