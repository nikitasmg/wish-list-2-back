package systemtemplate_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase/systemtemplate"
)

// Главная проверка пакета: templates.json разбирается и не содержит блоков,
// которые потом отвергнет конструктор.
func TestAll_LoadsAndValidates(t *testing.T) {
	templates, err := systemtemplate.All()
	require.NoError(t, err)
	require.Len(t, templates, 8, "восемь шаблонов из макета")

	seen := map[string]bool{}
	for _, tpl := range templates {
		assert.False(t, seen[tpl.ID], "id %q повторяется", tpl.ID)
		seen[tpl.ID] = true

		assert.NotEmpty(t, tpl.Name, "%s: без названия", tpl.ID)
		assert.NotEmpty(t, tpl.ColorScheme, "%s: без схемы", tpl.ID)
		assert.NotEmpty(t, tpl.SampleTitle, "%s: без названия-примера", tpl.ID)
		assert.NotEmpty(t, tpl.Blocks, "%s: пустой набор блоков", tpl.ID)

		for i, b := range tpl.Blocks {
			assert.True(t, entity.ValidBlockTypes[b.Type], "%s: блок %d тип %q", tpl.ID, i, b.Type)
			assert.False(t, entity.LegacyBlockTypes[b.Type], "%s: блок %d устаревший тип %q", tpl.ID, i, b.Type)
			// Раскладка v3: ряды до трёх колонок, блок в одной колонке. Остальное
			// (наезды, пропорции) проверяет тест валидации в пакете wishlist.
			assert.Less(t, b.Row, len(tpl.Rows), "%s: блок %d в ряду без настроек", tpl.ID, i)
			assert.Equal(t, 1, b.ColSpan)
			assert.NotNil(t, b.Data, "%s: блок %d без data", tpl.ID, i)
		}
	}
}

// Категории фильтров и категории шаблонов должны сойтись, иначе на странице
// выбора появится фильтр, под который ничего не попадает.
func TestAll_CategoriesMatchFilters(t *testing.T) {
	templates, err := systemtemplate.All()
	require.NoError(t, err)

	known := map[string]bool{}
	for _, c := range entity.SystemTemplateCategories {
		known[c.ID] = true
	}

	used := map[string]bool{}
	for _, tpl := range templates {
		assert.True(t, known[tpl.Category], "шаблон %q в неизвестной категории %q", tpl.ID, tpl.Category)
		used[tpl.Category] = true
	}

	for _, c := range entity.SystemTemplateCategories {
		assert.True(t, used[c.ID], "в категории %q нет ни одного шаблона", c.ID)
	}
}

// В каждом шаблоне должен быть вишлист — иначе страница создаётся без того,
// ради чего сервис существует.
func TestAll_EveryTemplateHasWishlist(t *testing.T) {
	templates, err := systemtemplate.All()
	require.NoError(t, err)

	for _, tpl := range templates {
		var hasWishlist, hasCover bool
		for _, b := range tpl.Blocks {
			switch b.Type {
			case "wishlist":
				hasWishlist = true
			case "cover":
				hasCover = true
			}
		}
		assert.True(t, hasWishlist, "%s: нет блока вишлиста", tpl.ID)
		assert.True(t, hasCover, "%s: нет обложки", tpl.ID)
	}
}

func TestGet(t *testing.T) {
	tpl, err := systemtemplate.Get("wedding")
	require.NoError(t, err)
	assert.Equal(t, "Свадьба", tpl.Name)

	_, err = systemtemplate.Get("не-существует")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "не найден")
}
