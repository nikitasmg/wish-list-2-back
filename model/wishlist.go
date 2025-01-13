package model

import (
	"github.com/google/uuid"
	"mime/multipart"
	"time"
)

// Wishlist представляет структуру списка желаемого
type Wishlist struct {
	ID          uuid.UUID `gorm:"primaryKey" json:"id"`                      // уникальный идентификатор списка желаемого
	Title       string    `gorm:"not null" json:"title" validate:"required"` // название списка
	Description string    `json:"description"`                               // описание списка
	Cover       string    `json:"cover"`
	CreatedAt   time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"not null" json:"updatedAt"` // ссылка на обложку
	UserID      uuid.UUID `gorm:"not null" json:"userId"`    // внешний ключ на пользователя
	ColorScheme string    `json:"colorScheme"`
	Present     []Present `gorm:"foreignKey:WishlistID;constraint:OnDelete:CASCADE" json:"-"`
}

type CreateWishlist struct {
	Title       string                `gorm:"not null" json:"title" validate:"required" form:"title"` // название списка
	Description string                `json:"description" form:"description"`                         // описание списка
	ColorScheme string                `json:"colorScheme" form:"colorScheme"`
	File        *multipart.FileHeader `json:"file" form:"file"`
}
