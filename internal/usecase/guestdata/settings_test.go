package guestdata_test

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
	"main/internal/usecase"
	mockrepo "main/mock/repo"
)

// withBlock — вишлист владельца с одним блоком; настройки блока лежат в data.
func withBlock(wr *mockrepo.MockWishlistRepo, wid uuid.UUID, blockType, data string) {
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner, Blocks: []entity.Block{
		{ID: "b1", Type: blockType, Data: json.RawMessage(data)},
	}}, nil)
}

// Ответ гостя

func TestSubmitRSVP_UnknownBlock(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "poll", `{}`)

	_, err := uc.SubmitRSVP(context.Background(), wid, "b1", uuid.New(), usecase.RSVPInput{Name: "Аня", Going: true})
	require.Error(t, err, "голосовать ответом гостя нельзя — блок другого типа")
	gr.AssertNotCalled(t, "UpsertRSVP", mock.Anything, mock.Anything)
}

func TestSubmitRSVP_AfterDeadline(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	withBlock(wr, wid, "rsvp", `{"deadline":"`+past+`"}`)

	_, err := uc.SubmitRSVP(context.Background(), wid, "b1", uuid.New(), usecase.RSVPInput{Name: "Аня", Going: true})
	require.ErrorIs(t, err, usecase.ErrClosed)
	gr.AssertNotCalled(t, "UpsertRSVP", mock.Anything, mock.Anything)
}

func TestSubmitRSVP_CustomQuestions(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "rsvp", `{"questions":[{"id":"cake","label":"Какой торт?","kind":"text"},{"id":"n","label":"Сколько?","kind":"number"},{"id":"car","label":"На машине?","kind":"bool"}]}`)
	gr.On("UpsertRSVP", mock.Anything, mock.MatchedBy(func(r entity.RSVPResponse) bool {
		return r.Answers["cake"] == "медовик" && r.Answers["n"] == "2" && r.Answers["car"] == "true"
	})).Return(nil)

	res, err := uc.SubmitRSVP(context.Background(), wid, "b1", uuid.New(), usecase.RSVPInput{
		Name: "Аня", Going: true,
		Answers: map[string]string{"cake": "  медовик ", "n": "2", "car": "true"},
	})
	require.NoError(t, err)
	assert.Equal(t, "медовик", res.Answers["cake"])
}

func TestSubmitRSVP_RejectsBadAnswers(t *testing.T) {
	cases := map[string]map[string]string{
		"неизвестный вопрос": {"who": "я"},
		"не число":           {"n": "много"},
		"не да/нет":          {"car": "может"},
	}
	for name, answers := range cases {
		t.Run(name, func(t *testing.T) {
			gr, wr, uc := newUC()
			wid := uuid.New()
			withBlock(wr, wid, "rsvp", `{"questions":[{"id":"n","label":"Сколько?","kind":"number"},{"id":"car","label":"На машине?","kind":"bool"}]}`)

			_, err := uc.SubmitRSVP(context.Background(), wid, "b1", uuid.New(), usecase.RSVPInput{Name: "Аня", Going: true, Answers: answers})
			require.Error(t, err)
			gr.AssertNotCalled(t, "UpsertRSVP", mock.Anything, mock.Anything)
		})
	}
}

func TestRSVPGuests(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "rsvp", `{"showGuests":true}`)
	gr.On("ListRSVP", mock.Anything, "b1").Return([]entity.RSVPResponse{
		{Name: "Аня", Going: true, PlusOne: 1},
		{Name: "Дима", Going: false},
		{Name: "Лена", Going: true, Kids: 2},
	}, nil)

	guests, err := uc.RSVPGuests(context.Background(), wid, "b1")
	require.NoError(t, err)
	assert.Equal(t, []string{"Аня", "Лена"}, guests.Names, "отказавшиеся в «Кто идёт» не попадают")
	assert.Equal(t, 5, guests.Total)
}

