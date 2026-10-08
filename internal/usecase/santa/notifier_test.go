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
	"main/pkg/telegram"
	mockrepo "main/mock/repo"
)

type notifierEnv struct {
	n     *Notifier
	sr    *mockrepo.MockSantaRepo
	ml    *fakeMailer
	tg    *fakeTG
	room  entity.SantaRoom
	giver entity.SantaParticipant
	ward  entity.SantaParticipant
}

func newNotifierEnv(t *testing.T) notifierEnv {
	t.Helper()
	sr := new(mockrepo.MockSantaRepo)
	ml := &fakeMailer{}
	tg := &fakeTG{}
	n := NewNotifier(sr, ml, tg, "https://santa.prosto-namekni.ru")
	n.now = func() time.Time { return chNow }
	room := entity.SantaRoom{ID: uuid.New(), Slug: "abcdefgh", Title: "Офис"}
	verified := chNow
	giver := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Аня", Channel: entity.SantaChannelEmail, Email: "anna@example.com", EmailVerifiedAt: &verified}
	chat := int64(55)
	ward := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Боря", Wishes: "носки", Channel: entity.SantaChannelTelegram, TgChatID: &chat}
	sr.On("GetParticipant", mock.Anything, giver.ID).Return(giver, nil)
	sr.On("GetParticipant", mock.Anything, ward.ID).Return(ward, nil)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	return notifierEnv{n: n, sr: sr, ml: ml, tg: tg, room: room, giver: giver, ward: ward}
}

func TestNotifier_DrawnByEmail(t *testing.T) {
	e := newNotifierEnv(t)
	note := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyDrawn, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	e.sr.On("GetAssignment", mock.Anything, e.room.ID, e.giver.ID).Return(entity.SantaAssignment{RoomID: e.room.ID, GiverID: e.giver.ID, ReceiverID: e.ward.ID}, nil)
	e.sr.On("MarkNotificationSent", mock.Anything, note.ID).Return(nil)

	sent, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, "anna@example.com", e.ml.to)
	assert.Contains(t, e.ml.text, "Боря")
	assert.Contains(t, e.ml.text, "носки")
	assert.Contains(t, e.ml.text, "https://santa.prosto-namekni.ru/r/abcdefgh")
}

func TestNotifier_WelcomeByTelegram(t *testing.T) {
	e := newNotifierEnv(t)
	note := entity.NewSantaNotification(e.ward.ID, entity.SantaNotifyWelcome, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	e.sr.On("MarkNotificationSent", mock.Anything, note.ID).Return(nil)

	_, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, 55, e.tg.chatID)
	assert.Contains(t, e.tg.text, "Офис")
}

func TestNotifier_RetrySchedule(t *testing.T) {
	for attempts, want := range map[int]*time.Duration{
		0: durp(time.Minute),
		1: durp(5 * time.Minute),
		2: durp(30 * time.Minute),
		3: nil, // четвёртая неудача — failed
	} {
		e := newNotifierEnv(t)
		e.ml.err = errors.New("smtp down")
		note := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyWelcome, chNow)
		note.Attempts = attempts
		e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
		var gotRetry *time.Time
		e.sr.On("MarkNotificationFailed", mock.Anything, note.ID, attempts+1, mock.Anything, "smtp down").Run(func(a mock.Arguments) {
			gotRetry = a.Get(3).(*time.Time)
		}).Return(nil)

		_, err := e.n.RunOnce(context.Background())
		require.NoError(t, err)
		if want == nil {
			assert.Nil(t, gotRetry, "попытка %d", attempts+1)
		} else {
			require.NotNil(t, gotRetry, "попытка %d", attempts+1)
			assert.Equal(t, chNow.Add(*want), *gotRetry)
		}
	}
}

func durp(d time.Duration) *time.Duration { return &d }

func TestNotifier_PermanentFailures(t *testing.T) {
	e := newNotifierEnv(t)
	gone := entity.NewSantaNotification(uuid.New(), entity.SantaNotifyWelcome, chNow)
	e.sr.On("GetParticipant", mock.Anything, gone.ParticipantID).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	noPair := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyDrawn, chNow)
	e.sr.On("GetAssignment", mock.Anything, e.room.ID, e.giver.ID).Return(entity.SantaAssignment{}, repo.ErrNotFound)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{gone, noPair}, nil)
	e.sr.On("MarkNotificationFailed", mock.Anything, gone.ID, 1, (*time.Time)(nil), mock.Anything).Return(nil)
	e.sr.On("MarkNotificationFailed", mock.Anything, noPair.ID, 1, (*time.Time)(nil), mock.Anything).Return(nil)

	sent, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, sent)
	// AssertExpectations не годится: ward из newNotifierEnv здесь не запрашивается.
	e.sr.AssertCalled(t, "MarkNotificationFailed", mock.Anything, gone.ID, 1, (*time.Time)(nil), mock.Anything)
	e.sr.AssertCalled(t, "MarkNotificationFailed", mock.Anything, noPair.ID, 1, (*time.Time)(nil), mock.Anything)
}

