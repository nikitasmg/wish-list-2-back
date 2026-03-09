package entity

import (
	"time"

	"github.com/google/uuid"
)

type Settings struct {
	ColorScheme          string
	ShowGiftAvailability bool
}

type Location struct {
	Name string
	Link string
	Time time.Time
}

type Wishlist struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Cover         string
	UserID        uuid.UUID
	Settings      Settings
	Location      Location
	PresentsCount uint
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
