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
		res := tx.Model(&SantaParticipantModel{}).Where("id = ?", participantID).Updates(map[string]any{
			"email":             email,
			"email_verified_at": nil,
			"updated_at":        time.Now(),
		})
		if res.Error != nil {
			if isUniqueViolation(res.Error, "idx_santa_participant_email") {
				return fmt.Errorf("santaRepo.SetEmail: %w", repo.ErrDuplicate)
			}
			return santaErr("santaRepo.SetEmail", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.SetEmail: %w", repo.ErrNotFound)
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

func (r *santaRepo) IncEmailCodeAttempts(ctx context.Context, participantID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&SantaEmailCodeModel{}).
		Where("participant_id = ?", participantID).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error; err != nil {
		return santaErr("santaRepo.IncEmailCodeAttempts", err)
	}
	return nil
}

func (r *santaRepo) DeleteEmailCode(ctx context.Context, participantID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&SantaEmailCodeModel{}, "participant_id = ?", participantID).Error; err != nil {
		return santaErr("santaRepo.DeleteEmailCode", err)
	}
	return nil
}

func (r *santaRepo) VerifyEmail(ctx context.Context, participantID uuid.UUID, at time.Time, welcome entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&SantaParticipantModel{}).Where("id = ?", participantID).Updates(map[string]any{
			"email_verified_at": at,
			"channel":           string(entity.SantaChannelEmail),
			"updated_at":        at,
		})
		if res.Error != nil {
			return santaErr("santaRepo.VerifyEmail", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.VerifyEmail: %w", repo.ErrNotFound)
		}
		if err := tx.Delete(&SantaEmailCodeModel{}, "participant_id = ?", participantID).Error; err != nil {
			return santaErr("santaRepo.VerifyEmail code", err)
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
