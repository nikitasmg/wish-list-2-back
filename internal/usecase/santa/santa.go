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
	santa       repo.SantaRepo
	users       repo.UserRepo
	shuffle     shuffleFunc
	now         func() time.Time
	mailer      usecase.Mailer
	tg          usecase.TelegramSender
	botUsername string
	emailQuota  *emailQuota
}

// Option подключает каналы уведомлений; без них use case работает как на этапе 1.
type Option func(*santaUseCase)

// WithMailer — почта для кодов. Без него RequestEmailCode отвечает
// ErrSantaUnavailable: в продакшене без SMTP мейлер не передают, чтобы код
// не «уходил» в лог с ответом «отправлено».
func WithMailer(m usecase.Mailer) Option { return func(uc *santaUseCase) { uc.mailer = m } }

// WithTelegram — бот для ответов на /start и ссылок. Пустой botUsername —
// ссылки не выдаются (ErrSantaUnavailable): его передают пустым и тогда,
// когда вебхук не зарегистрирован и по ссылке никто не ответит.
func WithTelegram(tg usecase.TelegramSender, botUsername string) Option {
	return func(uc *santaUseCase) {
		uc.tg = tg
		uc.botUsername = botUsername
	}
}

func New(santaRepo repo.SantaRepo, userRepo repo.UserRepo, opts ...Option) usecase.SantaUseCase {
	uc := &santaUseCase{
		santa: santaRepo, users: userRepo, shuffle: cryptoShuffle, now: time.Now,
		emailQuota: newEmailQuota(emailCodesPerHour, emailQuotaWindow),
	}
	for _, opt := range opts {
		opt(uc)
	}
	return uc
}

func (uc *santaUseCase) CreateRoom(ctx context.Context, ownerID uuid.UUID, in usecase.SantaRoomInput) (entity.SantaRoom, error) {
	now := uc.now()
	in = normalizeRoom(in)
	if err := validateRoom(in, now); err != nil {
		return entity.SantaRoom{}, err
	}
	organizer := normalizeProfile(usecase.SantaProfileInput{Name: in.OrganizerName, Wishes: in.OrganizerWishes})
	if in.OrganizerJoins {
		if err := validateProfile(organizer); err != nil {
			return entity.SantaRoom{}, err
		}
	}

	var room entity.SantaRoom
	created := false
	for attempt := 0; attempt < slugAttempts && !created; attempt++ {
		slug, err := uc.freeSlug(ctx)
		if err != nil {
			return entity.SantaRoom{}, err
		}
		room = entity.SantaRoom{
			ID: uuid.New(), OwnerID: ownerID, Slug: slug, Title: in.Title, Budget: in.Budget,
			ExchangeDate: in.ExchangeDate, DrawAt: in.DrawAt, Message: in.Message,
			Status: entity.SantaRoomOpen, CreatedAt: now, UpdatedAt: now,
		}
		err = uc.santa.CreateRoom(ctx, room)
		if err == nil {
			created = true
			break
		}
		// Адрес могли занять между проверкой и записью — тогда берём другой.
		if _, lookupErr := uc.santa.GetRoomBySlug(ctx, slug); lookupErr != nil {
			return entity.SantaRoom{}, fmt.Errorf("create room: %w", err)
		}
	}
	if !created {
		return entity.SantaRoom{}, errors.New("не удалось подобрать адрес комнаты")
	}

	if in.OrganizerJoins {
		if err := uc.addOrganizer(ctx, room, organizer, now); err != nil {
			// Комната без организатора-участника не нужна: откатываем вручную,
			// участники и пары уйдут каскадом.
			if delErr := uc.santa.DeleteRoom(ctx, room.ID); delErr != nil {
				return entity.SantaRoom{}, errors.Join(err, fmt.Errorf("rollback room: %w", delErr))
			}
			return entity.SantaRoom{}, err
		}
	}
	return room, nil
}

func (uc *santaUseCase) addOrganizer(ctx context.Context, room entity.SantaRoom, organizer usecase.SantaProfileInput, now time.Time) error {
	// Организатор входит по аккаунту, личная ссылка ему не нужна —
	// токен генерируется только ради непустого уникального хэша.
	_, hash, err := newToken()
	if err != nil {
		return err
	}
	owner := room.OwnerID
	p := entity.SantaParticipant{
		ID: uuid.New(), RoomID: room.ID, UserID: &owner, Name: organizer.Name,
		Wishes: organizer.Wishes, TokenHash: hash, CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.santa.CreateParticipant(ctx, p); err != nil {
		return fmt.Errorf("create organizer: %w", err)
	}
	return nil
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
			ID: p.ID, Name: p.Name, HasWishes: p.Wishes != "", HasWishlist: p.WishlistURL != "", Ready: p.Ready(),
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
	// Время жеребьёвки проверяем на «будущее», только если его меняют:
	// название комнаты с уже прошедшим draw_at должно править.
	check := in
	if sameTime(in.DrawAt, room.DrawAt) {
		check.DrawAt = nil
	}
	if err := validateRoom(check, uc.now()); err != nil {
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

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
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
	// Жеребьёвка могла пройти между проверкой и удалением.
	err = uc.santa.DeleteParticipant(ctx, p.ID)
	if errors.Is(err, repo.ErrStatusMismatch) {
		return usecase.ErrSantaDrawn
	}
	if errors.Is(err, repo.ErrNotFound) {
		return usecase.ErrSantaNotFound
	}
	return err
}
