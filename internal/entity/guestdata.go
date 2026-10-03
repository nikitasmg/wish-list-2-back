package entity

import (
	"time"

	"github.com/google/uuid"
)

// Данные, которые оставляют гости: ответы на приглашение, голоса, треки и
// поздравления.
//
// Это не «данные владельца внутри блока», а отдельные записи с авторством и
// дедупликацией, поэтому они живут в своих таблицах, а не в Block.Data.
// Привязка идёт к BlockID: на одной странице может быть два голосования, и
// позиция блока для этого не годится — она меняется при перестановке.

// RSVPResponse — ответ гостя на приглашение.
type RSVPResponse struct {
	ID         uuid.UUID `json:"id"`
	WishlistID uuid.UUID `json:"wishlistId"`
	BlockID    string    `json:"blockId"`
	// GuestID наружу не отдаётся: владельцу он ничего не говорит, а гостю
	// хватает флага Mine.
	GuestID uuid.UUID `json:"-"`

	Name     string `json:"name"`
	Going    bool   `json:"going"`
	PlusOne  int    `json:"plusOne"`
	Kids     int    `json:"kids"`
	Menu     string `json:"menu"`
	Transfer bool   `json:"transfer"`
	Comment  string `json:"comment"`
	// Answers — ответы на свои вопросы организатора, ключ — id вопроса.
	Answers map[string]string `json:"answers"`

	Mine      bool      `json:"mine"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RSVPSummary — сводка для организатора.
type RSVPSummary struct {
	Going       int            `json:"going"`
	NotGoing    int            `json:"notGoing"`
	PlusOnes    int            `json:"plusOnes"`
	Kids        int            `json:"kids"`
	Transfer    int            `json:"transfer"`
	TotalPeople int            `json:"totalPeople"`
	Responses   []RSVPResponse `json:"responses"`
}

// PollResults — итоги голосования. Список проголосовавших не отдаём: гостю
// обещана анонимность, а организатору нужны проценты, а не имена.
type PollResults struct {
	// Votes — голоса по id варианта. nil, когда результаты этому зрителю ещё
	// не показываются (Hidden).
	Votes   map[string]int `json:"votes"`
	Total   int            `json:"total"`
	MyVotes []string       `json:"myVotes"`
	// Hidden — результаты скрыты настройкой блока: «после голоса» или «только
	// организатору».
	Hidden bool `json:"hidden"`
	// Closed — голосование закрыто датой из настроек блока.
	Closed bool `json:"closed"`
	// GuestOptions — варианты, которые добавили гости (если владелец разрешил).
	GuestOptions []PollOption `json:"guestOptions"`
}

// PollOption — вариант, предложенный гостем.
type PollOption struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Mine   bool   `json:"mine"`
	Hidden bool   `json:"hidden"`
}

// PollGuestOption — вариант гостя для записи в базу.
type PollGuestOption struct {
	ID         uuid.UUID
	WishlistID uuid.UUID
	BlockID    string
	GuestID    uuid.UUID
	Text       string
}

// RSVPGuests — «Кто идёт»: только имена согласившихся и общее число людей.
// Ответы на вопросы и комментарии сюда не попадают — их видит организатор.
type RSVPGuests struct {
	Names []string `json:"names"`
	Total int      `json:"total"`
}

// PlaylistTrack — предложенный трек.
type PlaylistTrack struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Votes     int       `json:"votes"`
	VotedByMe bool      `json:"votedByMe"`
	Mine      bool      `json:"mine"`
	CreatedAt time.Time `json:"createdAt"`
}

// GuestbookEntry — запись в гостевой книге.
type GuestbookEntry struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Text     string    `json:"text"`
	PhotoURL string    `json:"photoUrl"`
	// Hidden видит только владелец: гостю скрытые записи не отдаются вовсе.
	Hidden    bool      `json:"hidden"`
	Mine      bool      `json:"mine"`
	CreatedAt time.Time `json:"createdAt"`
}

// Лимиты на пользовательский ввод. Без них гостевая книга и плейлист — это
// открытое текстовое поле в публичном интернете.
const (
	MaxGuestNameLen      = 60
	MaxRSVPCommentLen    = 300
	MaxRSVPMenuLen       = 120
	MaxTrackTitleLen     = 120
	MaxGuestbookTextLen  = 1000
	MaxPlusOne           = 10
	MaxKids              = 10
	MaxTracksPerGuest    = 5
	MaxGuestbookPerGuest = 5
	MaxRSVPAnswerLen     = 300
	MaxRSVPQuestions     = 10
	MaxPollOptionLen     = 80
	MaxPollOptionsGuest  = 3
)
