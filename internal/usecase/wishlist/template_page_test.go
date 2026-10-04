package wishlist_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	"main/internal/usecase/systemtemplate"
	mockminio "main/mock/minio"
	mockrepo "main/mock/repo"
)

func pageInput(tplID string, page usecase.TemplatePage) usecase.CreateFromSystemTemplateInput {
	return usecase.CreateFromSystemTemplateInput{TemplateID: tplID, Title: "Мой праздник", Page: &page}
}

func blocksOf(w entity.Wishlist, typ string) []entity.Block {
	var out []entity.Block
	for _, b := range w.Blocks {
		if b.Type == typ {
			out = append(out, b)
		}
	}
	return out
}

// Главное правило опросника: тексты-примеры шаблона на страницу не попадают.
// Человек написал «дома» — бара, настолок и чужого стоп-листа быть не должно.
func TestCreateFromSystemTemplate_PageDropsSampleContent(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("man", usecase.TemplatePage{
		Blocks: []string{"about", "place", "rsvp"},
		About:  "Посидим дома",
		Place:  usecase.TemplatePlace{Name: "Дома", Address: "ул. Садовая, 8"},
	}))
	raw := blocksJSON(t, w)
	for _, sample := range []string{"Подвал", "Вайнера", "настолки", "носки", "Стол забронирован"} {
		assert.NotContains(t, raw, sample)
	}
	assert.Empty(t, blocksOf(w, "list"), "стоп-лист не отмечен — его нет")

	cover, ok := findBlock(w, "cover")
	require.True(t, ok)
	assert.Equal(t, "Посидим дома", blockData(t, cover)["subtitle"])
	assert.Equal(t, "Мой праздник", cover.Title)

	loc, ok := findBlock(w, "location")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"name": "Дома", "address": "ул. Садовая, 8"}, blockData(t, loc))
	assert.Equal(t, "Где собираемся", loc.Title)

	_, ok = findBlock(w, "wishlist")
	assert.True(t, ok, "подарки есть всегда")
}

func TestCreateFromSystemTemplate_PageFillsLists(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("man", usecase.TemplatePage{
		Blocks: []string{"stop"},
		Stop:   []string{" подушка ", "", "плед"},
	}))
	lists := blocksOf(w, "list")
	require.Len(t, lists, 1)
	stop := lists[0]
	assert.Equal(t, "Пожалуйста, не дарите", stop.Title)
	assert.Equal(t, "tags", stop.View)
	assert.False(t, stop.Hidden)
	data := blockData(t, stop)
	assert.Equal(t, true, data["strike"])
	assert.Equal(t, []any{map[string]any{"v": "подушка"}, map[string]any{"v": "плед"}}, data["items"])
}

// В шаблоне девочки нет ни места, ни стоп-листа, зато есть «Любимое»,
// размеры, программа и дресс-код с примерами. Неотмеченное уходит,
// недостающее встаёт: про подарки — перед подарками, остальное — в конец.
func TestCreateFromSystemTemplate_PageAddsMissingBlocks(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("girl", usecase.TemplatePage{
		Blocks: []string{"place", "stop"},
		Place:  usecase.TemplatePlace{Name: "Дом"},
		Stop:   []string{"слаймы"},
	}))
	assert.Empty(t, blocksOf(w, "color_scheme"))
	assert.NotContains(t, blocksJSON(t, w), "фея")

	gifts, ok := findBlock(w, "wishlist")
	require.True(t, ok)
	loc, ok := findBlock(w, "location")
	require.True(t, ok)
	lists := blocksOf(w, "list")
	require.Len(t, lists, 1)
	assert.Equal(t, gifts.Row-1, lists[0].Row, "стоп-лист — прямо над подарками")
	assert.Greater(t, loc.Row, gifts.Row, "место — после подарков")
	assert.NotEmpty(t, loc.ID)
}

// После удаления блоков ряды идут подряд, а ряд, где остался один блок из
// двух, становится одноколоночным — иначе блок занял бы половину ширины.
func TestCreateFromSystemTemplate_PageCompactsRows(t *testing.T) {
	for _, id := range []string{"man", "woman", "girl", "wedding", "party"} {
		t.Run(id, func(t *testing.T) {
			w := createFromTemplateForTest(t, pageInput(id, usecase.TemplatePage{
				Blocks: []string{"place"},
				Place:  usecase.TemplatePlace{Name: "Дома"},
			}))
			maxRow := 0
			perRow := map[int][]entity.Block{}
			for _, b := range w.Blocks {
				perRow[b.Row] = append(perRow[b.Row], b)
				if b.Row > maxRow {
					maxRow = b.Row
				}
			}
			require.Len(t, w.Rows, maxRow+1)
			for r := 0; r <= maxRow; r++ {
				require.NotEmpty(t, perRow[r], "ряд %d пустой", r)
				assert.Equal(t, len(perRow[r]), w.Rows[r].ColumnsOrDefault(), "ряд %d", r)
				for i, b := range perRow[r] {
					assert.Equal(t, i, b.Col, "ряд %d", r)
				}
			}
		})
	}
}

// «Заполню потом»: блок создаётся, но гость его не видит, пока владелец не
// заполнит и не включит.
func TestCreateFromSystemTemplate_PageHidesEmptyBlocks(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("man", usecase.TemplatePage{Blocks: []string{"place", "stop", "rsvp"}}))
	loc, ok := findBlock(w, "location")
	require.True(t, ok)
	assert.True(t, loc.Hidden)
	lists := blocksOf(w, "list")
	require.Len(t, lists, 1)
	assert.True(t, lists[0].Hidden)
	rsvp, ok := findBlock(w, "rsvp")
	require.True(t, ok)
	assert.False(t, rsvp.Hidden, "ответу гостя заполнять нечего")

	cover, ok := findBlock(w, "cover")
	require.True(t, ok)
	_, has := blockData(t, cover)["subtitle"]
	assert.False(t, has, "без «пары слов» подзаголовка нет, а не пример из шаблона")
}

