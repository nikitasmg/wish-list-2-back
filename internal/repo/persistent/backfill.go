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

// BackfillPollChoices переносит голоса из poll_votes (один на гостя, вариант
// по индексу) в poll_choices (вариант по id).
//
// Индекс становится id: старые голосования хранят варианты строками, и фронт
// выдаёт им id по индексу — голоса продолжают указывать на тот же вариант.
// Перенесённые строки удаляются тем же запросом: иначе следующий запуск вернул
// бы голос, который гость с тех пор поменял.
func BackfillPollChoices(db *gorm.DB) error {
	const query = `
		WITH moved AS (
			DELETE FROM poll_votes
			RETURNING wishlist_id, block_id, guest_id, option_index, created_at
		)
		INSERT INTO poll_choices (wishlist_id, block_id, guest_id, option_id, created_at)
		SELECT wishlist_id, block_id, guest_id, option_index::text, created_at FROM moved
		ON CONFLICT DO NOTHING
	`
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("backfill poll choices: %w", err)
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

// BackfillSantaPendingEmail переносит неподтверждённые адреса участников
// Тайного Санты в pending_email.
//
// До этапа 3 новый адрес сразу писался в email со сбросом подтверждения;
// теперь в email только подтверждённый, а ждущий кода — в pending_email. Без
// переноса введённый код не подтвердил бы такой адрес.
//
// Идемпотентно: после переноса неподтверждённых email не остаётся.
func BackfillSantaPendingEmail(db *gorm.DB) error {
	const query = `
		UPDATE santa_participants
		SET pending_email = email, email = NULL
		WHERE email IS NOT NULL AND email_verified_at IS NULL
	`
	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("backfill santa pending_email: %w", err)
	}
	return nil
}
