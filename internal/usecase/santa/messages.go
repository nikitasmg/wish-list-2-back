package santa

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/pkg/telegram"
)

// message — одно уведомление во всех видах: письмо (HTML + текст) и Telegram.
type message struct {
	Subject    string
	HTML       string
	Text       string
	Telegram   string // HTML-разметка Telegram, пользовательский текст экранирован
	ButtonText string
	URL        string
}

var monthsGen = [...]string{"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря"}

func roomLink(publicURL, slug string) string {
	return strings.TrimRight(publicURL, "/") + "/r/" + slug
}

// organizerLink — комната в кабинете организатора на поддомене.
func organizerLink(publicURL string, roomID uuid.UUID) string {
	return strings.TrimRight(publicURL, "/") + "/rooms/" + roomID.String()
}

// budgetText: «до 3 000 ₽» (неразрывный пробел между разрядами) или «без лимита».
func budgetText(b *int) string {
	if b == nil {
		return "без лимита"
	}
	s := strconv.Itoa(*b)
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, " "...)
		}
		out = append(out, s[i])
	}
	return "до " + string(out) + " ₽"
}

// dayText: «27 декабря»; дата хранится как полночь UTC.
func dayText(d *time.Time) string {
	if d == nil {
		return ""
	}
	u := d.UTC()
	return fmt.Sprintf("%d %s", u.Day(), monthsGen[u.Month()-1])
}

type emailView struct {
	Heading     string
	Lines       []string
	Quote       string
	WishlistURL string
	ButtonText  string
	URL         string
	Footer      string
}

var emailTpl = template.Must(template.New("email").Parse(`<!doctype html>
<html lang="ru"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"></head>
<body style="margin:0;padding:24px;background:#0b1220;font-family:Arial,Helvetica,sans-serif;color:#f4f1ea">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#131c31;border-radius:16px;padding:28px">
<tr><td>
<p style="margin:0 0 4px;font-size:13px;color:#e7b14c">★ Тайный Санта · просто намекни</p>
<h1 style="margin:0 0 16px;font-size:22px;line-height:1.3">{{.Heading}}</h1>
{{range .Lines}}<p style="margin:0 0 10px;font-size:15px;line-height:1.5">{{.}}</p>{{end}}
{{if .Quote}}<p style="margin:16px 0;padding:14px 16px;border-radius:12px;background:#1b2640;font-size:15px;line-height:1.5;white-space:pre-line">{{.Quote}}</p>{{end}}
{{if .WishlistURL}}<p style="margin:0 0 16px;font-size:15px"><a href="{{.WishlistURL}}" style="color:#e7b14c">Вишлист</a></p>{{end}}
{{if .URL}}<p style="margin:24px 0 8px"><a href="{{.URL}}" style="display:inline-block;padding:12px 22px;border-radius:12px;background:#c92a47;color:#ffffff;text-decoration:none;font-weight:bold">{{.ButtonText}}</a></p>{{end}}
{{if .Footer}}<p style="margin:16px 0 0;font-size:12px;color:#9aa3b8">{{.Footer}}</p>{{end}}
</td></tr></table></td></tr></table></body></html>`))

// compose собирает письмо, текст и Telegram из одних частей. Всё, что
// пришло от людей, в HTML экранирует html/template, в Telegram — telegram.Escape.
func compose(subject string, v emailView, tg string) message {
	var buf bytes.Buffer
	if err := emailTpl.Execute(&buf, v); err != nil {
		// Шаблон статический; ошибка — баг, но письмо всё равно уйдёт текстом.
		buf.Reset()
	}
	var text strings.Builder
	text.WriteString(v.Heading + "\n\n")
	for _, l := range v.Lines {
		text.WriteString(l + "\n")
	}
	if v.Quote != "" {
		text.WriteString("\n" + v.Quote + "\n")
	}
	if v.WishlistURL != "" {
		text.WriteString("\nВишлист: " + v.WishlistURL + "\n")
	}
	if v.URL != "" {
		text.WriteString("\n" + v.ButtonText + ": " + v.URL + "\n")
	}
	if v.Footer != "" {
		text.WriteString("\n" + v.Footer + "\n")
	}
	return message{Subject: subject, HTML: buf.String(), Text: text.String(), Telegram: tg, ButtonText: v.ButtonText, URL: v.URL}
}

