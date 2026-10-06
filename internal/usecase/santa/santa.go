package santa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

const (
	maxParticipants = 100
	slugAttempts    = 5
)

type santaUseCase struct {
	santa   repo.SantaRepo
	users   repo.UserRepo
	shuffle shuffleFunc
	now     func() time.Time
}

func New(santaRepo repo.SantaRepo, userRepo repo.UserRepo) usecase.SantaUseCase {
	return &santaUseCase{santa: santaRepo, users: userRepo, shuffle: cryptoShuffle, now: time.Now}
}

func (uc *santaUseCase) CreateRoom(ctx context.Context, ownerID uuid.UUID, in usecase.SantaRoomInput) (entity.SantaRoom, error) {
	in = normalizeRoom(in)
	if err := validateRoom(in, uc.now()); err != nil {
		return entity.SantaRoom{}, err
	}
	organizer := normalizeProfile(usecase.SantaProfileInput{Name: in.OrganizerName, Wishes: in.OrganizerWishes})
	if in.OrganizerJoins {
		if err := validateProfile(organizer); err != nil {
			return entity.SantaRoom{}, err
		}
	}

	slug, err := uc.freeSlug(ctx)
	if err != nil {
		return entity.SantaRoom{}, err
	}
	now := uc.now()
	room := entity.SantaRoom{
		ID: uuid.New(), OwnerID: ownerID, Slug: slug, Title: in.Title, Budget: in.Budget,
		ExchangeDate: in.ExchangeDate, DrawAt: in.DrawAt, Message: in.Message,
		Status: entity.SantaRoomOpen, CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.santa.CreateRoom(ctx, room); err != nil {
		return entity.SantaRoom{}, fmt.Errorf("create room: %w", err)
	}

	if in.OrganizerJoins {
		// Организатор входит по аккаунту, личная ссылка ему не нужна —
		// токен генерируется только ради непустого уникального хэша.
		_, hash, err := newToken()
		if err != nil {
			return entity.SantaRoom{}, err
		}
		owner := ownerID
		p := entity.SantaParticipant{
			ID: uuid.New(), RoomID: room.ID, UserID: &owner, Name: organizer.Name,
			Wishes: organizer.Wishes, TokenHash: hash, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.santa.CreateParticipant(ctx, p); err != nil {
			return entity.SantaRoom{}, fmt.Errorf("create organizer: %w", err)
		}
	}
	return room, nil
}

func (uc *santaUseCase) freeSlug(ctx context.Context) (string, error) {
	for i := 0; i < slugAttempts; i++ {
		slug, err := newSlug()
		if err != nil {
			return "", err
		}
		_, err = uc.santa.GetRoomBySlug(ctx, slug)
		if errors.Is(err, repo.ErrNotFound) {
			return slug, nil
		}
		if err != nil {
			return "", fmt.Errorf("check slug: %w", err)
		}
	}
	return "", errors.New("не удалось подобрать адрес комнаты")
}

// ownedRoom отвечает «не найдено» и на чужую комнату: по id нельзя узнать,
// существует ли она.
func (uc *santaUseCase) ownedRoom(ctx context.Context, ownerID, roomID uuid.UUID) (entity.SantaRoom, error) {
	room, err := uc.santa.GetRoomByID(ctx, roomID)
	if errors.Is(err, repo.ErrNotFound) {
		return entity.SantaRoom{}, usecase.ErrSantaNotFound
	}
	if err != nil {
		return entity.SantaRoom{}, err
	}
	if room.OwnerID != ownerID {
		return entity.SantaRoom{}, usecase.ErrSantaNotFound
	}
	return room, nil
}

func (uc *santaUseCase) ListRooms(ctx context.Context, userID uuid.UUID) ([]usecase.SantaRoomSummary, error) {
	rooms, err := uc.santa.ListRoomsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rooms))
	for i, r := range rooms {
		ids[i] = r.ID
	}
	counts, err := uc.santa.CountParticipants(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.SantaRoomSummary, len(rooms))
	for i, r := range rooms {
		out[i] = usecase.SantaRoomSummary{SantaRoom: r, IsOwner: r.OwnerID == userID, ParticipantsCount: counts[r.ID]}
	}
	return out, nil
}

func (uc *santaUseCase) GetRoom(ctx context.Context, ownerID, roomID uuid.UUID) (usecase.SantaRoomDetails, error) {
	room, err := uc.ownedRoom(ctx, ownerID, roomID)
	if err != nil {
		return usecase.SantaRoomDetails{}, err
	}
	ps, err := uc.santa.ListParticipants(ctx, room.ID)
	if err != nil {
		return usecase.SantaRoomDetails{}, err
	}
	views := make([]usecase.SantaParticipantView, len(ps))
	for i, p := range ps {
		views[i] = usecase.SantaParticipantView{
			ID: p.ID, Name: p.Name, HasWishes: p.Wishes != "", HasWishlist: p.WishlistURL != "",
			IsOwner: p.UserID != nil && *p.UserID == room.OwnerID, CreatedAt: p.CreatedAt,
		}
	}
	return usecase.SantaRoomDetails{Room: room, Participants: views}, nil
}

func (uc *santaUseCase) UpdateRoom(ctx context.Context, ownerID, roomID uuid.UUID, in usecase.SantaRoomInput) (entity.SantaRoom, error) {
	room, err := uc.ownedRoom(ctx, ownerID, roomID)
	if err != nil {
		return entity.SantaRoom{}, err
	}
	if room.Status != entity.SantaRoomOpen {
		return entity.SantaRoom{}, usecase.ErrSantaDrawn
	}
	in = normalizeRoom(in)
	if err := validateRoom(in, uc.now()); err != nil {
		return entity.SantaRoom{}, err
	}
	room.Title = in.Title
	room.Budget = in.Budget
	room.ExchangeDate = in.ExchangeDate
	room.DrawAt = in.DrawAt
	room.Message = in.Message
	room.UpdatedAt = uc.now()
	if err := uc.santa.UpdateRoom(ctx, room); err != nil {
		// Комнату могли разыграть или удалить между проверкой и записью.
		if errors.Is(err, repo.ErrStatusMismatch) {
			return entity.SantaRoom{}, usecase.ErrSantaDrawn
		}
		if errors.Is(err, repo.ErrNotFound) {
			return entity.SantaRoom{}, usecase.ErrSantaNotFound
		}
		return entity.SantaRoom{}, err
	}
	return room, nil
}

func (uc *santaUseCase) DeleteRoom(ctx context.Context, ownerID, roomID uuid.UUID) error {
	if _, err := uc.ownedRoom(ctx, ownerID, roomID); err != nil {
		return err
	}
	return uc.santa.DeleteRoom(ctx, roomID)
}

func (uc *santaUseCase) RemoveParticipant(ctx context.Context, ownerID, roomID, participantID uuid.UUID) error {
	room, err := uc.ownedRoom(ctx, ownerID, roomID)
	if err != nil {
		return err
	}
	if room.Status != entity.SantaRoomOpen {
		return usecase.ErrSantaDrawn
	}
	p, err := uc.santa.GetParticipant(ctx, participantID)
	if errors.Is(err, repo.ErrNotFound) || (err == nil && p.RoomID != room.ID) {
		return usecase.ErrSantaNotFound
	}
	if err != nil {
		return err
	}
	return uc.santa.DeleteParticipant(ctx, p.ID)
}
