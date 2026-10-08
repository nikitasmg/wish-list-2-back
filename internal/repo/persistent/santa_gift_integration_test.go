//go:build integration

package persistent_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/repo/persistent"
)

func TestSantaRepo_SetGiftReady(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)

	require.NoError(t, r.SetGiftReady(ctx, room.ID, ids[0], true))
	p, err := r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	assert.True(t, p.GiftReady)

	require.NoError(t, r.SetGiftReady(ctx, room.ID, ids[0], false))
	p, err = r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	assert.False(t, p.GiftReady, "отметку можно снять")
}

func TestSantaRepo_SetGiftReadyNeedsPair(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	late := seedUnready(t, r, room.ID, 1)[0] // без канала — в жеребьёвку не попал, пары нет
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(time.Now())))

	assert.ErrorIs(t, r.SetGiftReady(ctx, room.ID, late, true), repo.ErrNotFound)

	open := seedRoom(t, r, uuid.New())
	pid := seedParticipants(t, r, open.ID, 1)[0]
	assert.ErrorIs(t, r.SetGiftReady(ctx, open.ID, pid, true), repo.ErrStatusMismatch, "до жеребьёвки")
}

func TestSantaRepo_RedrawResetsGiftReady(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)
	require.NoError(t, r.SetGiftReady(ctx, room.ID, ids[0], true))

	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomDrawn, circle(room.ID), drawnNote(time.Now())))

	p, err := r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	assert.False(t, p.GiftReady, "новый подопечный — подарка для него ещё нет")
}
