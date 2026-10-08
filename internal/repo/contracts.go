package repo

import (
	"context"
	"errors"
	"time"

	"main/internal/entity"

	"github.com/google/uuid"
)

type UserRepo interface {
	Create(ctx context.Context, user entity.User) error
	GetByUsername(ctx context.Context, username string) (entity.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (entity.User, error)
	Update(ctx context.Context, user entity.User) error
}

type WishlistRepo interface {
	Create(ctx context.Context, wishlist entity.Wishlist) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error)
	GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error)
	// UpdateMetadata пишет только настройки и возвращает сохранённую версию.
	// Полного Save модели здесь нет намеренно: он затирал бы блоки снимком,
	// прочитанным до правки.
	UpdateMetadata(ctx context.Context, id uuid.UUID, wishlist entity.Wishlist, expectedUpdatedAt time.Time) (entity.Wishlist, bool, error)
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementPresentsCount(ctx context.Context, id uuid.UUID) error
	DecrementPresentsCount(ctx context.Context, id uuid.UUID) error
	// ReservedCountsByUser — занятые подарки по каждому вишлисту пользователя,
	// одним запросом для карточек кабинета.
	ReservedCountsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uint, error)
	// RegisterView засчитывает просмотр, если этот гость его ещё не делал.
	RegisterView(ctx context.Context, wishlistID, guestID uuid.UUID) error
	// UpdateBlocks пишет блоки с проверкой версии. false — версия разошлась,
	// вишлист успели изменить в другом месте.
	// Ряды пишутся тем же запросом: их настройки адресуются индексом ряда и
	// без блоков смысла не имеют.
	UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block, rows []entity.RowSettings, blocksVersion int, expectedUpdatedAt time.Time) (bool, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type PresentRepo interface {
	Create(ctx context.Context, present entity.Present) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error)
	GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error)
	Update(ctx context.Context, present entity.Present) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error)
	// Reserve и Release — условные апдейты. Возвращают false, когда строка под
	// условие не подошла: подарок уже занят или бронь ставил другой гость.
	Reserve(ctx context.Context, id, guestID uuid.UUID, name string) (bool, error)
	Release(ctx context.Context, id, guestID uuid.UUID) (bool, error)
	// ClearMain снимает «главную мечту» со всех подарков вишлиста, кроме exceptID.
	ClearMain(ctx context.Context, wishlistID, exceptID uuid.UUID) error
	// Reorder проставляет sort_order по порядку ids в пределах вишлиста.
	Reorder(ctx context.Context, wishlistID uuid.UUID, ids []uuid.UUID) error
	SetGifted(ctx context.Context, id uuid.UUID, gifted bool) error
}

// GuestDataRepo — всё, что оставляют гости: ответы, голоса, треки, записи.
//
// Один интерфейс на четыре блока, а не четыре отдельных: у них общая механика
// (привязка к блоку, дедупликация по гостю) и общий вызывающий.
type GuestDataRepo interface {
	UpsertRSVP(ctx context.Context, response entity.RSVPResponse) error
	ListRSVP(ctx context.Context, blockID string) ([]entity.RSVPResponse, error)

	// ReplacePollChoices заменяет выбор гостя целиком: переголосование не
	// копит старые голоса.
	ReplacePollChoices(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, optionIDs []string) error
	CountPollChoices(ctx context.Context, blockID string, guestID uuid.UUID) (map[string]int, []string, error)
	CreatePollOption(ctx context.Context, option entity.PollGuestOption) error
	CountPollOptionsByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	ListPollOptions(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.PollOption, error)
	PollOptionWishlist(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error)
	SetPollOptionHidden(ctx context.Context, optionID uuid.UUID, hidden bool) error

	CountTracksByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	CreateTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) (uuid.UUID, error)
	ListTracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error)
	TrackBlockID(ctx context.Context, trackID uuid.UUID) (uuid.UUID, string, error)
	ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) (bool, error)

	CountGuestbookByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error)
	CreateGuestbookEntry(ctx context.Context, entry entity.GuestbookEntry, wishlistID uuid.UUID, blockID string, guestID uuid.UUID) error
	ListGuestbook(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.GuestbookEntry, error)
	GuestbookEntryWishlist(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error)
	SetGuestbookHidden(ctx context.Context, entryID uuid.UUID, hidden bool) error
}

type PresentMetaRepo interface {
	Upsert(ctx context.Context, meta entity.PresentMeta) error
}

type TemplateRepo interface {
	Create(ctx context.Context, template entity.Template) error
	GetByID(ctx context.Context, id uuid.UUID) (entity.Template, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Template, error)
	GetPublic(ctx context.Context, limit, offset int, userID uuid.UUID) ([]entity.TemplateWithAuthor, error)
	Update(ctx context.Context, template entity.Template) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	Like(ctx context.Context, userID, templateID uuid.UUID) (int, error)
	Unlike(ctx context.Context, userID, templateID uuid.UUID) (int, error)
}

// ErrNotFound — записи нет. Использование отличает по нему 404 от сбоя базы,
// не зная про gorm. Пока его возвращает только SantaRepo.
var ErrNotFound = errors.New("not found")

