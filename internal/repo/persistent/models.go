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
	ID          uuid.UUID `gorm:"primaryKey"`
	Title       string    `gorm:"not null"`
	Description string
	Reserved    bool
	// ReservedByGuest — кука гостя из middleware.GuestIdentity. Нужна, чтобы снять
	// бронь мог только тот, кто её поставил; наружу это поле не выходит.
	ReservedByGuest *string `gorm:"column:reserved_by_guest"`
	Cover           string
	Link            string
	Links           LinksJSON `gorm:"type:jsonb"`
	Price           *float64  `gorm:"type:decimal(10,2)"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
	WishlistID      uuid.UUID `gorm:"not null"`
}

func (PresentModel) TableName() string { return "presents" }

// SettingsJSON — JSON-тип для хранения настроек вишлиста
type SettingsJSON struct {
	ColorScheme          string            `json:"colorScheme"`
	ShowGiftAvailability bool              `json:"showGiftAvailability"`
	PresentsLayout       string            `json:"presentsLayout"`
	CustomScheme         *CustomSchemeJSON `json:"customScheme,omitempty"`
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
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	MobilePosition *int            `json:"mobile_position"`
	ColSpan        int             `json:"col_span"`
	RowSpan        int             `json:"row_span"`
	View           string          `json:"view"`
	Caption        string          `json:"caption"`
	Title          string          `json:"title"`
	Hidden         bool            `json:"hidden"`
	RevealAt       *time.Time      `json:"reveal_at"`
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

// LinksJSON — ссылки на магазины у подарка.
type LinksJSON []string

func (l *LinksJSON) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan LinksJSON")
	}
	return json.Unmarshal(bytes, l)
}

func (l LinksJSON) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	return json.Marshal(l)
}
