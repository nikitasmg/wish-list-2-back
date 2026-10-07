// Package telegram — минимальный клиент Bot API: отправка сообщений.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"time"
)

// Button — кнопка-ссылка под сообщением.
type Button struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type Client struct {
	token string
	base  string
	http  *http.Client
}

func New(token string) *Client {
	return NewWithBase(token, "https://api.telegram.org", &http.Client{Timeout: 10 * time.Second})
}

func NewWithBase(token, base string, hc *http.Client) *Client {
	return &Client{token: token, base: base, http: hc}
}

// SendMessage шлёт текст с parse_mode=HTML. Пользовательский текст —
// только через Escape.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, buttons []Button) error {
	body := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	var rows [][]Button
	for _, b := range buttons {
		if b.URL != "" && b.Text != "" {
			rows = append(rows, []Button{b})
		}
	}
	if len(rows) > 0 {
		body["reply_markup"] = map[string]any{"inline_keyboard": rows}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return errors.New("telegram: не удалось собрать запрос")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // url.Error содержит URL с токеном бота
		}
		return fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("telegram: status %d: %w", resp.StatusCode, err)
	}
	if !out.OK {
		return fmt.Errorf("telegram: %s (status %d)", out.Description, resp.StatusCode)
	}
	return nil
}

// Escape — экранирование пользовательского текста для parse_mode=HTML.
func Escape(s string) string { return html.EscapeString(s) }

// Log — бот для разработки: сообщения уходят в лог.
type Log struct{}

func NewLog() *Log { return &Log{} }

func (*Log) SendMessage(_ context.Context, chatID int64, text string, buttons []Button) error {
	log.Printf("telegram(log): chat=%d buttons=%v\n%s", chatID, buttons, text)
	return nil
}
