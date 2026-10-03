package entity

// SystemTemplate — встроенная заготовка страницы под повод.
//
// Это не то же самое, что Template: тот принадлежит пользователю, его можно
// сохранить из своего вишлиста и опубликовать в галерее. Системные заготовки
// идут вместе с релизом, одинаковы для всех и показываются только на экране
// создания вишлиста — поэтому у них нет ни владельца, ни публичности, ни
// лайков, и лежат они рядом с кодом, а не в базе.
type SystemTemplate struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	ColorScheme string `json:"colorScheme"`
	// SampleTitle — название-пример («Тёме — семь!»), которое подставляется
	// в превью и в поле «Для кого», пока пользователь не ввёл своё.
	SampleTitle string `json:"sampleTitle"`
	// SampleName — имя-пример для {name} в текстах блоков («Тёма»). Есть только
	// у шаблонов, где имя встречается не только на обложке.
	SampleName string        `json:"sampleName,omitempty"`
	Occasion   string        `json:"occasion"`
	Blocks     []Block       `json:"blocks"`
	Rows       []RowSettings `json:"rows,omitempty"`
	// Look — шрифт и узор, с которыми заготовка нарисована в макете.
	Look Look `json:"look"`
}

// SystemTemplateCategories — фильтры на экране выбора, в порядке показа.
var SystemTemplateCategories = []struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}{
	{ID: "bday", Name: "Дни рождения"},
	{ID: "kids", Name: "Детские"},
	{ID: "wedding", Name: "Свадьба"},
	{ID: "jubilee", Name: "Юбилей"},
	{ID: "party", Name: "Вечеринки"},
}

// Виды блоков, на которые опираются системные заготовки.
//
// Значение View зависит от типа блока, поэтому набор перечислен здесь — это
// контракт между заготовками и компонентами фронта:
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
var _ = SystemTemplateCategories
