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
	assert.Empty(t, p.Email, "до кода адрес не подтверждён")
	assert.Equal(t, "anna@example.com", p.PendingEmail)
	assert.False(t, p.Ready(), "до подтверждения не готов")

	code, err := r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "hash-"+id.String(), code.CodeHash)
	assert.Equal(t, 0, code.Attempts)

	for i := 1; i <= 5; i++ {
		taken, err := r.IncEmailCodeAttempts(ctx, id, 5)
		require.NoError(t, err)
		assert.True(t, taken, "попытка %d", i)
	}
	taken, err := r.IncEmailCodeAttempts(ctx, id, 5)
	require.NoError(t, err)
	assert.False(t, taken, "шестая попытка не занимается")
	code, err = r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 5, code.Attempts)

	// Код, который проверяли, уже заменён (другой адрес): подтверждение отклоняется,
	// новый адрес остаётся неподтверждённым.
	other := emailCode(id, now)
	other.CodeHash = "other-hash"
	require.NoError(t, r.SetEmail(ctx, id, "other@example.com", other))
	err = r.VerifyEmail(ctx, id, "hash-"+id.String(), now, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now))
	assert.ErrorIs(t, err, repo.ErrNotFound)
	p, err = r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.False(t, p.Ready())
	assert.EqualValues(t, 0, countNotes(t, db, id, entity.SantaNotifyWelcome))
	require.NoError(t, r.SetEmail(ctx, id, "anna@example.com", emailCode(id, now)))

	welcome := entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)
	require.NoError(t, r.VerifyEmail(ctx, id, "hash-"+id.String(), now, welcome))
	p, err = r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.True(t, p.Ready())
	assert.Equal(t, entity.SantaChannelEmail, p.Channel)
	assert.Equal(t, "anna@example.com", p.Email)
	assert.Empty(t, p.PendingEmail)
	_, err = r.GetEmailCode(ctx, id)
	assert.ErrorIs(t, err, repo.ErrNotFound, "код стёрт")
	assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyWelcome))
}

func TestSantaRepo_PendingEmailKeepsReady(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	id := seedUnready(t, r, room.ID, 1)[0]
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, r.SetEmail(ctx, id, "a@example.com", emailCode(id, now)))
	require.NoError(t, r.VerifyEmail(ctx, id, "hash-"+id.String(), now, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)))
	taken, err := r.IncEmailCodeAttempts(ctx, id, 5) // кода уже нет — тихо ничего
	require.NoError(t, err)
	assert.False(t, taken)

	later := now.Add(2 * time.Minute)
	require.NoError(t, r.SetEmail(ctx, id, "b@example.com", emailCode(id, later)))
	p, err := r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "a@example.com", p.Email, "старый адрес в силе до кода")
	assert.Equal(t, "b@example.com", p.PendingEmail)
	assert.NotNil(t, p.EmailVerifiedAt)
	assert.True(t, p.Ready(), "смена адреса не снимает готовность")

	code, err := r.GetEmailCode(ctx, id)
	require.NoError(t, err)
	assert.WithinDuration(t, later, code.SentAt, time.Second)
	assert.Equal(t, 0, code.Attempts, "новый код — счётчик попыток с нуля")

	require.NoError(t, r.VerifyEmail(ctx, id, "hash-"+id.String(), later, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, later)))
	p, err = r.GetParticipant(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "b@example.com", p.Email)
	assert.Empty(t, p.PendingEmail)
	assert.True(t, p.Ready())
}

func TestSantaRepo_EmailUniqueWithinRoom(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	now := time.Now().UTC()
	room := seedRoom(t, r, uuid.New())
	ids := seedUnready(t, r, room.ID, 2)
	require.NoError(t, r.SetEmail(ctx, ids[0], "same@example.com", emailCode(ids[0], now)))
	require.NoError(t, r.VerifyEmail(ctx, ids[0], "hash-"+ids[0].String(), now, entity.NewSantaNotification(ids[0], entity.SantaNotifyWelcome, now)))

	err := r.SetEmail(ctx, ids[1], "same@example.com", emailCode(ids[1], now))
	assert.ErrorIs(t, err, repo.ErrDuplicate, "подтверждённый адрес соседа занят")

	other := seedRoom(t, r, uuid.New())
	stranger := seedUnready(t, r, other.ID, 1)[0]
	assert.NoError(t, r.SetEmail(ctx, stranger, "same@example.com", emailCode(stranger, now)), "в другой комнате адрес свободен")
}

