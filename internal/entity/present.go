package entity

import (
	"time"

	"github.com/google/uuid"
)

type Present struct {
	ID          uuid.UUID
	Title       string
	Description string
	Reserved    bool
	Cover       string
	Link        string
	Price       *float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	WishlistID  uuid.UUID
}
