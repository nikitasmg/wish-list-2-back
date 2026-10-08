package santa

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/pkg/telegram"
)

const (
	notifyBatch       = 20
	notifyLease       = 2 * time.Minute
	maxNotifyAttempts = 4
	// Пачка не должна пережить аренду, иначе второй экземпляр возьмёт те же
	// строки и отправит дважды. Новую отправку начинаем, пока прошло не больше
	// половины аренды; каждая ограничена notifySendTimeout, отметка —
	// notifyMarkTimeout: lease/2 + send + mark < lease.
	notifySendTimeout = 45 * time.Second
	notifyMarkTimeout = 5 * time.Second
)

// retryDelays[i] — пауза после (i+1)-й неудачи; после maxNotifyAttempts — failed.
var retryDelays = []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute}

// errPermanent — повторять бессмысленно: участника нет, канал не подтверждён, пары нет.
var errPermanent = errors.New("permanent")

// Notifier разбирает outbox santa_notifications. Текст собирается в момент
// отправки из текущей базы: после перезапуска жеребьёвки уйдёт новая пара.
type Notifier struct {
	santa     repo.SantaRepo
	mailer    usecase.Mailer
	tg        usecase.TelegramSender
	publicURL string
	now       func() time.Time
}

func NewNotifier(santaRepo repo.SantaRepo, mailer usecase.Mailer, tg usecase.TelegramSender, publicURL string) *Notifier {
	return &Notifier{santa: santaRepo, mailer: mailer, tg: tg, publicURL: publicURL, now: time.Now}
}

// Run крутит RunOnce каждые tick до отмены ctx.
func (n *Notifier) Run(ctx context.Context, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		if _, err := n.RunOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("santa notifier: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// RunOnce берёт пачку под аренду и отправляет, пока не прошла половина
// аренды или не отменён ctx; не начатые строки вернутся в очередь сами,
// когда аренда истечёт.
func (n *Notifier) RunOnce(ctx context.Context) (int, error) {
	start := n.now()
	batch, err := n.santa.ClaimNotifications(ctx, start, notifyBatch, notifyLease)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, note := range batch {
		if ctx.Err() != nil || n.now().Sub(start) > notifyLease/2 {
			break
		}
		if n.process(ctx, note) {
			sent++
		}
	}
	return sent, nil
}

// process отправляет одно уведомление и отмечает итог; true — отправлено.
func (n *Notifier) process(ctx context.Context, note entity.SantaNotification) bool {
	sendCtx, cancelSend := context.WithTimeout(ctx, notifySendTimeout)
	err := n.deliver(sendCtx, note)
	cancelSend()

	// Отметку пишем и после отмены ctx при остановке: принятое сервером
	// письмо, оставшись pending, ушло бы повторно.
	markCtx, cancelMark := context.WithTimeout(context.WithoutCancel(ctx), notifyMarkTimeout)
	defer cancelMark()
	if err == nil {
		n.logMark("sent", note, n.santa.MarkNotificationSent(markCtx, note.ID))
		return true
	}
	attempts := note.Attempts + 1
	var retryAt *time.Time
	if !errors.Is(err, errPermanent) && attempts < maxNotifyAttempts {
		at := n.now().Add(retryDelays[attempts-1])
		retryAt = &at
	}
	n.logMark("failed", note, n.santa.MarkNotificationFailed(markCtx, note.ID, attempts, retryAt, err.Error()))
	return false
}

func (n *Notifier) logMark(what string, note entity.SantaNotification, err error) {
	switch {
	case err == nil:
	case errors.Is(err, repo.ErrNotFound):
		// Уже отмечено другим обработчиком или стёрто при схлопывании.
		log.Printf("santa notifier: mark %s %s: уже не pending", what, note.ID)
	default:
		log.Printf("santa notifier: mark %s %s: %v", what, note.ID, err)
	}
}

func (n *Notifier) deliver(ctx context.Context, note entity.SantaNotification) error {
	p, err := n.santa.GetParticipant(ctx, note.ParticipantID)
	if errors.Is(err, repo.ErrNotFound) {
		return fmt.Errorf("%w: участника нет", errPermanent)
	}
	if err != nil {
		return err
	}
	if !p.Ready() {
		return fmt.Errorf("%w: канал не подтверждён", errPermanent)
	}
	room, err := n.santa.GetRoomByID(ctx, p.RoomID)
	if errors.Is(err, repo.ErrNotFound) {
		return fmt.Errorf("%w: комнаты нет", errPermanent)
	}
	if err != nil {
		return err
	}
	msg, err := n.compose(ctx, note.Kind, room, p)
	if err != nil {
		return err
	}
	switch p.Channel {
	case entity.SantaChannelEmail:
		if n.mailer == nil {
			return fmt.Errorf("%w: почта не настроена", errPermanent)
		}
		return n.mailer.Send(ctx, p.Email, msg.Subject, msg.HTML, msg.Text)
	case entity.SantaChannelTelegram:
		if n.tg == nil {
			return fmt.Errorf("%w: бот не настроен", errPermanent)
		}
		_, err := n.tg.SendMessage(ctx, *p.TgChatID, msg.Telegram, []telegram.Button{{Text: msg.ButtonText, URL: msg.URL}})
		if errors.Is(err, telegram.ErrPermanent) {
			// Бот заблокирован или чата нет — повтор через минуту ничего не изменит.
			return fmt.Errorf("%w: %v", errPermanent, err)
		}
		return err
	}
	return fmt.Errorf("%w: нет канала", errPermanent)
}

func (n *Notifier) compose(ctx context.Context, kind entity.SantaNotificationKind, room entity.SantaRoom, p entity.SantaParticipant) (message, error) {
	link := roomLink(n.publicURL, room.Slug)
	switch kind {
	case entity.SantaNotifyWelcome:
		return welcomeMessage(room, link), nil
	case entity.SantaNotifyReminderFill:
		return reminderMessage(room, link), nil
	case entity.SantaNotifyDrawn, entity.SantaNotifyWishesUpdated:
		a, err := n.santa.GetAssignment(ctx, room.ID, p.ID)
		if errors.Is(err, repo.ErrNotFound) {
			return message{}, fmt.Errorf("%w: пары нет", errPermanent)
		}
		if err != nil {
			return message{}, err
		}
		ward, err := n.santa.GetParticipant(ctx, a.ReceiverID)
		if errors.Is(err, repo.ErrNotFound) {
			return message{}, fmt.Errorf("%w: подопечного нет", errPermanent)
		}
		if err != nil {
			return message{}, err
		}
		if kind == entity.SantaNotifyDrawn {
			return drawnMessage(room, ward, link), nil
		}
		return wishesUpdatedMessage(room, ward, link), nil
	}
	return message{}, fmt.Errorf("%w: неизвестный вид %q", errPermanent, kind)
}
