package santa

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"main/internal/usecase"
)

const (
	maxTitle   = 80
	maxMessage = 500
	maxBudget  = 1_000_000
	maxName    = 40
	maxWishes  = 1000
	maxURL     = 500
)

func invalid(msg string) error {
	return fmt.Errorf("%w: %s", usecase.ErrSantaInvalid, msg)
}

func normalizeRoom(in usecase.SantaRoomInput) usecase.SantaRoomInput {
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)
	in.OrganizerName = strings.TrimSpace(in.OrganizerName)
	in.OrganizerWishes = strings.TrimSpace(in.OrganizerWishes)
	return in
}

func validateRoom(in usecase.SantaRoomInput, now time.Time) error {
	if n := utf8.RuneCountInString(in.Title); n == 0 || n > maxTitle {
		return invalid("название — от 1 до 80 символов")
	}
	if utf8.RuneCountInString(in.Message) > maxMessage {
		return invalid("сообщение участникам — до 500 символов")
	}
	if in.Budget != nil && (*in.Budget < 0 || *in.Budget > maxBudget) {
		return invalid("бюджет — от 0 до 1 000 000 ₽")
	}
	if in.DrawAt != nil && !in.DrawAt.After(now) {
		return invalid("время жеребьёвки уже прошло")
	}
	return nil
}

func normalizeProfile(in usecase.SantaProfileInput) usecase.SantaProfileInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Wishes = strings.TrimSpace(in.Wishes)
	in.WishlistURL = strings.TrimSpace(in.WishlistURL)
	return in
}

func validateProfile(in usecase.SantaProfileInput) error {
	if n := utf8.RuneCountInString(in.Name); n == 0 || n > maxName {
		return invalid("имя — от 1 до 40 символов")
	}
	if utf8.RuneCountInString(in.Wishes) > maxWishes {
		return invalid("пожелания — до 1000 символов")
	}
	if in.WishlistURL != "" {
		// Ссылку увидит Санта и нажмёт её: javascript: и прочее сюда не пускаем.
		u, err := url.Parse(in.WishlistURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || utf8.RuneCountInString(in.WishlistURL) > maxURL {
			return invalid("ссылка на вишлист должна начинаться с http:// или https://")
		}
	}
	return nil
}
