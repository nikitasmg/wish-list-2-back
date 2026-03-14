package persistent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// UserModel — GORM-модель для таблицы "users"
type UserModel struct {
	ID       uuid.UUID `gorm:"primaryKey"`
	Username string    `gorm:"unique;not null"`
	Password string    `gorm:"not null"`
}

func (UserModel) TableName() string { return "users" }

// WishlistModel — GORM-модель для таблицы "wishlists"
type WishlistModel struct {
	ID            uuid.UUID    `gorm:"primaryKey"`
	Title         string       `gorm:"not null"`
	Description   string
	Cover         string
	UserID        uuid.UUID    `gorm:"not null"`
	Settings      SettingsJSON `gorm:"type:json"`
	Location      LocationJSON `gorm:"type:json"`
	PresentsCount uint
	ShortID       *string    `gorm:"uniqueIndex;column:short_id"`
	Blocks        BlocksJSON `gorm:"type:jsonb"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (WishlistModel) TableName() string { return "wishlists" }

// PresentModel — GORM-модель для таблицы "presents"
type PresentModel struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Title       string    `gorm:"not null"`
	Description string
	Reserved    bool
	Cover       string
	Link        string
	Price       *float64  `gorm:"type:decimal(10,2)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	WishlistID  uuid.UUID `gorm:"not null"`
}

func (PresentModel) TableName() string { return "presents" }

// ParseRateLimitModel — GORM-модель для таблицы "parse_rate_limits"
type ParseRateLimitModel struct {
	UserID      uuid.UUID `gorm:"primaryKey"`
	WindowStart time.Time `gorm:"not null"`
	Count       int       `gorm:"not null;default:0"`
}

func (ParseRateLimitModel) TableName() string { return "parse_rate_limits" }

// PresentMetaModel — GORM-модель для таблицы "present_meta"
type PresentMetaModel struct {
	PresentID   uuid.UUID `gorm:"primaryKey"`
	Source      string    `gorm:"not null"`
	OriginalURL string    `gorm:"not null"`
	Category    string
	Brand       string
	ParsedAt    time.Time `gorm:"not null"`
}

func (PresentMetaModel) TableName() string { return "present_meta" }

// SettingsJSON — JSON-тип для хранения настроек вишлиста
type SettingsJSON struct {
	ColorScheme          string `json:"colorScheme"`
	ShowGiftAvailability bool   `json:"showGiftAvailability"`
	PresentsLayout       string `json:"presentsLayout"`
}

func (s *SettingsJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan SettingsJSON")
	}
	return json.Unmarshal(bytes, s)
}

func (s SettingsJSON) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// LocationJSON — JSON-тип для хранения местоположения
type LocationJSON struct {
	Name string    `json:"name"`
	Link string    `json:"link"`
	Time time.Time `json:"time"`
}

func (l *LocationJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan LocationJSON")
	}
	return json.Unmarshal(bytes, l)
}

func (l LocationJSON) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// BlocksJSON — JSONB-тип для хранения массива блоков конструктора
type BlocksJSON []blockJSON

type blockJSON struct {
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	MobilePosition *int            `json:"mobile_position"`
	ColSpan        int             `json:"col_span"`
	RowSpan        int             `json:"row_span"`
	Data           json.RawMessage `json:"data"`
}

func (b *BlocksJSON) Scan(value interface{}) error {
	if value == nil {
		*b = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan BlocksJSON")
	}
	return json.Unmarshal(bytes, b)
}

func (b BlocksJSON) Value() (driver.Value, error) {
	if b == nil {
		return nil, nil
	}
	return json.Marshal(b)
}
