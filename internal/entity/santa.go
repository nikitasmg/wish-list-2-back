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
	// DrawAt — когда планировщик проведёт жеребьёвку сам; nil — только вручную.
	DrawAt  *time.Time      `json:"drawAt"`
	Message string          `json:"message"`
	Status  SantaRoomStatus `json:"status"`
	DrawnAt *time.Time      `json:"drawnAt"`
	// DrawFailedAt — жеребьёвка по DrawAt не прошла: готовых меньше трёх.
	// Сбрасывается новым временем жеребьёвки и самой жеребьёвкой.
	DrawFailedAt *time.Time `json:"drawFailedAt"`
	// LastRemindedAt — когда организатор последний раз нажал «Напомнить».
	LastRemindedAt *time.Time `json:"lastRemindedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
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
	// Channel — куда приходят уведомления; пусто — канал не выбран.
	Channel SantaChannel
	// Email — подтверждённый адрес в нижнем регистре; пусто — адреса нет.
	Email string
	// PendingEmail — новый адрес, ждущий кода. Пока он не подтверждён, письма
	// идут на Email, а готовность не меняется.
	PendingEmail    string
	EmailVerifiedAt *time.Time
	TgChatID        *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SantaAssignment — «GiverID дарит ReceiverID».
type SantaAssignment struct {
	RoomID     uuid.UUID
	GiverID    uuid.UUID
	ReceiverID uuid.UUID
}

// SantaChannel — куда участнику приходят уведомления.
type SantaChannel string

const (
	SantaChannelNone     SantaChannel = ""
	SantaChannelEmail    SantaChannel = "email"
	SantaChannelTelegram SantaChannel = "telegram"
)

// Ready — канал подтверждён: только такие участники попадают в жеребьёвку.
// То же условие в SQL — santaReadySQL в репозитории; меняются вместе.
func (p SantaParticipant) Ready() bool {
	switch p.Channel {
	case SantaChannelEmail:
		return p.Email != "" && p.EmailVerifiedAt != nil
	case SantaChannelTelegram:
		return p.TgChatID != nil
	}
	return false
}

// SantaEmailCode — код подтверждения почты. Сам код не хранится, только хэш.
type SantaEmailCode struct {
	ParticipantID uuid.UUID
	CodeHash      string
	ExpiresAt     time.Time
	Attempts      int
	SentAt        time.Time
}

// SantaTgLink — одноразовая ссылка t.me/<бот>?start=<токен>; хранится хэш токена.
type SantaTgLink struct {
	TokenHash     string
	ParticipantID uuid.UUID
	ExpiresAt     time.Time
}

type SantaNotificationKind string

const (
	// SantaNotifyWelcome — канал подтверждён.
	SantaNotifyWelcome SantaNotificationKind = "welcome"
	// SantaNotifyDrawn — жеребьёвка или перезапуск: кому дарить.
	SantaNotifyDrawn SantaNotificationKind = "drawn"
	// SantaNotifyReminderFill — организатор просит заполнить пожелания.
	SantaNotifyReminderFill SantaNotificationKind = "reminder_fill"
	// SantaNotifyWishesUpdated — подопечный поменял пожелания после жеребьёвки.
	SantaNotifyWishesUpdated SantaNotificationKind = "wishes_updated"
	// SantaNotifyDrawFailed — организатору: жеребьёвка по расписанию не прошла.
	SantaNotifyDrawFailed SantaNotificationKind = "draw_failed"
)

type SantaNotificationStatus string

const (
	SantaNotificationPending SantaNotificationStatus = "pending"
	SantaNotificationSent    SantaNotificationStatus = "sent"
	SantaNotificationFailed  SantaNotificationStatus = "failed"
)

// SantaNotification — запись outbox. Текст собирается при отправке из
// текущего состояния базы, поэтому Payload пока пустой — место на будущее.
type SantaNotification struct {
	ID            uuid.UUID
	ParticipantID uuid.UUID
	Kind          SantaNotificationKind
	Payload       map[string]string
	Status        SantaNotificationStatus
	Attempts      int
	NextTryAt     time.Time
	LastError     string
	CreatedAt     time.Time
}

// NewSantaNotification — уведомление в очередь «отправить сейчас».
func NewSantaNotification(participantID uuid.UUID, kind SantaNotificationKind, now time.Time) SantaNotification {
	return SantaNotification{
		ID: uuid.New(), ParticipantID: participantID, Kind: kind, Payload: map[string]string{},
		Status: SantaNotificationPending, NextTryAt: now, CreatedAt: now,
	}
}
