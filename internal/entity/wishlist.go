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
	// CustomScheme заполняется только при ColorScheme == "custom". Храним базу и
	// акцент, а не все семь токенов: правила подбора ещё будут меняться, и
	// сохранённые вишлисты не должны застывать на старой палитре.
	CustomScheme *CustomScheme `json:"customScheme,omitempty"`
}

// CustomScheme — «своя схема» из конструктора: тёмная или светлая база плюс
// акцент, остальные цвета фронт выводит из этой пары.
type CustomScheme struct {
	Base   string `json:"base"`   // "dark" | "light"
	Accent string `json:"accent"` // hex-цвет вида #RRGGBB
}

type Location struct {
	Name string    `json:"name"`
	Link string    `json:"link"`
	Time time.Time `json:"time"`
}

// Block — один блок конструктора вишлиста.
//
// View, Caption, Title, Hidden и RevealAt вынесены из Data отдельными полями:
// по ним фильтруется публичная выдача, а копаться ради этого в произвольном
// JSON каждого типа блока пришлось бы на каждом запросе.
type Block struct {
	// ID — стабильный идентификатор блока. К нему привязаны ответы гостей,
	// голоса и треки: позиция для этого не годится, она меняется при каждой
	// перестановке блоков.
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Position       int        `json:"position"`
	MobilePosition *int       `json:"mobilePosition"`
	ColSpan        int        `json:"colSpan"` // 1 or 2, default 1
	RowSpan        int        `json:"rowSpan"` // 1–3, default 1
	View           string     `json:"view"`    // вариант отображения внутри типа
	Caption        string     `json:"caption"`
	Title          string     `json:"title"`
	Hidden         bool       `json:"hidden"`   // владелец скрыл блок со страницы
	RevealAt       *time.Time `json:"revealAt"` // «секрет до даты»

	Data json.RawMessage `json:"data"`
}

// IsSecret — блок ещё не раскрылся: гостю вместо содержимого показывается таймер.
func (b Block) IsSecret(now time.Time) bool {
	return b.RevealAt != nil && now.Before(*b.RevealAt)
}

// ValidBlockTypes — типы блоков, которые бэк принимает и умеет отдать.
var ValidBlockTypes = map[string]bool{
	// основа
	"cover":      true,
	"text":       true,
	"text_image": true,
	"quote":      true,
	"media":      true,
	"video":      true,
	"divider":    true,
	// список: один блок, пять видов (теги, пары, плитки, время, таймлайн)
	"list": true,
	// о празднике
	"date":         true,
	"location":     true,
	"color_scheme": true,
	"timing":       true,
	"contact":      true,
	// подарки
	"wishlist": true,
	// гости
	"rsvp":      true,
	"poll":      true,
	"playlist":  true,
	"guestbook": true,
	// legacy — читаются у старых вишлистов, но новые такими не собирают
	"image":     true,
	"gallery":   true,
	"agenda":    true,
	"checklist": true,
}

// LegacyBlockTypes — блоки формата v1. Их заменили "list" (agenda, checklist)
// и "media" (image, gallery). Старые вишлисты продолжают их отдавать, новые
// собирать из них нельзя.
var LegacyBlockTypes = map[string]bool{
	"image":     true,
	"gallery":   true,
	"agenda":    true,
	"checklist": true,
}

// Версии формата блоков. Обратная совместимость не приоритет: v1 отдаётся как
// есть, и фронт рисует у незнакомых блоков плашку «блок из старой версии».
const (
	BlocksVersionLegacy  = 1
	BlocksVersionCurrent = 2
)

type Wishlist struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Cover         string    `json:"cover"`
	UserID        uuid.UUID `json:"userId"`
	Settings      Settings  `json:"settings"`
	Location      Location  `json:"location"`
	PresentsCount uint      `json:"presentsCount"`
	ShortID       string    `json:"shortId"` // короткий публичный ID вида abc-def-ghi (nullable в БД)
	Blocks        []Block   `json:"blocks"`  // nil = простой вишлист
	BlocksVersion int       `json:"blocksVersion"`

	// EventDate и Occasion живут отдельно от Location: дата праздника нужна для
	// обратного отсчёта, календаря и сортировки в кабинете даже тогда, когда
	// место не указано.
	EventDate *time.Time `json:"eventDate"`
	Occasion  string     `json:"occasion"`

	// ReservedCount считается на выдаче списка, в таблице вишлистов не хранится.
	ReservedCount uint `json:"reservedCount"`
	ViewsCount    uint `json:"viewsCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
