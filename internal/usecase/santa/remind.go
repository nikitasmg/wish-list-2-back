package santa

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"main/internal/usecase"
)

// Remind — реализация в задаче 6.
func (uc *santaUseCase) Remind(ctx context.Context, ownerID, roomID uuid.UUID) (usecase.SantaRemindResult, error) {
	return usecase.SantaRemindResult{}, errors.New("not implemented")
}
