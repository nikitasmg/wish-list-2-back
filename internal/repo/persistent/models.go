package persistent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
)

// UserModel — GORM-модель для таблицы "users"
type UserModel struct {
	ID          uuid.UUID `gorm:"primaryKey"`
	Username    string    `gorm:"unique;not null"`
	Password    string    `gorm:"not null"`
	DisplayName string
	Avatar      string
}

func (UserModel) TableName() string { return "users" }

// WishlistModel — GORM-модель для таблицы "wishlists"
type WishlistModel struct {
	ID            uuid.UUID `gorm:"primaryKey"`
	Title         string    `gorm:"not null"`
	Description   string
	Cover         string
	UserID        uuid.UUID    `gorm:"not null"`
	Settings      SettingsJSON `gorm:"type:json"`
	Location      LocationJSON `gorm:"type:json"`
	PresentsCount uint
	ShortID       *string    `gorm:"uniqueIndex;column:short_id"`
	Blocks        BlocksJSON `gorm:"type:jsonb"`
	// Rows — настройки рядов формата v3 по индексу ряда.
	Rows RowsJSON `gorm:"type:jsonb"`
	// BlocksVersion: 1 — формат до редизайна, 2 — текущий. Существующие строки
	// получают 1 через default, новые конструкторы пишут 2 явно.
	BlocksVersion int        `gorm:"default:1"`
	EventDate     *time.Time `gorm:"column:event_date;index"`
	Occasion      string
	ViewsCount    uint      `gorm:"column:views_count;default:0"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func (WishlistModel) TableName() string { return "wishlists" }

// WishlistViewModel — один просмотр публичной страницы одним гостем.
// Таблица нужна только для дедупликации: без неё счётчик накручивается
// перезагрузкой страницы.
type WishlistViewModel struct {
	WishlistID uuid.UUID `gorm:"primaryKey;column:wishlist_id"`
	GuestID    uuid.UUID `gorm:"primaryKey;column:guest_id"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (WishlistViewModel) TableName() string { return "wishlist_views" }

// PresentModel — GORM-модель для таблицы "presents"
type PresentModel struct {
	ID                uuid.UUID `gorm:"primaryKey"`
	Title             string    `gorm:"not null"`
	Description       string
	Reserved          bool
	ReservedByGuest   *string `gorm:"column:reserved_by_guest"`
	Cover             string
	Link              string
	Price             *float64        `gorm:"type:decimal(10,2)"`
	Type              string          `gorm:"not null;default:'single'"`
	ParticipantsCount int             `gorm:"not null;default:0"`
	Images            StringSliceJSON `gorm:"type:jsonb"`
	Links             StringSliceJSON `gorm:"type:jsonb"`
	CreatedAt         time.Time       `gorm:"autoCreateTime"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime"`
	WishlistID        uuid.UUID       `gorm:"not null"`
}

func (PresentModel) TableName() string { return "presents" }

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

// TemplateModel — GORM model for "templates" table
type TemplateModel struct {
	ID         uuid.UUID    `gorm:"primaryKey"`
	UserID     uuid.UUID    `gorm:"not null;index"`
	Name       string       `gorm:"not null"`
	Settings   SettingsJSON `gorm:"type:json"`
	Blocks     BlocksJSON   `gorm:"type:jsonb"`
	Rows       RowsJSON     `gorm:"type:jsonb"`
	IsPublic   bool         `gorm:"not null;default:false;index"`
	LikesCount int          `gorm:"not null;default:0"`
	CreatedAt  time.Time    `gorm:"autoCreateTime;index"`
	UpdatedAt  time.Time    `gorm:"autoUpdateTime"`
}

func (TemplateModel) TableName() string { return "templates" }

// TemplateLikeModel — GORM model for "template_likes" table
type TemplateLikeModel struct {
	UserID     uuid.UUID `gorm:"primaryKey;column:user_id"`
	TemplateID uuid.UUID `gorm:"primaryKey;column:template_id"`
	CreatedAt  time.Time `gorm:"not null;autoCreateTime"`
}

func (TemplateLikeModel) TableName() string { return "template_likes" }

// SettingsJSON — JSON-тип для хранения настроек вишлиста
type SettingsJSON struct {
	ColorScheme          string            `json:"colorScheme"`
	ShowGiftAvailability bool              `json:"showGiftAvailability"`
	PresentsLayout       string            `json:"presentsLayout"`
	CustomScheme         *CustomSchemeJSON `json:"customScheme,omitempty"`
	HeadingFont          string            `json:"headingFont,omitempty"`
	Pattern              string            `json:"pattern,omitempty"`
	MainDreamLarge       bool              `json:"mainDreamLarge,omitempty"`
	ConfettiOnReserve    bool              `json:"confettiOnReserve,omitempty"`
	LiveTimer            bool              `json:"liveTimer,omitempty"`
}

// CustomSchemeJSON — «своя схема»: база и акцент, остальное выводит фронт.
type CustomSchemeJSON struct {
	Base   string `json:"base"`
	Accent string `json:"accent"`
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
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Row        int             `json:"row"`
	Col        int             `json:"col"`
	ColSpan    int             `json:"col_span"`
	View       string          `json:"view"`
	Caption    string          `json:"caption"`
	Title      string          `json:"title"`
	Hidden     bool            `json:"hidden"`
	RevealAt   *time.Time      `json:"reveal_at"`
	SecretMode string          `json:"secret_mode,omitempty"`
	SecretText string          `json:"secret_text,omitempty"`
	Width      string          `json:"width,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// RowsJSON — настройки рядов (jsonb). nil — у вишлиста нет настроек рядов,
// все ряды по умолчанию.
type RowsJSON []entity.RowSettings

func (r *RowsJSON) Scan(value interface{}) error {
	if value == nil {
		*r = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan RowsJSON")
	}
	return json.Unmarshal(bytes, r)
}

func (r RowsJSON) Value() (driver.Value, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(r)
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

// StringSliceJSON — JSONB-тип для хранения массива строк (картинки/ссылки)
type StringSliceJSON []string

func (s *StringSliceJSON) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringSliceJSON")
	}
	return json.Unmarshal(bytes, s)
}

func (s StringSliceJSON) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// LinksJSON remains an alias for callers of the earlier redesign.
type LinksJSON = StringSliceJSON
