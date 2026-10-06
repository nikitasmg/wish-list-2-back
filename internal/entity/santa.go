package entity

import (
	"time"

	"github.com/google/uuid"
)

// SantaRoomStatus — этап жизни комнаты Тайного Санты.
type SantaRoomStatus string

const (
	// SantaRoomOpen — идёт сбор участников.
	SantaRoomOpen SantaRoomStatus = "open"
	// SantaRoomDrawn — пары вытянуты, состав заморожен.
	SantaRoomDrawn SantaRoomStatus = "drawn"
)

type SantaRoom struct {
	ID      uuid.UUID `json:"id"`
	OwnerID uuid.UUID `json:"ownerId"`
	// Slug — короткий адрес комнаты в ссылке-приглашении.
	Slug  string `json:"slug"`
	Title string `json:"title"`
	// Budget в рублях; nil — без лимита.
	Budget       *int       `json:"budget"`
	ExchangeDate *time.Time `json:"exchangeDate"`
	// DrawAt хранится с этапа 1, срабатывает с этапа 3.
	DrawAt    *time.Time      `json:"drawAt"`
	Message   string          `json:"message"`
	Status    SantaRoomStatus `json:"status"`
	DrawnAt   *time.Time      `json:"drawnAt"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type SantaParticipant struct {
	ID     uuid.UUID
	RoomID uuid.UUID
	// UserID — у вошедших пользователей; у гостей nil.
	UserID      *uuid.UUID
	Name        string
	Wishes      string
	WishlistURL string
	// TokenHash — sha256 секрета из личной ссылки. Сам секрет не хранится.
	TokenHash string
	GiftReady bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SantaAssignment — «GiverID дарит ReceiverID».
type SantaAssignment struct {
	RoomID     uuid.UUID
	GiverID    uuid.UUID
	ReceiverID uuid.UUID
}