func roomFacts(room entity.SantaRoom) []string {
	lines := []string{"Бюджет: " + budgetText(room.Budget) + "."}
	if day := dayText(room.ExchangeDate); day != "" {
		lines = append(lines, "Обмен подарками: "+day+".")
	}
	return lines
}

func tgFacts(room entity.SantaRoom) string {
	s := "Бюджет: " + budgetText(room.Budget)
	if day := dayText(room.ExchangeDate); day != "" {
		s += "\nОбмен подарками: " + day
	}
	return s
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://")
}

func emailCodeMessage(roomTitle, code string) message {
	return compose(
		"Код подтверждения: "+code,
		emailView{
			Heading: "Код подтверждения: " + code,
			Lines: []string{
				"Введите его на странице комнаты «" + roomTitle + "», чтобы получать сюда результат жеребьёвки.",
				"Код действует 15 минут.",
			},
			Footer: "Если вы не вступали в Тайного Санту, просто удалите это письмо.",
		},
		"",
	)
}

func welcomeMessage(room entity.SantaRoom, link string) message {
	return compose(
		"Вы в Тайном Санте «"+room.Title+"»",
		emailView{
			Heading:    "Готово — вы в комнате «" + room.Title + "»",
			Lines:      append([]string{"Когда организатор проведёт жеребьёвку, сюда придёт имя вашего подопечного."}, roomFacts(room)...),
			ButtonText: "Открыть комнату",
			URL:        link,
		},
		"🎄 Готово — вы в комнате <b>«"+telegram.Escape(room.Title)+"»</b>.\n\nКогда организатор проведёт жеребьёвку, сюда придёт имя вашего подопечного.\n\n"+tgFacts(room),
	)
}

func drawnMessage(room entity.SantaRoom, ward entity.SantaParticipant, link string) message {
	wishes := ward.Wishes
	wishlist := ""
	if isHTTPURL(ward.WishlistURL) {
		wishlist = ward.WishlistURL
	}
	lines := append([]string{"Вы — Тайный Санта для " + ward.Name + "."}, roomFacts(room)...)
	if wishes == "" && wishlist == "" {
		lines = append(lines, "Подопечный пока не написал пожеланий — мы сообщим, когда напишет.")
	}

	var tg strings.Builder
	tg.WriteString("🎁 Жеребьёвка в комнате <b>«" + telegram.Escape(room.Title) + "»</b> прошла.\n\n")
	tg.WriteString("Вы дарите подарок: <b>" + telegram.Escape(ward.Name) + "</b>\n\n")
	if wishes != "" {
		tg.WriteString("Пожелания:\n<blockquote>" + telegram.Escape(wishes) + "</blockquote>\n\n")
	} else {
		tg.WriteString("Пожеланий пока нет — сообщим, когда появятся.\n\n")
	}
	if wishlist != "" {
		tg.WriteString("Вишлист: " + telegram.Escape(wishlist) + "\n\n")
	}
	tg.WriteString(tgFacts(room))

	return compose(
		"Жеребьёвка прошла — «"+room.Title+"»",
		emailView{
			Heading:     "Вы дарите подарок: " + ward.Name,
			Lines:       lines,
			Quote:       wishes,
			WishlistURL: wishlist,
			ButtonText:  "Открыть конверт",
			URL:         link,
			Footer:      "Это тайна: никому не говорите, кому дарите.",
		},
		tg.String(),
	)
}

func reminderMessage(room entity.SantaRoom, link string) message {
	return compose(
		"Напишите пожелания — «"+room.Title+"»",
		emailView{
			Heading:    "Ваш Санта ждёт подсказку",
			Lines:      []string{"Организатор комнаты «" + room.Title + "» просит написать пожелания или приложить вишлист — так подарок точно попадёт в цель."},
			ButtonText: "Написать пожелания",
			URL:        link,
		},
		"✍️ Организатор комнаты <b>«"+telegram.Escape(room.Title)+"»</b> просит написать пожелания или приложить вишлист — так подарок точно попадёт в цель.",
	)
}

