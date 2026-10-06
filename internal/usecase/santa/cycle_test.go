package santa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
)

func newIDs(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.New()
	}
	return ids
}

func TestBuildCycle_TooFew(t *testing.T) {
	_, err := buildCycle(uuid.New(), newIDs(2), cryptoShuffle)
	assert.ErrorIs(t, err, usecase.ErrSantaTooFew)
}

func TestBuildCycle_ChainsInShuffledOrder(t *testing.T) {
	room := uuid.New()
	ids := newIDs(3)
	pairs, err := buildCycle(room, ids, func([]uuid.UUID) {})
	require.NoError(t, err)
	assert.Equal(t, []entity.SantaAssignment{
		{RoomID: room, GiverID: ids[0], ReceiverID: ids[1]},
		{RoomID: room, GiverID: ids[1], ReceiverID: ids[2]},
		{RoomID: room, GiverID: ids[2], ReceiverID: ids[0]},
	}, pairs)
}

func TestBuildCycle_DoesNotMutateInput(t *testing.T) {
	ids := newIDs(5)
	before := append([]uuid.UUID(nil), ids...)
	_, err := buildCycle(uuid.New(), ids, cryptoShuffle)
	require.NoError(t, err)
	assert.Equal(t, before, ids)
}

// Каждый дарит ровно одному, каждый получает ровно от одного, себя нет,
// и все стоят в одном круге — для любого размера комнаты.
func TestBuildCycle_AlwaysOneCircle(t *testing.T) {
	for n := 3; n <= 50; n++ {
		for run := 0; run < 200; run++ {
			ids := newIDs(n)
			pairs, err := buildCycle(uuid.New(), ids, cryptoShuffle)
			require.NoError(t, err)

			next := make(map[uuid.UUID]uuid.UUID, n)
			received := make(map[uuid.UUID]bool, n)
			for _, p := range pairs {
				require.NotEqual(t, p.GiverID, p.ReceiverID)
				_, dup := next[p.GiverID]
				require.False(t, dup, "дарит дважды")
				require.False(t, received[p.ReceiverID], "получает дважды")
				next[p.GiverID] = p.ReceiverID
				received[p.ReceiverID] = true
			}
			require.Len(t, next, n)

			cur := ids[0]
			for i := 1; i < n; i++ {
				cur = next[cur]
				require.NotEqual(t, ids[0], cur, "круг замкнулся раньше: пар больше одного круга")
			}
			require.Equal(t, ids[0], next[cur])
		}
	}
}

func TestCryptoShuffle_Varies(t *testing.T) {
	ids := newIDs(5)
	seen := map[uuid.UUID]bool{}
	for i := 0; i < 100; i++ {
		order := append([]uuid.UUID(nil), ids...)
		cryptoShuffle(order)
		seen[order[0]] = true
	}
	assert.Greater(t, len(seen), 1, "первый всегда один и тот же — перемешивания нет")
}
