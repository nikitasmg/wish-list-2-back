//go:build integration

package persistent_test

import (
	"context"
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

func countNotes(t *testing.T, db *gorm.DB, participantID uuid.UUID, kind entity.SantaNotificationKind) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&persistent.SantaNotificationModel{}).
		Where("participant_id = ? AND kind = ?", participantID, string(kind)).Count(&n).Error)
	return n
}

func emailCode(id uuid.UUID, now time.Time) entity.SantaEmailCode {
	return entity.SantaEmailCode{ParticipantID: id, CodeHash: "hash-" + id.String(), ExpiresAt: now.Add(15 * time.Minute), SentAt: now}
}

func TestSantaRepo_EmailVerification(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, r.SetEmail(ctx, id, "anna@example.com", emailCode(id, now)))
	p, err := r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "anna@example.com", p.Email)
	assert.False(t, p.Ready(), "до подтверждения не готов")

	code, err := r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "hash-"+id.String(), code.CodeHash)
	assert.Equal(t, 0, code.Attempts)

	require.NoError(t, r.IncEmailCodeAttempts(ctx, id))
	code, err = r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 1, code.Attempts)

	welcome := entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)
	require.NoError(t, r.VerifyEmail(ctx, id, now, welcome))
	p, err = r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.True(t, p.Ready())
	assert.Equal(t, entity.SantaChannelEmail, p.Channel)
	_, err = r.GetEmailCode(ctx, id)
	assert.ErrorIs(t, err, repo.ErrNotFound, "код стёрт")
	assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyWelcome))
}

func TestSantaRepo_SetEmailResetsVerificationAndCode(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, r.SetEmail(ctx, id, "a@example.com", emailCode(id, now)))
	require.NoError(t, r.VerifyEmail(ctx, id, now, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)))
	require.NoError(t, r.IncEmailCodeAttempts(ctx, id)) // кода уже нет — тихо ничего

	later := now.Add(2 * time.Minute)
	require.NoError(t, r.SetEmail(ctx, id, "b@example.com", emailCode(id, later)))
	p, err := r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "b@example.com", p.Email)
	assert.Nil(t, p.EmailVerifiedAt, "новый адрес надо подтвердить заново")
	assert.False(t, p.Ready())

	code, err := r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.WithinDuration(t, later, code.SentAt, time.Second)
	assert.Equal(t, 0, code.Attempts, "новый код — счётчик попыток с нуля")
}

func TestSantaRepo_EmailUniqueWithinRoom(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	now := time.Now().UTC()
	room := seedRoom(t, r, uuid.New())
	ids := seedUnready(t, r, room.ID, 2)
	require.NoError(t, r.SetEmail(ctx, ids[0], "same@example.com", emailCode(ids[0], now)))

	err := r.SetEmail(ctx, ids[1], "same@example.com", emailCode(ids[1], now))
	assert.ErrorIs(t, err, repo.ErrDuplicate)

	other := seedRoom(t, r, uuid.New())
	stranger := seedUnready(t, r, other.ID, 1)[0]
	assert.NoError(t, r.SetEmail(ctx, stranger, "same@example.com", emailCode(stranger, now)), "в другой комнате адрес свободен")
}

func TestSantaRepo_SetEmailMissingParticipant(t *testing.T) {
	r := persistent.NewSantaRepo(setupSantaDB(t))
	id := uuid.New()
	err := r.SetEmail(context.Background(), id, "x@example.com", emailCode(id, time.Now()))
	assert.ErrorIs(t, err, repo.ErrNotFound)
}

func TestSantaRepo_LinkTelegram(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC()
	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "tok", ParticipantID: id, ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "tok2", ParticipantID: id, ExpiresAt: now.Add(time.Hour)}))

	welcome := func(p entity.SantaParticipant) entity.SantaNotification {
		return entity.NewSantaNotification(p.ID, entity.SantaNotifyWelcome, now)
	}
	p, err := r.LinkTelegram(ctx, "tok", 777, now, welcome)
	require.NoError(t, err)
	assert.Equal(t, id, p.ID)
	require.NotNil(t, p.TgChatID)
	assert.EqualValues(t, 777, *p.TgChatID)
	assert.True(t, p.Ready())
	assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyWelcome))

	_, err = r.LinkTelegram(ctx, "tok", 777, now, welcome)
	assert.ErrorIs(t, err, repo.ErrNotFound, "ссылка одноразовая")
	_, err = r.LinkTelegram(ctx, "tok2", 777, now, welcome)
	assert.ErrorIs(t, err, repo.ErrNotFound, "остальные ссылки участника тоже стёрты")
}

func TestSantaRepo_LinkTelegramExpired(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC()
	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "old", ParticipantID: id, ExpiresAt: now.Add(-time.Minute)}))

	_, err := r.LinkTelegram(ctx, "old", 1, now, func(p entity.SantaParticipant) entity.SantaNotification {
		return entity.NewSantaNotification(p.ID, entity.SantaNotifyWelcome, now)
	})
	assert.ErrorIs(t, err, repo.ErrNotFound)
	p, err := r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, p.TgChatID)
}

func TestSantaRepo_DeleteParticipantCascadesChannelData(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC()
	require.NoError(t, r.SetEmail(ctx, id, "c@example.com", emailCode(id, now)))
	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "t", ParticipantID: id, ExpiresAt: now.Add(time.Hour)}))
	require.NoError(t, r.VerifyEmail(ctx, id, now, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)))

	require.NoError(t, r.DeleteParticipant(ctx, id))
	assert.EqualValues(t, 0, countNotes(t, db, id, entity.SantaNotifyWelcome))
	var links int64
	require.NoError(t, db.Model(&persistent.SantaTgLinkModel{}).Where("participant_id = ?", id).Count(&links).Error)
	assert.EqualValues(t, 0, links)
}
