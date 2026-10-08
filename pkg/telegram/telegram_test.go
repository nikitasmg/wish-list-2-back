package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/botTOKEN/sendMessage", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		b, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(b, &got))
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":321}}`))
	}))
	defer srv.Close()

	c := NewWithBase("TOKEN", srv.URL, srv.Client())
	id, err := c.SendMessage(context.Background(), 42, "<b>Привет</b>", []Button{{Text: "Открыть", URL: "https://santa.prosto-namekni.ru/r/abcdefgh"}})
	require.NoError(t, err)
	assert.EqualValues(t, 321, id)
	assert.EqualValues(t, 42, got["chat_id"])
	assert.Equal(t, "HTML", got["parse_mode"])
	assert.Equal(t, true, got["disable_web_page_preview"])
	kb := got["reply_markup"].(map[string]any)["inline_keyboard"].([]any)
	btn := kb[0].([]any)[0].(map[string]any)
	assert.Equal(t, "Открыть", btn["text"])
}

func TestSendMessageNoButtons(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(b, &got))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	_, err := NewWithBase("T", srv.URL, srv.Client()).SendMessage(context.Background(), 1, "x", nil)
	require.NoError(t, err)
	_, has := got["reply_markup"]
	assert.False(t, has)
}

func TestSendMessageAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
	}))
	defer srv.Close()
	_, err := NewWithBase("T", srv.URL, srv.Client()).SendMessage(context.Background(), 1, "x", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
	assert.ErrorIs(t, err, ErrPermanent)
}

func TestEscape(t *testing.T) {
	assert.Equal(t, "&lt;b&gt;Аня &amp; Ко&lt;/b&gt;", Escape("<b>Аня & Ко</b>"))
}

func TestSendMessageErrorDoesNotLeakToken(t *testing.T) {
	c := NewWithBase("SECRET-TOKEN", "http://127.0.0.1:1", &http.Client{})
	_, err := c.SendMessage(context.Background(), 1, "x", nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-TOKEN")
}

func TestSendMessagePermanentErrors(t *testing.T) {
	cases := []struct {
		status    int
		body      string
		permanent bool
	}{
		{http.StatusForbidden, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`, true},
		{http.StatusForbidden, `{"ok":false,"error_code":403,"description":"Forbidden: user is deactivated"}`, true},
		{http.StatusBadRequest, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`, true},
		{http.StatusBadRequest, `{"ok":false,"error_code":400,"description":"Bad Request: message is too long"}`, false},
		{http.StatusTooManyRequests, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 5"}`, false},
		{http.StatusInternalServerError, `{"ok":false,"error_code":500,"description":"Internal Server Error"}`, false},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		_, err := NewWithBase("T", srv.URL, srv.Client()).SendMessage(context.Background(), 1, "x", nil)
		srv.Close()
		require.Error(t, err, tc.body)
		assert.Equal(t, tc.permanent, errors.Is(err, ErrPermanent), tc.body)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, tc.body)
		assert.Equal(t, tc.status, apiErr.Code)
	}
}
