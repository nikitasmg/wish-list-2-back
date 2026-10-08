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

// insertNotifications кладёт уведомления в outbox внутри уже открытой транзакции.
func insertNotifications(tx *gorm.DB, notes ...entity.SantaNotification) error {
	if len(notes) == 0 {
		return nil
	}
	models := make([]SantaNotificationModel, len(notes))
	for i, n := range notes {
		models[i] = toSantaNotificationModel(n)
	}
	if err := tx.Create(&models).Error; err != nil {
		return santaErr("santaRepo.insertNotifications", err)
	}
	return nil
}

func (r *santaRepo) SetEmail(ctx context.Context, participantID uuid.UUID, email string, code entity.SantaEmailCode) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p SantaParticipantModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", participantID).Error; err != nil {
			return santaErr("santaRepo.SetEmail", err)
		}
		// В email лежат только подтверждённые адреса. Чужой неподтверждённый
		// адрес не мешает: достанется тому, кто первым введёт код.
		var taken int64
		if err := tx.Model(&SantaParticipantModel{}).
			Where("room_id = ? AND id <> ? AND email = ?", p.RoomID, p.ID, email).
			Count(&taken).Error; err != nil {
			return santaErr("santaRepo.SetEmail check", err)
		}
		if taken > 0 {
			return fmt.Errorf("santaRepo.SetEmail: %w", repo.ErrDuplicate)
		}
		// Подтверждённый адрес и готовность не трогаем, пока новый не подтверждён.
		if err := tx.Model(&SantaParticipantModel{}).Where("id = ?", participantID).Updates(map[string]any{
			"pending_email": email,
			"updated_at":    time.Now(),
		}).Error; err != nil {
			return santaErr("santaRepo.SetEmail", err)
		}
		m := SantaEmailCodeModel{
			ParticipantID: participantID, CodeHash: code.CodeHash, ExpiresAt: code.ExpiresAt,
			Attempts: 0, SentAt: code.SentAt,
		}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&m).Error; err != nil {
			return santaErr("santaRepo.SetEmail code", err)
		}
		return nil
	})
}

func (r *santaRepo) GetEmailCode(ctx context.Context, participantID uuid.UUID) (entity.SantaEmailCode, error) {
	var m SantaEmailCodeModel
	if err := r.db.WithContext(ctx).First(&m, "participant_id = ?", participantID).Error; err != nil {
		return entity.SantaEmailCode{}, santaErr("santaRepo.GetEmailCode", err)
	}
	return entity.SantaEmailCode{
		ParticipantID: m.ParticipantID, CodeHash: m.CodeHash, ExpiresAt: m.ExpiresAt,
		Attempts: m.Attempts, SentAt: m.SentAt,
	}, nil
}

func (r *santaRepo) IncEmailCodeAttempts(ctx context.Context, participantID uuid.UUID, max int) (bool, error) {
	res := r.db.WithContext(ctx).Model(&SantaEmailCodeModel{}).
		Where("participant_id = ? AND attempts < ?", participantID, max).
		UpdateColumn("attempts", gorm.Expr("attempts + 1"))
	if res.Error != nil {
		return false, santaErr("santaRepo.IncEmailCodeAttempts", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *santaRepo) DeleteEmailCode(ctx context.Context, participantID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&SantaEmailCodeModel{}, "participant_id = ?", participantID).Error; err != nil {
		return santaErr("santaRepo.DeleteEmailCode", err)
	}
	return nil
}

func (r *santaRepo) VerifyEmail(ctx context.Context, participantID uuid.UUID, codeHash string, at time.Time, welcome entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		del := tx.Delete(&SantaEmailCodeModel{}, "participant_id = ? AND code_hash = ?", participantID, codeHash)
		if del.Error != nil {
			return santaErr("santaRepo.VerifyEmail code", del.Error)
		}
		if del.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.VerifyEmail: %w", repo.ErrNotFound)
		}
		// SET читает старые значения строки: email получает прежний pending_email.
		res := tx.Model(&SantaParticipantModel{}).
			Where("id = ? AND pending_email IS NOT NULL", participantID).
			Updates(map[string]any{
				"email":             gorm.Expr("pending_email"),
				"pending_email":     nil,
				"email_verified_at": at,
				"channel":           string(entity.SantaChannelEmail),
				"updated_at":        at,
			})
		if res.Error != nil {
			// Тот же адрес успел подтвердить другой участник комнаты.
			if isUniqueViolation(res.Error, "idx_santa_participant_email") {
				return fmt.Errorf("santaRepo.VerifyEmail: %w", repo.ErrDuplicate)
			}
			return santaErr("santaRepo.VerifyEmail", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.VerifyEmail: %w", repo.ErrNotFound)
		}
		return insertNotifications(tx, welcome)
	})
}

func (r *santaRepo) CreateTgLink(ctx context.Context, link entity.SantaTgLink) error {
	m := SantaTgLinkModel{TokenHash: link.TokenHash, ParticipantID: link.ParticipantID, ExpiresAt: link.ExpiresAt}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return santaErr("santaRepo.CreateTgLink", err)
	}
	return nil
}

