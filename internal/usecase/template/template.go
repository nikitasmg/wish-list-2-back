// Package template отдаёт заготовки страниц под повод.
//
// Данные лежат в templates.json рядом с кодом и вшиваются в бинарник через
// go:embed: шаблоны правятся вместе с релизом, поэтому ни таблицы, ни админки,
// ни отдельного файла в образе им не нужно.
package template

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
	templates []entity.Template
	loadErr   error
)

// All — все шаблоны в порядке показа.
func All() ([]entity.Template, error) {
	load()
	if loadErr != nil {
		return nil, loadErr
	}
	return templates, nil
}

// ByID — шаблон по идентификатору.
func ByID(id string) (entity.Template, error) {
	load()
	if loadErr != nil {
		return entity.Template{}, loadErr
	}
	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}
	return entity.Template{}, fmt.Errorf("шаблон %q не найден", id)
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
