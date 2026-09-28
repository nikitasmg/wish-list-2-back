package entity

// Template — заготовка страницы под повод: цветовая схема плюс набор блоков
// с текстами-подсказками.
//
// Шаблоны меняются редко и одинаковы для всех пользователей, поэтому лежат
// рядом с кодом, а не в базе: таблица потребовала бы админку, миграций и
// синхронизации между окружениями ради данных, которые правятся вместе с
// релизом.
type Template struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	ColorScheme string `json:"colorScheme"`
	// SampleTitle — название-пример («Тёме — семь!»), которое подставляется
	// в превью и в поле «Для кого», пока пользователь не ввёл своё.
	SampleTitle string  `json:"sampleTitle"`
	Occasion    string  `json:"occasion"`
	Blocks      []Block `json:"blocks"`
}

// TemplateCategories — фильтры на странице выбора, в порядке показа.
var TemplateCategories = []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}{
	{ID: "bday", Name: "Дни рождения"},
	{ID: "kids", Name: "Детские"},
	{ID: "wedding", Name: "Свадьба"},
	{ID: "jubilee", Name: "Юбилей"},
	{ID: "party", Name: "Вечеринки"},
}

// Виды блоков, на которые опираются шаблоны.
//
// Значение View зависит от типа блока, поэтому набор перечислен здесь — это
// контракт между шаблонами и компонентами фронта:
//
//	cover:    center | left | number | photo | circle | arch
//	list:     tags | pairs | tiles | schedule | timeline
//	media:    single | row
//	wishlist: cards | list | tiles
//
// Форма data по типам:
//
//	cover:        { number?: string, subtitle?: string }
//	text:         { html: string }
//	quote:        { text: string, author?: string }
//	media:        { images: string[], captions?: string[] }
//	video:        { url: string }
//	list:         { items: [{ k?: string, t?: string, v: string }], strike?: bool }
//	location:     { name: string, address?: string, link?: string }
//	color_scheme: { colors: string[] }
//	contact:      { name: string, role?: string, telegram?: string, phone?: string }
//	wishlist:     {}
//	rsvp:         { fields: string[] }   // plusOne, kids, menu, transfer, who
//	poll:         { question: string, options: string[] }
//	playlist:     { votes: bool }
//	guestbook:    { photos: bool }
var _ = TemplateCategories
