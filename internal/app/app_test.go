package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"main/pkg/mailer"
	"main/pkg/telegram"
)

func TestNotifierChannels(t *testing.T) {
	logMail, logBot := mailer.NewLog(), telegram.NewLog()

	m, b := notifierChannels(true, false, false, logMail, logBot)
	assert.Nil(t, m, "прод без SMTP — письма не «уходят» в лог")
	assert.Nil(t, b, "прод без токена — сообщения не «уходят» в лог")

	m, b = notifierChannels(true, true, true, logMail, logBot)
	assert.NotNil(t, m)
	assert.NotNil(t, b)

	m, b = notifierChannels(false, false, false, logMail, logBot)
	assert.NotNil(t, m, "в разработке лог — норма")
	assert.NotNil(t, b)
}
