package santa

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	mockrepo "main/mock/repo"
)

func TestSetGiftReady_Marks(t *testing.T) {
	e := newChatEnv(t)
	e.sr.On("SetGiftReady", mock.Anything, e.room.ID, e.p.ID, true).Return(nil)
	// me(): карточка с подопечным после отметки.
	e.sr.On("CountParticipants", mock.Anything, mock.Anything).Return(map[uuid.UUID]int{e.room.ID: 3}, nil)
	e.uc.users.(*mockrepo.MockUserRepo).On("GetByID", mock.Anything, mock.Anything).Return(entity.User{}, repo.ErrNotFound)
	e.sr.On("GetAssignment", mock.Anything, e.room.ID, e.p.ID).Return(e.toWard, nil)
	e.sr.On("GetParticipant", mock.Anything, e.toWard.ReceiverID).Return(entity.SantaParticipant{ID: e.toWard.ReceiverID, Name: "Маша"}, nil)
	e.sr.On("CountUnread", mock.Anything, e.room.ID, e.p.ID).Return(0, 0, nil)

	me, err := e.uc.SetGiftReady(context.Background(), "abcdefgh", tokAuth, true)
	require.NoError(t, err)
	assert.True(t, me.GiftReady)
	e.sr.AssertCalled(t, "SetGiftReady", mock.Anything, e.room.ID, e.p.ID, true)
}

func TestSetGiftReady_NotInDraw(t *testing.T) {
	e := newChatEnv(t)
	e.sr.On("SetGiftReady", mock.Anything, e.room.ID, e.p.ID, true).Return(repo.ErrNotFound)
	_, err := e.uc.SetGiftReady(context.Background(), "abcdefgh", tokAuth, true)
	assert.ErrorIs(t, err, usecase.ErrSantaNotInDraw)
}

func TestSetGiftReady_BeforeDraw(t *testing.T) {
	uc, _, _, _, _, _ := channelUC(t) // комната open
	_, err := uc.SetGiftReady(context.Background(), "abcdefgh", tokAuth, true)
	assert.ErrorIs(t, err, usecase.ErrSantaNotDrawn)
}

func TestSetGiftReady_RedrawMeanwhile(t *testing.T) {
	e := newChatEnv(t)
	e.sr.On("SetGiftReady", mock.Anything, e.room.ID, e.p.ID, true).Return(repo.ErrStatusMismatch)
	_, err := e.uc.SetGiftReady(context.Background(), "abcdefgh", tokAuth, true)
	assert.ErrorIs(t, err, usecase.ErrSantaNotDrawn)
}
