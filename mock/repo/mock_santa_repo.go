package mockrepo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"main/internal/entity"
	"main/internal/repo"
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

func (m *MockSantaRepo) UpdateParticipant(ctx context.Context, p entity.SantaParticipant, notes ...entity.SantaNotification) error {
	return m.Called(ctx, p, notes).Error(0)
}

func (m *MockSantaRepo) DeleteParticipant(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockSantaRepo) GetAssignment(ctx context.Context, roomID, giverID uuid.UUID) (entity.SantaAssignment, error) {
	args := m.Called(ctx, roomID, giverID)
	return args.Get(0).(entity.SantaAssignment), args.Error(1)
}

func (m *MockSantaRepo) Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func([]uuid.UUID) ([]entity.SantaAssignment, error), note func(uuid.UUID) entity.SantaNotification) error {
	return m.Called(ctx, roomID, expected, build, note).Error(0)
}

func (m *MockSantaRepo) GetGiver(ctx context.Context, roomID, receiverID uuid.UUID) (entity.SantaAssignment, error) {
	args := m.Called(ctx, roomID, receiverID)
	a, _ := args.Get(0).(entity.SantaAssignment)
	return a, args.Error(1)
}

func (m *MockSantaRepo) Remind(ctx context.Context, roomID uuid.UUID, now time.Time, cooldown time.Duration, notes []entity.SantaNotification) error {
	return m.Called(ctx, roomID, now, cooldown, notes).Error(0)
}

func (m *MockSantaRepo) ClaimNotifications(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]entity.SantaNotification, error) {
	args := m.Called(ctx, now, limit, lease)
	notes, _ := args.Get(0).([]entity.SantaNotification)
	return notes, args.Error(1)
}

func (m *MockSantaRepo) MarkNotificationSent(ctx context.Context, id uuid.UUID, tgMessageID *int64) error {
	return m.Called(ctx, id, tgMessageID).Error(0)
}

func (m *MockSantaRepo) MarkNotificationFailed(ctx context.Context, id uuid.UUID, attempts int, retryAt *time.Time, lastErr string) error {
	return m.Called(ctx, id, attempts, retryAt, lastErr).Error(0)
}

func (m *MockSantaRepo) SetEmail(ctx context.Context, participantID uuid.UUID, email string, code entity.SantaEmailCode) error {
	return m.Called(ctx, participantID, email, code).Error(0)
}

func (m *MockSantaRepo) GetEmailCode(ctx context.Context, participantID uuid.UUID) (entity.SantaEmailCode, error) {
	args := m.Called(ctx, participantID)
	code, _ := args.Get(0).(entity.SantaEmailCode)
	return code, args.Error(1)
}

func (m *MockSantaRepo) IncEmailCodeAttempts(ctx context.Context, participantID uuid.UUID, max int) (bool, error) {
	args := m.Called(ctx, participantID, max)
	return args.Bool(0), args.Error(1)
}

func (m *MockSantaRepo) DeleteEmailCode(ctx context.Context, participantID uuid.UUID) error {
	return m.Called(ctx, participantID).Error(0)
}

func (m *MockSantaRepo) VerifyEmail(ctx context.Context, participantID uuid.UUID, codeHash string, at time.Time, welcome entity.SantaNotification) error {
	return m.Called(ctx, participantID, codeHash, at, welcome).Error(0)
}

func (m *MockSantaRepo) CreateTgLink(ctx context.Context, link entity.SantaTgLink) error {
	return m.Called(ctx, link).Error(0)
}

func (m *MockSantaRepo) LinkTelegram(ctx context.Context, tokenHash string, chatID int64, now time.Time, welcome func(entity.SantaParticipant) entity.SantaNotification) (entity.SantaParticipant, error) {
	args := m.Called(ctx, tokenHash, chatID, now, welcome)
	p, _ := args.Get(0).(entity.SantaParticipant)
	return p, args.Error(1)
}

func (m *MockSantaRepo) PurgeStale(ctx context.Context, now time.Time, keepNotes time.Duration) (int64, error) {
	args := m.Called(ctx, now, keepNotes)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSantaRepo) DueDrawRooms(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	args := m.Called(ctx, now, limit)
	ids, _ := args.Get(0).([]uuid.UUID)
	return ids, args.Error(1)
}

func (m *MockSantaRepo) DrawScheduled(ctx context.Context, roomID uuid.UUID, now time.Time, minReady int, build func([]uuid.UUID) ([]entity.SantaAssignment, error), note func(uuid.UUID) entity.SantaNotification, failNote func(uuid.UUID) entity.SantaNotification) (repo.ScheduledDrawOutcome, error) {
	args := m.Called(ctx, roomID, now, minReady, build, note, failNote)
	return args.Get(0).(repo.ScheduledDrawOutcome), args.Error(1)
}

func (m *MockSantaRepo) CreateMessage(ctx context.Context, msg entity.SantaMessage, since time.Time, limit int, note entity.SantaNotification) error {
	return m.Called(ctx, msg, since, limit, note).Error(0)
}

func (m *MockSantaRepo) GetMessage(ctx context.Context, id uuid.UUID) (entity.SantaMessage, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.SantaMessage), args.Error(1)
}

func (m *MockSantaRepo) ListMessages(ctx context.Context, roomID, giverID, receiverID uuid.UUID, limit int) ([]entity.SantaMessage, error) {
	args := m.Called(ctx, roomID, giverID, receiverID, limit)
	msgs, _ := args.Get(0).([]entity.SantaMessage)
	return msgs, args.Error(1)
}

func (m *MockSantaRepo) MarkMessagesRead(ctx context.Context, roomID, giverID, receiverID uuid.UUID, fromGiver bool, at time.Time) error {
	return m.Called(ctx, roomID, giverID, receiverID, fromGiver, at).Error(0)
}

func (m *MockSantaRepo) CountUnread(ctx context.Context, roomID, participantID uuid.UUID) (int, int, error) {
	args := m.Called(ctx, roomID, participantID)
	return args.Int(0), args.Int(1), args.Error(2)
}

func (m *MockSantaRepo) FindChatNotification(ctx context.Context, chatID, tgMessageID int64) (entity.SantaNotification, error) {
	args := m.Called(ctx, chatID, tgMessageID)
	return args.Get(0).(entity.SantaNotification), args.Error(1)
}
