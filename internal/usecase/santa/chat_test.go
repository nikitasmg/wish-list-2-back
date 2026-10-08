package santa

import (
	"context"
	"encoding/json"
	"strings"
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

type chatEnv struct {
	uc        *santaUseCase
	sr        *mockrepo.MockSantaRepo
	tg        *fakeTG
	room      entity.SantaRoom
	p         entity.SantaParticipant
	toWard    entity.SantaAssignment // p дарит
	fromSanta entity.SantaAssignment // p получает
}

// newChatEnv — разыгранная комната «abcdefgh», участник p по токену «tok».
// Пары (GetAssignment/GetGiver) каждый тест задаёт сам.
func newChatEnv(t *testing.T) chatEnv {
	t.Helper()
	sr := new(mockrepo.MockSantaRepo)
	tg := &fakeTG{}
	uc := New(sr, new(mockrepo.MockUserRepo), WithTelegram(tg, "santa_namekni_bot")).(*santaUseCase)
	uc.now = func() time.Time { return chNow }
	room := entity.SantaRoom{ID: uuid.New(), Slug: "abcdefgh", Title: "Офис", Status: entity.SantaRoomDrawn}
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Аня", TokenHash: hashToken("tok")}
	sr.On("GetRoomBySlug", mock.Anything, "abcdefgh").Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, hashToken("tok")).Return(p, nil)
	return chatEnv{
		uc: uc, sr: sr, tg: tg, room: room, p: p,
		toWard:    entity.SantaAssignment{RoomID: room.ID, GiverID: p.ID, ReceiverID: uuid.New()},
		fromSanta: entity.SantaAssignment{RoomID: room.ID, GiverID: uuid.New(), ReceiverID: p.ID},
	}
}

func (e chatEnv) withPairs() chatEnv {
	e.sr.On("GetAssignment", mock.Anything, e.room.ID, e.p.ID).Return(e.toWard, nil)
	e.sr.On("GetGiver", mock.Anything, e.room.ID, e.p.ID).Return(e.fromSanta, nil)
	return e
}

func TestGetChat_ReceiverThread(t *testing.T) {
	e := newChatEnv(t).withPairs()
	ward := e.toWard.ReceiverID
	e.sr.On("ListMessages", mock.Anything, e.room.ID, e.p.ID, ward, chatHistory).Return([]entity.SantaMessage{
		{ID: uuid.New(), GiverID: e.p.ID, ReceiverID: ward, FromGiver: true, Body: "Какой размер?", CreatedAt: chNow},
		{ID: uuid.New(), GiverID: e.p.ID, ReceiverID: ward, FromGiver: false, Body: "M", CreatedAt: chNow},
	}, nil)
	e.sr.On("MarkMessagesRead", mock.Anything, e.room.ID, e.p.ID, ward, false, chNow).Return(nil)

	chat, err := e.uc.GetChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatReceiver)
	require.NoError(t, err)
	assert.Equal(t, usecase.SantaChatReceiver, chat.With)
	require.Len(t, chat.Messages, 2)
	assert.True(t, chat.Messages[0].Mine)
	assert.False(t, chat.Messages[1].Mine)
	e.sr.AssertCalled(t, "MarkMessagesRead", mock.Anything, e.room.ID, e.p.ID, ward, false, chNow)
}

func TestGetChat_SantaThreadHidesGiver(t *testing.T) {
	e := newChatEnv(t).withPairs()
	santa := e.fromSanta.GiverID
	e.sr.On("ListMessages", mock.Anything, e.room.ID, santa, e.p.ID, chatHistory).Return([]entity.SantaMessage{
		{ID: uuid.New(), RoomID: e.room.ID, GiverID: santa, ReceiverID: e.p.ID, FromGiver: true, Body: "Привет от Санты", CreatedAt: chNow},
	}, nil)
	e.sr.On("MarkMessagesRead", mock.Anything, e.room.ID, santa, e.p.ID, true, chNow).Return(nil)

	chat, err := e.uc.GetChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatSanta)
	require.NoError(t, err)
	require.Len(t, chat.Messages, 1)
	assert.False(t, chat.Messages[0].Mine)
	raw, err := json.Marshal(chat)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), santa.String(), "id Санты не уходит подопечному")
}

func TestGetChat_BeforeDraw(t *testing.T) {
	uc, _, _, _, _, _ := channelUC(t) // комната open
	_, err := uc.GetChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatSanta)
	assert.ErrorIs(t, err, usecase.ErrSantaNotDrawn)
}

func TestGetChat_NotInDraw(t *testing.T) {
	e := newChatEnv(t)
	e.sr.On("GetGiver", mock.Anything, e.room.ID, e.p.ID).Return(entity.SantaAssignment{}, repo.ErrNotFound)
	_, err := e.uc.GetChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatSanta)
	assert.ErrorIs(t, err, usecase.ErrSantaNotInDraw)
}

