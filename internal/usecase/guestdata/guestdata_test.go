package guestdata_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	"main/internal/usecase/guestdata"
	mockrepo "main/mock/repo"
)

var (
	owner    = uuid.New()
	stranger = uuid.New()
)

func newUC() (*mockrepo.MockGuestDataRepo, *mockrepo.MockWishlistRepo, usecase.GuestDataUseCase) {
	gr := &mockrepo.MockGuestDataRepo{}
	wr := &mockrepo.MockWishlistRepo{}
	return gr, wr, guestdata.New(gr, wr)
}

// RSVP

func TestSubmitRSVP_RequiresGuestCookie(t *testing.T) {
	gr, _, uc := newUC()

	_, err := uc.SubmitRSVP(context.Background(), uuid.New(), "b1", uuid.Nil, usecase.RSVPInput{
		Name: "Аня", Going: true,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "куки")
	gr.AssertNotCalled(t, "UpsertRSVP", mock.Anything, mock.Anything)
}

func TestSubmitRSVP_RequiresName(t *testing.T) {
	gr, _, uc := newUC()

	_, err := uc.SubmitRSVP(context.Background(), uuid.New(), "b1", uuid.New(), usecase.RSVPInput{
		Name: "   ", Going: true,
	})

	require.Error(t, err)
	gr.AssertNotCalled(t, "UpsertRSVP", mock.Anything, mock.Anything)
}

// «Не приду» не должно тащить за собой спутников: иначе организатор считает
// по сводке людей, которых не будет.
func TestSubmitRSVP_NotGoingClearsCompanions(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "rsvp", `{}`)
	gr.On("UpsertRSVP", mock.Anything, mock.Anything).Return(nil)

	result, err := uc.SubmitRSVP(context.Background(), wid, "b1", uuid.New(), usecase.RSVPInput{
		Name: "Аня", Going: false, PlusOne: 2, Kids: 1, Transfer: true,
	})

	require.NoError(t, err)
	assert.Zero(t, result.PlusOne)
	assert.Zero(t, result.Kids)
	assert.False(t, result.Transfer)
}

