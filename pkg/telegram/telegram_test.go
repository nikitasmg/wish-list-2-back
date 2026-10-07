package telegram

import (
	"context"
	"encoding/json"
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
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	c := NewWithBase("TOKEN", srv.URL, srv.Client())
	err := c.SendMessage(context.Background(), 42, "<b>Привет</b>", []Button{{Text: "Открыть", URL: "https://santa.prosto-namekni.ru/r/abcdefgh"}})
	require.NoError(t, err)
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
	require.NoError(t, NewWithBase("T", srv.URL, srv.Client()).SendMessage(context.Background(), 1, "x", nil))
	_, has := got["reply_markup"]
	assert.False(t, has)
}

func TestSendMessageAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`))
	}))
	defer srv.Close()
	err := NewWithBase("T", srv.URL, srv.Client()).SendMessage(context.Background(), 1, "x", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked")
}

func TestEscape(t *testing.T) {
	assert.Equal(t, "&lt;b&gt;Аня &amp; Ко&lt;/b&gt;", Escape("<b>Аня & Ко</b>"))
}

func TestSendMessageErrorDoesNotLeakToken(t *testing.T) {
	c := NewWithBase("SECRET-TOKEN", "http://127.0.0.1:1", &http.Client{})
	err := c.SendMessage(context.Background(), 1, "x", nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "SECRET-TOKEN")
}
