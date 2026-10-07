package santa_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

func TestGetInvite_UnknownSlug(t *testing.T) {
	sr, _, uc := newUC()
	sr.On("GetRoomBySlug", mock.Anything, "nope").Return(entity.SantaRoom{}, repo.ErrNotFound)

	_, err := uc.GetInvite(ctx, "nope")

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

func TestGetInvite_OrganizerNameFallsBackToUsername(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 5}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{Username: "nikita"}, nil)

	inv, err := uc.GetInvite(ctx, room.Slug)

	require.NoError(t, err)
	assert.Equal(t, "nikita", inv.OrganizerName)
	assert.Equal(t, 5, inv.ParticipantsCount)
	assert.Equal(t, entity.SantaRoomOpen, inv.Status)
}

func TestJoin_AfterDrawIsClosed(t *testing.T) {
	sr, _, uc := newUC()
	room := drawnRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
	sr.AssertNotCalled(t, "CreateParticipant", mock.Anything, mock.Anything)
}

func TestJoin_RejectsScriptLink(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: "Маша", WishlistURL: "javascript:alert(1)"})

	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestJoin_UserCannotJoinTwice(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	user := uuid.New()
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByUser", mock.Anything, room.ID, user).Return(entity.SantaParticipant{ID: uuid.New()}, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{UserID: &user}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaAlreadyJoined)
}

func TestJoin_FullRoom(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 100}, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
}

func TestJoin_ReturnsTokenAndStoresOnlyHash(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	var stored entity.SantaParticipant
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 2}, nil)
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(1).(entity.SantaParticipant)
	}).Return(nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{DisplayName: "Никита"}, nil)

	res, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: " Маша ", Wishes: "чай"})

	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.NotEqual(t, res.Token, stored.TokenHash)
	assert.Len(t, stored.TokenHash, 64)
	assert.Equal(t, tokenHash(res.Token), stored.TokenHash)
	assert.Nil(t, stored.UserID)
	assert.Equal(t, "Маша", res.Me.Name)
	assert.Nil(t, res.Me.Receiver)
}

func TestGetMe_WrongTokenIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(entity.SantaParticipant{}, repo.ErrNotFound)

	_, err := uc.GetMe(ctx, room.Slug, usecase.SantaAuth{Token: "stale"})

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

func TestGetMe_NoCredentialsIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)

	_, err := uc.GetMe(ctx, room.Slug, usecase.SantaAuth{})

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

func TestGetMe_FallsBackToAccount(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	user := uuid.New()
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, UserID: &user, Name: "Никита"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	sr.On("GetParticipantByUser", mock.Anything, room.ID, user).Return(p, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 3}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)

	me, err := uc.GetMe(ctx, room.Slug, usecase.SantaAuth{Token: "old", UserID: &user})

	require.NoError(t, err)
	assert.Equal(t, p.ID, me.ParticipantID)
}

func TestGetMe_DrawnShowsReceiver(t *testing.T) {
	sr, ur, uc := newUC()
	room := drawnRoom(uuid.New())
	me := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Артём"}
	ward := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Маша", Wishes: "чай", WishlistURL: "https://x.ru/w"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(me, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 3}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)
	sr.On("GetAssignment", mock.Anything, room.ID, me.ID).Return(entity.SantaAssignment{RoomID: room.ID, GiverID: me.ID, ReceiverID: ward.ID}, nil)
	sr.On("GetParticipant", mock.Anything, ward.ID).Return(ward, nil)

	got, err := uc.GetMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"})

	require.NoError(t, err)
	require.NotNil(t, got.Receiver)
	assert.Equal(t, usecase.SantaReceiver{Name: "Маша", Wishes: "чай", WishlistURL: "https://x.ru/w"}, *got.Receiver)
}

