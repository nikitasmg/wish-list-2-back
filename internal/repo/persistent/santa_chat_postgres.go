package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"main/internal/entity"
	"main/internal/repo"
)

// authoredSQL — сообщения, написанные участником (оба ? — его id): как Сантой
// своему подопечному и как подопечным своему Санте.
const authoredSQL = "((giver_id = ? AND from_giver) OR (receiver_id = ? AND NOT from_giver))"

func (r *santaRepo) CreateMessage(ctx context.Context, msg entity.SantaMessage, since time.Time, limit int, note entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// FOR SHARE против FOR UPDATE жеребьёвки: перезапуск ждёт вставку
		// (и сотрёт сообщение вместе с остальными) или вставка видит новые пары.
		var room SantaRoomModel
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&room, "id = ?", msg.RoomID).Error; err != nil {
			return santaErr("santaRepo.CreateMessage lock", err)
		}
		if room.Status != string(entity.SantaRoomDrawn) {
			return repo.ErrStatusMismatch
		}
		var pairs int64
		if err := tx.Model(&SantaAssignmentModel{}).
			Where("room_id = ? AND giver_id = ? AND receiver_id = ?", msg.RoomID, msg.GiverID, msg.ReceiverID).
			Count(&pairs).Error; err != nil {
			return santaErr("santaRepo.CreateMessage pair", err)
		}
		if pairs == 0 {
			return fmt.Errorf("santaRepo.CreateMessage: %w", repo.ErrNotFound)
		}
		// Два одновременных сообщения могут проскочить лимит на одно — не страшно.
		author := msg.AuthorID()
		var sent int64
		if err := tx.Model(&SantaMessageModel{}).
			Where("created_at > ?", since).
			Where(authoredSQL, author, author).
			Count(&sent).Error; err != nil {
			return santaErr("santaRepo.CreateMessage limit", err)
		}
		if sent >= int64(limit) {
			return repo.ErrTooSoon
		}
		m := toSantaMessageModel(msg)
		if err := tx.Create(&m).Error; err != nil {
			return santaErr("santaRepo.CreateMessage", err)
		}
		return insertNotifications(tx, note)
	})
}

func (r *santaRepo) GetMessage(ctx context.Context, id uuid.UUID) (entity.SantaMessage, error) {
	var m SantaMessageModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.SantaMessage{}, santaErr("santaRepo.GetMessage", err)
	}
	return toSantaMessageEntity(m), nil
}

func (r *santaRepo) ListMessages(ctx context.Context, roomID, giverID, receiverID uuid.UUID, limit int) ([]entity.SantaMessage, error) {
	var models []SantaMessageModel
	// Последние limit: берём с конца и разворачиваем.
	if err := r.db.WithContext(ctx).
		Where("room_id = ? AND giver_id = ? AND receiver_id = ?", roomID, giverID, receiverID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, santaErr("santaRepo.ListMessages", err)
	}
	out := make([]entity.SantaMessage, len(models))
	for i, m := range models {
		out[len(models)-1-i] = toSantaMessageEntity(m)
	}
	return out, nil
}

func (r *santaRepo) MarkMessagesRead(ctx context.Context, roomID, giverID, receiverID uuid.UUID, fromGiver bool, at time.Time) error {
	if err := r.db.WithContext(ctx).Model(&SantaMessageModel{}).
		Where("room_id = ? AND giver_id = ? AND receiver_id = ? AND from_giver = ? AND read_at IS NULL",
			roomID, giverID, receiverID, fromGiver).
		Update("read_at", at).Error; err != nil {
		return santaErr("santaRepo.MarkMessagesRead", err)
	}
	return nil
}

func (r *santaRepo) CountUnread(ctx context.Context, roomID, participantID uuid.UUID) (int, int, error) {
	var row struct {
		FromSanta    int
		FromReceiver int
	}
	if err := r.db.WithContext(ctx).Model(&SantaMessageModel{}).
		Select("COUNT(*) FILTER (WHERE receiver_id = ? AND from_giver) AS from_santa, "+
			"COUNT(*) FILTER (WHERE giver_id = ? AND NOT from_giver) AS from_receiver", participantID, participantID).
		Where("room_id = ? AND read_at IS NULL", roomID).
		Scan(&row).Error; err != nil {
		return 0, 0, santaErr("santaRepo.CountUnread", err)
	}
	return row.FromSanta, row.FromReceiver, nil
}

func (r *santaRepo) FindChatNotification(ctx context.Context, chatID, tgMessageID int64) (entity.SantaNotification, error) {
	var m SantaNotificationModel
	// message_id уникален только внутри чата — сверяем и чат получателя.
	if err := r.db.WithContext(ctx).
		Select("santa_notifications.*").
		Joins("JOIN santa_participants p ON p.id = santa_notifications.participant_id").
		Where("santa_notifications.kind = ? AND santa_notifications.tg_message_id = ? AND p.tg_chat_id = ?",
			string(entity.SantaNotifyChatMessage), tgMessageID, chatID).
		Order("santa_notifications.created_at DESC").
		Take(&m).Error; err != nil {
		return entity.SantaNotification{}, santaErr("santaRepo.FindChatNotification", err)
	}
	return toSantaNotificationEntity(m), nil
}