func TestSubmitRSVP_RejectsTooManyCompanions(t *testing.T) {
	_, _, uc := newUC()

	_, err := uc.SubmitRSVP(context.Background(), uuid.New(), "b1", uuid.New(), usecase.RSVPInput{
		Name: "Аня", Going: true, PlusOne: entity.MaxPlusOne + 1,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "не больше")
}

func TestOwnerRSVPSummary_CountsPeople(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()

	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	gr.On("ListRSVP", mock.Anything, "b1").Return([]entity.RSVPResponse{
		{Name: "Аня", Going: true, PlusOne: 1, Transfer: true},
		{Name: "Лена", Going: true, Kids: 2},
		{Name: "Дима", Going: false, PlusOne: 3},
	}, nil)

	summary, err := uc.OwnerRSVPSummary(context.Background(), owner, wid, "b1")

	require.NoError(t, err)
	assert.Equal(t, 2, summary.Going)
	assert.Equal(t, 1, summary.NotGoing)
	assert.Equal(t, 1, summary.PlusOnes)
	assert.Equal(t, 2, summary.Kids)
	assert.Equal(t, 1, summary.Transfer)
	// 2 гостя + 1 спутник + 2 ребёнка; спутники отказавшегося не считаются
	assert.Equal(t, 5, summary.TotalPeople)
}

func TestOwnerRSVPSummary_RejectsStranger(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	_, err := uc.OwnerRSVPSummary(context.Background(), stranger, wid, "b1")

	require.ErrorIs(t, err, usecase.ErrForbidden)
	gr.AssertNotCalled(t, "ListRSVP", mock.Anything, mock.Anything)
}

// Голосование

func TestVote_RequiresGuestCookie(t *testing.T) {
	gr, _, uc := newUC()

	_, err := uc.Vote(context.Background(), uuid.New(), "b1", uuid.Nil, []string{"0"})

	require.Error(t, err)
	gr.AssertNotCalled(t, "ReplacePollChoices", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Плейлист

func TestSuggestTrack_EnforcesPerGuestLimit(t *testing.T) {
	gr, _, uc := newUC()
	guest := uuid.New()

	gr.On("CountTracksByGuest", mock.Anything, "b1", guest).Return(entity.MaxTracksPerGuest, nil)

	_, err := uc.SuggestTrack(context.Background(), uuid.New(), "b1", guest, "Кино — Группа крови")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "не больше")
	gr.AssertNotCalled(t, "CreateTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSuggestTrack_TrimsAndCreates(t *testing.T) {
	gr, _, uc := newUC()
	guest, wid := uuid.New(), uuid.New()

	gr.On("CountTracksByGuest", mock.Anything, "b1", guest).Return(0, nil)
	gr.On("CreateTrack", mock.Anything, wid, "b1", guest, "Земфира — Хочешь?").Return(uuid.New(), nil)
	gr.On("ListTracks", mock.Anything, "b1", guest).Return([]entity.PlaylistTrack{}, nil)

	_, err := uc.SuggestTrack(context.Background(), wid, "b1", guest, "  Земфира — Хочешь?  ")

	require.NoError(t, err)
	gr.AssertExpectations(t)
}

func TestSuggestTrack_RejectsEmptyTitle(t *testing.T) {
	gr, _, uc := newUC()

	_, err := uc.SuggestTrack(context.Background(), uuid.New(), "b1", uuid.New(), "   ")

	require.Error(t, err)
	gr.AssertNotCalled(t, "CreateTrack", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// Гостевая книга

func TestGuestbook_HidesHiddenEntriesFromGuests(t *testing.T) {
	gr, _, uc := newUC()
	guest := uuid.New()

	// Ключевое: гостю запрашиваем список с includeHidden=false. Иначе «скрыть»
	// ничего не скрывает — запись просто не рисуется, но видна в ответе API.
	gr.On("ListGuestbook", mock.Anything, "b1", guest, false).
		Return([]entity.GuestbookEntry{{Text: "Поздравляю!"}}, nil)

	entries, err := uc.Guestbook(context.Background(), "b1", guest)

	require.NoError(t, err)
	assert.Len(t, entries, 1)
	gr.AssertExpectations(t)
}

func TestOwnerGuestbook_IncludesHidden(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()

	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)
	gr.On("ListGuestbook", mock.Anything, "b1", uuid.Nil, true).
		Return([]entity.GuestbookEntry{{Text: "Скрытое", Hidden: true}}, nil)

	entries, err := uc.OwnerGuestbook(context.Background(), owner, wid, "b1")

	require.NoError(t, err)
	assert.True(t, entries[0].Hidden)
}

func TestOwnerSetGuestbookHidden_RejectsStranger(t *testing.T) {
	gr, wr, uc := newUC()
	wid, entryID := uuid.New(), uuid.New()

	gr.On("GuestbookEntryWishlist", mock.Anything, entryID).Return(wid, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	err := uc.OwnerSetGuestbookHidden(context.Background(), stranger, entryID, true)

	require.ErrorIs(t, err, usecase.ErrForbidden)
	gr.AssertNotCalled(t, "SetGuestbookHidden", mock.Anything, mock.Anything, mock.Anything)
}

func TestAddGuestbookEntry_RejectsTooLongText(t *testing.T) {
	gr, _, uc := newUC()
	guest := uuid.New()
	gr.On("CountGuestbookByGuest", mock.Anything, "b1", guest).Return(0, nil)

	_, err := uc.AddGuestbookEntry(context.Background(), uuid.New(), "b1", guest, usecase.GuestbookInput{
		Name: "Лена",
		Text: strings.Repeat("я", entity.MaxGuestbookTextLen+1),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "длиннее")
	gr.AssertNotCalled(t, "CreateGuestbookEntry", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