func wishesUpdatedMessage(room entity.SantaRoom, ward entity.SantaParticipant, link string) message {
	wishlist := ""
	if isHTTPURL(ward.WishlistURL) {
		wishlist = ward.WishlistURL
	}
	tg := "📝 " + telegram.Escape(ward.Name) + " обновил(а) пожелания в комнате <b>«" + telegram.Escape(room.Title) + "»</b>."
	if ward.Wishes != "" {
		tg += "\n\n<blockquote>" + telegram.Escape(ward.Wishes) + "</blockquote>"
	}
	if wishlist != "" {
		tg += "\n\nВишлист: " + telegram.Escape(wishlist)
	}
	return compose(
		ward.Name+" обновил(а) пожелания",
		emailView{
			Heading:     ward.Name + " обновил(а) пожелания",
			Lines:       []string{"Ваш подопечный в комнате «" + room.Title + "» поменял пожелания."},
			Quote:       ward.Wishes,
			WishlistURL: wishlist,
			ButtonText:  "Открыть конверт",
			URL:         link,
		},
		tg,
	)
}

func drawFailedMessage(room entity.SantaRoom, link string) message {
	return compose(
		"Жеребьёвка не прошла — «"+room.Title+"»",
		emailView{
			Heading: "Жеребьёвка по расписанию не прошла",
			Lines: []string{
				"В комнате «" + room.Title + "» меньше трёх участников с подтверждённой почтой или Telegram.",
				"Попросите остальных подключить канал, а потом проведите жеребьёвку вручную или назначьте новое время.",
			},
			ButtonText: "Открыть комнату",
			URL:        link,
		},
		"⚠️ Жеребьёвка по расписанию в комнате <b>«"+telegram.Escape(room.Title)+"»</b> не прошла: меньше трёх участников с подтверждённой почтой или Telegram.\n\nПопросите остальных подключить канал, а потом проведите жеребьёвку вручную или назначьте новое время.",
	)
}

func botHelloText() string {
	return "Привет! Я бот Тайного Санты от «просто намекни».\n\nЧтобы получать сюда результат жеребьёвки, откройте страницу своей комнаты и нажмите «Подключить Telegram»."
}

func botLinkExpiredText() string {
	return "Эта ссылка устарела или уже использована. Откройте страницу комнаты и нажмите «Подключить Telegram» ещё раз."
}

// chatFromSantaMessage — подопечному о сообщении Санты. Имени Санты здесь нет
// и быть не должно: оно тайна до обмена подарками.
func chatFromSantaMessage(room entity.SantaRoom, body, link string) message {
	return compose(
		"Тайный Санта написал вам — «"+room.Title+"»",
		emailView{
			Heading:    "Вам пишет ваш Тайный Санта",
			Lines:      []string{"Сообщение в комнате «" + room.Title + "». Кто он — останется тайной до обмена подарками."},
			Quote:      body,
			ButtonText: "Ответить",
			URL:        link,
			Footer:     "Ответить можно только на странице комнаты: ответ на это письмо Санта не получит.",
		},
		"💬 Вам пишет ваш <b>Тайный Санта</b> (комната «"+telegram.Escape(room.Title)+"»):\n\n<blockquote>"+telegram.Escape(body)+"</blockquote>\n\nЧтобы ответить, ответьте на это сообщение (Reply) — Санта получит ответ и не узнает ваш Telegram.",
	)
}

// chatFromWardMessage — Санте о сообщении подопечного.
func chatFromWardMessage(room entity.SantaRoom, wardName, body, link string) message {
	return compose(
		wardName+" написал(а) вам — «"+room.Title+"»",
		emailView{
			Heading:    "Вам пишет подопечный: " + wardName,
			Lines:      []string{"Сообщение в комнате «" + room.Title + "». Ваше имя подопечный не знает."},
			Quote:      body,
			ButtonText: "Ответить",
			URL:        link,
			Footer:     "Ответить можно только на странице комнаты: ответ на это письмо не дойдёт.",
		},
		"💬 Вам пишет подопечный <b>"+telegram.Escape(wardName)+"</b> (комната «"+telegram.Escape(room.Title)+"»):\n\n<blockquote>"+telegram.Escape(body)+"</blockquote>\n\nЧтобы ответить, ответьте на это сообщение (Reply) — подопечный не узнает, кто вы.",
	)
}

func botChatSentText() string { return "Отправлено ✓" }

func botChatUnknownText() string {
	return "Не понял, кому это. Чтобы написать в чат Тайного Санты, ответьте (Reply) на сообщение из чата или напишите на странице комнаты."
}

func botChatClosedText() string {
	return "Этот чат закрыт: организатор перезапустил жеребьёвку или удалил комнату."
}

func botChatLimitText() string {
	return "Не больше 30 сообщений в час — продолжите чуть позже."
}

func botChatTooLongText() string {
	return "Сообщение длиннее 1000 символов — сократите и отправьте ещё раз."
}
