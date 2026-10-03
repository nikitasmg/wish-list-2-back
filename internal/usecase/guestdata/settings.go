package guestdata

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
)

// Настройки гостевых блоков живут в data блока, а значит в руках владельца и
// фронта. Здесь они читаются заново на каждом запросе: поменял организатор
// дедлайн или закрыл голосование — следующий гость упрётся уже в новое правило.

type rsvpQuestion struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"` // text | number | bool
}

type rsvpSettings struct {
	Questions  []rsvpQuestion `json:"questions"`
	Deadline   flexTime       `json:"deadline"`
	ShowGuests bool           `json:"showGuests"`
}

type pollSettings struct {
	Options      []string // id вариантов владельца
	Multiple     bool
	Results      string // all | after_vote | owner
	ClosesAt     *time.Time
	GuestOptions bool
}

// flexTime — дата из формы: RFC3339 или просто день. День означает «до конца
// этого дня», иначе «ответить до 10 октября» закрывалось бы в полночь девятого.
type flexTime struct{ t *time.Time }

func (f *flexTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil || s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		f.t = &t
		return nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		end := t.Add(24 * time.Hour)
		f.t = &end
	}
	return nil
}

// findBlock — блок нужного типа на этом вишлисте. Без проверки типа ответ
// гостя можно было бы записать в блок голосования, и наоборот.
func (uc *guestDataUseCase) findBlock(ctx context.Context, wishlistID uuid.UUID, blockID, blockType string) (entity.Wishlist, entity.Block, error) {
	w, err := uc.wishlistRepo.GetByID(ctx, wishlistID)
	if err != nil {
		return entity.Wishlist{}, entity.Block{}, fmt.Errorf("wishlist not found: %w", err)
	}
	for _, b := range w.Blocks {
		if b.ID == blockID {
			if b.Type != blockType {
				return entity.Wishlist{}, entity.Block{}, fmt.Errorf("блок %s — не %s", blockID, blockType)
			}
			return w, b, nil
		}
	}
	return entity.Wishlist{}, entity.Block{}, fmt.Errorf("блок %s не найден", blockID)
}

func parseRSVPSettings(data json.RawMessage) rsvpSettings {
	var s rsvpSettings
	_ = json.Unmarshal(data, &s) // кривые настройки = настройки по умолчанию
	return s
}

func parsePollSettings(data json.RawMessage) pollSettings {
	var raw struct {
		Options      []json.RawMessage `json:"options"`
		Multiple     bool              `json:"multiple"`
		Results      string            `json:"results"`
		ClosesAt     flexTime          `json:"closesAt"`
		GuestOptions bool              `json:"guestOptions"`
	}
	_ = json.Unmarshal(data, &raw)

	s := pollSettings{Multiple: raw.Multiple, Results: raw.Results, ClosesAt: raw.ClosesAt.t, GuestOptions: raw.GuestOptions}
	for i, option := range raw.Options {
		// Старые голосования хранили варианты строками, и голоса ссылались на
		// индекс. Индекс и становится id — так старые голоса не теряются.
		var text string
		if json.Unmarshal(option, &text) == nil {
			s.Options = append(s.Options, strconv.Itoa(i))
			continue
		}
		var obj struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(option, &obj) == nil && obj.ID != "" {
			s.Options = append(s.Options, obj.ID)
		}
	}
	return s
}

func (s pollSettings) closed(now time.Time) bool {
	return s.ClosesAt != nil && !now.Before(*s.ClosesAt)
}

// validateAnswers — ответы только на заданные вопросы и в их формате.
func validateAnswers(questions []rsvpQuestion, answers map[string]string) (map[string]string, error) {
	if len(answers) == 0 {
		return nil, nil
	}
	kinds := make(map[string]string, len(questions))
	for _, q := range questions {
		kinds[q.ID] = q.Kind
	}

	cleaned := make(map[string]string, len(answers))
	for id, value := range answers {
		kind, ok := kinds[id]
		if !ok {
			return nil, fmt.Errorf("вопроса %q нет в приглашении", id)
		}
		value = trimSpace(value)
		if err := checkLen("ответ", value, entity.MaxRSVPAnswerLen); err != nil {
			return nil, err
		}
		switch kind {
		case "number":
			if value == "" {
				break
			}
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n > 99 {
				return nil, fmt.Errorf("в ответе на %q нужно число", id)
			}
		case "bool":
			if value != "" && value != "true" && value != "false" {
				return nil, fmt.Errorf("в ответе на %q нужно да или нет", id)
			}
		}
		if value != "" {
			cleaned[id] = value
		}
	}
	return cleaned, nil
}
