package wishlist

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"main/internal/entity"
	"main/internal/usecase"
)

// Страница из ответов опросника.
//
// Шаблон даёт оформление и раскладку, а содержимое блоков — только то, что
// человек выбрал и написал. Тексты-примеры («Бар «Подвал»», чужой стоп-лист)
// в вишлист не попадают: на странице они выглядели бы как его собственные.

// Ключи каталога блоков опросника. cover и gifts есть на странице всегда.
const (
	pageCover     = "cover"
	pageGifts     = "gifts"
	pageAbout     = "about"
	pagePlace     = "place"
	pageProgram   = "program"
	pageDress     = "dress"
	pageContact   = "contact"
	pageLikes     = "likes"
	pageStop      = "stop"
	pageSizes     = "sizes"
	pageRSVP      = "rsvp"
	pagePlaylist  = "playlist"
	pageGuestbook = "guestbook"
)

// pageCatalogOrder — порядок, в котором недостающие блоки встают на страницу.
var pageCatalogOrder = []string{pageLikes, pageStop, pageSizes, pageProgram, pagePlace, pageDress, pageContact, pageRSVP, pagePlaylist, pageGuestbook}

// pageBeforeGifts — блоки про подарки встают прямо над списком подарков.
var pageBeforeGifts = map[string]bool{pageLikes: true, pageStop: true, pageSizes: true}

var pageKnownKeys = map[string]bool{
	pageAbout: true, pagePlace: true, pageProgram: true, pageDress: true, pageContact: true,
	pageLikes: true, pageStop: true, pageSizes: true, pageRSVP: true, pagePlaylist: true, pageGuestbook: true,
}

type pageBlockSpec struct {
	Type, View, Caption, Title string
	// KeepData — data шаблона сохраняется: у гостевых блоков там настройки,
	// а не факты о празднике.
	KeepData    bool
	DefaultData map[string]any
}

var pageSpecs = map[string]pageBlockSpec{
	pagePlace:     {Type: "location", Caption: "Место", Title: "Где собираемся"},
	pageProgram:   {Type: "list", View: "schedule", Caption: "Программа", Title: "Как пройдёт праздник"},
	pageDress:     {Type: "color_scheme", View: "circles", Caption: "Дресс-код", Title: "Цвета праздника"},
	pageContact:   {Type: "contact", Caption: "Контакты", Title: "Остались вопросы?"},
	pageLikes:     {Type: "list", View: "tags", Caption: "Любимое", Title: "Если хочется угадать"},
	pageStop:      {Type: "list", View: "tags", Caption: "Стоп-лист", Title: "Пожалуйста, не дарите"},
	pageSizes:     {Type: "list", View: "tiles", Caption: "Размеры", Title: "Чтобы точно подошло"},
	pageRSVP:      {Type: "rsvp", Caption: "Ответ гостя", Title: "Придёте?", KeepData: true, DefaultData: map[string]any{"fields": []string{}}},
	pagePlaylist:  {Type: "playlist", Caption: "Плейлист", Title: "Закажите песню на вечер", KeepData: true, DefaultData: map[string]any{"votes": true}},
	pageGuestbook: {Type: "guestbook", Caption: "Поздравления", Title: "Пара тёплых слов", KeepData: true, DefaultData: map[string]any{}},
}

const (
	maxPageAbout   = 500
	maxPageText    = 200
	maxPageTags    = 12
	maxPageProgram = 8
	maxPageColors  = 5
)

var pageHex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// pageBlockKey — какому ключу каталога соответствует блок шаблона. Пусто —
// блок с примером, которого в опроснике нет (цитата, фото, видео,
// голосование, история): на страницу из ответов он не попадает.
func pageBlockKey(b entity.Block) string {
	switch b.Type {
	case "cover":
		return pageCover
	case "wishlist":
		return pageGifts
	case "location":
		return pagePlace
	case "color_scheme":
		return pageDress
	case "contact", "rsvp", "playlist", "guestbook":
		return b.Type
	case "list":
		switch b.View {
		case "schedule":
			return pageProgram
		case "tiles":
			return pageSizes
		case "pairs":
			return pageLikes
		case "tags":
			var data struct {
				Strike bool `json:"strike"`
			}
			_ = json.Unmarshal(b.Data, &data)
			if data.Strike {
				return pageStop
			}
			return pageLikes
		}
	}
	return ""
}

func checkLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s: не длиннее %d символов", field, max)
	}
	return nil
}

