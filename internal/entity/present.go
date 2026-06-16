package entity

import (
	"time"

	"github.com/google/uuid"
)

type Present struct {
	ID                uuid.UUID `json:"id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	Reserved          bool      `json:"reserved"`
	Cover             string    `json:"cover"`
	Link              string    `json:"link"`
	Price             *float64  `json:"price"`
	Type              string    `json:"type"` // "single" | "group" | "multi"
	ParticipantsCount int       `json:"participantsCount"`
	Images            []string  `json:"images"`
	Links             []string  `json:"links"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	WishlistID        uuid.UUID `json:"wishlistId"`
}
