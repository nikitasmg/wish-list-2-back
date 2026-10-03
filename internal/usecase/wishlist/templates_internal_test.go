package wishlist

import (
	"testing"

	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase/systemtemplate"
)

// Каждая системная заготовка проходит ту же проверку, что и сохранение из
// конструктора: иначе вишлист по шаблону создастся, а первое же
// автосохранение упадёт на раскладке.
func TestSystemTemplatesPassValidation(t *testing.T) {
	templates, err := systemtemplate.All()
	require.NoError(t, err)
	require.Len(t, templates, 8)
	for _, tpl := range templates {
		t.Run(tpl.ID, func(t *testing.T) {
			require.NoError(t, validateNewBlocks(tpl.Blocks, tpl.Rows))
			require.NoError(t, validateLook(tpl.Look))
			require.NotEmpty(t, tpl.Look.HeadingFont, "у заготовки свой шрифт из макета")
			require.Len(t, tpl.Rows, maxRow(tpl.Blocks)+1, "настройки есть у каждого ряда")
		})
	}
}

func maxRow(blocks []entity.Block) int {
	max := -1
	for _, b := range blocks {
		if b.Row > max {
			max = b.Row
		}
	}
	return max
}