func TestRSVPGuests_HiddenUnlessEnabled(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "rsvp", `{}`)

	_, err := uc.RSVPGuests(context.Background(), wid, "b1")
	require.ErrorIs(t, err, usecase.ErrForbidden)
	gr.AssertNotCalled(t, "ListRSVP", mock.Anything, mock.Anything)
}

// Голосование

const pollOptions = `"options":[{"id":"a","text":"Шоколадный"},{"id":"b","text":"Медовик"}]`

func TestVote_SingleChoice(t *testing.T) {
	gr, wr, uc := newUC()
	wid, guest := uuid.New(), uuid.New()
	withBlock(wr, wid, "poll", `{`+pollOptions+`}`)
	gr.On("ListPollOptions", mock.Anything, "b1", guest, false).Return([]entity.PollOption{}, nil)
	gr.On("ReplacePollChoices", mock.Anything, wid, "b1", guest, []string{"b"}).Return(nil)
	gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{"b": 1}, []string{"b"}, nil)

	res, err := uc.Vote(context.Background(), wid, "b1", guest, []string{"b"})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"b": 1}, res.Votes)
	assert.Equal(t, []string{"b"}, res.MyVotes)

	_, err = uc.Vote(context.Background(), wid, "b1", guest, []string{"a", "b"})
	require.Error(t, err, "один вариант — значит один")

	_, err = uc.Vote(context.Background(), wid, "b1", guest, []string{"zzz"})
	require.Error(t, err, "несуществующий вариант")
}

func TestVote_MultipleChoice(t *testing.T) {
	gr, wr, uc := newUC()
	wid, guest := uuid.New(), uuid.New()
	withBlock(wr, wid, "poll", `{`+pollOptions+`,"multiple":true}`)
	gr.On("ListPollOptions", mock.Anything, "b1", guest, false).Return([]entity.PollOption{}, nil)
	gr.On("ReplacePollChoices", mock.Anything, wid, "b1", guest, []string{"a", "b"}).Return(nil)
	gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{"a": 1, "b": 1}, []string{"a", "b"}, nil)

	res, err := uc.Vote(context.Background(), wid, "b1", guest, []string{"a", "b", "a"})
	require.NoError(t, err)
	assert.Equal(t, 2, res.Total)
}

// Старые голосования хранили варианты строками: их id — индекс.
func TestVote_LegacyStringOptions(t *testing.T) {
	gr, wr, uc := newUC()
	wid, guest := uuid.New(), uuid.New()
	withBlock(wr, wid, "poll", `{"options":["Да","Нет"]}`)
	gr.On("ListPollOptions", mock.Anything, "b1", guest, false).Return([]entity.PollOption{}, nil)
	gr.On("ReplacePollChoices", mock.Anything, wid, "b1", guest, []string{"1"}).Return(nil)
	gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{"1": 1}, []string{"1"}, nil)

	_, err := uc.Vote(context.Background(), wid, "b1", guest, []string{"1"})
	require.NoError(t, err)
}

func TestVote_Closed(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	past := time.Now().Add(-time.Minute).Format(time.RFC3339)
	withBlock(wr, wid, "poll", `{`+pollOptions+`,"closesAt":"`+past+`"}`)

	_, err := uc.Vote(context.Background(), wid, "b1", uuid.New(), []string{"a"})
	require.ErrorIs(t, err, usecase.ErrClosed)
	gr.AssertNotCalled(t, "ReplacePollChoices", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPollResults_Visibility(t *testing.T) {
	cases := []struct {
		name       string
		results    string
		mine       []string
		viewer     uuid.UUID
		wantHidden bool
	}{
		{name: "всем сразу", results: "all", wantHidden: false},
		{name: "после голоса — не голосовал", results: "after_vote", wantHidden: true},
		{name: "после голоса — голосовал", results: "after_vote", mine: []string{"a"}, wantHidden: false},
		{name: "только мне — гость", results: "owner", mine: []string{"a"}, wantHidden: true},
		{name: "только мне — владелец", results: "owner", viewer: owner, wantHidden: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gr, wr, uc := newUC()
			wid, guest := uuid.New(), uuid.New()
			withBlock(wr, wid, "poll", `{`+pollOptions+`,"results":"`+tc.results+`"}`)
			gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{"a": 3}, tc.mine, nil)
			gr.On("ListPollOptions", mock.Anything, "b1", guest, tc.viewer == owner).Return([]entity.PollOption{}, nil)

			res, err := uc.PollResults(context.Background(), wid, "b1", guest, tc.viewer)
			require.NoError(t, err)
			assert.Equal(t, tc.wantHidden, res.Hidden)
			if tc.wantHidden {
				assert.Nil(t, res.Votes, "скрытые результаты не уходят в ответ")
				assert.Zero(t, res.Total)
			} else {
				assert.Equal(t, 3, res.Votes["a"])
			}
		})
	}
}

