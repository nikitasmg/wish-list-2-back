package santa_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

type buildFunc = func([]uuid.UUID) ([]entity.SantaAssignment, error)

func TestDraw_StrangerIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)

	err := uc.Draw(ctx, uuid.New(), room.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
	sr.AssertNotCalled(t, "Draw", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDraw_BuildsOneCircle(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	var build buildFunc
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("Draw", mock.Anything, room.ID, entity.SantaRoomOpen, mock.Anything).Run(func(args mock.Arguments) {
		build = args.Get(3).(buildFunc)
	}).Return(nil)

	require.NoError(t, uc.Draw(ctx, owner, room.ID))

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	pairs, err := build(ids)
	require.NoError(t, err)
	require.Len(t, pairs, 4)
	for _, p := range pairs {
		assert.Equal(t, room.ID, p.RoomID)
		assert.NotEqual(t, p.GiverID, p.ReceiverID)
	}

	_, err = build(ids[:2])
	assert.ErrorIs(t, err, usecase.ErrSantaTooFew)
}

func TestDraw_SecondCallIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("Draw", mock.Anything, room.ID, entity.SantaRoomOpen, mock.Anything).Return(repo.ErrStatusMismatch)

	err := uc.Draw(ctx, owner, room.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
}

func TestRedraw_BeforeDrawIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("Draw", mock.Anything, room.ID, entity.SantaRoomDrawn, mock.Anything).Return(repo.ErrStatusMismatch)

	err := uc.Redraw(ctx, owner, room.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaNotDrawn)
}
