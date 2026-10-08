package santa

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	mockrepo "main/mock/repo"
)

func newSchedulerEnv() (*Scheduler, *mockrepo.MockSantaRepo) {
	sr := new(mockrepo.MockSantaRepo)
	s := NewScheduler(sr)
	s.now = func() time.Time { return chNow }
	return s, sr
}

func TestScheduler_DrawsDueRooms(t *testing.T) {
	s, sr := newSchedulerEnv()
	a, b := uuid.New(), uuid.New()
	sr.On("DueDrawRooms", mock.Anything, chNow, scheduleBatch).Return([]uuid.UUID{a, b}, nil)
	sr.On("DrawScheduled", mock.Anything, a, chNow, minParticipants, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		build := args.Get(4).(func([]uuid.UUID) ([]entity.SantaAssignment, error))
		pairs, err := build([]uuid.UUID{uuid.New(), uuid.New(), uuid.New()})
		require.NoError(t, err)
		require.Len(t, pairs, 3)
		assert.Equal(t, a, pairs[0].RoomID, "пары своей комнаты, а не последней в цикле")
		drawn := args.Get(5).(func(uuid.UUID) entity.SantaNotification)(uuid.New())
		assert.Equal(t, entity.SantaNotifyDrawn, drawn.Kind)
		failed := args.Get(6).(func(uuid.UUID) entity.SantaNotification)(uuid.New())
		assert.Equal(t, entity.SantaNotifyDrawFailed, failed.Kind)
	}).Return(repo.ScheduledDrawDone, nil)
	sr.On("DrawScheduled", mock.Anything, b, chNow, minParticipants, mock.Anything, mock.Anything, mock.Anything).Return(repo.ScheduledDrawTooFew, nil)

	drawn, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, drawn)
	sr.AssertExpectations(t)
}

func TestScheduler_OneRoomErrorDoesNotStopOthers(t *testing.T) {
	s, sr := newSchedulerEnv()
	a, b := uuid.New(), uuid.New()
	sr.On("DueDrawRooms", mock.Anything, chNow, scheduleBatch).Return([]uuid.UUID{a, b}, nil)
	sr.On("DrawScheduled", mock.Anything, a, chNow, minParticipants, mock.Anything, mock.Anything, mock.Anything).Return(repo.ScheduledDrawSkipped, errors.New("deadlock"))
	sr.On("DrawScheduled", mock.Anything, b, chNow, minParticipants, mock.Anything, mock.Anything, mock.Anything).Return(repo.ScheduledDrawDone, nil)

	drawn, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, drawn)
}

func TestScheduler_DueError(t *testing.T) {
	s, sr := newSchedulerEnv()
	sr.On("DueDrawRooms", mock.Anything, chNow, scheduleBatch).Return(nil, errors.New("db down"))
	_, err := s.RunOnce(context.Background())
	assert.Error(t, err)
}
