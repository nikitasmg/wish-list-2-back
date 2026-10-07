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
	"main/internal/usecase"
	mockrepo "main/mock/repo"
	"main/pkg/telegram"
)

type fakeMailer struct {
	to, subject, text string
	err               error
	calls             int
}

func (f *fakeMailer) Send(_ context.Context, to, subject, _, text string) error {
	f.calls++
	f.to, f.subject, f.text = to, subject, text
	return f.err
}

type fakeTG struct {
	chatID int64
	text   string
	calls  int
	err    error
}

func (f *fakeTG) SendMessage(_ context.Context, chatID int64, text string, _ []telegram.Button) error {
	f.calls++
	f.chatID, f.text = chatID, text
	return f.err
}

var chNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

// channelUC — use case с фиксированным временем, комнатой slug "abcdefgh" и
// участником по токену "tok".
func channelUC(t *testing.T) (*santaUseCase, *mockrepo.MockSantaRepo, *fakeMailer, *fakeTG, entity.SantaRoom, entity.SantaParticipant) {
	t.Helper()
	sr := new(mockrepo.MockSantaRepo)
	ur := new(mockrepo.MockUserRepo)
	ml := &fakeMailer{}
	tg := &fakeTG{}
	uc := New(sr, ur, WithMailer(ml), WithTelegram(tg, "santa_namekni_bot")).(*santaUseCase)
	uc.now = func() time.Time { return chNow }
	room := entity.SantaRoom{ID: uuid.New(), Slug: "abcdefgh", Title: "Офис", Status: entity.SantaRoomOpen}
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Аня", TokenHash: hashToken("tok")}
	sr.On("GetRoomBySlug", mock.Anything, "abcdefgh").Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, hashToken("tok")).Return(p, nil)
	return uc, sr, ml, tg, room, p
}

var tokAuth = usecase.SantaAuth{Token: "tok"}

func TestRequestEmailCode_SendsCodeAndStoresHash(t *testing.T) {
	uc, sr, ml, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(entity.SantaEmailCode{}, repo.ErrNotFound)
	var stored entity.SantaEmailCode
	sr.On("SetEmail", mock.Anything, p.ID, "anna@example.com", mock.Anything).Run(func(a mock.Arguments) {
		stored = a.Get(3).(entity.SantaEmailCode)
	}).Return(nil)

	require.NoError(t, uc.RequestEmailCode(context.Background(), "abcdefgh", tokAuth, "  Anna@Example.COM "))
	assert.Equal(t, 1, ml.calls)
	assert.Equal(t, "anna@example.com", ml.to, "адрес хранится и используется в нижнем регистре")
	code := ml.subject[len(ml.subject)-6:]
	assert.Regexp(t, `^\d{6}$`, code)
	assert.Equal(t, hashEmailCode(p.ID, code), stored.CodeHash, "в базе хэш, а не код")
	assert.NotContains(t, stored.CodeHash, code)
	assert.Equal(t, chNow.Add(emailCodeTTL), stored.ExpiresAt)
	assert.Equal(t, chNow, stored.SentAt)
}

