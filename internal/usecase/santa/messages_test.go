package santa

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"main/internal/entity"
)

func intp(v int) *int { return &v }

func TestBudgetAndDay(t *testing.T) {
	assert.Equal(t, "до 3\u00a0000 ₽", budgetText(intp(3000)))
	assert.Equal(t, "до 500 ₽", budgetText(intp(500)))
	assert.Equal(t, "без лимита", budgetText(nil))
	d := time.Date(2026, 12, 27, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, "27 декабря", dayText(&d))
	assert.Equal(t, "", dayText(nil))
}

func TestRoomLink(t *testing.T) {
	assert.Equal(t, "https://santa.prosto-namekni.ru/r/abcdefgh", roomLink("https://santa.prosto-namekni.ru/", "abcdefgh"))
}

func TestDrawnMessageEscapesUserText(t *testing.T) {
	room := entity.SantaRoom{Title: "Офис <script>", Slug: "abcdefgh", Budget: intp(2000)}
	ward := entity.SantaParticipant{Name: "Аня & Ко", Wishes: "<b>книги</b>\nчай", WishlistURL: "javascript:alert(1)"}
	m := drawnMessage(room, ward, "https://santa.prosto-namekni.ru/r/abcdefgh")

	assert.Contains(t, m.Subject, "Офис <script>", "тема — простой текст, её кодирует mime")
	assert.NotContains(t, m.HTML, "<script>")
	assert.NotContains(t, m.HTML, "<b>книги</b>")
	assert.Contains(t, m.HTML, "Аня &amp; Ко")
	assert.NotContains(t, m.HTML, "javascript:alert", "html/template вырезает опасную ссылку")
	assert.NotContains(t, m.Telegram, "<b>книги</b>")
	assert.Contains(t, m.Telegram, "&lt;b&gt;книги&lt;/b&gt;")
	assert.Contains(t, m.Telegram, "Аня &amp; Ко")
	assert.Contains(t, m.Text, "Аня & Ко")
	assert.Contains(t, m.Text, "до 2\u00a0000 ₽")
	assert.Equal(t, "https://santa.prosto-namekni.ru/r/abcdefgh", m.URL)
	assert.NotEmpty(t, m.ButtonText)
}

func TestDrawnMessageWithoutWishes(t *testing.T) {
	m := drawnMessage(entity.SantaRoom{Title: "Офис"}, entity.SantaParticipant{Name: "Боря"}, "https://x/r/a")
	assert.Contains(t, m.Text, "пока не написал пожеланий")
	assert.Contains(t, m.Telegram, "Боря")
}

func TestEmailCodeMessage(t *testing.T) {
	m := emailCodeMessage("Офис", "042137")
	assert.Contains(t, m.Subject, "042137")
	assert.Contains(t, m.Text, "042137")
	assert.Contains(t, m.HTML, "042137")
	assert.Contains(t, m.Text, "15 минут")
}

func TestAllMessagesHaveBothBodies(t *testing.T) {
	room := entity.SantaRoom{Title: "Офис", Slug: "abcdefgh"}
	ward := entity.SantaParticipant{Name: "Боря", Wishes: "носки"}
	for name, m := range map[string]message{
		"welcome":  welcomeMessage(room, "https://x/r/abcdefgh"),
		"reminder": reminderMessage(room, "https://x/r/abcdefgh"),
		"updated":  wishesUpdatedMessage(room, ward, "https://x/r/abcdefgh"),
	} {
		assert.NotEmpty(t, m.Subject, name)
		assert.True(t, strings.HasPrefix(m.HTML, "<!doctype html>"), name)
		assert.NotEmpty(t, m.Text, name)
		assert.NotEmpty(t, m.Telegram, name)
		assert.Equal(t, "https://x/r/abcdefgh", m.URL, name)
	}
	assert.NotEmpty(t, botHelloText())
	assert.NotEmpty(t, botLinkExpiredText())
}

func TestDrawFailedMessage(t *testing.T) {
	room := entity.SantaRoom{ID: uuid.New(), Title: "<Офис>"}
	link := organizerLink("https://santa.prosto-namekni.ru/", room.ID)
	assert.Equal(t, "https://santa.prosto-namekni.ru/rooms/"+room.ID.String(), link)

	msg := drawFailedMessage(room, link)
	assert.Contains(t, msg.Subject, "не прошла")
	assert.Contains(t, msg.Text, link)
	assert.Contains(t, msg.Telegram, "&lt;Офис&gt;")
	assert.NotContains(t, msg.HTML, "<Офис>")
}
