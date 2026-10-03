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
	// Look — «Оформление» конструктора. Встроено, а не вложено: в JSON поля
	// лежат рядом с colorScheme, как их и читает фронт.
	Look
}

// Look — шрифт заголовков, узор фона и «живость» страницы.
type Look struct {
	HeadingFont       string `json:"headingFont"`       // accent | strict | soft | poster | elegant | classic; пусто — accent
	Pattern           string `json:"pattern"`           // none | stars | confetti | lines; пусто — none
	MainDreamLarge    bool   `json:"mainDreamLarge"`    // главная мечта крупно
	ConfettiOnReserve bool   `json:"confettiOnReserve"` // конфетти при брони
	LiveTimer         bool   `json:"liveTimer"`         // таймер до праздника тикает секундами
}

// Допустимые значения оформления. Пустая строка — значение по умолчанию у
// вишлистов, созданных до появления поля.
var (
	HeadingFonts = map[string]bool{"": true, "accent": true, "strict": true, "soft": true, "poster": true, "elegant": true, "classic": true}
	Patterns     = map[string]bool{"": true, "none": true, "stars": true, "confetti": true, "lines": true}
)

// CustomScheme —«своя схема» из конструктора: тёмная или светлая база плюс
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

// Block — один блок конструктора вишлиста (координатная модель).
//
// Раскладка — Row/Col/ColSpan, как в продакшене. Остальное добавил редизайн:
// View, Caption, Title, Hidden и RevealAt вынесены из Data отдельными полями,
// потому что по ним фильтруется публичная выдача, а копаться ради этого в
// произвольном JSON каждого типа блока пришлось бы на каждом запросе.
type Block struct {
	// ID — стабильный идентификатор блока. К нему привязаны ответы гостей,
	// голоса и треки: координаты для этого не годятся, они меняются при
	// каждой перестановке блоков.
	ID       string     `json:"id"`
	Type     string     `json:"type"`
	Row      int        `json:"row"`     // номер ряда, с нуля
	Col      int        `json:"col"`     // колонка в ряду, 0–2
	ColSpan  int        `json:"colSpan"` // ширина в колонках, 1–3
	View     string     `json:"view"`    // вариант отображения внутри типа
	Caption  string     `json:"caption"`
	Title    string     `json:"title"`
	Hidden   bool       `json:"hidden"`   // владелец скрыл блок со страницы
	RevealAt *time.Time `json:"revealAt"` // «секрет до даты»
	// SecretMode — что гость видит до RevealAt: timer (замок и таймер, по
	// умолчанию), lock (только замок, без даты), hidden (блока нет вовсе).
	SecretMode string `json:"secretMode"`
	// SecretText — надпись на замке. Лежит вне Data, потому что Data до даты
	// вырезается целиком.
	SecretText string `json:"secretText"`
	// Width — ширина содержимого одиночного блока: narrow | full; пусто — full.
	Width string          `json:"width"`
	Data  json.RawMessage `json:"data"`
}

// SecretModes — допустимые режимы секрета; пусто — timer.
var SecretModes = map[string]bool{"": true, "timer": true, "lock": true, "hidden": true}

// BlockWidths — допустимые ширины одиночного блока; пусто — full.
var BlockWidths = map[string]bool{"": true, "narrow": true, "full": true}

// RowSettings — настройки ряда. Ряд — это все блоки с одинаковым Row; запись
// в Wishlist.Rows берётся по индексу ряда. Индекс годится, потому что блоки и
// ряды пишутся одним запросом под одной версией и разъехаться не могут.
type RowSettings struct {
	Columns       int    `json:"columns"`       // 1–3; 0 — две, как в формате v2
	Ratio         string `json:"ratio"`         // 1:1 | 2:1 | 1:2 | 1:1:1; пусто — поровну
	Height        string `json:"height"`        // equal | auto; пусто — auto
	Gap           string `json:"gap"`           // s | m | l; пусто — m
	MobileReverse bool   `json:"mobileReverse"` // на телефоне правая колонка первой
}

// ColumnsOrDefault — ряд без настроек ведёт себя как сетка v2 из двух колонок.
func (r RowSettings) ColumnsOrDefault() int {
	if r.Columns == 0 {
		return 2
	}
	return r.Columns
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
// v3 — ряды до трёх колонок с настройками в Wishlist.Rows; v2 читается как v3
// без настроек рядов.
const (
	BlocksVersionLegacy  = 1
	BlocksVersionCurrent = 3
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
	// Rows — настройки рядов по индексу Row. Короче числа рядов — остальные по
	// умолчанию.
	Rows          []RowSettings `json:"rows"`
	BlocksVersion int           `json:"blocksVersion"`

	// EventDate и Occasion живут отдельно от Location: дата праздника нужна для
	// обратного отсчёта, календаря и сортировки в кабинете даже тогда, когда
	// место не указано.
	EventDate *time.Time `json:"eventDate"`
	Occasion  string     `json:"occasion"`
	// TemplateName — имя шаблона, из которого создан вишлист («ДР мальчика»).
	// Только для подписи в конструкторе: связи с шаблоном нет, он мог
	// измениться или исчезнуть.
	TemplateName string `json:"templateName"`

	// ReservedCount считается на выдаче списка, в таблице вишлистов не хранится.
	ReservedCount uint `json:"reservedCount"`
	ViewsCount    uint `json:"viewsCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
