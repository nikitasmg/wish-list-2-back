package santa

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

const remindCooldown = 12 * time.Hour

// Remind — напоминание тем, у кого пусто и в пожеланиях, и в вишлисте. Без
// подтверждённого канала сообщение не дойдёт — таких только считаем, их
// организатор видит в списке и зовёт сам.
func (uc *santaUseCase) Remind(ctx context.Context, ownerID, roomID uuid.UUID) (usecase.SantaRemindResult, error) {
	room, err := uc.ownedRoom(ctx, ownerID, roomID)
	if err != nil {
		return usecase.SantaRemindResult{}, err
	}
	now := uc.now()
	if room.LastRemindedAt != nil && now.Sub(*room.LastRemindedAt) < remindCooldown {
		return usecase.SantaRemindResult{}, usecase.ErrSantaTooSoon
	}
	ps, err := uc.santa.ListParticipants(ctx, room.ID)
	if err != nil {
		return usecase.SantaRemindResult{}, err
	}
	var res usecase.SantaRemindResult
	var notes []entity.SantaNotification
	for _, p := range ps {
		if !p.Ready() {
			res.Unreachable++
			continue
		}
		if p.Wishes == "" && p.WishlistURL == "" {
			notes = append(notes, entity.NewSantaNotification(p.ID, entity.SantaNotifyReminderFill, now))
			res.Sent++
		}
	}
	// Некому слать — время напоминания не тратим.
	if len(notes) == 0 {
		return res, nil
	}
	if err := uc.santa.Remind(ctx, room.ID, now, remindCooldown, notes); err != nil {
		if errors.Is(err, repo.ErrTooSoon) {
			return usecase.SantaRemindResult{}, usecase.ErrSantaTooSoon
		}
		return usecase.SantaRemindResult{}, err
	}
	return res, nil
}
