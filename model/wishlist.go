package model

import (
	"github.com/google/uuid"
	"time"
)

// WishList представляет структуру списка желаемого
type WishList struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`                      // уникальный идентификатор списка желаемого
	Title       string    `gorm:"not null" json:"title" validate:"required"` // название списка
	Description string    `json:"description"`                               // описание списка
	Cover       string    `json:"cover"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`         // ссылка на обложку
	UserID      uuid.UUID `gorm:"not null foreignKey" json:"user_id"` // внешний ключ на пользователя
}
