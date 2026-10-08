package santa

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

// mapRoomWriteErr переводит ошибки записи репозитория (гонка с жеребьёвкой
// или удалением) в ошибки use case; прочие возвращает как есть.
func mapRoomWriteErr(err error) error {
	switch {
	case errors.Is(err, repo.ErrStatusMismatch):
		return usecase.ErrSantaDrawn
	case errors.Is(err, repo.ErrNotFound):
		return usecase.ErrSantaNotFound
	}
	return err
}

func (uc *santaUseCase) roomBySlug(ctx context.Context, slug string) (entity.SantaRoom, error) {
	room, err := uc.santa.GetRoomBySlug(ctx, slug)
	if errors.Is(err, repo.ErrNotFound) {
		return entity.SantaRoom{}, usecase.ErrSantaNotFound
	}
	return room, err
}

func (uc *santaUseCase) invite(ctx context.Context, room entity.SantaRoom) (usecase.SantaInvite, error) {
	counts, err := uc.santa.CountParticipants(ctx, []uuid.UUID{room.ID})
	if err != nil {
		return usecase.SantaInvite{}, err
	}
	// Имя организатора — украшение приглашения: если пользователь не
	// нашёлся, карточка всё равно открывается; остальные сбои не глотаем.
	// userRepo.GetByID отдаёт gorm.ErrRecordNotFound, а не repo.ErrNotFound.
	organizer := ""
	owner, err := uc.users.GetByID(ctx, room.OwnerID)
	switch {
	case err == nil:
		organizer = owner.DisplayName
		if organizer == "" {
			organizer = owner.Username
		}
	case errors.Is(err, repo.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
	default:
		return usecase.SantaInvite{}, fmt.Errorf("organizer: %w", err)
	}
	return usecase.SantaInvite{
		Slug: room.Slug, Title: room.Title, OrganizerName: organizer, Budget: room.Budget,
		ExchangeDate: room.ExchangeDate, DrawAt: room.DrawAt, Message: room.Message,
		ParticipantsCount: counts[room.ID], Status: room.Status, DrawnAt: room.DrawnAt,
	}, nil
}

// participant: сначала секрет из личной ссылки, потом аккаунт. Устаревший
// токен в браузере не мешает войти по аккаунту.
func (uc *santaUseCase) participant(ctx context.Context, room entity.SantaRoom, auth usecase.SantaAuth) (entity.SantaParticipant, error) {
	if auth.Token != "" {
		p, err := uc.santa.GetParticipantByToken(ctx, room.ID, hashToken(auth.Token))
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return entity.SantaParticipant{}, err
		}
	}
	if auth.UserID != nil {
		p, err := uc.santa.GetParticipantByUser(ctx, room.ID, *auth.UserID)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return entity.SantaParticipant{}, err
		}
	}
	return entity.SantaParticipant{}, usecase.ErrSantaNotFound
}

func (uc *santaUseCase) me(ctx context.Context, room entity.SantaRoom, p entity.SantaParticipant) (usecase.SantaMe, error) {
	inv, err := uc.invite(ctx, room)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	me := usecase.SantaMe{
		ParticipantID: p.ID, Name: p.Name, Wishes: p.Wishes, WishlistURL: p.WishlistURL, Room: inv,
		Notify: notifyView(p),
	}
	if room.Status != entity.SantaRoomDrawn {
		return me, nil
	}
	a, err := uc.santa.GetAssignment(ctx, room.ID, p.ID)
	if errors.Is(err, repo.ErrNotFound) {
		return me, nil
	}
	if err != nil {
		return usecase.SantaMe{}, err
	}
	ward, err := uc.santa.GetParticipant(ctx, a.ReceiverID)
	if err != nil {
		return usecase.SantaMe{}, fmt.Errorf("receiver: %w", err)
	}
	me.Receiver = &usecase.SantaReceiver{Name: ward.Name, Wishes: ward.Wishes, WishlistURL: ward.WishlistURL}
	return me, nil
}

func (uc *santaUseCase) GetInvite(ctx context.Context, slug string) (usecase.SantaInvite, error) {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return usecase.SantaInvite{}, err
	}
	return uc.invite(ctx, room)
}