// Без возраста у обложки «цифрой» осталась бы «7» из примера.
func TestCreateFromSystemTemplate_PageCoverNumber(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("boy", usecase.TemplatePage{}))
	cover, ok := findBlock(w, "cover")
	require.True(t, ok)
	assert.Equal(t, "center", cover.View)
	_, has := blockData(t, cover)["number"]
	assert.False(t, has)

	in := pageInput("boy", usecase.TemplatePage{})
	in.Age = 9
	w = createFromTemplateForTest(t, in)
	cover, _ = findBlock(w, "cover")
	assert.Equal(t, "number", cover.View)
	assert.Equal(t, "9", blockData(t, cover)["number"])
}

// У ответа гостя и плейлиста в data — настройки, а не факты: их оставляем.
func TestCreateFromSystemTemplate_PageKeepsGuestSettings(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("woman", usecase.TemplatePage{Blocks: []string{"playlist", "rsvp"}}))
	pl, ok := findBlock(w, "playlist")
	require.True(t, ok)
	assert.Equal(t, true, blockData(t, pl)["votes"])
	assert.Empty(t, blocksOf(w, "quote"), "цитата — пример, её нет")
}

func TestCreateFromSystemTemplate_PageFillsDressContactSizesProgram(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("wedding", usecase.TemplatePage{
		Blocks:  []string{"dress", "contact", "sizes", "program"},
		Dress:   usecase.TemplateDress{Colors: []usecase.TemplateColor{{Hex: "#141414", Name: "Чёрный"}}, Note: "Без белого"},
		Contact: usecase.TemplateContact{Name: "Оля", Way: "@olya"},
		Sizes:   usecase.TemplateSizes{Clothes: "S", Shoes: "37"},
		Program: []usecase.TemplateTimeEntry{{T: "15:00", V: "Регистрация"}, {T: "", V: ""}},
	}))
	dress, ok := findBlock(w, "color_scheme")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"colors": []any{map[string]any{"hex": "#141414", "name": "Чёрный"}}, "showNames": true, "note": "Без белого"}, blockData(t, dress))

	contact, ok := findBlock(w, "contact")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"name": "Оля", "telegram": "@olya"}, blockData(t, contact))
	assert.NotContains(t, blocksJSON(t, w), "Ольга")

	var sizes, program entity.Block
	for _, b := range blocksOf(w, "list") {
		switch b.View {
		case "tiles":
			sizes = b
		case "schedule":
			program = b
		}
	}
	assert.Equal(t, []any{map[string]any{"k": "одежда", "v": "S"}, map[string]any{"k": "обувь", "v": "37"}}, blockData(t, sizes)["items"])
	assert.Equal(t, []any{map[string]any{"t": "15:00", "v": "Регистрация"}}, blockData(t, program)["items"])
	assert.Empty(t, blocksOf(w, "location"), "место не отмечено")
}

func TestCreateFromSystemTemplate_PageContactPhone(t *testing.T) {
	w := createFromTemplateForTest(t, pageInput("man", usecase.TemplatePage{
		Blocks: []string{"contact"}, Contact: usecase.TemplateContact{Name: "Оля", Way: "+7 900 000-00-00"},
	}))
	contact, ok := findBlock(w, "contact")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"name": "Оля", "phone": "+7 900 000-00-00"}, blockData(t, contact))
}

func TestCreateFromSystemTemplate_PageValidation(t *testing.T) {
	many := make([]string, 13)
	for i := range many {
		many[i] = "x"
	}
	cases := map[string]usecase.TemplatePage{
		"неизвестный блок": {Blocks: []string{"bar"}},
		"длинный список":   {Blocks: []string{"stop"}, Stop: many},
		"неверный цвет":    {Blocks: []string{"dress"}, Dress: usecase.TemplateDress{Colors: []usecase.TemplateColor{{Hex: "red"}}}},
		"длинное место":    {Blocks: []string{"place"}, Place: usecase.TemplatePlace{Name: strings.Repeat("я", 201)}},
		"длинный текст":    {Blocks: []string{"about"}, About: strings.Repeat("я", 501)},
	}
	for name, page := range cases {
		t.Run(name, func(t *testing.T) {
			wr := &mockrepo.MockWishlistRepo{}
			wr.On("CountByUserID", mock.Anything, mock.Anything).Return(int64(0), nil).Maybe()
			uc := newWishlistUC(wr, &mockminio.MockFileStorage{})
			_, err := uc.CreateFromSystemTemplate(context.Background(), uuid.New(), pageInput("man", page))
			require.Error(t, err)
			wr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateFromSystemTemplate_PageDoesNotMutateTemplate(t *testing.T) {
	before, err := systemtemplate.Get("girl")
	require.NoError(t, err)
	beforeJSON, _ := json.Marshal(before)

	createFromTemplateForTest(t, pageInput("girl", usecase.TemplatePage{Blocks: []string{"place", "stop"}, Stop: []string{"a"}}))

	after, err := systemtemplate.Get("girl")
	require.NoError(t, err)
	afterJSON, _ := json.Marshal(after)
	assert.JSONEq(t, string(beforeJSON), string(afterJSON))
}
