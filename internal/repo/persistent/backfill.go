package persistent

import (
	"fmt"

	"gorm.io/gorm"
)

// BackfillEventDate переносит дату праздника из location.time в отдельную
// колонку event_date.
//
// AutoMigrate умеет добавить колонку, но не заполнить её, а без переноса все
// существующие вишлисты остались бы без даты: обратный отсчёт, календарь и
// сортировка в кабинете показывали бы пустоту.
//
// Идемпотентно: трогает только строки, где event_date ещё не заполнена.
// Нулевое время Go сериализуется как 0001-01-01, и такие строки тоже
// пропускаются — это «дата не указана», а не 1 января первого года.
func BackfillEventDate(db *gorm.DB) error {
	const query = `
		UPDATE wishlists
		SET event_date = (location->>'time')::timestamptz
		WHERE event_date IS NULL
		  AND location->>'time' IS NOT NULL
		  AND location->>'time' <> ''
		  AND (location->>'time')::timestamptz > '1970-01-01'::timestamptz
	`

	result := db.Exec(query)
	if result.Error != nil {
		return fmt.Errorf("backfill event_date: %w", result.Error)
	}
	return nil
}

// BackfillPresentLinks переносит единственную ссылку подарка в массив links.
//
// Идемпотентно: заполняются только строки, где links ещё пуст, а link не пуст.
func BackfillPresentLinks(db *gorm.DB) error {
	const query = `
		UPDATE presents
		SET links = jsonb_build_array(link)
		WHERE links IS NULL
		  AND link IS NOT NULL
		  AND link <> ''
	`

	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("backfill present links: %w", err)
	}
	return nil
}