// ErrDuplicate — нарушена уникальность (участник уже в комнате).
var ErrDuplicate = errors.New("duplicate")

// ErrStatusMismatch — комната не в том статусе, которого ждал вызов.
var ErrStatusMismatch = errors.New("status mismatch")

// ScheduledDrawOutcome — чем кончилась попытка жеребьёвки по расписанию.
type ScheduledDrawOutcome string

const (
	// ScheduledDrawSkipped — комната уже не ждёт: разыграна, время сняли или
	// перенесли, её держит другой экземпляр.
	ScheduledDrawSkipped ScheduledDrawOutcome = "skipped"
	ScheduledDrawDone    ScheduledDrawOutcome = "drawn"
	// ScheduledDrawTooFew — готовых меньше minReady: draw_at снят,
	// draw_failed_at поставлен.
	ScheduledDrawTooFew ScheduledDrawOutcome = "too_few"
)

// ErrTooSoon — повтор раньше разрешённого (напоминание организатора).
var ErrTooSoon = errors.New("too soon")

type SantaRepo interface {
	CreateRoom(ctx context.Context, room entity.SantaRoom) error
	GetRoomByID(ctx context.Context, id uuid.UUID) (entity.SantaRoom, error)
	GetRoomBySlug(ctx context.Context, slug string) (entity.SantaRoom, error)
	// ListRoomsByUser — комнаты, где пользователь владелец или участник, новые сверху.
	ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SantaRoom, error)
	// UpdateRoom пишет название, бюджет, даты, сообщение и draw_failed_at;
	// только в open, иначе ErrStatusMismatch.
	UpdateRoom(ctx context.Context, room entity.SantaRoom) error
	// DeleteRoom удаляет комнату вместе с участниками и парами.
	DeleteRoom(ctx context.Context, id uuid.UUID) error

	// CreateParticipant под блокировкой комнаты: не open — ErrStatusMismatch,
	// комнаты нет — ErrNotFound.
	CreateParticipant(ctx context.Context, p entity.SantaParticipant) error
	GetParticipant(ctx context.Context, id uuid.UUID) (entity.SantaParticipant, error)
	// GetParticipantByToken ищет только внутри комнаты: токен из другой
	// комнаты здесь «не найден».
	GetParticipantByToken(ctx context.Context, roomID uuid.UUID, tokenHash string) (entity.SantaParticipant, error)
	GetParticipantByUser(ctx context.Context, roomID, userID uuid.UUID) (entity.SantaParticipant, error)
	// ListParticipants — в порядке вступления.
	ListParticipants(ctx context.Context, roomID uuid.UUID) ([]entity.SantaParticipant, error)
	CountParticipants(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]int, error)
	// UpdateParticipant пишет имя/пожелания/вишлист и кладёт notes в очередь —
	// одной транзакцией. Несданные (pending) wishes_updated того же получателя
	// стираются: Санта получит одно сообщение о последней правке.
	UpdateParticipant(ctx context.Context, p entity.SantaParticipant, notes ...entity.SantaNotification) error
	// DeleteParticipant под блокировкой комнаты: не open — ErrStatusMismatch,
	// участника нет — ErrNotFound.
	DeleteParticipant(ctx context.Context, id uuid.UUID) error

	// Каналы уведомлений.
	// SetEmail запоминает новый адрес в pending_email и кладёт код — одной
	// транзакцией; подтверждённый адрес и готовность не меняются. Адрес
	// подтверждён у другого участника комнаты — ErrDuplicate; участника нет —
	// ErrNotFound.
	SetEmail(ctx context.Context, participantID uuid.UUID, email string, code entity.SantaEmailCode) error
	GetEmailCode(ctx context.Context, participantID uuid.UUID) (entity.SantaEmailCode, error)
	// IncEmailCodeAttempts атомарно занимает попытку проверки кода: attempts+1,
	// пока attempts < max. false — попыток не осталось или кода нет.
	IncEmailCodeAttempts(ctx context.Context, participantID uuid.UUID, max int) (bool, error)
	DeleteEmailCode(ctx context.Context, participantID uuid.UUID) error
	// VerifyEmail: pending_email становится подтверждённым email, канал —
	// почта, код стёрт, приветствие в очереди — одной транзакцией. Код
	// стирается по codeHash: если за это время код заменён или стёрт —
	// ErrNotFound, ничего не меняется. Тот же адрес успел подтвердить другой
	// участник комнаты — ErrDuplicate.
	VerifyEmail(ctx context.Context, participantID uuid.UUID, codeHash string, at time.Time, welcome entity.SantaNotification) error
	CreateTgLink(ctx context.Context, link entity.SantaTgLink) error
	// LinkTelegram по одноразовой ссылке: чат записан, канал — Telegram,
	// ссылки участника стёрты, приветствие в очереди. Ссылки нет или она
	// истекла — ErrNotFound.
	LinkTelegram(ctx context.Context, tokenHash string, chatID int64, now time.Time, welcome func(entity.SantaParticipant) entity.SantaNotification) (entity.SantaParticipant, error)

	GetAssignment(ctx context.Context, roomID, giverID uuid.UUID) (entity.SantaAssignment, error)
	// GetGiver — кто дарит receiverID; пар нет — ErrNotFound.
	GetGiver(ctx context.Context, roomID, receiverID uuid.UUID) (entity.SantaAssignment, error)
	// Draw в одной транзакции: блокирует комнату, проверяет статус expected
	// (иначе ErrStatusMismatch), стирает старые пары и несданные уведомления
	// drawn участников комнаты, отдаёт build id ГОТОВЫХ участников (канал
	// подтверждён) в порядке вступления, пишет пары, кладёт note каждому
	// дарящему и ставит status=drawn. Ошибка build откатывает всё и
	// возвращается как есть. Сбрасывает draw_failed_at.
	Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func(ids []uuid.UUID) ([]entity.SantaAssignment, error), note func(giverID uuid.UUID) entity.SantaNotification) error
	// DueDrawRooms — id открытых комнат с draw_at <= now, ранние первыми.
	DueDrawRooms(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error)
	// DrawScheduled — жеребьёвка по расписанию одной транзакцией. Комната
	// берётся FOR UPDATE SKIP LOCKED и только если всё ещё open с
	// draw_at <= now, иначе ScheduledDrawSkipped. Готовых меньше minReady —
	// draw_at = NULL, draw_failed_at = now и failNote организатору, если он
	// готовый участник комнаты (ScheduledDrawTooFew). Иначе — как Draw
	// (ScheduledDrawDone).
	DrawScheduled(ctx context.Context, roomID uuid.UUID, now time.Time, minReady int, build func(ids []uuid.UUID) ([]entity.SantaAssignment, error), note func(giverID uuid.UUID) entity.SantaNotification, failNote func(organizerID uuid.UUID) entity.SantaNotification) (ScheduledDrawOutcome, error)
	// Remind под блокировкой комнаты: напоминали позже now-cooldown —
	// ErrTooSoon; иначе ставит last_reminded_at=now и кладёт notes.
	Remind(ctx context.Context, roomID uuid.UUID, now time.Time, cooldown time.Duration, notes []entity.SantaNotification) error

	// Outbox.
	// ClaimNotifications берёт до limit созревших pending-уведомлений
	// (FOR UPDATE SKIP LOCKED) и сдвигает им next_try_at на now+lease: второй
	// обработчик их не возьмёт, а упавший — отдаст через lease.
	ClaimNotifications(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]entity.SantaNotification, error)
	// MarkNotificationSent/Failed меняют только pending-уведомление; уже
	// отмеченное или стёртое — ErrNotFound. tgMessageID — message_id в
	// Telegram (nil для почты).
	MarkNotificationSent(ctx context.Context, id uuid.UUID, tgMessageID *int64) error
	// MarkNotificationFailed: retryAt == nil — окончательно failed, иначе
	// снова pending к retryAt.
	MarkNotificationFailed(ctx context.Context, id uuid.UUID, attempts int, retryAt *time.Time, lastErr string) error
	// PurgeStale стирает истёкшие ссылки Telegram и коды почты и отработанные
	// (sent/failed) уведомления, созданные раньше now-keepNotes. Возвращает,
	// сколько строк стёрто.
	PurgeStale(ctx context.Context, now time.Time, keepNotes time.Duration) (int64, error)

	// Анонимный чат пары «Санта → подопечный».
	// CreateMessage пишет сообщение и уведомление получателю одной транзакцией
	// под блокировкой комнаты (FOR SHARE): перезапуск жеребьёвки не проскочит
	// между проверкой пары и вставкой. Комната не drawn — ErrStatusMismatch;
	// такой пары нет — ErrNotFound; автор написал limit сообщений позже since —
	// ErrTooSoon.
	CreateMessage(ctx context.Context, msg entity.SantaMessage, since time.Time, limit int, note entity.SantaNotification) error
	GetMessage(ctx context.Context, id uuid.UUID) (entity.SantaMessage, error)
	// ListMessages — последние limit сообщений пары по возрастанию времени.
	ListMessages(ctx context.Context, roomID, giverID, receiverID uuid.UUID, limit int) ([]entity.SantaMessage, error)
	// MarkMessagesRead отмечает прочитанными непрочитанные сообщения пары,
	// написанные одной стороной: fromGiver — Сантой, иначе подопечным.
	MarkMessagesRead(ctx context.Context, roomID, giverID, receiverID uuid.UUID, fromGiver bool, at time.Time) error
	// CountUnread — непрочитанные участником: от его Санты и от его подопечного.
	CountUnread(ctx context.Context, roomID, participantID uuid.UUID) (fromSanta, fromReceiver int, err error)
	// FindChatNotification — уведомление chat_message, ушедшее в Telegram-чат
	// chatID сообщением tgMessageID; нет — ErrNotFound.
	FindChatNotification(ctx context.Context, chatID, tgMessageID int64) (entity.SantaNotification, error)
}
