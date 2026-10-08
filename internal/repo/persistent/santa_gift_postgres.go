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

func (r *santaRepo) SetGiftReady(ctx context.Context, roomID, participantID uuid.UUID, ready bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// FOR SHARE против FOR UPDATE жеребьёвки: перезапуск сбросит отметку
		// после нас или мы увидим новые пары.
		var room SantaRoomModel
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return santaErr("santaRepo.SetGiftReady lock", err)
		}
		if room.Status != string(entity.SantaRoomDrawn) {
			return repo.ErrStatusMismatch
		}
		res := tx.Model(&SantaParticipantModel{}).
			Where("id = ? AND room_id = ?", participantID, roomID).
			Where("EXISTS (SELECT 1 FROM santa_assignments a WHERE a.room_id = ? AND a.giver_id = ?)", roomID, participantID).
			Updates(map[string]any{"gift_ready": ready, "updated_at": time.Now()})
		if res.Error != nil {
			return santaErr("santaRepo.SetGiftReady", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.SetGiftReady: %w", repo.ErrNotFound)
		}
		return nil
	})
}
