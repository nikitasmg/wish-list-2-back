package santa

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

const (
	emailCodeTTL      = 15 * time.Minute
	emailCodeCooldown = time.Minute
	emailCodeAttempts = 5
	tgLinkTTL         = 24 * time.Hour
)

// unavailableError — канал не настроен на сервере: текст для человека,
// errors.Is — usecase.ErrSantaUnavailable (503).
type unavailableError struct{ msg string }

func (e unavailableError) Error() string { return e.msg }
func (e unavailableError) Unwrap() error { return usecase.ErrSantaUnavailable }

func unavailable(msg string) error { return unavailableError{msg: msg} }

// normalizeEmail: нижний регистр, только голый адрес без имени.
func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", invalid("укажите адрес почты")
	}
	if utf8.RuneCountInString(email) > 254 {
		return "", invalid("слишком длинный адрес почты")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@")+1:], ".") {
		return "", invalid("проверьте адрес почты")
	}
	return email, nil
}

func (uc *santaUseCase) participantBySlug(ctx context.Context, slug string, auth usecase.SantaAuth) (entity.SantaRoom, entity.SantaParticipant, error) {
	room, err := uc.roomBySlug(ctx, slug)
	if err != nil {
		return entity.SantaRoom{}, entity.SantaParticipant{}, err
	}
	p, err := uc.participant(ctx, room, auth)
	if err != nil {
		return entity.SantaRoom{}, entity.SantaParticipant{}, err
	}
	return room, p, nil
}

func (uc *santaUseCase) RequestEmailCode(ctx context.Context, slug string, auth usecase.SantaAuth, rawEmail string) error {
	if uc.mailer == nil {
		return unavailable("отправка почты пока не настроена — подключите Telegram или попробуйте позже")
	}
	room, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return err
	}
	email, err := normalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	now := uc.now()
	prev, err := uc.santa.GetEmailCode(ctx, p.ID)
	switch {
	case err == nil:
		if now.Sub(prev.SentAt) < emailCodeCooldown {
			return usecase.ErrSantaTooSoon
		}
	case !errors.Is(err, repo.ErrNotFound):
		return err
	}
	if !uc.emailQuota.take(email, now) {
		return usecase.ErrSantaEmailLimit
	}
	code, err := newEmailCode()
	if err != nil {
		return err
	}
	rec := entity.SantaEmailCode{
		ParticipantID: p.ID, CodeHash: hashEmailCode(p.ID, code),
		ExpiresAt: now.Add(emailCodeTTL), SentAt: now,
	}
	if err := uc.santa.SetEmail(ctx, p.ID, email, rec); err != nil {
		switch {
		case errors.Is(err, repo.ErrDuplicate):
			return usecase.ErrSantaEmailTaken
		case errors.Is(err, repo.ErrNotFound):
			return usecase.ErrSantaNotFound
		}
		return err
	}
	msg := emailCodeMessage(room.Title, code)
	if err := uc.mailer.Send(ctx, email, msg.Subject, msg.HTML, msg.Text); err != nil {
		// Письмо не ушло — код бесполезен; стираем, чтобы повтор не ждал минуту.
		if delErr := uc.santa.DeleteEmailCode(ctx, p.ID); delErr != nil {
			log.Printf("santa: drop email code: %v", delErr)
		}
		return fmt.Errorf("send email code: %w", err)
	}
	return nil
}

func (uc *santaUseCase) VerifyEmail(ctx context.Context, slug string, auth usecase.SantaAuth, rawCode string) (usecase.SantaMe, error) {
	room, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	rec, err := uc.santa.GetEmailCode(ctx, p.ID)
	if errors.Is(err, repo.ErrNotFound) {
		return usecase.SantaMe{}, invalid("кода нет — запросите новый")
	}
	if err != nil {
		return usecase.SantaMe{}, err
	}
	now := uc.now()
	if !rec.ExpiresAt.After(now) {
		return usecase.SantaMe{}, invalid("код устарел — запросите новый")
	}
	// Попытка занимается атомарно ДО сравнения: параллельные запросы не
	// получат больше emailCodeAttempts проверок.
	taken, err := uc.santa.IncEmailCodeAttempts(ctx, p.ID, emailCodeAttempts)
	if err != nil {
		return usecase.SantaMe{}, err
	}
	if !taken {
		return usecase.SantaMe{}, invalid("слишком много попыток — запросите новый код")
	}
	code := strings.TrimSpace(rawCode)
	if subtle.ConstantTimeCompare([]byte(hashEmailCode(p.ID, code)), []byte(rec.CodeHash)) != 1 {
		return usecase.SantaMe{}, invalid("неверный код")
	}
	welcome := entity.NewSantaNotification(p.ID, entity.SantaNotifyWelcome, now)
	if err := uc.santa.VerifyEmail(ctx, p.ID, rec.CodeHash, now, welcome); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// Код успели заменить или стереть — проверенный код уже не действует.
			return usecase.SantaMe{}, invalid("код устарел — запросите новый")
		}
		return usecase.SantaMe{}, err
	}
	p.EmailVerifiedAt = &now
	p.Channel = entity.SantaChannelEmail
	return uc.me(ctx, room, p)
}

func (uc *santaUseCase) TelegramLink(ctx context.Context, slug string, auth usecase.SantaAuth) (string, error) {
	if uc.botUsername == "" {
		return "", unavailable("подключение Telegram пока не настроено — укажите почту или попробуйте позже")
	}
	_, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return "", err
	}
	raw, hash, err := newToken()
	if err != nil {
		return "", err
	}
	link := entity.SantaTgLink{TokenHash: hash, ParticipantID: p.ID, ExpiresAt: uc.now().Add(tgLinkTTL)}
	if err := uc.santa.CreateTgLink(ctx, link); err != nil {
		return "", err
	}
	return fmt.Sprintf("https://t.me/%s?start=%s", uc.botUsername, raw), nil
}

func (uc *santaUseCase) TelegramStart(ctx context.Context, chatID int64, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return uc.tgReply(ctx, chatID, botHelloText())
	}
	now := uc.now()
	_, err := uc.santa.LinkTelegram(ctx, hashToken(token), chatID, now, func(p entity.SantaParticipant) entity.SantaNotification {
		return entity.NewSantaNotification(p.ID, entity.SantaNotifyWelcome, now)
	})
	if errors.Is(err, repo.ErrNotFound) {
		return uc.tgReply(ctx, chatID, botLinkExpiredText())
	}
	return err
}

func (uc *santaUseCase) tgReply(ctx context.Context, chatID int64, text string) error {
	if uc.tg == nil {
		return nil
	}
	// Сбой ответа не должен ронять вебхук: Telegram повторил бы апдейт.
	if _, err := uc.tg.SendMessage(ctx, chatID, text, nil); err != nil {
		log.Printf("santa: telegram reply: %v", err)
	}
	return nil
}
