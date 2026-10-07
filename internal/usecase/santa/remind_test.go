package santa

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	mockrepo "main/mock/repo"
)

func remindUC(t *testing.T, room entity.SantaRoom, ps []entity.SantaParticipant) (*santaUseCase, *mockrepo.MockSantaRepo) {
	t.Helper()
	sr := new(mockrepo.MockSantaRepo)
	uc := New(sr, new(mockrepo.MockUserRepo)).(*santaUseCase)
	uc.now = func() time.Time { return chNow }
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("ListParticipants", mock.Anything, room.ID).Return(ps, nil)
	return uc, sr
}

func readyP(roomID uuid.UUID, wishes string) entity.SantaParticipant {
	chat := int64(1)
	return entity.SantaParticipant{ID: uuid.New(), RoomID: roomID, Name: "x", Wishes: wishes, Channel: entity.SantaChannelTelegram, TgChatID: &chat}
}

func TestRemind_OnlyReadyWithoutWishes(t *testing.T) {
	owner := uuid.New()
	room := entity.SantaRoom{ID: uuid.New(), OwnerID: owner, Status: entity.SantaRoomOpen}
	empty := readyP(room.ID, "")
	filled := readyP(room.ID, "книги")
	unready := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID}
	uc, sr := remindUC(t, room, []entity.SantaParticipant{empty, filled, unready})
	sr.On("Remind", mock.Anything, room.ID, chNow, remindCooldown, mock.MatchedBy(func(ns []entity.SantaNotification) bool {
		return len(ns) == 1 && ns[0].ParticipantID == empty.ID && ns[0].Kind == entity.SantaNotifyReminderFill
	})).Return(nil)

	res, err := uc.Remind(context.Background(), owner, room.ID)
	require.NoError(t, err)
	assert.Equal(t, usecase.SantaRemindResult{Sent: 1, Unreachable: 1}, res)
}

func TestRemind_NobodyToRemindKeepsCooldown(t *testing.T) {
	owner := uuid.New()
	room := entity.SantaRoom{ID: uuid.New(), OwnerID: owner, Status: entity.SantaRoomOpen}
	uc, sr := remindUC(t, room, []entity.SantaParticipant{readyP(room.ID, "книги")})
	res, err := uc.Remind(context.Background(), owner, room.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Sent)
	sr.AssertNotCalled(t, "Remind", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRemind_TooSoon(t *testing.T) {
	owner := uuid.New()
	last := chNow.Add(-time.Hour)
	room := entity.SantaRoom{ID: uuid.New(), OwnerID: owner, Status: entity.SantaRoomOpen, LastRemindedAt: &last}
	uc, sr := remindUC(t, room, nil)
	_, err := uc.Remind(context.Background(), owner, room.ID)
	assert.ErrorIs(t, err, usecase.ErrSantaTooSoon)
	sr.AssertNotCalled(t, "ListParticipants", mock.Anything, mock.Anything)
}

func TestRemind_RaceTooSoonFromRepo(t *testing.T) {
	owner := uuid.New()
	room := entity.SantaRoom{ID: uuid.New(), OwnerID: owner, Status: entity.SantaRoomOpen}
	uc, sr := remindUC(t, room, []entity.SantaParticipant{readyP(room.ID, "")})
	sr.On("Remind", mock.Anything, room.ID, chNow, remindCooldown, mock.Anything).Return(repo.ErrTooSoon)
	_, err := uc.Remind(context.Background(), owner, room.ID)
	assert.ErrorIs(t, err, usecase.ErrSantaTooSoon)
}

func TestRemind_StrangerIsNotFound(t *testing.T) {
	room := entity.SantaRoom{ID: uuid.New(), OwnerID: uuid.New(), Status: entity.SantaRoomOpen}
	uc, _ := remindUC(t, room, nil)
	_, err := uc.Remind(context.Background(), uuid.New(), room.ID)
	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}