func TestRequestEmailCode_Cooldown(t *testing.T) {
	uc, sr, ml, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(entity.SantaEmailCode{SentAt: chNow.Add(-30 * time.Second)}, nil)
	err := uc.RequestEmailCode(context.Background(), "abcdefgh", tokAuth, "a@example.com")
	assert.ErrorIs(t, err, usecase.ErrSantaTooSoon)
	assert.Equal(t, 0, ml.calls)
	sr.AssertNotCalled(t, "SetEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestEmailCode_BadAddress(t *testing.T) {
	uc, sr, _, _, _, _ := channelUC(t)
	for _, bad := range []string{"", "не адрес", "Аня <a@example.com>", "a@"} {
		err := uc.RequestEmailCode(context.Background(), "abcdefgh", tokAuth, bad)
		assert.ErrorIs(t, err, usecase.ErrSantaInvalid, bad)
	}
	sr.AssertNotCalled(t, "SetEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestRequestEmailCode_Taken(t *testing.T) {
	uc, sr, ml, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(entity.SantaEmailCode{}, repo.ErrNotFound)
	sr.On("SetEmail", mock.Anything, p.ID, "a@example.com", mock.Anything).Return(repo.ErrDuplicate)
	err := uc.RequestEmailCode(context.Background(), "abcdefgh", tokAuth, "a@example.com")
	assert.ErrorIs(t, err, usecase.ErrSantaEmailTaken)
	assert.Equal(t, 0, ml.calls)
}

func TestRequestEmailCode_SendFailureDropsCode(t *testing.T) {
	uc, sr, ml, _, _, p := channelUC(t)
	ml.err = errors.New("smtp down")
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(entity.SantaEmailCode{}, repo.ErrNotFound)
	sr.On("SetEmail", mock.Anything, p.ID, "a@example.com", mock.Anything).Return(nil)
	sr.On("DeleteEmailCode", mock.Anything, p.ID).Return(nil)
	err := uc.RequestEmailCode(context.Background(), "abcdefgh", tokAuth, "a@example.com")
	assert.Error(t, err)
	sr.AssertCalled(t, "DeleteEmailCode", mock.Anything, p.ID)
}

func TestRequestEmailCode_NotParticipant(t *testing.T) {
	uc, sr, _, _, room, _ := channelUC(t)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, hashToken("other")).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	err := uc.RequestEmailCode(context.Background(), "abcdefgh", usecase.SantaAuth{Token: "other"}, "a@example.com")
	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

func validCode(p entity.SantaParticipant, code string, attempts int) entity.SantaEmailCode {
	return entity.SantaEmailCode{ParticipantID: p.ID, CodeHash: hashEmailCode(p.ID, code), ExpiresAt: chNow.Add(5 * time.Minute), Attempts: attempts, SentAt: chNow}
}

func TestVerifyEmail_Success(t *testing.T) {
	uc, sr, _, _, room, p := channelUC(t)
	p.Email = "a@example.com"
	sr.ExpectedCalls = nil // перенастроить участника с адресом
	sr.On("GetRoomBySlug", mock.Anything, "abcdefgh").Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, hashToken("tok")).Return(p, nil)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(validCode(p, "123456", 0), nil)
	sr.On("IncEmailCodeAttempts", mock.Anything, p.ID, emailCodeAttempts).Return(true, nil)
	sr.On("VerifyEmail", mock.Anything, p.ID, hashEmailCode(p.ID, "123456"), chNow, mock.MatchedBy(func(n entity.SantaNotification) bool {
		return n.ParticipantID == p.ID && n.Kind == entity.SantaNotifyWelcome
	})).Return(nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	uc.users.(*mockrepo.MockUserRepo).On("GetByID", mock.Anything, mock.Anything).Return(entity.User{}, repo.ErrNotFound)

	me, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, " 123456 ")
	require.NoError(t, err)
	assert.True(t, me.Notify.Ready)
	assert.True(t, me.Notify.EmailVerified)
	assert.Equal(t, entity.SantaChannelEmail, me.Notify.Channel)
}

func TestVerifyEmail_WrongCodeCountsAttempt(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(validCode(p, "123456", 1), nil)
	sr.On("IncEmailCodeAttempts", mock.Anything, p.ID, emailCodeAttempts).Return(true, nil)
	_, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, "654321")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
	sr.AssertCalled(t, "IncEmailCodeAttempts", mock.Anything, p.ID, emailCodeAttempts)
	sr.AssertNotCalled(t, "VerifyEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyEmail_LockedAfterFiveAttempts(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(validCode(p, "123456", emailCodeAttempts), nil)
	sr.On("IncEmailCodeAttempts", mock.Anything, p.ID, emailCodeAttempts).Return(false, nil)
	_, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, "123456")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid, "даже верный код после 5 попыток не принимается")
	sr.AssertNotCalled(t, "VerifyEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyEmail_CodeReplacedMeanwhile(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(validCode(p, "123456", 0), nil)
	sr.On("IncEmailCodeAttempts", mock.Anything, p.ID, emailCodeAttempts).Return(true, nil)
	sr.On("VerifyEmail", mock.Anything, p.ID, hashEmailCode(p.ID, "123456"), chNow, mock.Anything).Return(repo.ErrNotFound)
	_, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, "123456")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestVerifyEmail_Expired(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	code := validCode(p, "123456", 0)
	code.ExpiresAt = chNow.Add(-time.Second)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(code, nil)
	_, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, "123456")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestVerifyEmail_NoCode(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	sr.On("GetEmailCode", mock.Anything, p.ID).Return(entity.SantaEmailCode{}, repo.ErrNotFound)
	_, err := uc.VerifyEmail(context.Background(), "abcdefgh", tokAuth, "123456")
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestTelegramLink(t *testing.T) {
	uc, sr, _, _, _, p := channelUC(t)
	var link entity.SantaTgLink
	sr.On("CreateTgLink", mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
		link = a.Get(1).(entity.SantaTgLink)
	}).Return(nil)

	url, err := uc.TelegramLink(context.Background(), "abcdefgh", tokAuth)
	require.NoError(t, err)
	assert.Regexp(t, `^https://t\.me/santa_namekni_bot\?start=[A-Za-z0-9_-]{43}$`, url)
	raw := url[len("https://t.me/santa_namekni_bot?start="):]
	assert.Equal(t, hashToken(raw), link.TokenHash, "в базе хэш токена")
	assert.Equal(t, p.ID, link.ParticipantID)
	assert.Equal(t, chNow.Add(tgLinkTTL), link.ExpiresAt)
}

