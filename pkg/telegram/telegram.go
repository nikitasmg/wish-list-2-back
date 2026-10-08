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
	"strings"
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

// ErrPermanent — повтор не поможет: бот заблокирован или пользователь удалён
// (403), чата нет (400 «chat not found»). Проверка — errors.Is(err, ErrPermanent).
var ErrPermanent = errors.New("telegram: постоянная ошибка")

// APIError — отказ Bot API (ok=false в ответе).
type APIError struct {
	Code        int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: %s (status %d)", e.Description, e.Code)
}

// Unwrap даёт ErrPermanent для ошибок, которые повтор не исправит. Прочие 400
// (длина текста, разметка) остаются временными: текст собирается заново при
// каждой попытке и может стать другим.
func (e *APIError) Unwrap() error {
	switch {
	case e.Code == http.StatusForbidden,
		e.Code == http.StatusBadRequest && strings.Contains(strings.ToLower(e.Description), "chat not found"):
		return ErrPermanent
	}
	return nil
}

// SendMessage шлёт текст с parse_mode=HTML и возвращает message_id
// отправленного сообщения: по нему бот узнаёт ответ (reply_to_message).
// Пользовательский текст — только через Escape.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, buttons []Button) (int64, error) {
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
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return 0, errors.New("telegram: не удалось собрать запрос")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err // url.Error содержит URL с токеном бота
		}
		return 0, fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Result      struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("telegram: status %d: %w", resp.StatusCode, err)
	}
	if !out.OK {
		code := out.ErrorCode
		if code == 0 {
			code = resp.StatusCode
		}
		return 0, &APIError{Code: code, Description: out.Description}
	}
	return out.Result.MessageID, nil
}

// Escape — экранирование пользовательского текста для parse_mode=HTML.
func Escape(s string) string { return html.EscapeString(s) }

// Log — бот для разработки: сообщения уходят в лог.
type Log struct{}

func NewLog() *Log { return &Log{} }

func (*Log) SendMessage(_ context.Context, chatID int64, text string, buttons []Button) (int64, error) {
	log.Printf("telegram(log): chat=%d buttons=%v\n%s", chatID, buttons, text)
	return 0, nil
}