// cleanTags обрезает пробелы и выкидывает пустые и повторы.
func cleanTags(field string, values []string) ([]map[string]any, error) {
	if len(values) > maxPageTags {
		return nil, fmt.Errorf("%s: не больше %d пунктов", field, maxPageTags)
	}
	seen := map[string]bool{}
	items := []map[string]any{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		if err := checkLen(field, v, maxPageText); err != nil {
			return nil, err
		}
		seen[v] = true
		items = append(items, map[string]any{"v": v})
	}
	return items, nil
}

// pageContent — готовые data для каждого ключа и признак «есть содержимое».
type pageContent struct {
	about string
	data  map[string]map[string]any
	empty map[string]bool
}

func preparePage(p usecase.TemplatePage) (pageContent, error) {
	c := pageContent{data: map[string]map[string]any{}, empty: map[string]bool{}}
	for _, k := range p.Blocks {
		if !pageKnownKeys[k] {
			return c, fmt.Errorf("неизвестный блок %q", k)
		}
	}

	c.about = strings.TrimSpace(p.About)
	if err := checkLen("пара слов гостям", c.about, maxPageAbout); err != nil {
		return c, err
	}

	place := map[string]any{}
	for field, value := range map[string]string{"name": p.Place.Name, "address": p.Place.Address, "note": p.Place.Note} {
		value = strings.TrimSpace(value)
		if err := checkLen("место", value, maxPageText); err != nil {
			return c, err
		}
		if value != "" {
			place[field] = value
		}
	}
	c.data[pagePlace] = place
	c.empty[pagePlace] = place["name"] == nil && place["address"] == nil

	if len(p.Program) > maxPageProgram {
		return c, fmt.Errorf("программа: не больше %d пунктов", maxPageProgram)
	}
	program := []map[string]any{}
	for _, e := range p.Program {
		t, v := strings.TrimSpace(e.T), strings.TrimSpace(e.V)
		if err := checkLen("программа", t+v, maxPageText); err != nil {
			return c, err
		}
		if v == "" {
			continue
		}
		item := map[string]any{"v": v}
		if t != "" {
			item["t"] = t
		}
		program = append(program, item)
	}
	c.data[pageProgram] = map[string]any{"items": program}
	c.empty[pageProgram] = len(program) == 0

	if len(p.Dress.Colors) > maxPageColors {
		return c, fmt.Errorf("дресс-код: не больше %d цветов", maxPageColors)
	}
	colors := []map[string]any{}
	for _, col := range p.Dress.Colors {
		if !pageHex.MatchString(col.Hex) {
			return c, fmt.Errorf("дресс-код: цвет %q должен быть в виде #RRGGBB", col.Hex)
		}
		name := strings.TrimSpace(col.Name)
		if err := checkLen("дресс-код", name, maxPageText); err != nil {
			return c, err
		}
		colors = append(colors, map[string]any{"hex": col.Hex, "name": name})
	}
	dress := map[string]any{"colors": colors, "showNames": true}
	note := strings.TrimSpace(p.Dress.Note)
	if err := checkLen("дресс-код", note, maxPageText); err != nil {
		return c, err
	}
	if note != "" {
		dress["note"] = note
	}
	c.data[pageDress] = dress
	c.empty[pageDress] = len(colors) == 0

	name, way := strings.TrimSpace(p.Contact.Name), strings.TrimSpace(p.Contact.Way)
	if err := checkLen("контакт", name+way, maxPageText); err != nil {
		return c, err
	}
	contact := map[string]any{}
	if name != "" {
		contact["name"] = name
	}
	if way != "" {
		if strings.HasPrefix(way, "@") || strings.Contains(way, "t.me/") {
			contact["telegram"] = way
		} else {
			contact["phone"] = way
		}
	}
	c.data[pageContact] = contact
	c.empty[pageContact] = len(contact) == 0

	likes, err := cleanTags("что люблю", p.Likes)
	if err != nil {
		return c, err
	}
	c.data[pageLikes] = map[string]any{"items": likes}
	c.empty[pageLikes] = len(likes) == 0

	stop, err := cleanTags("стоп-лист", p.Stop)
	if err != nil {
		return c, err
	}
	c.data[pageStop] = map[string]any{"items": stop, "strike": true}
	c.empty[pageStop] = len(stop) == 0

	sizes := []map[string]any{}
	for _, s := range []struct{ k, v string }{{"одежда", p.Sizes.Clothes}, {"обувь", p.Sizes.Shoes}, {"рост", p.Sizes.Height}, {"кольцо", p.Sizes.Ring}} {
		v := strings.TrimSpace(s.v)
		if err := checkLen("размеры", v, maxPageText); err != nil {
			return c, err
		}
		if v != "" {
			sizes = append(sizes, map[string]any{"k": s.k, "v": v})
		}
	}
	c.data[pageSizes] = map[string]any{"items": sizes}
	c.empty[pageSizes] = len(sizes) == 0

	return c, nil
}

