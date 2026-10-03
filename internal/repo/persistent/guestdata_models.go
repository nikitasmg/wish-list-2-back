package persistent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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
	// Answers — ответы на свои вопросы организатора, ключ — id вопроса.
	Answers AnswersJSON `gorm:"type:jsonb"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (RSVPResponseModel) TableName() string { return "rsvp_responses" }

// PollChoiceModel — выбранный гостем вариант. Строка на каждый вариант:
// голосование бывает с несколькими ответами. Вариант адресуется стабильным id,
// а не индексом — варианты перетаскивают и удаляют.
type PollChoiceModel struct {
	WishlistID uuid.UUID `gorm:"index;not null"`
	BlockID    string    `gorm:"primaryKey"`
	GuestID    uuid.UUID `gorm:"primaryKey"`
	OptionID   string    `gorm:"primaryKey"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (PollChoiceModel) TableName() string { return "poll_choices" }

// PollGuestOptionModel — вариант, который предложил гость. Владелец может его
// скрыть.
type PollGuestOptionModel struct {
	ID         uuid.UUID `gorm:"primaryKey"`
	WishlistID uuid.UUID `gorm:"index;not null"`
	BlockID    string    `gorm:"index;not null"`
	GuestID    uuid.UUID `gorm:"index;not null"`
	Text       string    `gorm:"not null"`
	Hidden     bool      `gorm:"not null;default:false"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (PollGuestOptionModel) TableName() string { return "poll_guest_options" }

// AnswersJSON — ответы гостя на вопросы организатора (jsonb).
type AnswersJSON map[string]string

func (a *AnswersJSON) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan AnswersJSON")
	}
	return json.Unmarshal(bytes, a)
}

func (a AnswersJSON) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// PollVoteModel — голос формата до v3: один на гостя, вариант по индексу.
// Таблица осталась ради переноса в poll_choices (BackfillPollChoices).
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