func TestTelegramLink_NotConfigured(t *testing.T) {
	sr := new(mockrepo.MockSantaRepo)
	uc := New(sr, new(mockrepo.MockUserRepo))
	_, err := uc.TelegramLink(context.Background(), "abcdefgh", tokAuth)
	assert.Error(t, err)
}

func TestTelegramStart_Links(t *testing.T) {
	uc, sr, _, tg, _, p := channelUC(t)
	sr.On("LinkTelegram", mock.Anything, hashToken("RAW"), int64(77), chNow, mock.Anything).Run(func(a mock.Arguments) {
		welcome := a.Get(4).(func(entity.SantaParticipant) entity.SantaNotification)
		n := welcome(p)
		assert.Equal(t, entity.SantaNotifyWelcome, n.Kind)
		assert.Equal(t, p.ID, n.ParticipantID)
	}).Return(p, nil)
	require.NoError(t, uc.TelegramStart(context.Background(), 77, "RAW"))
	assert.Equal(t, 0, tg.calls, "приветствие уйдёт через очередь, не напрямую")
}

func TestTelegramStart_ExpiredLink(t *testing.T) {
	uc, sr, _, tg, _, _ := channelUC(t)
	sr.On("LinkTelegram", mock.Anything, hashToken("OLD"), int64(77), chNow, mock.Anything).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	require.NoError(t, uc.TelegramStart(context.Background(), 77, "OLD"))
	assert.Equal(t, 1, tg.calls)
	assert.Equal(t, botLinkExpiredText(), tg.text)
}

func TestTelegramStart_ReplyFailureIsNotAnError(t *testing.T) {
	uc, sr, _, tg, _, _ := channelUC(t)
	tg.err = errors.New("telegram down")
	sr.On("LinkTelegram", mock.Anything, hashToken("OLD"), int64(77), chNow, mock.Anything).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	assert.NoError(t, uc.TelegramStart(context.Background(), 77, "OLD"), "вебхук не должен отвечать 500 и провоцировать повтор")
	assert.Equal(t, 1, tg.calls)
}

func TestTelegramStart_NoToken(t *testing.T) {
	uc, sr, _, tg, _, _ := channelUC(t)
	require.NoError(t, uc.TelegramStart(context.Background(), 77, ""))
	assert.Equal(t, botHelloText(), tg.text)
	sr.AssertNotCalled(t, "LinkTelegram", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
