package mockrepo

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockGuestDataRepo struct {
	mock.Mock
}

func (m *MockGuestDataRepo) UpsertRSVP(ctx context.Context, response entity.RSVPResponse) error {
	args := m.Called(ctx, response)
	return args.Error(0)
}

func (m *MockGuestDataRepo) ListRSVP(ctx context.Context, blockID string) ([]entity.RSVPResponse, error) {
	args := m.Called(ctx, blockID)
	if v, ok := args.Get(0).([]entity.RSVPResponse); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGuestDataRepo) ReplacePollChoices(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, optionIDs []string) error {
	return m.Called(ctx, wishlistID, blockID, guestID, optionIDs).Error(0)
}

func (m *MockGuestDataRepo) CountPollChoices(ctx context.Context, blockID string, guestID uuid.UUID) (map[string]int, []string, error) {
	args := m.Called(ctx, blockID, guestID)
	counts, _ := args.Get(0).(map[string]int)
	mine, _ := args.Get(1).([]string)
	return counts, mine, args.Error(2)
}

func (m *MockGuestDataRepo) CreatePollOption(ctx context.Context, option entity.PollGuestOption) error {
	return m.Called(ctx, option).Error(0)
}

func (m *MockGuestDataRepo) CountPollOptionsByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	args := m.Called(ctx, blockID, guestID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockGuestDataRepo) ListPollOptions(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.PollOption, error) {
	args := m.Called(ctx, blockID, guestID, includeHidden)
	opts, _ := args.Get(0).([]entity.PollOption)
	return opts, args.Error(1)
}

func (m *MockGuestDataRepo) PollOptionWishlist(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, optionID)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Error(1)
}

func (m *MockGuestDataRepo) SetPollOptionHidden(ctx context.Context, optionID uuid.UUID, hidden bool) error {
	return m.Called(ctx, optionID, hidden).Error(0)
}

func (m *MockGuestDataRepo) CountTracksByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	args := m.Called(ctx, blockID, guestID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockGuestDataRepo) CreateTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) (uuid.UUID, error) {
	args := m.Called(ctx, wishlistID, blockID, guestID, title)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Error(1)
}

func (m *MockGuestDataRepo) ListTracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	args := m.Called(ctx, blockID, guestID)
	if v, ok := args.Get(0).([]entity.PlaylistTrack); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGuestDataRepo) TrackBlockID(ctx context.Context, trackID uuid.UUID) (uuid.UUID, string, error) {
	args := m.Called(ctx, trackID)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.String(1), args.Error(2)
}

func (m *MockGuestDataRepo) ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) (bool, error) {
	args := m.Called(ctx, trackID, guestID)
	return args.Bool(0), args.Error(1)
}

func (m *MockGuestDataRepo) CountGuestbookByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	args := m.Called(ctx, blockID, guestID)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockGuestDataRepo) CreateGuestbookEntry(ctx context.Context, entry entity.GuestbookEntry, wishlistID uuid.UUID, blockID string, guestID uuid.UUID) error {
	args := m.Called(ctx, entry, wishlistID, blockID, guestID)
	return args.Error(0)
}

func (m *MockGuestDataRepo) ListGuestbook(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.GuestbookEntry, error) {
	args := m.Called(ctx, blockID, guestID, includeHidden)
	if v, ok := args.Get(0).([]entity.GuestbookEntry); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGuestDataRepo) GuestbookEntryWishlist(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, entryID)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Error(1)
}

func (m *MockGuestDataRepo) SetGuestbookHidden(ctx context.Context, entryID uuid.UUID, hidden bool) error {
	args := m.Called(ctx, entryID, hidden)
	return args.Error(0)
}