func TestSantaRepo_VerifyEmailTakenMeanwhile(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	now := time.Now().UTC()
	room := seedRoom(t, r, uuid.New())
	ids := seedUnready(t, r, room.ID, 2)
	// Оба ждут код на один адрес — неподтверждённый адрес никого не блокирует.
	require.NoError(t, r.SetEmail(ctx, ids[0], "same@example.com", emailCode(ids[0], now)))
	require.NoError(t, r.SetEmail(ctx, ids[1], "same@example.com", emailCode(ids[1], now)))

	require.NoError(t, r.VerifyEmail(ctx, ids[0], "hash-"+ids[0].String(), now, entity.NewSantaNotification(ids[0], entity.SantaNotifyWelcome, now)))
	err := r.VerifyEmail(ctx, ids[1], "hash-"+ids[1].String(), now, entity.NewSantaNotification(ids[1], entity.SantaNotifyWelcome, now))
	assert.ErrorIs(t, err, repo.ErrDuplicate)

	p, err := r.GetParticipant(ctx, ids[1])
	require.NoError(t, err)
	assert.False(t, p.Ready())
	assert.Equal(t, "same@example.com", p.PendingEmail, "откат: ничего не поменялось")
	assert.EqualValues(t, 0, countNotes(t, db, ids[1], entity.SantaNotifyWelcome))
}

func TestBackfillSantaPendingEmail(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ids := seedUnready(t, r, room.ID, 2)
	now := time.Now().UTC()
	// Как писал этап 2: новый адрес сразу в email, подтверждение сброшено.
	require.NoError(t, db.Model(&persistent.SantaParticipantModel{}).Where("id = ?", ids[0]).
		Update("email", "old@example.com").Error)
	require.NoError(t, db.Model(&persistent.SantaParticipantModel{}).Where("id = ?", ids[1]).
		Updates(map[string]any{"email": "ok@example.com", "email_verified_at": now, "channel": "email"}).Error)

	require.NoError(t, persistent.BackfillSantaPendingEmail(db))
	require.NoError(t, persistent.BackfillSantaPendingEmail(db), "повторный запуск безопасен")

	p, err := r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	assert.Empty(t, p.Email)
	assert.Equal(t, "old@example.com", p.PendingEmail)
	q, err := r.GetParticipant(ctx, ids[1])
	require.NoError(t, err)
	assert.Equal(t, "ok@example.com", q.Email)
	assert.Empty(t, q.PendingEmail)
	assert.True(t, q.Ready())
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
	require.NoError(t, r.VerifyEmail(ctx, id, "hash-"+id.String(), now, entity.NewSantaNotification(id, entity.SantaNotifyWelcome, now)))

	require.NoError(t, r.DeleteParticipant(ctx, id))
	assert.EqualValues(t, 0, countNotes(t, db, id, entity.SantaNotifyWelcome))
	var links int64
	require.NoError(t, db.Model(&persistent.SantaTgLinkModel{}).Where("participant_id = ?", id).Count(&links).Error)
	assert.EqualValues(t, 0, links)
}

func drawnNote(now time.Time) func(uuid.UUID) entity.SantaNotification {
	return func(giverID uuid.UUID) entity.SantaNotification {
		return entity.NewSantaNotification(giverID, entity.SantaNotifyDrawn, now)
	}
}

func TestSantaRepo_DrawOnlyReadyAndEnqueues(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ready := seedParticipants(t, r, room.ID, 3)
	unready := seedUnready(t, r, room.ID, 1)[0]

	var got []uuid.UUID
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, func(in []uuid.UUID) ([]entity.SantaAssignment, error) {
		got = in
		return circle(room.ID)(in)
	}, drawnNote(time.Now())))
	assert.Equal(t, ready, got, "неготовый в жеребьёвку не попал")
	for _, id := range ready {
		assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyDrawn))
	}
	assert.EqualValues(t, 0, countNotes(t, db, unready, entity.SantaNotifyDrawn))
	_, err := r.GetAssignment(ctx, room.ID, unready)
	assert.ErrorIs(t, err, repo.ErrNotFound)
}

func TestSantaRepo_RedrawDropsPendingDrawn(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(time.Now())))
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomDrawn, circle(room.ID), drawnNote(time.Now())))
	for _, id := range ids {
		assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyDrawn), "несданное старое drawn стёрто, новое одно")
	}
}