func TestUpdateMe_RenameAfterDrawIsClosed(t *testing.T) {
	sr, _, uc := newUC()
	room := drawnRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Артём"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)

	_, err := uc.UpdateMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Тёма"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
	sr.AssertNotCalled(t, "UpdateParticipant", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateMe_WishesAfterDrawAreSaved(t *testing.T) {
	sr, ur, uc := newUC()
	room := drawnRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Артём"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("UpdateParticipant", mock.Anything, mock.MatchedBy(func(u entity.SantaParticipant) bool {
		return u.Wishes == "кофе" && u.Name == "Артём"
	}), mock.Anything).Return(nil)
	sr.On("CountParticipants", mock.Anything, mock.Anything).Return(map[uuid.UUID]int{}, nil)
	ur.On("GetByID", mock.Anything, mock.Anything).Return(entity.User{}, nil)
	sr.On("GetAssignment", mock.Anything, room.ID, p.ID).Return(entity.SantaAssignment{}, repo.ErrNotFound)
	sr.On("GetGiver", mock.Anything, mock.Anything, mock.Anything).Return(entity.SantaAssignment{}, repo.ErrNotFound)

	_, err := uc.UpdateMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Артём", Wishes: "кофе"})

	require.NoError(t, err)
	sr.AssertExpectations(t)
}

func TestLeaveMe_AfterDrawIsClosed(t *testing.T) {
	sr, _, uc := newUC()
	room := drawnRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)

	err := uc.LeaveMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
	sr.AssertNotCalled(t, "DeleteParticipant", mock.Anything, mock.Anything)
}

func TestLeaveMe_Deletes(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("DeleteParticipant", mock.Anything, p.ID).Return(nil)

	require.NoError(t, uc.LeaveMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}))
}

func TestJoin_DrawnConcurrentlyIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 2}, nil)
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Return(repo.ErrStatusMismatch)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
}

func TestLeaveMe_DrawnConcurrentlyIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("DeleteParticipant", mock.Anything, p.ID).Return(repo.ErrStatusMismatch)

	err := uc.LeaveMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"})

	assert.ErrorIs(t, err, usecase.ErrSantaDrawn)
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestGetMe_LooksUpByTokenHash(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Маша"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, tokenHash("raw-secret")).Return(p, nil).Once()
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)

	me, err := uc.GetMe(ctx, room.Slug, usecase.SantaAuth{Token: "raw-secret"})

	require.NoError(t, err)
	assert.Equal(t, p.ID, me.ParticipantID)
	sr.AssertExpectations(t)
}

func TestJoin_StoresUserID(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	user := uuid.New()
	var stored entity.SantaParticipant
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByUser", mock.Anything, room.ID, user).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(1).(entity.SantaParticipant)
	}).Return(nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{UserID: &user}, usecase.SantaProfileInput{Name: "Маша"})

	require.NoError(t, err)
	require.NotNil(t, stored.UserID)
	assert.Equal(t, user, *stored.UserID)
}

func TestUpdateMe_VanishedParticipantIsNotFound(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Маша"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("UpdateParticipant", mock.Anything, mock.Anything, mock.Anything).Return(repo.ErrNotFound)

	_, err := uc.UpdateMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Маша", Wishes: "чай"})

	assert.ErrorIs(t, err, usecase.ErrSantaNotFound)
}

func TestInvite_OrganizerLookupFailureIsReturned(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, errors.New("db down"))

	_, err := uc.GetInvite(ctx, room.Slug)

	assert.Error(t, err)
}

func TestInvite_MissingOrganizerKeepsCardOpen(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, fmt.Errorf("userRepo.GetByID: %w", gorm.ErrRecordNotFound))

	inv, err := uc.GetInvite(ctx, room.Slug)

	require.NoError(t, err)
	assert.Equal(t, "", inv.OrganizerName)
}

func TestJoin_WithValidTokenIsAlreadyJoined(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, tokenHash("tok")).Return(entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID}, nil)

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaAlreadyJoined)
	sr.AssertNotCalled(t, "CreateParticipant", mock.Anything, mock.Anything)
}

func TestJoin_StaleTokenDoesNotBlock(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, tokenHash("stale")).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Return(nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)

	res, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{Token: "stale"}, usecase.SantaProfileInput{Name: "Маша"})

	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
}

func TestJoin_DuplicateUserRaceIsConflict(t *testing.T) {
	sr, _, uc := newUC()
	room := openRoom(uuid.New())
	user := uuid.New()
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByUser", mock.Anything, room.ID, user).Return(entity.SantaParticipant{}, repo.ErrNotFound)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil)
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Return(fmt.Errorf("santaRepo.CreateParticipant: %w", repo.ErrDuplicate))

	_, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{UserID: &user}, usecase.SantaProfileInput{Name: "Маша"})

	assert.ErrorIs(t, err, usecase.ErrSantaAlreadyJoined)
}