func TestGetChat_BadWith(t *testing.T) {
	e := newChatEnv(t)
	_, err := e.uc.GetChat(context.Background(), "abcdefgh", tokAuth, "everyone")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestSendChat_ToReceiver(t *testing.T) {
	e := newChatEnv(t).withPairs()
	ward := e.toWard.ReceiverID
	var saved entity.SantaMessage
	var note entity.SantaNotification
	e.sr.On("CreateMessage", mock.Anything, mock.Anything, chNow.Add(-time.Hour), chatPerHour, mock.Anything).Run(func(a mock.Arguments) {
		saved = a.Get(1).(entity.SantaMessage)
		note = a.Get(4).(entity.SantaNotification)
	}).Return(nil)

	got, err := e.uc.SendChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatReceiver, "  Какой размер?  ")
	require.NoError(t, err)
	assert.True(t, got.Mine)
	assert.Equal(t, "Какой размер?", got.Body)
	assert.Equal(t, e.p.ID, saved.GiverID)
	assert.Equal(t, ward, saved.ReceiverID)
	assert.True(t, saved.FromGiver)
	assert.Equal(t, chNow, saved.CreatedAt)
	assert.Equal(t, ward, note.ParticipantID)
	assert.Equal(t, entity.SantaNotifyChatMessage, note.Kind)
	assert.Equal(t, saved.ID.String(), note.Payload[entity.SantaPayloadMessageID])
}

func TestSendChat_Validation(t *testing.T) {
	e := newChatEnv(t).withPairs()
	for _, body := range []string{"", "   ", strings.Repeat("я", maxChatBody+1)} {
		_, err := e.uc.SendChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatSanta, body)
		assert.ErrorIs(t, err, usecase.ErrSantaInvalid, "длина %d", len([]rune(body)))
	}
	e.sr.AssertNotCalled(t, "CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSendChat_RepoErrors(t *testing.T) {
	for repoErr, want := range map[error]error{
		repo.ErrTooSoon:        usecase.ErrSantaChatLimit,
		repo.ErrNotFound:       usecase.ErrSantaNotInDraw,
		repo.ErrStatusMismatch: usecase.ErrSantaNotDrawn,
	} {
		e := newChatEnv(t).withPairs()
		e.sr.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(repoErr)
		_, err := e.uc.SendChat(context.Background(), "abcdefgh", tokAuth, usecase.SantaChatSanta, "Спасибо!")
		assert.ErrorIs(t, err, want, repoErr.Error())
	}
}

// replyEnv — уведомление о сообщении Санты (santa → ward) ушло подопечному в
// чат 77 сообщением 500.
func replyEnv(t *testing.T) (chatEnv, entity.SantaMessage) {
	e := newChatEnv(t)
	santa, ward := uuid.New(), uuid.New()
	orig := entity.SantaMessage{ID: uuid.New(), RoomID: e.room.ID, GiverID: santa, ReceiverID: ward, FromGiver: true, Body: "Привет", CreatedAt: chNow}
	note := entity.NewSantaNotification(ward, entity.SantaNotifyChatMessage, chNow)
	note.Payload[entity.SantaPayloadMessageID] = orig.ID.String()
	e.sr.On("FindChatNotification", mock.Anything, int64(77), int64(500)).Return(note, nil)
	return e, orig
}

func TestTelegramReply_FromWardToSanta(t *testing.T) {
	e, orig := replyEnv(t)
	e.sr.On("GetMessage", mock.Anything, orig.ID).Return(orig, nil)
	e.sr.On("CreateMessage", mock.Anything, mock.MatchedBy(func(m entity.SantaMessage) bool {
		return !m.FromGiver && m.GiverID == orig.GiverID && m.ReceiverID == orig.ReceiverID && m.Body == "Спасибо!"
	}), chNow.Add(-time.Hour), chatPerHour, mock.MatchedBy(func(n entity.SantaNotification) bool {
		return n.ParticipantID == orig.GiverID && n.Kind == entity.SantaNotifyChatMessage
	})).Return(nil)

	require.NoError(t, e.uc.TelegramReply(context.Background(), 77, 500, "Спасибо!"))
	assert.Equal(t, botChatSentText(), e.tg.text)
	assert.EqualValues(t, 77, e.tg.chatID)
	e.sr.AssertNumberOfCalls(t, "CreateMessage", 1)
}

func TestTelegramReply_UnknownMessage(t *testing.T) {
	e := newChatEnv(t)
	e.sr.On("FindChatNotification", mock.Anything, int64(77), int64(9)).Return(entity.SantaNotification{}, repo.ErrNotFound)
	require.NoError(t, e.uc.TelegramReply(context.Background(), 77, 9, "Спасибо!"))
	assert.Equal(t, botChatUnknownText(), e.tg.text)
	e.sr.AssertNotCalled(t, "CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTelegramReply_AfterRedrawIsClosed(t *testing.T) {
	e, orig := replyEnv(t)
	e.sr.On("GetMessage", mock.Anything, orig.ID).Return(entity.SantaMessage{}, repo.ErrNotFound) // перезапуск стёр переписку
	require.NoError(t, e.uc.TelegramReply(context.Background(), 77, 500, "Спасибо!"))
	assert.Equal(t, botChatClosedText(), e.tg.text)
	e.sr.AssertNotCalled(t, "CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestTelegramReply_Limit(t *testing.T) {
	e, orig := replyEnv(t)
	e.sr.On("GetMessage", mock.Anything, orig.ID).Return(orig, nil)
	e.sr.On("CreateMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(repo.ErrTooSoon)
	require.NoError(t, e.uc.TelegramReply(context.Background(), 77, 500, "Спасибо!"))
	assert.Equal(t, botChatLimitText(), e.tg.text)
}