func TestNotifier_UnreadyChannelIsPermanent(t *testing.T) {
	e := newNotifierEnv(t)
	plain := entity.SantaParticipant{ID: uuid.New(), RoomID: e.room.ID, Name: "Без канала"}
	e.sr.On("GetParticipant", mock.Anything, plain.ID).Return(plain, nil)
	note := entity.NewSantaNotification(plain.ID, entity.SantaNotifyReminderFill, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	e.sr.On("MarkNotificationFailed", mock.Anything, note.ID, 1, (*time.Time)(nil), mock.Anything).Return(nil)
	_, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, e.ml.calls+e.tg.calls)
}

func TestNotifier_ClaimError(t *testing.T) {
	e := newNotifierEnv(t)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return(nil, errors.New("db down"))
	_, err := e.n.RunOnce(context.Background())
	assert.Error(t, err)
}

func TestNotifier_LeaseBudgetFitsLease(t *testing.T) {
	assert.Less(t, notifyLease/2+notifySendTimeout+notifyMarkTimeout, notifyLease,
		"последняя отправка пачки должна закончиться до конца аренды")
}

func TestNotifier_StopsBatchAfterHalfLease(t *testing.T) {
	e := newNotifierEnv(t)
	calls := 0
	// start=0, проверка перед 1-й=35с, перед 2-й=70с > lease/2.
	e.n.now = func() time.Time {
		at := chNow.Add(time.Duration(calls) * 35 * time.Second)
		calls++
		return at
	}
	first := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyWelcome, chNow)
	second := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyReminderFill, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{first, second}, nil)
	e.sr.On("MarkNotificationSent", mock.Anything, first.ID).Return(nil)

	sent, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, 1, e.ml.calls, "вторая не отправлена — вернётся после аренды")
	e.sr.AssertNotCalled(t, "MarkNotificationSent", mock.Anything, second.ID)
	e.sr.AssertNotCalled(t, "MarkNotificationFailed", mock.Anything, second.ID, mock.Anything, mock.Anything, mock.Anything)
}

func TestNotifier_SendDeadlineAndMarkAfterShutdown(t *testing.T) {
	e := newNotifierEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.ml.hook = func(sendCtx context.Context) {
		deadline, ok := sendCtx.Deadline()
		assert.True(t, ok, "у отправки есть дедлайн")
		assert.WithinDuration(t, time.Now().Add(notifySendTimeout), deadline, 5*time.Second)
		cancel() // остановка сервера во время отправки
	}
	first := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyWelcome, chNow)
	second := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyReminderFill, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{first, second}, nil)
	var markErr error
	e.sr.On("MarkNotificationSent", mock.Anything, first.ID).Run(func(a mock.Arguments) {
		markErr = a.Get(0).(context.Context).Err()
	}).Return(nil)

	sent, err := e.n.RunOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.NoError(t, markErr, "отметка пишется и после отмены ctx")
	assert.Equal(t, 1, e.ml.calls, "после отмены новых отправок нет")
}

func TestNotifier_MarkAlreadyHandledIsQuiet(t *testing.T) {
	e := newNotifierEnv(t)
	note := entity.NewSantaNotification(e.giver.ID, entity.SantaNotifyWelcome, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	e.sr.On("MarkNotificationSent", mock.Anything, note.ID).Return(repo.ErrNotFound)
	sent, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
}

func TestNotifier_TelegramBlockedIsPermanent(t *testing.T) {
	e := newNotifierEnv(t)
	e.tg.err = &telegram.APIError{Code: 403, Description: "Forbidden: bot was blocked by the user"}
	note := entity.NewSantaNotification(e.ward.ID, entity.SantaNotifyWelcome, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	e.sr.On("MarkNotificationFailed", mock.Anything, note.ID, 1, (*time.Time)(nil), mock.Anything).Return(nil)

	_, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	e.sr.AssertCalled(t, "MarkNotificationFailed", mock.Anything, note.ID, 1, (*time.Time)(nil), mock.Anything)
}

func TestNotifier_TelegramTransientRetries(t *testing.T) {
	e := newNotifierEnv(t)
	e.tg.err = &telegram.APIError{Code: 429, Description: "Too Many Requests: retry after 5"}
	note := entity.NewSantaNotification(e.ward.ID, entity.SantaNotifyWelcome, chNow)
	e.sr.On("ClaimNotifications", mock.Anything, chNow, notifyBatch, notifyLease).Return([]entity.SantaNotification{note}, nil)
	retry := chNow.Add(time.Minute)
	e.sr.On("MarkNotificationFailed", mock.Anything, note.ID, 1, &retry, mock.Anything).Return(nil)

	_, err := e.n.RunOnce(context.Background())
	require.NoError(t, err)
	e.sr.AssertCalled(t, "MarkNotificationFailed", mock.Anything, note.ID, 1, &retry, mock.Anything)
}