func setData(b *entity.Block, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode %s data: %w", b.Type, err)
	}
	b.Data = raw
	return nil
}

// fillPageBlock ставит в блок ответы: вид и подпись — из каталога, data —
// из ответов, пустой блок скрывается до заполнения.
func fillPageBlock(b *entity.Block, key string, c pageContent) error {
	spec := pageSpecs[key]
	b.Caption, b.Title = spec.Caption, spec.Title
	if spec.View != "" && (b.View == "" || spec.Type == "list") {
		b.View = spec.View
	}
	b.RevealAt, b.SecretMode, b.SecretText = nil, "", ""
	if spec.KeepData {
		if len(b.Data) == 0 {
			return setData(b, spec.DefaultData)
		}
		return nil
	}
	b.Hidden = c.empty[key]
	return setData(b, c.data[key])
}

// buildTemplatePage собирает страницу из блоков шаблона (уже копии) и
// ответов опросника. Раскладка шаблона сохраняется, лишнее убирается,
// недостающее добавляется отдельными рядами, ряды уплотняются.
func buildTemplatePage(blocks []entity.Block, rows []entity.RowSettings, title string, age int, p usecase.TemplatePage) ([]entity.Block, []entity.RowSettings, error) {
	content, err := preparePage(p)
	if err != nil {
		return nil, nil, err
	}
	chosen := map[string]bool{pageCover: true, pageGifts: true}
	for _, k := range p.Blocks {
		chosen[k] = true
	}

	sorted := append([]entity.Block(nil), blocks...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Row != sorted[j].Row {
			return sorted[i].Row < sorted[j].Row
		}
		return sorted[i].Col < sorted[j].Col
	})

	type pageRow struct {
		settings entity.RowSettings
		blocks   []entity.Block
	}
	var layout []pageRow
	seen := map[string]bool{}
	lastRow := -1
	for _, b := range sorted {
		key := pageBlockKey(b)
		if key == "" || !chosen[key] || seen[key] {
			continue
		}
		seen[key] = true

		switch key {
		case pageCover:
			b.Title = title
			patch := map[string]any{"subtitle": nil, "number": nil}
			if content.about != "" {
				patch["subtitle"] = content.about
			}
			if b.View == "number" {
				if age > 0 {
					patch["number"] = strconv.Itoa(age)
				} else {
					b.View = "center"
				}
			}
			if err := patchBlockData(&b, patch); err != nil {
				return nil, nil, err
			}
		case pageGifts:
			b.Title = "Подарки"
		default:
			if err := fillPageBlock(&b, key, content); err != nil {
				return nil, nil, err
			}
		}

		if b.Row != lastRow {
			settings := entity.RowSettings{}
			if b.Row < len(rows) {
				settings = rows[b.Row]
			}
			layout = append(layout, pageRow{settings: settings})
			lastRow = b.Row
		}
		layout[len(layout)-1].blocks = append(layout[len(layout)-1].blocks, b)
	}

	giftsAt := len(layout)
	for i, r := range layout {
		for _, b := range r.blocks {
			if b.Type == "wishlist" {
				giftsAt = i
			}
		}
	}
	var before, after []pageRow
	for _, key := range pageCatalogOrder {
		if !chosen[key] || seen[key] {
			continue
		}
		spec := pageSpecs[key]
		b := entity.Block{Type: spec.Type, ColSpan: 1}
		if err := fillPageBlock(&b, key, content); err != nil {
			return nil, nil, err
		}
		row := pageRow{settings: entity.RowSettings{Columns: 1}, blocks: []entity.Block{b}}
		if pageBeforeGifts[key] {
			before = append(before, row)
		} else {
			after = append(after, row)
		}
	}
	layout = append(append(append(append([]pageRow(nil), layout[:giftsAt]...), before...), layout[giftsAt:]...), after...)

	var outBlocks []entity.Block
	outRows := make([]entity.RowSettings, 0, len(layout))
	for i, r := range layout {
		settings := r.settings
		if n := len(r.blocks); n != settings.ColumnsOrDefault() {
			settings.Columns = n
			settings.Ratio = ""
		}
		for j, b := range r.blocks {
			b.Row, b.Col, b.ColSpan = i, j, 1
			outBlocks = append(outBlocks, b)
		}
		outRows = append(outRows, settings)
	}
	return outBlocks, outRows, nil
}
