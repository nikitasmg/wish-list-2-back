package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"mime/multipart"
	"time"
)

type Location struct {
	Name string    `form:"location[name]" json:"name"`
	Link string    `form:"location[link]" json:"link"`
	Time time.Time `form:"location[time]" json:"time"`
}

// Implement the scanner interfaces for Location
func (l *Location) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan Location")
	}
	return json.Unmarshal(bytes, &l)
}

func (l Location) Value() (driver.Value, error) {
	return json.Marshal(l)
}

type Settings struct {
	ColorScheme          string `form:"settings[colorScheme]" json:"colorScheme"`
	ShowGiftAvailability bool   `form:"settings[showGiftAvailability]" json:"showGiftAvailability"`
}

// Implement the scanner interfaces for Settings
func (s *Settings) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan Settings")
	}
	return json.Unmarshal(bytes, &s)
}

func (s Settings) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Wishlist представляет структуру списка желаемого
type Wishlist struct {
	ID            uuid.UUID `gorm:"primaryKey" json:"id"`                      // уникальный идентификатор списка желаемого
	Title         string    `gorm:"not null" json:"title" validate:"required"` // название списка
	Description   string    `json:"description"`                               // описание списка
	Cover         string    `json:"cover"`
	UserID        uuid.UUID `gorm:"not null" json:"userId"` // внешний ключ на пользователя
	Settings      Settings  `gorm:"type:json" json:"settings"`
	Location      Location  `gorm:"type:json" json:"location"`
	PresentsCount uint      `json:"presentsCount"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
	Present       []Present `gorm:"foreignKey:WishlistID;constraint:OnDelete:CASCADE" json:"-"`
}

type CreateWishlist struct {
	Title       string                `gorm:"not null" json:"title" validate:"required" form:"title"` // название списка
	Description string                `json:"description" form:"description"`                         // описание списка
	Settings    Settings              `gorm:"type:json" json:"settings" form:"settings"`
	Location    Location              `gorm:"type:json" json:"location" form:"location"`
	File        *multipart.FileHeader `json:"file" form:"file"`
}
