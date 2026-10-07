package santa

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

const slugAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// newSlug — 8 символов адреса комнаты: 62^8 вариантов, перебором не найти.
func newSlug() (string, error) {
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(slugAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = slugAlphabet[n.Int64()]
	}
	return string(b), nil
}

// newToken — секрет личной ссылки участника и его хэш для базы.
func newToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw), nil
}

// hashToken — sha256 без соли: токен сам по себе случайный, словарь к нему
// не подобрать, а поиск по хэшу должен быть точным.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newEmailCode — 6 цифр из crypto/rand.
func newEmailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashEmailCode привязывает код к участнику: одинаковые коды разных людей
// дают разные хэши.
func hashEmailCode(participantID uuid.UUID, code string) string {
	sum := sha256.Sum256([]byte(participantID.String() + ":" + code))
	return hex.EncodeToString(sum[:])
}
