package entity

import (
	"time"

	"github.com/google/uuid"
)

type PresentMeta struct {
	PresentID   uuid.UUID
	Source      string
	OriginalURL string
	Category    string
	Brand       string
	ParsedAt    time.Time
}