func (uc *santaUseCase) Join(ctx context.Context, slug string, auth usecase.SantaAuth, in usecase.SantaProfileInput) (usecase.SantaJoinResult, error) {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return usecase.SantaJoinResult{}, err
	}
	if room.Status != entity.SantaRoomOpen {
		return usecase.SantaJoinResult{}, usecase.ErrSantaDrawn
	}
	in = normalizeProfile(in)
	if err := validateProfile(in); err != nil {
		return usecase.SantaJoinResult{}, err
	}
	// Токен этого браузера уже ведёт к участнику комнаты: второй раз не пускаем,
	// иначе один человек попадёт в жеребьёвку дважды.
	if auth.Token != "" {
		_, err := uc.santa.GetParticipantByToken(ctx, room.ID, hashToken(auth.Token))
		if err == nil {
			return usecase.SantaJoinResult{}, usecase.ErrSantaAlreadyJoined
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return usecase.SantaJoinResult{}, err
		}
	}
	userID := auth.UserID
	if userID != nil {
		_, err := uc.santa.GetParticipantByUser(ctx, room.ID, *userID)
		if err == nil {
			return usecase.SantaJoinResult{}, usecase.ErrSantaAlreadyJoined
		}
		if !errors.Is(err, repo.ErrNotFound) {
			return usecase.SantaJoinResult{}, err
		}
	}
	counts, err := uc.santa.CountParticipants(ctx, []uuid.UUID{room.ID})
	if err != nil {
		return usecase.SantaJoinResult{}, err
	}
	if counts[room.ID] >= maxParticipants {
		return usecase.SantaJoinResult{}, invalid("в комнате уже 100 участников")
	}

	raw, hash, err := newToken()
	if err != nil {
		return usecase.SantaJoinResult{}, err
	}
	now := uc.now()
	p := entity.SantaParticipant{
		ID: uuid.New(), RoomID: room.ID, UserID: userID, Name: in.Name, Wishes: in.Wishes,
		WishlistURL: in.WishlistURL, TokenHash: hash, CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.santa.CreateParticipant(ctx, p); err != nil {
		// Жеребьёвка могла пройти между проверкой и вставкой.
		if errors.Is(err, repo.ErrDuplicate) {
			return usecase.SantaJoinResult{}, usecase.ErrSantaAlreadyJoined
		}
		if mapped := mapRoomWriteErr(err); mapped != err {
			return usecase.SantaJoinResult{}, mapped
		}
		return usecase.SantaJoinResult{}, fmt.Errorf("join: %w", err)
	}
	me, err := uc.me(ctx, room, p)
	if err != nil {
		// Участник уже записан, а токен отдаётся один раз: потерять его из-за
		// сбоя при сборке карточки нельзя. Отдаём минимальную карточку из
		// того, что есть под рукой; полную фронт получит запросом /me.
		log.Printf("santa: join: me: %v", err)
		me = usecase.SantaMe{
			ParticipantID: p.ID, Name: p.Name, Wishes: p.Wishes, WishlistURL: p.WishlistURL,
			Notify: notifyView(p),
			Room: usecase.SantaInvite{
				Slug: room.Slug, Title: room.Title, Budget: room.Budget, ExchangeDate: room.ExchangeDate,
				DrawAt: room.DrawAt, Message: room.Message, Status: room.Status, DrawnAt: room.DrawnAt,
			},
		}
	}
	return usecase.SantaJoinResult{Token: raw, Me: me}, nil
}

func (uc *santaUseCase) GetMe(ctx context.Context, slug string, auth usecase.SantaAuth) (usecase.SantaMe, error) {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	p, err := uc.participant(ctx, room, auth)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	return uc.me(ctx, room, p)
}

func (uc *santaUseCase) UpdateMe(ctx context.Context, slug string, auth usecase.SantaAuth, in usecase.SantaProfileInput) (usecase.SantaMe, error) {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	p, err := uc.participant(ctx, room, auth)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	in = normalizeProfile(in)
	if err := validateProfile(in); err != nil {
		return usecase.SantaMe{}, err
	}
	if room.Status != entity.SantaRoomOpen && in.Name != p.Name {
		return usecase.SantaMe{}, usecase.ErrSantaDrawn
	}
	changed := p.Wishes != in.Wishes || p.WishlistURL != in.WishlistURL
	p.Name = in.Name
	p.Wishes = in.Wishes
	p.WishlistURL = in.WishlistURL
	p.UpdatedAt = uc.now()

	var notes []entity.SantaNotification
	if room.Status == entity.SantaRoomDrawn && changed {
		a, err := uc.santa.GetGiver(ctx, room.ID, p.ID)
		switch {
		case err == nil:
			notes = append(notes, entity.NewSantaNotification(a.GiverID, entity.SantaNotifyWishesUpdated, p.UpdatedAt))
		case !errors.Is(err, repo.ErrNotFound):
			return usecase.SantaMe{}, err
		}
	}
	if err := uc.santa.UpdateParticipant(ctx, p, notes...); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return usecase.SantaMe{}, usecase.ErrSantaNotFound
		}
		return usecase.SantaMe{}, err
	}
	return uc.me(ctx, room, p)
}

func (uc *santaUseCase) LeaveMe(ctx context.Context, slug string, auth usecase.SantaAuth) error {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return err
	}
	if room.Status != entity.SantaRoomOpen {
		return usecase.ErrSantaDrawn
	}
	p, err := uc.participant(ctx, room, auth)
	if err != nil {
		return err
	}
	// Жеребьёвка могла пройти между проверкой и удалением.
	return mapRoomWriteErr(uc.santa.DeleteParticipant(ctx, p.ID))
}

func notifyView(p entity.SantaParticipant) usecase.SantaNotifyView {
	return usecase.SantaNotifyView{
		Channel:       p.Channel,
		Email:         p.Email,
		EmailVerified: p.Email != "" && p.EmailVerifiedAt != nil,
		PendingEmail:  p.PendingEmail,
		EmailPending:  p.PendingEmail != "",
		Telegram:      p.TgChatID != nil,
		Ready:         p.Ready(),
	}
}
