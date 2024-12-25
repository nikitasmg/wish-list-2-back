package model

import (
	"github.com/google/uuid"
	"time"
)

// Present представляет структуру подарка
type Present struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`                      // уникальный идентификатор подарка
	Title       string    `gorm:"not null" json:"title" validate:"required"` // название подарка
	Description string    `json:"description"`                               // описание подарка
	Reserved    bool      `json:"reserved"`                                  // статус резервирования
	Cover       string    `json:"cover"`
	Link        string    `gorm:"not null" json:"link" validate:"required"` // ссылка на обложку
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
	WishListID  uuid.UUID `gorm:"not null foreignKey" json:"wish_list_id"` // внешний ключ на WishList
}