func (r *santaRepo) LinkTelegram(ctx context.Context, tokenHash string, chatID int64, now time.Time, welcome func(entity.SantaParticipant) entity.SantaNotification) (entity.SantaParticipant, error) {
	var out entity.SantaParticipant
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var link SantaTgLinkModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&link, "token_hash = ?", tokenHash).Error; err != nil {
			return santaErr("santaRepo.LinkTelegram link", err)
		}
		if !link.ExpiresAt.After(now) {
			return fmt.Errorf("santaRepo.LinkTelegram: %w", repo.ErrNotFound)
		}
		if err := tx.Model(&SantaParticipantModel{}).Where("id = ?", link.ParticipantID).Updates(map[string]any{
			"tg_chat_id": chatID,
			"channel":    string(entity.SantaChannelTelegram),
			"updated_at": now,
		}).Error; err != nil {
			return santaErr("santaRepo.LinkTelegram participant", err)
		}
		if err := tx.Delete(&SantaTgLinkModel{}, "participant_id = ?", link.ParticipantID).Error; err != nil {
			return santaErr("santaRepo.LinkTelegram cleanup", err)
		}
		var m SantaParticipantModel
		if err := tx.First(&m, "id = ?", link.ParticipantID).Error; err != nil {
			return santaErr("santaRepo.LinkTelegram reload", err)
		}
		out = toSantaParticipantEntity(m)
		return insertNotifications(tx, welcome(out))
	})
	if err != nil {
		return entity.SantaParticipant{}, err
	}
	return out, nil
}

func (r *santaRepo) Remind(ctx context.Context, roomID uuid.UUID, now time.Time, cooldown time.Duration, notes []entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var room SantaRoomModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return santaErr("santaRepo.Remind lock", err)
		}
		if room.LastRemindedAt != nil && room.LastRemindedAt.After(now.Add(-cooldown)) {
			return repo.ErrTooSoon
		}
		if err := tx.Model(&SantaRoomModel{}).Where("id = ?", roomID).
			Update("last_reminded_at", now).Error; err != nil {
			return santaErr("santaRepo.Remind mark", err)
		}
		return insertNotifications(tx, notes...)
	})
}

func (r *santaRepo) ClaimNotifications(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]entity.SantaNotification, error) {
	var models []SantaNotificationModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND next_try_at <= ?", string(entity.SantaNotificationPending), now).
			Order("next_try_at, id").
			Limit(limit).
			Find(&models).Error; err != nil {
			return santaErr("santaRepo.ClaimNotifications", err)
		}
		if len(models) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, len(models))
		for i, m := range models {
			ids[i] = m.ID
		}
		if err := tx.Model(&SantaNotificationModel{}).Where("id IN ?", ids).
			Update("next_try_at", now.Add(lease)).Error; err != nil {
			return santaErr("santaRepo.ClaimNotifications lease", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]entity.SantaNotification, len(models))
	for i, m := range models {
		out[i] = toSantaNotificationEntity(m)
	}
	return out, nil
}

// markPending меняет только ещё pending-уведомление: отмеченное другим
// обработчиком или стёртое при схлопывании не трогаем — ErrNotFound.
func (r *santaRepo) markPending(ctx context.Context, op string, id uuid.UUID, updates map[string]any) error {
	res := r.db.WithContext(ctx).Model(&SantaNotificationModel{}).
		Where("id = ? AND status = ?", id, string(entity.SantaNotificationPending)).
		Updates(updates)
	if res.Error != nil {
		return santaErr(op, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, repo.ErrNotFound)
	}
	return nil
}

func (r *santaRepo) MarkNotificationSent(ctx context.Context, id uuid.UUID, tgMessageID *int64) error {
	return r.markPending(ctx, "santaRepo.MarkNotificationSent", id, map[string]any{
		"status":        string(entity.SantaNotificationSent),
		"last_error":    "",
		"tg_message_id": tgMessageID,
	})
}

func (r *santaRepo) MarkNotificationFailed(ctx context.Context, id uuid.UUID, attempts int, retryAt *time.Time, lastErr string) error {
	updates := map[string]any{"attempts": attempts, "last_error": lastErr}
	if retryAt == nil {
		updates["status"] = string(entity.SantaNotificationFailed)
	} else {
		updates["status"] = string(entity.SantaNotificationPending)
		updates["next_try_at"] = *retryAt
	}
	return r.markPending(ctx, "santaRepo.MarkNotificationFailed", id, updates)
}

func (r *santaRepo) PurgeStale(ctx context.Context, now time.Time, keepNotes time.Duration) (int64, error) {
	db := r.db.WithContext(ctx)
	done := []string{string(entity.SantaNotificationSent), string(entity.SantaNotificationFailed)}
	steps := []struct {
		op    string
		model any
		where string
		args  []any
	}{
		// Ссылка и код с истёкшим сроком уже ничего не подтверждают.
		{"links", &SantaTgLinkModel{}, "expires_at <= ?", []any{now}},
		{"codes", &SantaEmailCodeModel{}, "expires_at <= ?", []any{now}},
		// pending не трогаем: они ещё в работе, сколько бы ни ждали.
		{"notifications", &SantaNotificationModel{}, "status IN ? AND created_at < ?", []any{done, now.Add(-keepNotes)}},
	}
	var total int64
	for _, s := range steps {
		res := db.Where(s.where, s.args...).Delete(s.model)
		if res.Error != nil {
			return total, santaErr("santaRepo.PurgeStale "+s.op, res.Error)
		}
		total += res.RowsAffected
	}
	return total, nil
}
