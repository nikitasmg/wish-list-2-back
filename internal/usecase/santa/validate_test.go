package santa

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"main/internal/usecase"
)

func ptr(v int) *int { return &v }

func TestValidateRoom_Boundaries(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		in   usecase.SantaRoomInput
		ok   bool
	}{
		{"budget 0", usecase.SantaRoomInput{Title: "a", Budget: ptr(0)}, true},
		{"budget max", usecase.SantaRoomInput{Title: "a", Budget: ptr(1_000_000)}, true},
		{"title 80", usecase.SantaRoomInput{Title: strings.Repeat("я", 80)}, true},
		{"title 81", usecase.SantaRoomInput{Title: strings.Repeat("я", 81)}, false},
		{"message 500", usecase.SantaRoomInput{Title: "a", Message: strings.Repeat("я", 500)}, true},
		{"message 501", usecase.SantaRoomInput{Title: "a", Message: strings.Repeat("я", 501)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRoom(c.in, now)
			if c.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, usecase.ErrSantaInvalid)
			}
		})
	}
}

func TestValidateProfile_WishlistURL(t *testing.T) {
	for _, bad := range []string{"javascript:alert(1)", "ftp://x"} {
		err := validateProfile(usecase.SantaProfileInput{Name: "Маша", WishlistURL: bad})
		assert.ErrorIs(t, err, usecase.ErrSantaInvalid, bad)
	}
	assert.NoError(t, validateProfile(usecase.SantaProfileInput{Name: "Маша", WishlistURL: "https://x.ru/w"}))
}
