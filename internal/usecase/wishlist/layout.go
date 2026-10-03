package wishlist

import (
	"fmt"

	"main/internal/entity"
)

// ratiosByColumns — пропорции, которые макет предлагает для каждого числа
// колонок. Разделитель на холсте прилипает только к ним, поэтому произвольная
// дробь — это ошибка клиента, а не новая раскладка.
var ratiosByColumns = map[int]map[string]bool{
	1: {"": true},
	2: {"": true, "1:1": true, "2:1": true, "1:2": true},
	3: {"": true, "1:1:1": true},
}

var (
	rowHeights = map[string]bool{"": true, "equal": true, "auto": true}
	rowGaps    = map[string]bool{"": true, "s": true, "m": true, "l": true}
)

// validateLayout — блоки укладываются в ряды без наездов, настройки рядов
// осмысленны.
//
// Ряд без записи в rows считается двухколоночным: так читается формат v2, где
// настроек рядов не было вовсе.
func validateLayout(blocks []entity.Block, rows []entity.RowSettings) error {
	maxRow := -1
	for _, b := range blocks {
		if b.Row > maxRow {
			maxRow = b.Row
		}
	}
	if len(rows) > maxRow+1 {
		return fmt.Errorf("rows: настроек %d, а рядов %d", len(rows), maxRow+1)
	}

	for i, r := range rows {
		if r.Columns < 0 || r.Columns > 3 {
			return fmt.Errorf("rows[%d]: columns должно быть от 1 до 3", i)
		}
		if !ratiosByColumns[r.ColumnsOrDefault()][r.Ratio] {
			return fmt.Errorf("rows[%d]: ratio %q не подходит для %d колонок", i, r.Ratio, r.ColumnsOrDefault())
		}
		if !rowHeights[r.Height] {
			return fmt.Errorf("rows[%d]: неизвестная height %q", i, r.Height)
		}
		if !rowGaps[r.Gap] {
			return fmt.Errorf("rows[%d]: неизвестный gap %q", i, r.Gap)
		}
	}

	occupied := map[[2]int]int{}
	for i, b := range blocks {
		if b.Row < 0 {
			return fmt.Errorf("block[%d]: row должен быть >= 0", i)
		}
		columns := 2
		if b.Row < len(rows) {
			columns = rows[b.Row].ColumnsOrDefault()
		}
		span := b.ColSpan
		if span < 1 {
			span = 1
		}
		if b.Col < 0 || b.Col+span > columns {
			return fmt.Errorf("block[%d]: col %d с colSpan %d выходит за ряд из %d колонок", i, b.Col, span, columns)
		}
		for c := b.Col; c < b.Col+span; c++ {
			cell := [2]int{b.Row, c}
			if other, taken := occupied[cell]; taken {
				return fmt.Errorf("block[%d]: ячейка ряд %d, колонка %d уже занята блоком %d", i, b.Row, c, other)
			}
			occupied[cell] = i
		}
	}
	return nil
}

// validateLook — оформление приходит из формы и уходит в CSS-классы, поэтому
// принимаются только известные значения.
func validateLook(look entity.Look) error {
	if !entity.HeadingFonts[look.HeadingFont] {
		return fmt.Errorf("неизвестный headingFont %q", look.HeadingFont)
	}
	if !entity.Patterns[look.Pattern] {
		return fmt.Errorf("неизвестный pattern %q", look.Pattern)
	}
	return nil
}
