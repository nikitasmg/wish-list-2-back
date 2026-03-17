package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Settings struct {
	ColorScheme          string `json:"colorScheme"`
	ShowGiftAvailability bool   `json:"showGiftAvailability"`
	PresentsLayout       string `json:"presentsLayout"` // "list" | "grid3" | "grid2", default "list"
}

type Location struct {
	Name string    `json:"name"`
	Link string    `json:"link"`
	Time time.Time `json:"time"`
}

// Block — один блок конструктора вишлиста (координатная модель)
type Block struct {
	Type    string          `json:"type"`
	Row     int             `json:"row"`     // 0-based row in grid
	Col     int             `json:"col"`     // 0 = left, 1 = right
	ColSpan int             `json:"colSpan"` // 1 or 2
	Data    json.RawMessage `json:"data"`
}

// ValidBlockTypes — допустимые типы блоков (валидируются на бэке)
var ValidBlockTypes = map[string]bool{
	"text":         true,
	"text_image":   true,
	"image":        true,
	"date":         true,
	"location":     true,
	"color_scheme": true,
	"timing":       true,
	"agenda":       true,
	"gallery":      true,
	"quote":        true,
	"divider":      true,
	"contact":      true,
	"video":        true,
	"checklist":    true,
}

type Wishlist struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Cover         string    `json:"cover"`
	UserID        uuid.UUID `json:"userId"`
	Settings      Settings  `json:"settings"`
	Location      Location  `json:"location"`
	PresentsCount uint      `json:"presentsCount"`
	ShortID       string    `json:"shortId"`  // короткий публичный ID вида abc-def-ghi (nullable в БД)
	Blocks        []Block   `json:"blocks"`   // nil = простой вишлист
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
