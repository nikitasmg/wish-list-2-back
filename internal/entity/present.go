package entity

import (
	"time"

	"github.com/google/uuid"
)

type Present struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Reserved    bool      `json:"reserved"`
	Cover       string    `json:"cover"`
	Link        string    `json:"link"`
	Price       *float64  `json:"price"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	WishlistID  uuid.UUID `json:"wishlistId"`
}