func TestAddPollOption(t *testing.T) {
	gr, wr, uc := newUC()
	wid, guest := uuid.New(), uuid.New()
	withBlock(wr, wid, "poll", `{`+pollOptions+`,"guestOptions":true}`)
	gr.On("CountPollOptionsByGuest", mock.Anything, "b1", guest).Return(int64(0), nil)
	gr.On("CreatePollOption", mock.Anything, mock.MatchedBy(func(o entity.PollGuestOption) bool {
		return o.Text == "Торт-вулкан" && o.BlockID == "b1" && o.WishlistID == wid && o.GuestID == guest
	})).Return(nil)
	gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{}, []string(nil), nil)
	gr.On("ListPollOptions", mock.Anything, "b1", guest, false).Return([]entity.PollOption{{ID: "x", Text: "Торт-вулкан", Mine: true}}, nil)

	res, err := uc.AddPollOption(context.Background(), wid, "b1", guest, "  Торт-вулкан ")
	require.NoError(t, err)
	assert.Len(t, res.GuestOptions, 1)
}

func TestAddPollOption_DisabledByOwner(t *testing.T) {
	gr, wr, uc := newUC()
	wid := uuid.New()
	withBlock(wr, wid, "poll", `{`+pollOptions+`}`)

	_, err := uc.AddPollOption(context.Background(), wid, "b1", uuid.New(), "Свой")
	require.ErrorIs(t, err, usecase.ErrForbidden)
	gr.AssertNotCalled(t, "CreatePollOption", mock.Anything, mock.Anything)
}

// Вариант гостя можно выбрать, скрытый — нельзя.
func TestVote_GuestOption(t *testing.T) {
	gr, wr, uc := newUC()
	wid, guest := uuid.New(), uuid.New()
	withBlock(wr, wid, "poll", `{`+pollOptions+`,"guestOptions":true}`)
	gid := uuid.NewString()
	gr.On("ListPollOptions", mock.Anything, "b1", guest, false).Return([]entity.PollOption{{ID: gid, Text: "Свой"}}, nil)
	gr.On("ReplacePollChoices", mock.Anything, wid, "b1", guest, []string{gid}).Return(nil)
	gr.On("CountPollChoices", mock.Anything, "b1", guest).Return(map[string]int{gid: 1}, []string{gid}, nil)

	_, err := uc.Vote(context.Background(), wid, "b1", guest, []string{gid})
	require.NoError(t, err)
}

func TestOwnerSetPollOptionHidden_RejectsStranger(t *testing.T) {
	gr, wr, uc := newUC()
	wid, optionID := uuid.New(), uuid.New()
	gr.On("PollOptionWishlist", mock.Anything, optionID).Return(wid, nil)
	wr.On("GetByID", mock.Anything, wid).Return(entity.Wishlist{ID: wid, UserID: owner}, nil)

	err := uc.OwnerSetPollOptionHidden(context.Background(), stranger, optionID, true)
	require.ErrorIs(t, err, usecase.ErrForbidden)
	gr.AssertNotCalled(t, "SetPollOptionHidden", mock.Anything, mock.Anything, mock.Anything)
}
