package shortid

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// Generate returns a short ID in the format "abc-def-ghi"
func Generate() (string, error) {
	result := make([]byte, 11)
	result[3] = '-'
	result[7] = '-'

	positions := []int{0, 1, 2, 4, 5, 6, 8, 9, 10}
	for _, i := range positions {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		result[i] = alphabet[n.Int64()]
	}

	return string(result), nil
}
