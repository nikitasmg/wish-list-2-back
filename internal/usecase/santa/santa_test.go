package santa_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/internal/usecase/santa"
	mockrepo "main/mock/repo"
)

var ctx = context.Background()

func newUC() (*mockrepo.MockSantaRepo, *mockrepo.MockUserRepo, usecase.SantaUseCase) {
	sr := &mockrepo.MockSantaRepo{}
	ur := &mockrepo.MockUserRepo{}
	return sr, ur, santa.New(sr, ur)
}

func openRoom(owner uuid.UUID) entity.SantaRoom {
	return entity.SantaRoom{ID: uuid.New(), OwnerID: owner, Slug: "AbCd2345", Title: "Офис", Status: entity.SantaRoomOpen}
}

func drawnRoom(owner uuid.UUID) entity.SantaRoom {
	r := openRoom(owner)
	r.Status = entity.SantaRoomDrawn
	return r
}

func intPtr(v int) *int { return &v }

func TestCreateRoom_RejectsEmptyTitle(t *testing.T) {
	sr, _, uc := newUC()
	_, err := uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "   "})
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
	sr.AssertNotCalled(t, "CreateRoom", mock.Anything, mock.Anything)
}

func TestCreateRoom_RejectsBadBudget(t *testing.T) {
	_, _, uc := newUC()
	_, err := uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "Офис", Budget: intPtr(-1)})
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
	_, err = uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "Офис", Budget: intPtr(1_000_001)})
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestCreateRoom_RejectsPastDrawAt(t *testing.T) {
	_, _, uc := newUC()
	past := time.Now().Add(-time.Hour)
	_, err := uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "Офис", DrawAt: &past})
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestCreateRoom_OrganizerNeedsName(t *testing.T) {
	sr, _, uc := newUC()
	_, err := uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "Офис", OrganizerJoins: true})
	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
	sr.AssertNotCalled(t, "CreateRoom", mock.Anything, mock.Anything)
}

func TestCreateRoom_AddsOrganizerAsParticipant(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	sr.On("GetRoomBySlug", mock.Anything, mock.Anything).Return(entity.SantaRoom{}, repo.ErrNotFound)
	sr.On("CreateRoom", mock.Anything, mock.MatchedBy(func(r entity.SantaRoom) bool {
		return r.OwnerID == owner && r.Status == entity.SantaRoomOpen
	})).Return(nil)
	sr.On("CreateParticipant", mock.Anything, mock.MatchedBy(func(p entity.SantaParticipant) bool {
		return p.UserID != nil && *p.UserID == owner && p.Name == "Никита" && p.TokenHash != ""
	})).Return(nil)

	room, err := uc.CreateRoom(ctx, owner, usecase.SantaRoomInput{
		Title: "  Офис  ", Budget: intPtr(3000), OrganizerJoins: true, OrganizerName: " Никита ",
	})

	require.NoError(t, err)
	assert.Equal(t, "Офис", room.Title)
	assert.Len(t, room.Slug, 8)
	sr.AssertExpectations(t)
}

func TestCreateRoom_WithoutOrganizer(t *testing.T) {
	sr, _, uc := newUC()
	sr.On("GetRoomBySlug", mock.Anything, mock.Anything).Return(entity.SantaRoom{}, repo.ErrNotFound)
	sr.On("CreateRoom", mock.Anything, mock.Anything).Return(nil)

	_, err := uc.CreateRoom(ctx, uuid.New(), usecase.SantaRoomInput{Title: "Офис"})

	require.NoError(t, err)
	sr.AssertNotCalled(t, "CreateParticipant", mock.Anything, mock.Anything)
}

func TestListRooms_MarksOwnership(t *testing.T) {
	sr, _, uc := newUC()
	me := uuid.New()
	mine, theirs := openRoom(me), openRoom(uuid.New())
	sr.On("ListRoomsByUser", mock.Anything, me).Return([]entity.SantaRoom{mine, theirs}, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{mine.ID, theirs.ID}).
		Return(map[uuid.UUID]int{mine.ID: 4}, nil)

	rooms, err := uc.ListRooms(ctx, me)

	require.NoError(t, err)
	require.Len(t, rooms, 2)
	assert.True(t, rooms[0].IsOwner)
	assert.Equal(t, 4, rooms[0].ParticipantsCount)
	assert.False(t, rooms[1].IsOwner)
	assert.Equal(t, 0, rooms[1].ParticipantsCount)
}

