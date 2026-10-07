package persistent

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
)

type SantaRoomModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	OwnerID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Slug           string    `gorm:"not null;uniqueIndex"`
	Title          string    `gorm:"not null"`
	Budget         *int
	ExchangeDate   *time.Time `gorm:"type:date"`
	DrawAt         *time.Time
	Message        string `gorm:"not null;default:''"`
	Status         string `gorm:"not null;default:open"`
	DrawnAt        *time.Time
	LastRemindedAt *time.Time
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`

	// Связи нужны ради внешних ключей с каскадом при AutoMigrate; в запросах не используются.
	Participants []SantaParticipantModel `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE"`
	Assignments  []SantaAssignmentModel  `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE"`
}

func (SantaRoomModel) TableName() string { return "santa_rooms" }

type SantaParticipantModel struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	// Входит в два уникальных индекса: (room_id, user_id) и (room_id, email).
	RoomID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_santa_participant_user;uniqueIndex:idx_santa_participant_email"`
	// NULL у гостей: уникальность (room_id, user_id) их не задевает.
	UserID      *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_santa_participant_user"`
	Name        string     `gorm:"not null"`
	Wishes      string     `gorm:"not null;default:''"`
	WishlistURL string     `gorm:"column:wishlist_url;not null;default:''"`
	TokenHash   string     `gorm:"not null;uniqueIndex"`
	GiftReady   bool       `gorm:"not null;default:false"`
	Channel     string     `gorm:"not null;default:''"`
	// NULL, пока адреса нет: уникальность (room_id, email) пустых не задевает.
	Email           *string `gorm:"uniqueIndex:idx_santa_participant_email"`
	EmailVerifiedAt *time.Time
	TgChatID        *int64
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	Given    []SantaAssignmentModel `gorm:"foreignKey:GiverID;constraint:OnDelete:CASCADE"`
	Received []SantaAssignmentModel `gorm:"foreignKey:ReceiverID;constraint:OnDelete:CASCADE"`
	// Ради внешних ключей с каскадом; в запросах не используются.
	EmailCode     *SantaEmailCodeModel     `gorm:"foreignKey:ParticipantID;constraint:OnDelete:CASCADE"`
	TgLinks       []SantaTgLinkModel       `gorm:"foreignKey:ParticipantID;constraint:OnDelete:CASCADE"`
	Notifications []SantaNotificationModel `gorm:"foreignKey:ParticipantID;constraint:OnDelete:CASCADE"`
}

func (SantaParticipantModel) TableName() string { return "santa_participants" }

type SantaAssignmentModel struct {
	RoomID     uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex:idx_santa_receiver"`
	GiverID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReceiverID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_santa_receiver"`
}

func (SantaAssignmentModel) TableName() string { return "santa_assignments" }

func toSantaRoomModel(r entity.SantaRoom) SantaRoomModel {
	return SantaRoomModel{
		ID: r.ID, OwnerID: r.OwnerID, Slug: r.Slug, Title: r.Title, Budget: r.Budget,
		ExchangeDate: r.ExchangeDate, DrawAt: r.DrawAt, Message: r.Message,
		Status: string(r.Status), DrawnAt: r.DrawnAt, LastRemindedAt: r.LastRemindedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toSantaRoomEntity(m SantaRoomModel) entity.SantaRoom {
	return entity.SantaRoom{
		ID: m.ID, OwnerID: m.OwnerID, Slug: m.Slug, Title: m.Title, Budget: m.Budget,
		ExchangeDate: m.ExchangeDate, DrawAt: m.DrawAt, Message: m.Message,
		Status: entity.SantaRoomStatus(m.Status), DrawnAt: m.DrawnAt, LastRemindedAt: m.LastRemindedAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toSantaParticipantModel(p entity.SantaParticipant) SantaParticipantModel {
	var email *string
	if p.Email != "" {
		e := p.Email
		email = &e
	}
	return SantaParticipantModel{
		ID: p.ID, RoomID: p.RoomID, UserID: p.UserID, Name: p.Name, Wishes: p.Wishes,
		WishlistURL: p.WishlistURL, TokenHash: p.TokenHash, GiftReady: p.GiftReady,
		Channel: string(p.Channel), Email: email, EmailVerifiedAt: p.EmailVerifiedAt, TgChatID: p.TgChatID,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toSantaParticipantEntity(m SantaParticipantModel) entity.SantaParticipant {
	email := ""
	if m.Email != nil {
		email = *m.Email
	}
	return entity.SantaParticipant{
		ID: m.ID, RoomID: m.RoomID, UserID: m.UserID, Name: m.Name, Wishes: m.Wishes,
		WishlistURL: m.WishlistURL, TokenHash: m.TokenHash, GiftReady: m.GiftReady,
		Channel: entity.SantaChannel(m.Channel), Email: email, EmailVerifiedAt: m.EmailVerifiedAt, TgChatID: m.TgChatID,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type SantaEmailCodeModel struct {
	ParticipantID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CodeHash      string    `gorm:"not null"`
	ExpiresAt     time.Time `gorm:"not null"`
	Attempts      int       `gorm:"not null;default:0"`
	SentAt        time.Time `gorm:"not null"`
}

func (SantaEmailCodeModel) TableName() string { return "santa_email_codes" }

type SantaTgLinkModel struct {
	TokenHash     string    `gorm:"primaryKey"`
	ParticipantID uuid.UUID `gorm:"type:uuid;not null;index"`
	ExpiresAt     time.Time `gorm:"not null"`
}

func (SantaTgLinkModel) TableName() string { return "santa_tg_links" }

type SantaNotificationModel struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ParticipantID uuid.UUID `gorm:"type:uuid;not null;index"`
	Kind          string    `gorm:"not null"`
	// JSON-объект строкой: text, а не jsonb — payload пока не читается в SQL.
	Payload   string    `gorm:"type:text;not null;default:'{}'"`
	Status    string    `gorm:"not null;default:pending;index:idx_santa_notification_due,priority:1"`
	Attempts  int       `gorm:"not null;default:0"`
	NextTryAt time.Time `gorm:"not null;index:idx_santa_notification_due,priority:2"`
	LastError string    `gorm:"not null;default:''"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (SantaNotificationModel) TableName() string { return "santa_notifications" }

func toSantaNotificationModel(n entity.SantaNotification) SantaNotificationModel {
	payload := "{}"
	if len(n.Payload) > 0 {
		if b, err := json.Marshal(n.Payload); err == nil {
			payload = string(b)
		}
	}
	return SantaNotificationModel{
		ID: n.ID, ParticipantID: n.ParticipantID, Kind: string(n.Kind), Payload: payload,
		Status: string(n.Status), Attempts: n.Attempts, NextTryAt: n.NextTryAt,
		LastError: n.LastError, CreatedAt: n.CreatedAt,
	}
}

func toSantaNotificationEntity(m SantaNotificationModel) entity.SantaNotification {
	payload := map[string]string{}
	_ = json.Unmarshal([]byte(m.Payload), &payload)
	return entity.SantaNotification{
		ID: m.ID, ParticipantID: m.ParticipantID, Kind: entity.SantaNotificationKind(m.Kind), Payload: payload,
		Status: entity.SantaNotificationStatus(m.Status), Attempts: m.Attempts, NextTryAt: m.NextTryAt,
		LastError: m.LastError, CreatedAt: m.CreatedAt,
	}
}
