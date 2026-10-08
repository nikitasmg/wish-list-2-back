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

// drawnRoom — разыгранная комната из трёх по кругу circle: ids[0]→ids[1]→ids[2]→ids[0].
func drawnRoom(t *testing.T, r repo.SantaRepo) (entity.SantaRoom, []uuid.UUID) {
	t.Helper()
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	require.NoError(t, r.Draw(context.Background(), room.ID, entity.SantaRoomOpen, circle(room.ID), drawnNote(time.Now())))
	return room, ids
}

func chatMsg(roomID, giver, receiver uuid.UUID, fromGiver bool, body string, at time.Time) entity.SantaMessage {
	return entity.SantaMessage{ID: uuid.New(), RoomID: roomID, GiverID: giver, ReceiverID: receiver, FromGiver: fromGiver, Body: body, CreatedAt: at}
}

func chatNote(m entity.SantaMessage) entity.SantaNotification {
	n := entity.NewSantaNotification(m.RecipientID(), entity.SantaNotifyChatMessage, m.CreatedAt)
	n.Payload[entity.SantaPayloadMessageID] = m.ID.String()
	return n
}

func post(t *testing.T, r repo.SantaRepo, m entity.SantaMessage) {
	t.Helper()
	require.NoError(t, r.CreateMessage(context.Background(), m, m.CreatedAt.Add(-time.Hour), 30, chatNote(m)))
}

func TestSantaRepo_CreateAndListMessages(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC().Truncate(time.Second)

	first := chatMsg(room.ID, ids[0], ids[1], true, "Какой размер?", now)
	second := chatMsg(room.ID, ids[0], ids[1], false, "M", now.Add(time.Second))
	third := chatMsg(room.ID, ids[0], ids[1], true, "Понял", now.Add(2*time.Second))
	for _, m := range []entity.SantaMessage{first, second, third} {
		post(t, r, m)
	}

	all, err := r.ListMessages(ctx, room.ID, ids[0], ids[1], 10)
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, []string{"Какой размер?", "M", "Понял"}, []string{all[0].Body, all[1].Body, all[2].Body})
	last, err := r.ListMessages(ctx, room.ID, ids[0], ids[1], 2)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{second.ID, third.ID}, []uuid.UUID{last[0].ID, last[1].ID}, "последние два, по возрастанию")

	got, err := r.GetMessage(ctx, second.ID)
	require.NoError(t, err)
	assert.False(t, got.FromGiver)
	assert.EqualValues(t, 2, countNotes(t, db, ids[1], entity.SantaNotifyChatMessage), "подопечному — два от Санты")
	assert.EqualValues(t, 1, countNotes(t, db, ids[0], entity.SantaNotifyChatMessage))
}

func TestSantaRepo_CreateMessageNeedsPairAndDrawnRoom(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC()

	wrong := chatMsg(room.ID, ids[0], ids[2], true, "Привет", now) // ids[0] дарит ids[1]
	assert.ErrorIs(t, r.CreateMessage(ctx, wrong, now.Add(-time.Hour), 30, chatNote(wrong)), repo.ErrNotFound)

	open := seedRoom(t, r, uuid.New())
	pids := seedParticipants(t, r, open.ID, 2)
	early := chatMsg(open.ID, pids[0], pids[1], true, "Привет", now)
	assert.ErrorIs(t, r.CreateMessage(ctx, early, now.Add(-time.Hour), 30, chatNote(early)), repo.ErrStatusMismatch)
}

func TestSantaRepo_CreateMessageLimit(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC().Truncate(time.Second)

	// ids[1] пишет и как подопечный (своему Санте ids[0]), и как Санта (ids[2]) —
	// лимит общий.
	asWard := chatMsg(room.ID, ids[0], ids[1], false, "раз", now)
	asSanta := chatMsg(room.ID, ids[1], ids[2], true, "два", now.Add(time.Second))
	require.NoError(t, r.CreateMessage(ctx, asWard, now.Add(-time.Hour), 2, chatNote(asWard)))
	require.NoError(t, r.CreateMessage(ctx, asSanta, now.Add(-time.Hour), 2, chatNote(asSanta)))

	third := chatMsg(room.ID, ids[1], ids[2], true, "три", now.Add(2*time.Second))
	assert.ErrorIs(t, r.CreateMessage(ctx, third, now.Add(-time.Hour), 2, chatNote(third)), repo.ErrTooSoon)
	assert.NoError(t, r.CreateMessage(ctx, third, now.Add(time.Second), 2, chatNote(third)), "старые вышли из окна")

	other := chatMsg(room.ID, ids[0], ids[1], true, "чужой лимит не мой", now.Add(3*time.Second))
	assert.NoError(t, r.CreateMessage(ctx, other, now.Add(-time.Hour), 2, chatNote(other)))
}