func TestSantaRepo_UpdateParticipantEnqueues(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 2)
	p, err := r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	p.Wishes = "Книги"
	require.NoError(t, r.UpdateParticipant(ctx, p, entity.NewSantaNotification(ids[1], entity.SantaNotifyWishesUpdated, time.Now())))
	assert.EqualValues(t, 1, countNotes(t, db, ids[1], entity.SantaNotifyWishesUpdated))
}

func TestSantaRepo_UpdateParticipantCollapsesWishesUpdated(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	p, err := r.GetParticipant(ctx, ids[0])
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)
	save := func(receiver uuid.UUID, wishes string) {
		p.Wishes = wishes
		require.NoError(t, r.UpdateParticipant(ctx, p, entity.NewSantaNotification(receiver, entity.SantaNotifyWishesUpdated, now)))
	}

	// Сданное уведомление остаётся в истории.
	save(ids[1], "книги")
	claimed, err := r.ClaimNotifications(ctx, now, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.NoError(t, r.MarkNotificationSent(ctx, claimed[0].ID, nil))

	// Три быстрых сохранения — одно pending; взятое в работу тоже стирается,
	// и его отметка уже ничего не меняет.
	save(ids[1], "носки")
	inFlight, err := r.ClaimNotifications(ctx, now, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, inFlight, 1)
	save(ids[1], "шарф")
	save(ids[1], "варежки")
	assert.EqualValues(t, 2, countNotes(t, db, ids[1], entity.SantaNotifyWishesUpdated), "одно сданное + одно свежее")
	assert.ErrorIs(t, r.MarkNotificationSent(ctx, inFlight[0].ID, nil), repo.ErrNotFound)

	// Уведомления других получателей не задеты.
	save(ids[2], "свечи")
	assert.EqualValues(t, 1, countNotes(t, db, ids[2], entity.SantaNotifyWishesUpdated))
	assert.EqualValues(t, 2, countNotes(t, db, ids[1], entity.SantaNotifyWishesUpdated))
}

func TestSantaRepo_GetGiver(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	_, err := r.GetGiver(ctx, room.ID, ids[1])
	assert.ErrorIs(t, err, repo.ErrNotFound, "до жеребьёвки дарящего нет")

	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(time.Now())))
	a, err := r.GetGiver(ctx, room.ID, ids[1])
	require.NoError(t, err)
	assert.Equal(t, ids[0], a.GiverID)
}

func TestSantaRepo_RemindCooldown(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	id := seedParticipants(t, r, room.ID, 1)[0]
	now := time.Now().UTC().Truncate(time.Second)
	note := func() []entity.SantaNotification {
		return []entity.SantaNotification{entity.NewSantaNotification(id, entity.SantaNotifyReminderFill, now)}
	}

	require.NoError(t, r.Remind(ctx, room.ID, now, 12*time.Hour, note()))
	saved, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	require.NotNil(t, saved.LastRemindedAt)
	assert.WithinDuration(t, now, *saved.LastRemindedAt, time.Second)

	err = r.Remind(ctx, room.ID, now.Add(11*time.Hour), 12*time.Hour, note())
	assert.ErrorIs(t, err, repo.ErrTooSoon)
	assert.EqualValues(t, 1, countNotes(t, db, id, entity.SantaNotifyReminderFill), "второе напоминание не легло")

	require.NoError(t, r.Remind(ctx, room.ID, now.Add(13*time.Hour), 12*time.Hour, note()))
	assert.EqualValues(t, 2, countNotes(t, db, id, entity.SantaNotifyReminderFill))
}

func TestSantaRepo_ClaimLeasesNotifications(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(now)))

	first, err := r.ClaimNotifications(ctx, now, 10, 2*time.Minute)
	require.NoError(t, err)
	assert.Len(t, first, 3)

	again, err := r.ClaimNotifications(ctx, now.Add(time.Minute), 10, 2*time.Minute)
	require.NoError(t, err)
	assert.Empty(t, again, "под арендой — второй обработчик не берёт")

	afterLease, err := r.ClaimNotifications(ctx, now.Add(3*time.Minute), 10, 2*time.Minute)
	require.NoError(t, err)
	assert.Len(t, afterLease, 3, "аренда истекла — снова в работе")
}

