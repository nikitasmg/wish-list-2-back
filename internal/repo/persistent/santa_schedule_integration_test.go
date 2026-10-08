//go:build integration

package persistent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/repo/persistent"
)

func setDrawAt(t *testing.T, db *gorm.DB, roomID uuid.UUID, at time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&persistent.SantaRoomModel{}).Where("id = ?", roomID).Update("draw_at", at).Error)
}

// seedOwner — организатор-участник комнаты; ready — с подтверждённым Telegram.
func seedOwner(t *testing.T, r repo.SantaRepo, room entity.SantaRoom, ready bool) uuid.UUID {
	t.Helper()
	owner := room.OwnerID
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, UserID: &owner, Name: "Организатор", TokenHash: uuid.NewString()}
	if ready {
		chatSeq++
		chat := chatSeq
		p.Channel, p.TgChatID = entity.SantaChannelTelegram, &chat
	}
	require.NoError(t, r.CreateParticipant(context.Background(), p))
	return p.ID
}

func failNote(now time.Time) func(uuid.UUID) entity.SantaNotification {
	return func(id uuid.UUID) entity.SantaNotification {
		return entity.NewSantaNotification(id, entity.SantaNotifyDrawFailed, now)
	}
}

func drawScheduled(r repo.SantaRepo, roomID uuid.UUID, now time.Time) (repo.ScheduledDrawOutcome, error) {
	return r.DrawScheduled(context.Background(), roomID, now, 3, circle(roomID), drawnNote(now), failNote(now))
}

func TestSantaRepo_DueDrawRooms(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	due := seedRoom(t, r, uuid.New())
	setDrawAt(t, db, due.ID, now.Add(-time.Minute))
	future := seedRoom(t, r, uuid.New())
	setDrawAt(t, db, future.ID, now.Add(time.Hour))
	drawn := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, drawn.ID, 3)
	require.NoError(t, r.Draw(ctx, drawn.ID, entity.SantaRoomOpen, circle(drawn.ID), drawnNote(now)))
	setDrawAt(t, db, drawn.ID, now.Add(-time.Minute))
	seedRoom(t, r, uuid.New()) // без времени

	ids, err := r.DueDrawRooms(ctx, now, 10)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{due.ID}, ids)
}

func TestSantaRepo_DrawScheduledDraws(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	setDrawAt(t, db, room.ID, now.Add(-time.Minute))

	outcome, err := drawScheduled(r, room.ID, now)
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawDone, outcome)
	got, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.SantaRoomDrawn, got.Status)
	assert.Nil(t, got.DrawFailedAt)
	for _, id := range ids {
		assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyDrawn))
	}
}

func TestSantaRepo_DrawScheduledOnce(t *testing.T) {
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	setDrawAt(t, db, room.ID, now.Add(-time.Minute))

	// Два экземпляра бэка в один тик.
	var wg sync.WaitGroup
	outcomes := make([]repo.ScheduledDrawOutcome, 2)
	for i := range outcomes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			o, err := drawScheduled(r, room.ID, now)
			assert.NoError(t, err)
			outcomes[i] = o
		}(i)
	}
	wg.Wait()
	assert.ElementsMatch(t, []repo.ScheduledDrawOutcome{repo.ScheduledDrawDone, repo.ScheduledDrawSkipped}, outcomes)

	again, err := drawScheduled(r, room.ID, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawSkipped, again, "разыгранную комнату не трогаем")
	for _, id := range ids {
		assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyDrawn), "одно «кому дарить»")
	}
}

func TestSantaRepo_DrawScheduledTooFew(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	owner := seedOwner(t, r, room, true)
	seedParticipants(t, r, room.ID, 1)
	seedUnready(t, r, room.ID, 2)
	setDrawAt(t, db, room.ID, now.Add(-time.Minute))

	outcome, err := drawScheduled(r, room.ID, now)
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawTooFew, outcome)
	got, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.SantaRoomOpen, got.Status)
	assert.Nil(t, got.DrawAt, "время снято — планировщик не повторяет каждую минуту")
	require.NotNil(t, got.DrawFailedAt)
	assert.WithinDuration(t, now, *got.DrawFailedAt, time.Second)
	assert.EqualValues(t, 1, countNotes(t, db, owner, entity.SantaNotifyDrawFailed))
	_, err = r.GetAssignment(ctx, room.ID, owner)
	assert.ErrorIs(t, err, repo.ErrNotFound)

	again, err := drawScheduled(r, room.ID, now.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawSkipped, again)
	assert.EqualValues(t, 1, countNotes(t, db, owner, entity.SantaNotifyDrawFailed), "предупреждение одно")
}

func TestSantaRepo_DrawScheduledTooFewOwnerUnready(t *testing.T) {
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	owner := seedOwner(t, r, room, false)
	seedParticipants(t, r, room.ID, 2)
	setDrawAt(t, db, room.ID, now.Add(-time.Minute))

	outcome, err := drawScheduled(r, room.ID, now)
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawTooFew, outcome)
	assert.EqualValues(t, 0, countNotes(t, db, owner, entity.SantaNotifyDrawFailed), "без канала слать некуда — только плашка")
}

func TestSantaRepo_DrawScheduledNotDueYet(t *testing.T) {
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	setDrawAt(t, db, room.ID, now.Add(time.Hour)) // организатор перенёс время

	outcome, err := drawScheduled(r, room.ID, now)
	require.NoError(t, err)
	assert.Equal(t, repo.ScheduledDrawSkipped, outcome)
}

func TestSantaRepo_DrawClearsDrawFailed(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	require.NoError(t, db.Model(&persistent.SantaRoomModel{}).Where("id = ?", room.ID).Update("draw_failed_at", now).Error)

	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(now)))
	got, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Nil(t, got.DrawFailedAt)
}

func TestSantaRepo_UpdateRoomWritesDrawFailed(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	room := seedRoom(t, r, uuid.New())
	require.NoError(t, db.Model(&persistent.SantaRoomModel{}).Where("id = ?", room.ID).Update("draw_failed_at", now).Error)

	got, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	got.Title = "Новое"
	require.NoError(t, r.UpdateRoom(ctx, got))
	kept, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.NotNil(t, kept.DrawFailedAt, "правка названия плашку не снимает")

	kept.DrawFailedAt = nil
	require.NoError(t, r.UpdateRoom(ctx, kept))
	cleared, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Nil(t, cleared.DrawFailedAt)
}
