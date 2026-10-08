package santa

import (
	"context"
	"errors"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

func (uc *santaUseCase) SetGiftReady(ctx context.Context, slug string, auth usecase.SantaAuth, ready bool) (usecase.SantaMe, error) {
	room, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	if room.Status != entity.SantaRoomDrawn {
		return usecase.SantaMe{}, usecase.ErrSantaNotDrawn
	}
	if err := uc.santa.SetGiftReady(ctx, room.ID, p.ID, ready); err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			return usecase.SantaMe{}, usecase.ErrSantaNotInDraw
		case errors.Is(err, repo.ErrStatusMismatch):
			return usecase.SantaMe{}, usecase.ErrSantaNotDrawn
		}
		return usecase.SantaMe{}, err
	}
	p.GiftReady = ready
	return uc.me(ctx, room, p)
}
