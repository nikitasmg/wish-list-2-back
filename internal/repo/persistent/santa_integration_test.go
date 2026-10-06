//go:build integration

package persistent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/repo/persistent"
)

func setupSantaDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDB(t)
	require.NoError(t, db.AutoMigrate(
		&persistent.SantaRoomModel{},
		&persistent.SantaParticipantModel{},
		&persistent.SantaAssignmentModel{},
	))
	return db
}

func seedRoom(t *testing.T, r repo.SantaRepo, owner uuid.UUID) entity.SantaRoom {
	t.Helper()
	room := entity.SantaRoom{
		ID: uuid.New(), OwnerID: owner, Slug: uuid.NewString()[:8],
		Title: "Офис", Status: entity.SantaRoomOpen,
	}
	require.NoError(t, r.CreateRoom(context.Background(), room))
	return room
}

func seedParticipants(t *testing.T, r repo.SantaRepo, roomID uuid.UUID, n int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		p := entity.SantaParticipant{ID: uuid.New(), RoomID: roomID, Name: "Участник", TokenHash: uuid.NewString()}
		require.NoError(t, r.CreateParticipant(context.Background(), p))
		ids[i] = p.ID
	}
	return ids
}

func circle(roomID uuid.UUID) func([]uuid.UUID) ([]entity.SantaAssignment, error) {
	return func(ids []uuid.UUID) ([]entity.SantaAssignment, error) {
		out := make([]entity.SantaAssignment, len(ids))
		for i, id := range ids {
			out[i] = entity.SantaAssignment{RoomID: roomID, GiverID: id, ReceiverID: ids[(i+1)%len(ids)]}
		}
		return out, nil
	}
}

func TestSantaRepo_DrawWritesCycleAndLocksStatus(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)

	var got []uuid.UUID
	err := r.Draw(ctx, room.ID, entity.SantaRoomOpen, func(in []uuid.UUID) ([]entity.SantaAssignment, error) {
		got = in
		return circle(room.ID)(in)
	})
	require.NoError(t, err)
	assert.Equal(t, ids, got, "участники приходят в порядке вступления")

	saved, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.SantaRoomDrawn, saved.Status)
	assert.NotNil(t, saved.DrawnAt)

	a, err := r.GetAssignment(ctx, room.ID, ids[0])
	require.NoError(t, err)
	assert.Equal(t, ids[1], a.ReceiverID)

	// Вторая жеребьёвка с ожиданием open — отказ, пары прежние.
	err = r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID))
	assert.ErrorIs(t, err, repo.ErrStatusMismatch)

	a, err = r.GetAssignment(ctx, room.ID, ids[0])
	require.NoError(t, err)
	assert.Equal(t, ids[1], a.ReceiverID, "пары не изменились")
}

func TestSantaRepo_RedrawReplacesPairs(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID)))

	reversed := func(in []uuid.UUID) ([]entity.SantaAssignment, error) {
		out := make([]entity.SantaAssignment, len(in))
		for i, id := range in {
			out[i] = entity.SantaAssignment{RoomID: room.ID, GiverID: id, ReceiverID: in[(i+len(in)-1)%len(in)]}
		}
		return out, nil
	}
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomDrawn, reversed))

	a, err := r.GetAssignment(ctx, room.ID, ids[0])
	require.NoError(t, err)
	assert.Equal(t, ids[2], a.ReceiverID)
}

func TestSantaRepo_DrawRollsBackOnBuildError(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 2)

	boom := errors.New("boom")
	err := r.Draw(ctx, room.ID, entity.SantaRoomOpen, func([]uuid.UUID) ([]entity.SantaAssignment, error) {
		return nil, boom
	})
	assert.ErrorIs(t, err, boom)

	saved, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.SantaRoomOpen, saved.Status)
	_, err = r.GetAssignment(ctx, room.ID, ids[0])
	assert.ErrorIs(t, err, repo.ErrNotFound)
}

func TestSantaRepo_ListRoomsByUser(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	me := uuid.New()
	mine := seedRoom(t, r, me)
	joined := seedRoom(t, r, uuid.New())
	seedRoom(t, r, uuid.New()) // чужая, без меня

	require.NoError(t, r.CreateParticipant(ctx, entity.SantaParticipant{
		ID: uuid.New(), RoomID: joined.ID, UserID: &me, Name: "Я", TokenHash: uuid.NewString(),
	}))

	rooms, err := r.ListRoomsByUser(ctx, me)
	require.NoError(t, err)
	got := []uuid.UUID{}
	for _, room := range rooms {
		got = append(got, room.ID)
	}
	assert.ElementsMatch(t, []uuid.UUID{mine.ID, joined.ID}, got)
}

