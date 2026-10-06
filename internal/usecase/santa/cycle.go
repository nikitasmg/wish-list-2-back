package santa

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/usecase"
)

const minParticipants = 3

// shuffleFunc перемешивает срез на месте. В тестах подменяется.
type shuffleFunc func([]uuid.UUID)

// cryptoShuffle — Фишер–Йетс на crypto/rand: по результату нельзя
// предсказать пары, даже зная время жеребьёвки.
func cryptoShuffle(ids []uuid.UUID) {
	for i := len(ids) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			panic(fmt.Sprintf("crypto/rand: %v", err))
		}
		j := int(n.Int64())
		ids[i], ids[j] = ids[j], ids[i]
	}
}

// buildCycle ставит участников в один круг: i-й дарит (i+1)-му, последний —
// первому. Не случайная перестановка: в ней кто-то вытянул бы себя, а
// компания распадалась бы на пары «дарим друг другу».
func buildCycle(roomID uuid.UUID, ids []uuid.UUID, shuffle shuffleFunc) ([]entity.SantaAssignment, error) {
	if len(ids) < minParticipants {
		return nil, usecase.ErrSantaTooFew
	}
	order := append([]uuid.UUID(nil), ids...)
	shuffle(order)
	pairs := make([]entity.SantaAssignment, len(order))
	for i, giver := range order {
		pairs[i] = entity.SantaAssignment{RoomID: roomID, GiverID: giver, ReceiverID: order[(i+1)%len(order)]}
	}
	return pairs, nil
}
