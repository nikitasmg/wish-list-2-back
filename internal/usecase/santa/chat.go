package santa

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

const (
	maxChatBody = 1000
	chatPerHour = 30
	// chatHistory — сколько последних сообщений отдаём странице.
	chatHistory = 200
)

// chatPair — пара, в которой участник переписывается, и его сторона в ней:
// asGiver — он Санта (пишет подопечному).
func (uc *santaUseCase) chatPair(ctx context.Context, room entity.SantaRoom, p entity.SantaParticipant, with usecase.SantaChatWith) (entity.SantaAssignment, bool, error) {
	var (
		a       entity.SantaAssignment
		err     error
		asGiver bool
	)
	switch with {
	case usecase.SantaChatReceiver:
		asGiver = true
	case usecase.SantaChatSanta:
	default:
		return entity.SantaAssignment{}, false, invalid("with — receiver или santa")
	}
	if room.Status != entity.SantaRoomDrawn {
		return entity.SantaAssignment{}, false, usecase.ErrSantaNotDrawn
	}
	if asGiver {
		a, err = uc.santa.GetAssignment(ctx, room.ID, p.ID)
	} else {
		a, err = uc.santa.GetGiver(ctx, room.ID, p.ID)
	}
	if errors.Is(err, repo.ErrNotFound) {
		return entity.SantaAssignment{}, false, usecase.ErrSantaNotInDraw
	}
	if err != nil {
		return entity.SantaAssignment{}, false, err
	}
	return a, asGiver, nil
}

// chatView — сообщение без id сторон: подопечный не должен узнать Санту.
func chatView(m entity.SantaMessage, asGiver bool) usecase.SantaChatMessage {
	return usecase.SantaChatMessage{ID: m.ID, Mine: m.FromGiver == asGiver, Body: m.Body, CreatedAt: m.CreatedAt}
}

func (uc *santaUseCase) GetChat(ctx context.Context, slug string, auth usecase.SantaAuth, with usecase.SantaChatWith) (usecase.SantaChat, error) {
	room, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return usecase.SantaChat{}, err
	}
	pair, asGiver, err := uc.chatPair(ctx, room, p, with)
	if err != nil {
		return usecase.SantaChat{}, err
	}
	msgs, err := uc.santa.ListMessages(ctx, room.ID, pair.GiverID, pair.ReceiverID, chatHistory)
	if err != nil {
		return usecase.SantaChat{}, err
	}
	// Прочитаны входящие — написанные другой стороной.
	if err := uc.santa.MarkMessagesRead(ctx, room.ID, pair.GiverID, pair.ReceiverID, !asGiver, uc.now()); err != nil {
		return usecase.SantaChat{}, err
	}
	out := usecase.SantaChat{With: with, Messages: make([]usecase.SantaChatMessage, len(msgs))}
	for i, m := range msgs {
		out.Messages[i] = chatView(m, asGiver)
	}
	return out, nil
}

func (uc *santaUseCase) SendChat(ctx context.Context, slug string, auth usecase.SantaAuth, with usecase.SantaChatWith, body string) (usecase.SantaChatMessage, error) {
	room, p, err := uc.participantBySlug(ctx, slug, auth)
	if err != nil {
		return usecase.SantaChatMessage{}, err
	}
	pair, asGiver, err := uc.chatPair(ctx, room, p, with)
	if err != nil {
		return usecase.SantaChatMessage{}, err
	}
	msg, err := uc.postMessage(ctx, pair, asGiver, body)
	if err != nil {
		return usecase.SantaChatMessage{}, err
	}
	return chatView(msg, asGiver), nil
}

// postMessage — общий путь страницы и бота: проверка текста, лимит, запись
// сообщения и уведомления получателю.
func (uc *santaUseCase) postMessage(ctx context.Context, pair entity.SantaAssignment, fromGiver bool, body string) (entity.SantaMessage, error) {
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n == 0 || n > maxChatBody {
		return entity.SantaMessage{}, invalid("сообщение — от 1 до 1000 символов")
	}
	now := uc.now()
	msg := entity.SantaMessage{
		ID: uuid.New(), RoomID: pair.RoomID, GiverID: pair.GiverID, ReceiverID: pair.ReceiverID,
		FromGiver: fromGiver, Body: body, CreatedAt: now,
	}
	note := entity.NewSantaNotification(msg.RecipientID(), entity.SantaNotifyChatMessage, now)
	note.Payload[entity.SantaPayloadMessageID] = msg.ID.String()
	err := uc.santa.CreateMessage(ctx, msg, now.Add(-time.Hour), chatPerHour, note)
	switch {
	case err == nil:
		return msg, nil
	case errors.Is(err, repo.ErrTooSoon):
		return entity.SantaMessage{}, usecase.ErrSantaChatLimit
	case errors.Is(err, repo.ErrNotFound):
		// Пару сменил перезапуск жеребьёвки.
		return entity.SantaMessage{}, usecase.ErrSantaNotInDraw
	case errors.Is(err, repo.ErrStatusMismatch):
		return entity.SantaMessage{}, usecase.ErrSantaNotDrawn
	}
	return entity.SantaMessage{}, err
}

func (uc *santaUseCase) TelegramReply(ctx context.Context, chatID, replyToMessageID int64, text string) error {
	note, err := uc.santa.FindChatNotification(ctx, chatID, replyToMessageID)
	if errors.Is(err, repo.ErrNotFound) {
		return uc.tgReply(ctx, chatID, botChatUnknownText())
	}
	if err != nil {
		return err
	}
	origID, err := uuid.Parse(note.Payload[entity.SantaPayloadMessageID])
	if err != nil {
		return uc.tgReply(ctx, chatID, botChatUnknownText())
	}
	orig, err := uc.santa.GetMessage(ctx, origID)
	if errors.Is(err, repo.ErrNotFound) || (err == nil && orig.RecipientID() != note.ParticipantID) {
		// Перезапуск жеребьёвки стёр переписку, комнату удалили.
		return uc.tgReply(ctx, chatID, botChatClosedText())
	}
	if err != nil {
		return err
	}
	// Отвечает получатель исходного сообщения — другая сторона той же пары.
	pair := entity.SantaAssignment{RoomID: orig.RoomID, GiverID: orig.GiverID, ReceiverID: orig.ReceiverID}
	_, err = uc.postMessage(ctx, pair, !orig.FromGiver, text)
	switch {
	case err == nil:
		return uc.tgReply(ctx, chatID, botChatSentText())
	case errors.Is(err, usecase.ErrSantaNotInDraw), errors.Is(err, usecase.ErrSantaNotDrawn):
		return uc.tgReply(ctx, chatID, botChatClosedText())
	case errors.Is(err, usecase.ErrSantaChatLimit):
		return uc.tgReply(ctx, chatID, botChatLimitText())
	case errors.Is(err, usecase.ErrSantaInvalid):
		// Пустой текст вебхук не передаёт — остаётся только «слишком длинно».
		return uc.tgReply(ctx, chatID, botChatTooLongText())
	}
	return err
}
