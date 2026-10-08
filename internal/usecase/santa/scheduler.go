package santa

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
)

// scheduleBatch — сколько созревших комнат разыгрывать за тик.
const scheduleBatch = 20

// Scheduler проводит жеребьёвки по draw_at. Несколько экземпляров бэка друг
// другу не мешают: репозиторий берёт комнату FOR UPDATE SKIP LOCKED и
// перепроверяет статус и время.
type Scheduler struct {
	santa   repo.SantaRepo
	shuffle shuffleFunc
	now     func() time.Time
}

func NewScheduler(santaRepo repo.SantaRepo) *Scheduler {
	return &Scheduler{santa: santaRepo, shuffle: cryptoShuffle, now: time.Now}
}

// Run крутит RunOnce каждые tick до отмены ctx.
func (s *Scheduler) Run(ctx context.Context, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		if _, err := s.RunOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("santa scheduler: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce разыгрывает созревшие комнаты; сбой одной не мешает остальным.
// Возвращает, сколько комнат разыграно.
func (s *Scheduler) RunOnce(ctx context.Context) (int, error) {
	now := s.now()
	ids, err := s.santa.DueDrawRooms(ctx, now, scheduleBatch)
	if err != nil {
		return 0, err
	}
	drawn := 0
	for _, roomID := range ids {
		if ctx.Err() != nil {
			break
		}
		outcome, err := s.santa.DrawScheduled(ctx, roomID, now, minParticipants,
			func(ids []uuid.UUID) ([]entity.SantaAssignment, error) {
				return buildCycle(roomID, ids, s.shuffle)
			},
			func(giverID uuid.UUID) entity.SantaNotification {
				return entity.NewSantaNotification(giverID, entity.SantaNotifyDrawn, now)
			},
			func(organizerID uuid.UUID) entity.SantaNotification {
				return entity.NewSantaNotification(organizerID, entity.SantaNotifyDrawFailed, now)
			},
		)
		switch {
		case err != nil:
			log.Printf("santa scheduler: room %s: %v", roomID, err)
		case outcome == repo.ScheduledDrawDone:
			drawn++
		case outcome == repo.ScheduledDrawTooFew:
			log.Printf("santa scheduler: room %s: мало готовых участников, время жеребьёвки снято", roomID)
		}
	}
	return drawn, nil
}
