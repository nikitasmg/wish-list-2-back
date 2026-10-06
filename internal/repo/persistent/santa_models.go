package persistent

import (
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
)

type SantaRoomModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	OwnerID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Slug         string    `gorm:"not null;uniqueIndex"`
	Title        string    `gorm:"not null"`
	Budget       *int
	ExchangeDate *time.Time `gorm:"type:date"`
	DrawAt       *time.Time
	Message      string `gorm:"not null;default:''"`
	Status       string `gorm:"not null;default:open"`
	DrawnAt      *time.Time
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (SantaRoomModel) TableName() string { return "santa_rooms" }

type SantaParticipantModel struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoomID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_santa_participant_user"`
	// NULL у гостей: уникальность (room_id, user_id) их не задевает.
	UserID      *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_santa_participant_user"`
	Name        string     `gorm:"not null"`
	Wishes      string     `gorm:"not null;default:''"`
	WishlistURL string     `gorm:"column:wishlist_url;not null;default:''"`
	TokenHash   string     `gorm:"not null;uniqueIndex"`
	GiftReady   bool       `gorm:"not null;default:false"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
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
		Status: string(r.Status), DrawnAt: r.DrawnAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toSantaRoomEntity(m SantaRoomModel) entity.SantaRoom {
	return entity.SantaRoom{
		ID: m.ID, OwnerID: m.OwnerID, Slug: m.Slug, Title: m.Title, Budget: m.Budget,
		ExchangeDate: m.ExchangeDate, DrawAt: m.DrawAt, Message: m.Message,
		Status: entity.SantaRoomStatus(m.Status), DrawnAt: m.DrawnAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toSantaParticipantModel(p entity.SantaParticipant) SantaParticipantModel {
	return SantaParticipantModel{
		ID: p.ID, RoomID: p.RoomID, UserID: p.UserID, Name: p.Name, Wishes: p.Wishes,
		WishlistURL: p.WishlistURL, TokenHash: p.TokenHash, GiftReady: p.GiftReady,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toSantaParticipantEntity(m SantaParticipantModel) entity.SantaParticipant {
	return entity.SantaParticipant{
		ID: m.ID, RoomID: m.RoomID, UserID: m.UserID, Name: m.Name, Wishes: m.Wishes,
		WishlistURL: m.WishlistURL, TokenHash: m.TokenHash, GiftReady: m.GiftReady,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
