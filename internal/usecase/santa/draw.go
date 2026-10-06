package santa

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

func (uc *santaUseCase) Draw(ctx context.Context, ownerID, roomID uuid.UUID) error {
	return uc.draw(ctx, ownerID, roomID, entity.SantaRoomOpen, usecase.ErrSantaDrawn)
}

func (uc *santaUseCase) Redraw(ctx context.Context, ownerID, roomID uuid.UUID) error {
	return uc.draw(ctx, ownerID, roomID, entity.SantaRoomDrawn, usecase.ErrSantaNotDrawn)
}

// draw: статус проверяет репозиторий под блокировкой строки, а не мы до
// вызова — иначе два одновременных нажатия прошли бы оба.
func (uc *santaUseCase) draw(ctx context.Context, ownerID, roomID uuid.UUID, expected entity.SantaRoomStatus, wrongStatus error) error {
	room, err := uc.ownedRoom(ctx, ownerID, roomID)
	if err != nil {
		return err
	}
	err = uc.santa.Draw(ctx, room.ID, expected, func(ids []uuid.UUID) ([]entity.SantaAssignment, error) {
		return buildCycle(room.ID, ids, uc.shuffle)
	})
	if errors.Is(err, repo.ErrStatusMismatch) {
		return wrongStatus
	}
	return err
}
