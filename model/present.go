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
	CreatedAt   time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null" json:"updatedAt"`
	WishlistID  uuid.UUID `gorm:"not null" json:"wishlistId"` // внешний ключ на Wishlist
}

type CreatePresent struct {
	Title       string `gorm:"not null" json:"title" form:"title" validate:"required"` // название подарка
	Description string `json:"description" form:"description"`                         // описание подарка
	Link        string `gorm:"not null" json:"link" form:"link" validate:"required"`   // ссылка на обложку
	File        []byte `gorm:"not null" json:"file" form:"file"`
}