func TestSantaRepo_UnreadAndMarkRead(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC().Truncate(time.Second)
	post(t, r, chatMsg(room.ID, ids[0], ids[1], true, "1", now))
	post(t, r, chatMsg(room.ID, ids[0], ids[1], true, "2", now.Add(time.Second)))
	post(t, r, chatMsg(room.ID, ids[0], ids[1], false, "3", now.Add(2*time.Second)))

	fromSanta, fromReceiver, err := r.CountUnread(ctx, room.ID, ids[1])
	require.NoError(t, err)
	assert.Equal(t, 2, fromSanta, "ids[1] получил два от своего Санты ids[0]")
	assert.Equal(t, 0, fromReceiver)
	fromSanta, fromReceiver, err = r.CountUnread(ctx, room.ID, ids[0])
	require.NoError(t, err)
	assert.Equal(t, 0, fromSanta)
	assert.Equal(t, 1, fromReceiver, "ids[0] получил один от подопечного ids[1]")

	require.NoError(t, r.MarkMessagesRead(ctx, room.ID, ids[0], ids[1], true, now.Add(time.Minute)))
	fromSanta, _, err = r.CountUnread(ctx, room.ID, ids[1])
	require.NoError(t, err)
	assert.Equal(t, 0, fromSanta)
	_, fromReceiver, err = r.CountUnread(ctx, room.ID, ids[0])
	require.NoError(t, err)
	assert.Equal(t, 1, fromReceiver, "чужие входящие не тронуты")
}

func TestSantaRepo_RedrawClearsMessages(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC().Truncate(time.Second)
	msg := chatMsg(room.ID, ids[0], ids[1], true, "Привет", now)
	post(t, r, msg)

	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomDrawn, circle(room.ID), drawnNote(now)))

	_, err := r.GetMessage(ctx, msg.ID)
	assert.ErrorIs(t, err, repo.ErrNotFound, "переписка старой пары стёрта")
	assert.EqualValues(t, 0, countNotes(t, db, ids[1], entity.SantaNotifyChatMessage), "несданное уведомление о ней тоже")
}

func TestSantaRepo_FindChatNotification(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room, ids := drawnRoom(t, r)
	now := time.Now().UTC().Truncate(time.Second)
	msg := chatMsg(room.ID, ids[0], ids[1], true, "Привет", now)
	post(t, r, msg)
	claimed, err := r.ClaimNotifications(ctx, now.Add(time.Second), 10, time.Minute)
	require.NoError(t, err)
	var note entity.SantaNotification
	for _, n := range claimed {
		if n.Kind == entity.SantaNotifyChatMessage {
			note = n
		}
	}
	require.Equal(t, ids[1], note.ParticipantID)
	tgID := int64(555)
	require.NoError(t, r.MarkNotificationSent(ctx, note.ID, &tgID))

	ward, err := r.GetParticipant(ctx, ids[1])
	require.NoError(t, err)
	found, err := r.FindChatNotification(ctx, *ward.TgChatID, 555)
	require.NoError(t, err)
	assert.Equal(t, note.ID, found.ID)
	assert.Equal(t, msg.ID.String(), found.Payload[entity.SantaPayloadMessageID])

	_, err = r.FindChatNotification(ctx, *ward.TgChatID+1, 555)
	assert.ErrorIs(t, err, repo.ErrNotFound, "message_id уникален только внутри чата")
	_, err = r.FindChatNotification(ctx, *ward.TgChatID, 556)
	assert.ErrorIs(t, err, repo.ErrNotFound)
}
