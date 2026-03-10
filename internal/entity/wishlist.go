package entity

import (
	"encoding/json"
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

// Block — один блок конструктора вишлиста
type Block struct {
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	MobilePosition *int            `json:"mobile_position"`
	Data           json.RawMessage `json:"data"`
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
	ShortID       string  // короткий публичный ID вида abc-def-ghi (nullable в БД)
	Blocks        []Block // nil = простой вишлист
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
