package mailer

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildMessage(t *testing.T) {
	from, err := mail.ParseAddress("Тайный Санта <santa@prosto-namekni.ru>")
	require.NoError(t, err)
	raw, err := buildMessage(from, "anna@example.com", "Код: 123456", "<p>Привет</p>", "Привет", time.Date(2026, 12, 1, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	require.NoError(t, err)
	dec := new(mime.WordDecoder)
	subject, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	require.NoError(t, err)
	assert.Equal(t, "Код: 123456", subject)
	fromHdr, err := dec.DecodeHeader(msg.Header.Get("From"))
	require.NoError(t, err)
	assert.Contains(t, fromHdr, "santa@prosto-namekni.ru")
	assert.Equal(t, "anna@example.com", msg.Header.Get("To"))
	assert.NotEmpty(t, msg.Header.Get("Message-Id"))

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	require.NoError(t, err)
	assert.Equal(t, "multipart/alternative", mediaType)

	mr := multipart.NewReader(msg.Body, params["boundary"])
	bodies := map[string]string{}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		ct, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		b, err := io.ReadAll(part) // multipart.Reader снимает quoted-printable сам
		require.NoError(t, err)
		bodies[ct] = string(b)
	}
	assert.Equal(t, "Привет", bodies["text/plain"])
	assert.Equal(t, "<p>Привет</p>", bodies["text/html"])
}

func TestNewSMTPRejectsBadFrom(t *testing.T) {
	_, err := NewSMTP(Config{Host: "smtp.example.com", Port: 587, From: "не адрес"})
	assert.Error(t, err)
}
