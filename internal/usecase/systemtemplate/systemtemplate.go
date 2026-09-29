// Package systemtemplate отдаёт встроенные заготовки страниц под повод.
//
// Данные лежат в templates.json рядом с кодом и вшиваются в бинарник через
// go:embed: шаблоны правятся вместе с релизом, поэтому ни таблицы, ни админки,
// ни отдельного файла в образе им не нужно.
package systemtemplate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"main/internal/entity"
)

//go:embed templates.json
var templatesJSON []byte

var (
	once      sync.Once
	templates []entity.SystemTemplate
	loadErr   error
)

// All — все шаблоны в порядке показа.
func All() ([]entity.SystemTemplate, error) {
	load()
	if loadErr != nil {
		return nil, loadErr
	}
	return templates, nil
}

// Get — шаблон по идентификатору.
func Get(id string) (entity.SystemTemplate, error) {
	load()
	if loadErr != nil {
		return entity.SystemTemplate{}, loadErr
	}
	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}
	return entity.SystemTemplate{}, fmt.Errorf("шаблон %q не найден", id)
}

func load() {
	once.Do(func() {
		if err := json.Unmarshal(templatesJSON, &templates); err != nil {
			loadErr = fmt.Errorf("parse templates.json: %w", err)
			return
		}
		// Опечатка в типе блока превратилась бы в шаблон, который невозможно
		// применить: CreateConstructor отверг бы его на этапе валидации, а
		// пользователь увидел бы невнятную ошибку вместо готовой страницы.
		for _, t := range templates {
			for i, b := range t.Blocks {
				if b.Row < 0 || b.Col < 0 || b.Col > 1 || b.ColSpan < 1 || b.ColSpan > 2 || b.Col+b.ColSpan > 2 {
					loadErr = fmt.Errorf("шаблон %q, блок %d: неверные координаты", t.ID, i)
					return
				}
				if !entity.ValidBlockTypes[b.Type] {
					loadErr = fmt.Errorf("шаблон %q, блок %d: неизвестный тип %q", t.ID, i, b.Type)
					return
				}
				if entity.LegacyBlockTypes[b.Type] {
					loadErr = fmt.Errorf("шаблон %q, блок %d: тип %q устарел", t.ID, i, b.Type)
					return
				}
			}
		}
	})
}