func TestSantaRepo_MarkNotification(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(now)))
	batch, err := r.ClaimNotifications(ctx, now, 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, batch, 3)

	require.NoError(t, r.MarkNotificationSent(ctx, batch[0].ID, nil))
	retry := now.Add(5 * time.Minute)
	require.NoError(t, r.MarkNotificationFailed(ctx, batch[1].ID, 2, &retry, "timeout"))
	require.NoError(t, r.MarkNotificationFailed(ctx, batch[2].ID, 4, nil, "gone"))

	var rows []persistent.SantaNotificationModel
	require.NoError(t, db.Order("id").Find(&rows).Error)
	byID := map[uuid.UUID]persistent.SantaNotificationModel{}
	for _, row := range rows {
		byID[row.ID] = row
	}
	assert.Equal(t, "sent", byID[batch[0].ID].Status)
	assert.Equal(t, "pending", byID[batch[1].ID].Status)
	assert.Equal(t, 2, byID[batch[1].ID].Attempts)
	assert.WithinDuration(t, retry, byID[batch[1].ID].NextTryAt, time.Second)
	assert.Equal(t, "timeout", byID[batch[1].ID].LastError)
	assert.Equal(t, "failed", byID[batch[2].ID].Status)

	due, err := r.ClaimNotifications(ctx, now.Add(10*time.Minute), 10, time.Minute)
	require.NoError(t, err)
	require.Len(t, due, 1, "только отложенное pending")
	assert.Equal(t, batch[1].ID, due[0].ID)

	// Уже отмеченное не перезаписывается: обработчик, опоздавший с отметкой,
	// не вернёт sent в pending и не оживит failed.
	assert.ErrorIs(t, r.MarkNotificationFailed(ctx, batch[0].ID, 1, &retry, "late"), repo.ErrNotFound)
	assert.ErrorIs(t, r.MarkNotificationSent(ctx, batch[2].ID, nil), repo.ErrNotFound)
	assert.ErrorIs(t, r.MarkNotificationSent(ctx, uuid.New(), nil), repo.ErrNotFound)
	var sentRow, failedRow persistent.SantaNotificationModel
	require.NoError(t, db.First(&sentRow, "id = ?", batch[0].ID).Error)
	assert.Equal(t, "sent", sentRow.Status)
	assert.Empty(t, sentRow.LastError)
	require.NoError(t, db.First(&failedRow, "id = ?", batch[2].ID).Error)
	assert.Equal(t, "failed", failedRow.Status)
}

func TestSantaRepo_PurgeStale(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	ids := seedUnready(t, r, room.ID, 2)
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "old", ParticipantID: ids[0], ExpiresAt: now.Add(-time.Hour)}))
	require.NoError(t, r.CreateTgLink(ctx, entity.SantaTgLink{TokenHash: "fresh", ParticipantID: ids[0], ExpiresAt: now.Add(time.Hour)}))
	expired := emailCode(ids[0], now.Add(-time.Hour)) // истёк 45 минут назад
	require.NoError(t, r.SetEmail(ctx, ids[0], "a@example.com", expired))
	require.NoError(t, r.SetEmail(ctx, ids[1], "b@example.com", emailCode(ids[1], now)))

	month := 30 * 24 * time.Hour
	note := func(status string, age time.Duration) persistent.SantaNotificationModel {
		return persistent.SantaNotificationModel{
			ID: uuid.New(), ParticipantID: ids[0], Kind: string(entity.SantaNotifyWelcome), Payload: "{}",
			Status: status, NextTryAt: now, CreatedAt: now.Add(-age),
		}
	}
	oldSent, oldFailed, oldPending, freshSent := note("sent", month+time.Hour), note("failed", month+time.Hour), note("pending", month+time.Hour), note("sent", 24*time.Hour)
	require.NoError(t, db.Create(&[]persistent.SantaNotificationModel{oldSent, oldFailed, oldPending, freshSent}).Error)

	deleted, err := r.PurgeStale(ctx, now, month)
	require.NoError(t, err)
	assert.EqualValues(t, 4, deleted, "ссылка, код и два отработанных уведомления")

	var links []persistent.SantaTgLinkModel
	require.NoError(t, db.Find(&links).Error)
	require.Len(t, links, 1)
	assert.Equal(t, "fresh", links[0].TokenHash)
	_, err = r.GetEmailCode(ctx, ids[0])
	assert.ErrorIs(t, err, repo.ErrNotFound, "истёкший код стёрт")
	_, err = r.GetEmailCode(ctx, ids[1])
	assert.NoError(t, err, "живой код остался")
	var left []uuid.UUID
	require.NoError(t, db.Model(&persistent.SantaNotificationModel{}).Order("id").Pluck("id", &left).Error)
	assert.ElementsMatch(t, []uuid.UUID{oldPending.ID, freshSent.ID}, left, "pending не трогаем, свежие храним")
}