func TestSantaRepo_ParticipantLookups(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	other := seedRoom(t, r, uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Маша", TokenHash: "hash-1"}
	require.NoError(t, r.CreateParticipant(ctx, p))
	seedParticipants(t, r, room.ID, 2)

	found, err := r.GetParticipantByToken(ctx, room.ID, "hash-1")
	require.NoError(t, err)
	assert.Equal(t, p.ID, found.ID)

	_, err = r.GetParticipantByToken(ctx, other.ID, "hash-1")
	assert.ErrorIs(t, err, repo.ErrNotFound, "токен чужой комнаты не подходит")

	_, err = r.GetParticipantByToken(ctx, room.ID, "nope")
	assert.ErrorIs(t, err, repo.ErrNotFound)

	counts, err := r.CountParticipants(ctx, []uuid.UUID{room.ID, other.ID})
	require.NoError(t, err)
	assert.Equal(t, 3, counts[room.ID])
	assert.Equal(t, 0, counts[other.ID])
}

func TestSantaRepo_UserJoinsRoomOnce(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	user := uuid.New()
	first := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, UserID: &user, Name: "А", TokenHash: uuid.NewString()}
	second := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, UserID: &user, Name: "Б", TokenHash: uuid.NewString()}
	require.NoError(t, r.CreateParticipant(ctx, first))
	err := r.CreateParticipant(ctx, second)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code, "unique violation")
	_, err = r.GetParticipant(ctx, second.ID)
	assert.ErrorIs(t, err, repo.ErrNotFound)
}

func TestSantaRepo_DeleteRoomRemovesEverything(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	ids := seedParticipants(t, r, room.ID, 3)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID)))

	require.NoError(t, r.DeleteRoom(ctx, room.ID))

	_, err := r.GetRoomByID(ctx, room.ID)
	assert.ErrorIs(t, err, repo.ErrNotFound)
	_, err = r.GetParticipant(ctx, ids[0])
	assert.ErrorIs(t, err, repo.ErrNotFound)
}

func TestSantaRepo_UpdateRoom(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	budget := 1500
	room.Title, room.Budget, room.Message = "Новый год", &budget, "Привет"
	require.NoError(t, r.UpdateRoom(ctx, room))

	saved, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, "Новый год", saved.Title)
	assert.Equal(t, "Привет", saved.Message)
	require.NotNil(t, saved.Budget)
	assert.Equal(t, 1500, *saved.Budget)
	assert.Equal(t, entity.SantaRoomOpen, saved.Status)
}

func TestSantaRepo_UpdateRoomRejectedWhenDrawn(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 2)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID)))

	room.Title = "Поздно"
	assert.ErrorIs(t, r.UpdateRoom(ctx, room), repo.ErrStatusMismatch)

	saved, err := r.GetRoomByID(ctx, room.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.SantaRoomDrawn, saved.Status)
	assert.Equal(t, "Офис", saved.Title)
}

func TestSantaRepo_UpdateMissing(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	assert.ErrorIs(t, r.UpdateRoom(ctx, entity.SantaRoom{ID: uuid.New(), Title: "x"}), repo.ErrNotFound)
	assert.ErrorIs(t, r.UpdateParticipant(ctx, entity.SantaParticipant{ID: uuid.New(), Name: "x"}), repo.ErrNotFound)
}

func TestSantaRepo_UpdateParticipantKeepsToken(t *testing.T) {
	ctx := context.Background()
	r := persistent.NewSantaRepo(setupSantaDB(t))
	room := seedRoom(t, r, uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "А", TokenHash: "h"}
	require.NoError(t, r.CreateParticipant(ctx, p))
	require.NoError(t, r.UpdateParticipant(ctx, entity.SantaParticipant{ID: p.ID, Name: "Б", Wishes: "книга", WishlistURL: "u"}))

	got, err := r.GetParticipant(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Б", got.Name)
	assert.Equal(t, "книга", got.Wishes)
	assert.Equal(t, "h", got.TokenHash)
	assert.Equal(t, room.ID, got.RoomID)
}

func TestSantaRepo_DeleteRoomCascadesAssignments(t *testing.T) {
	ctx := context.Background()
	db := setupSantaDB(t)
	r := persistent.NewSantaRepo(db)
	room := seedRoom(t, r, uuid.New())
	seedParticipants(t, r, room.ID, 3)
	require.NoError(t, r.Draw(ctx, room.ID, entity.SantaRoomOpen, circle(room.ID)))

	require.NoError(t, r.DeleteRoom(ctx, room.ID))

	var n int64
	require.NoError(t, db.Model(&persistent.SantaAssignmentModel{}).Where("room_id = ?", room.ID).Count(&n).Error)
	assert.Zero(t, n)
	require.NoError(t, db.Model(&persistent.SantaParticipantModel{}).Where("room_id = ?", room.ID).Count(&n).Error)
	assert.Zero(t, n)
}