func TestJoin_MeFailureStillReturnsToken(t *testing.T) {
	sr, ur, uc := newUC()
	room := openRoom(uuid.New())
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 1}, nil).Once()
	sr.On("CreateParticipant", mock.Anything, mock.Anything).Return(nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int(nil), errors.New("db down"))
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil).Maybe()

	res, err := uc.Join(ctx, room.Slug, usecase.SantaAuth{}, usecase.SantaProfileInput{Name: "Маша"})

	require.NoError(t, err)
	assert.NotEmpty(t, res.Token)
	assert.Equal(t, "Маша", res.Me.Name)
	assert.Equal(t, room.Slug, res.Me.Room.Slug)
}

func TestGetInvite_ExposesDrawnAt(t *testing.T) {
	sr, ur, uc := newUC()
	room := drawnRoom(uuid.New())
	at := time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC)
	room.DrawnAt = &at
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("CountParticipants", mock.Anything, []uuid.UUID{room.ID}).Return(map[uuid.UUID]int{room.ID: 3}, nil)
	ur.On("GetByID", mock.Anything, room.OwnerID).Return(entity.User{}, nil)

	inv, err := uc.GetInvite(ctx, room.Slug)

	require.NoError(t, err)
	require.NotNil(t, inv.DrawnAt)
	assert.True(t, at.Equal(*inv.DrawnAt))
}

func TestUpdateMe_AfterDrawNotifiesSanta(t *testing.T) {
	sr, ur, uc := newUC()
	room := drawnRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Аня", Wishes: "старое"}
	santaID := uuid.New()
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("GetGiver", mock.Anything, room.ID, p.ID).Return(entity.SantaAssignment{RoomID: room.ID, GiverID: santaID, ReceiverID: p.ID}, nil)
	sr.On("UpdateParticipant", mock.Anything, mock.Anything, mock.MatchedBy(func(ns []entity.SantaNotification) bool {
		return len(ns) == 1 && ns[0].ParticipantID == santaID && ns[0].Kind == entity.SantaNotifyWishesUpdated
	})).Return(nil)
	sr.On("CountParticipants", mock.Anything, mock.Anything).Return(map[uuid.UUID]int{}, nil)
	sr.On("GetAssignment", mock.Anything, room.ID, p.ID).Return(entity.SantaAssignment{}, repo.ErrNotFound)
	ur.On("GetByID", mock.Anything, mock.Anything).Return(entity.User{}, nil)

	_, err := uc.UpdateMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Аня", Wishes: "новое"})

	require.NoError(t, err)
	sr.AssertExpectations(t)
}

func TestUpdateMe_AfterDrawSameWishesNoNotify(t *testing.T) {
	sr, ur, uc := newUC()
	room := drawnRoom(uuid.New())
	p := entity.SantaParticipant{ID: uuid.New(), RoomID: room.ID, Name: "Аня", Wishes: "то же"}
	sr.On("GetRoomBySlug", mock.Anything, room.Slug).Return(room, nil)
	sr.On("GetParticipantByToken", mock.Anything, room.ID, mock.Anything).Return(p, nil)
	sr.On("UpdateParticipant", mock.Anything, mock.Anything, []entity.SantaNotification(nil)).Return(nil)
	sr.On("CountParticipants", mock.Anything, mock.Anything).Return(map[uuid.UUID]int{}, nil)
	sr.On("GetAssignment", mock.Anything, room.ID, p.ID).Return(entity.SantaAssignment{}, repo.ErrNotFound)
	ur.On("GetByID", mock.Anything, mock.Anything).Return(entity.User{}, nil)

	_, err := uc.UpdateMe(ctx, room.Slug, usecase.SantaAuth{Token: "tok"}, usecase.SantaProfileInput{Name: "Аня", Wishes: "то же"})

	require.NoError(t, err)
	sr.AssertNotCalled(t, "GetGiver", mock.Anything, mock.Anything, mock.Anything)
}
