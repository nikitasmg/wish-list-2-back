package entity

import (
	"time"

	"github.com/google/uuid"
)

type Present struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Reserved    bool      `json:"reserved"`
	Cover       string    `json:"cover"`
	// Links — магазины, где подарок можно купить. Название магазина отдельным
	// полем не храним: это хост ссылки, и оно разъехалось бы с URL при первой
	// же правке — фронт берёт подпись из самой ссылки.
	Links []string `json:"links"`
	// Link — первая из Links. Остаётся в ответе, пока фронт не переедет на
	// массив; при записи её значение попадает в Links[0].
	Link       string    `json:"link"`
	Price      *float64  `json:"price"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	WishlistID uuid.UUID `json:"wishlistId"`

	// ReservedByGuest — кука гостя, забронировавшего подарок. Наружу не отдаётся
	// никогда: владелец видит только факт брони, «Маша не узнает, кто что дарит».
	ReservedByGuest string `json:"-"`
	// ReservedByMe — вычисляется на выдаче для текущего гостя, в базе не хранится.
	ReservedByMe bool `json:"reservedByMe"`
}

// MaxPresentDescriptionLen — лимит описания подарка в символах (не в байтах:
// описания пишут по-русски, и 500 байт это всего 250 букв).
const MaxPresentDescriptionLen = 500

// MaxPresentLinks — сколько магазинов помещается в карточку подарка.
const MaxPresentLinks = 5