func TestGetRoom_StrangerGetsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)

	_, err := uc.GetRoom(ctx, uuid.New(), room.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
	sr.AssertNotCalled(t, "ListParticipants", mock.Anything, mock.Anything)
}

func TestGetRoom_MissingIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	id := uuid.New()
	sr.On("GetRoomByID", mock.Anything, id).Return(entity.SantaRoom{}, repo.ErrNotFound)

	_, err := uc.GetRoom(ctx, uuid.New(), id)

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

// Организатор видит, кто заполнил пожелания, но не сам текст.
func TestGetRoom_HidesWishesText(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("ListParticipants", mock.Anything, room.ID).Return([]entity.SantaParticipant{
		{ID: uuid.New(), RoomID: room.ID, UserID: &owner, Name: "Никита", Wishes: "секретный чай"},
		{ID: uuid.New(), RoomID: room.ID, Name: "Маша", WishlistURL: "https://example.com/w"},
	}, nil)

	details, err := uc.GetRoom(ctx, owner, room.ID)

	require.NoError(t, err)
	require.Len(t, details.Participants, 2)
	assert.True(t, details.Participants[0].IsOwner)
	assert.True(t, details.Participants[0].HasWishes)
	assert.True(t, details.Participants[1].HasWishlist)
	raw, err := json.Marshal(details)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "секретный чай")
	assert.NotContains(t, string(raw), "example.com")
}

func TestUpdateRoom_AfterDrawIsClosed(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := drawnRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)

	_, err := uc.UpdateRoom(ctx, owner, room.ID, usecase.SantaRoomInput{Title: "Новое"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
	sr.AssertNotCalled(t, "UpdateRoom", mock.Anything, mock.Anything)
}

// Жеребьёвка прошла между проверкой и записью: репозиторий вернёт
// ErrStatusMismatch, наружу это должно выйти как ErrSantaDrawn.
func TestUpdateRoom_DrawnConcurrentlyIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("UpdateRoom", mock.Anything, mock.Anything).Return(repo.ErrStatusMismatch)

	_, err := uc.UpdateRoom(ctx, owner, room.ID, usecase.SantaRoomInput{Title: "Новое"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
}

func TestUpdateRoom_SavesFields(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("UpdateRoom", mock.Anything, mock.MatchedBy(func(r entity.SantaRoom) bool {
		return r.Title == "Новое" && r.Budget != nil && *r.Budget == 5000 && r.Slug == room.Slug
	})).Return(nil)

	updated, err := uc.UpdateRoom(ctx, owner, room.ID, usecase.SantaRoomInput{Title: "Новое", Budget: intPtr(5000)})

	require.NoError(t, err)
	assert.Equal(t, "Новое", updated.Title)
	sr.AssertExpectations(t)
}

func TestDeleteRoom_Stranger(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)

	err := uc.DeleteRoom(ctx, uuid.New(), room.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
	sr.AssertNotCalled(t, "DeleteRoom", mock.Anything, mock.Anything)
}

func TestRemoveParticipant_FromAnotherRoomIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	stranger := entity.SantaParticipant{ID: uuid.New(), RoomID: uuid.New()}
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("GetParticipant", mock.Anything, stranger.ID).Return(stranger, nil)

	err := uc.RemoveParticipant(ctx, owner, room.ID, stranger.ID)

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
	sr.AssertNotCalled(t, "DeleteParticipant", mock.Anything, mock.Anything)
}

func TestRemoveParticipant_AfterDrawIsClosed(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := drawnRoom(owner)
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)

	err := uc.RemoveParticipant(ctx, owner, room.ID, uuid.New())

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
}

func TestRemoveParticipant_Deletes(t *testing.T) {
	sr, _, uc := newUC()
	owner := uuid.New()
	room := openRoom(owner)
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID}
	sr.On("GetRoomByID", mock.Anything, room.ID).Return(room, nil)
	sr.On("GetParticipant", mock.Anything, p.ID).Return(p, nil)
	sr.On("DeleteParticipant", mock.Anything, p.ID).Return(nil)

	require.NoError(t, uc.RemoveParticipant(ctx, owner, room.ID, p.ID))
	sr.AssertExpectations(t)
}
