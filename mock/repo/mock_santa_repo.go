package mockrepo

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
)

type MockSantaRepo struct {
	mock.Mock
}

func (m *MockSantaRepo) CreateRoom(ctx context.Context, room entity.SantaRoom) error {
	return m.Called(ctx, room).Error(0)
}

func (m *MockSantaRepo) GetRoomByID(ctx context.Context, id uuid.UUID) (entity.SantaRoom, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.SantaRoom), args.Error(1)
}

func (m *MockSantaRepo) GetRoomBySlug(ctx context.Context, slug string) (entity.SantaRoom, error) {
	args := m.Called(ctx, slug)
	return args.Get(0).(entity.SantaRoom), args.Error(1)
}

func (m *MockSantaRepo) ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SantaRoom, error) {
	args := m.Called(ctx, userID)
	rooms, _ := args.Get(0).([]entity.SantaRoom)
	return rooms, args.Error(1)
}

func (m *MockSantaRepo) UpdateRoom(ctx context.Context, room entity.SantaRoom) error {
	return m.Called(ctx, room).Error(0)
}

func (m *MockSantaRepo) DeleteRoom(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSantaRepo) CreateParticipant(ctx context.Context, p entity.SantaParticipant) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockSantaRepo) GetParticipant(ctx context.Context, id uuid.UUID) (entity.SantaParticipant, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.SantaParticipant), args.Error(1)
}

func (m *MockSantaRepo) GetParticipantByToken(ctx context.Context, roomID uuid.UUID, tokenHash string) (entity.SantaParticipant, error) {
	args := m.Called(ctx, roomID, tokenHash)
	return args.Get(0).(entity.SantaParticipant), args.Error(1)
}

func (m *MockSantaRepo) GetParticipantByUser(ctx context.Context, roomID, userID uuid.UUID) (entity.SantaParticipant, error) {
	args := m.Called(ctx, roomID, userID)
	return args.Get(0).(entity.SantaParticipant), args.Error(1)
}

func (m *MockSantaRepo) ListParticipants(ctx context.Context, roomID uuid.UUID) ([]entity.SantaParticipant, error) {
	args := m.Called(ctx, roomID)
	ps, _ := args.Get(0).([]entity.SantaParticipant)
	return ps, args.Error(1)
}

func (m *MockSantaRepo) CountParticipants(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	args := m.Called(ctx, roomIDs)
	counts, _ := args.Get(0).(map[uuid.UUID]int)
	return counts, args.Error(1)
}

func (m *MockSantaRepo) UpdateParticipant(ctx context.Context, p entity.SantaParticipant) error {
	return m.Called(ctx, p).Error(0)
}

func (m *MockSantaRepo) DeleteParticipant(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSantaRepo) GetAssignment(ctx context.Context, roomID, giverID uuid.UUID) (entity.SantaAssignment, error) {
	args := m.Called(ctx, roomID, giverID)
	return args.Get(0).(entity.SantaAssignment), args.Error(1)
}

func (m *MockSantaRepo) Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func([]uuid.UUID) ([]entity.SantaAssignment, error)) error {
	return m.Called(ctx, roomID, expected, build).Error(0)
}
