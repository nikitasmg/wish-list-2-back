package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"main/internal/entity"
	"main/internal/repo"
)

type santaRepo struct {
	db *gorm.DB
}

func NewSantaRepo(db *gorm.DB) *santaRepo {
	return &santaRepo{db: db}
}

// santaErr подменяет «записи нет» gorm на repo.ErrNotFound.
func santaErr(op string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%s: %w", op, repo.ErrNotFound)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// isUniqueViolation — нарушен именно этот уникальный индекс.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

func (r *santaRepo) CreateRoom(ctx context.Context, room entity.SantaRoom) error {
	m := toSantaRoomModel(room)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return santaErr("santaRepo.CreateRoom", err)
	}
	return nil
}

func (r *santaRepo) GetRoomByID(ctx context.Context, id uuid.UUID) (entity.SantaRoom, error) {
	var m SantaRoomModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.SantaRoom{}, santaErr("santaRepo.GetRoomByID", err)
	}
	return toSantaRoomEntity(m), nil
}

func (r *santaRepo) GetRoomBySlug(ctx context.Context, slug string) (entity.SantaRoom, error) {
	var m SantaRoomModel
	if err := r.db.WithContext(ctx).First(&m, "slug = ?", slug).Error; err != nil {
		return entity.SantaRoom{}, santaErr("santaRepo.GetRoomBySlug", err)
	}
	return toSantaRoomEntity(m), nil
}

func (r *santaRepo) ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]entity.SantaRoom, error) {
	joined := r.db.Model(&SantaParticipantModel{}).Select("room_id").Where("user_id = ?", userID)
	var models []SantaRoomModel
	if err := r.db.WithContext(ctx).
		Where("owner_id = ? OR id IN (?)", userID, joined).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, santaErr("santaRepo.ListRoomsByUser", err)
	}
	rooms := make([]entity.SantaRoom, len(models))
	for i, m := range models {
		rooms[i] = toSantaRoomEntity(m)
	}
	return rooms, nil
}

func (r *santaRepo) UpdateRoom(ctx context.Context, room entity.SantaRoom) error {
	res := r.db.WithContext(ctx).Model(&SantaRoomModel{}).
		Where("id = ? AND status = ?", room.ID, string(entity.SantaRoomOpen)).
		Updates(map[string]any{
			"title":          room.Title,
			"budget":         room.Budget,
			"exchange_date":  room.ExchangeDate,
			"draw_at":        room.DrawAt,
			"draw_failed_at": room.DrawFailedAt,
			"message":        room.Message,
			"updated_at":     time.Now(),
		})
	if res.Error != nil {
		return santaErr("santaRepo.UpdateRoom", res.Error)
	}
	if res.RowsAffected == 0 {
		var n int64
		if err := r.db.WithContext(ctx).Model(&SantaRoomModel{}).Where("id = ?", room.ID).Count(&n).Error; err != nil {
			return santaErr("santaRepo.UpdateRoom", err)
		}
		if n == 0 {
			return fmt.Errorf("santaRepo.UpdateRoom: %w", repo.ErrNotFound)
		}
		return repo.ErrStatusMismatch
	}
	return nil
}

func (r *santaRepo) DeleteRoom(ctx context.Context, id uuid.UUID) error {
	// Участники и пары удаляются каскадом внешних ключей.
	if err := r.db.WithContext(ctx).Delete(&SantaRoomModel{}, "id = ?", id).Error; err != nil {
		return santaErr("santaRepo.DeleteRoom", err)
	}
	return nil
}

// lockOpenRoom берёт комнату под FOR SHARE и проверяет, что она открыта.
// Жеребьёвка берёт FOR UPDATE, поэтому вступление и выход не проскочат
// между её проверкой статуса и коммитом.
func lockOpenRoom(tx *gorm.DB, op string, roomID uuid.UUID) error {
	var room SantaRoomModel
	if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&room, "id = ?", roomID).Error; err != nil {
		return santaErr(op, err)
	}
	if room.Status != string(entity.SantaRoomOpen) {
		return repo.ErrStatusMismatch
	}
	return nil
}

func (r *santaRepo) CreateParticipant(ctx context.Context, p entity.SantaParticipant) error {
	m := toSantaParticipantModel(p)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockOpenRoom(tx, "santaRepo.CreateParticipant lock", p.RoomID); err != nil {
			return err
		}
		if err := tx.Create(&m).Error; err != nil {
			// Параллельное вступление того же аккаунта в ту же комнату.
			if isUniqueViolation(err, "idx_santa_participant_user") {
				return fmt.Errorf("santaRepo.CreateParticipant: %w", repo.ErrDuplicate)
			}
			return santaErr("santaRepo.CreateParticipant", err)
		}
		return nil
	})
}

func (r *santaRepo) getParticipant(ctx context.Context, op string, query string, args ...any) (entity.SantaParticipant, error) {
	var m SantaParticipantModel
	if err := r.db.WithContext(ctx).Where(query, args...).First(&m).Error; err != nil {
		return entity.SantaParticipant{}, santaErr(op, err)
	}
	return toSantaParticipantEntity(m), nil
}

func (r *santaRepo) GetParticipant(ctx context.Context, id uuid.UUID) (entity.SantaParticipant, error) {
	return r.getParticipant(ctx, "santaRepo.GetParticipant", "id = ?", id)
}

func (r *santaRepo) GetParticipantByToken(ctx context.Context, roomID uuid.UUID, tokenHash string) (entity.SantaParticipant, error) {
	return r.getParticipant(ctx, "santaRepo.GetParticipantByToken", "room_id = ? AND token_hash = ?", roomID, tokenHash)
}

func (r *santaRepo) GetParticipantByUser(ctx context.Context, roomID, userID uuid.UUID) (entity.SantaParticipant, error) {
	return r.getParticipant(ctx, "santaRepo.GetParticipantByUser", "room_id = ? AND user_id = ?", roomID, userID)
}

func (r *santaRepo) ListParticipants(ctx context.Context, roomID uuid.UUID) ([]entity.SantaParticipant, error) {
	var models []SantaParticipantModel
	if err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at, id").
		Find(&models).Error; err != nil {
		return nil, santaErr("santaRepo.ListParticipants", err)
	}
	out := make([]entity.SantaParticipant, len(models))
	for i, m := range models {
		out[i] = toSantaParticipantEntity(m)
	}
	return out, nil
}

func (r *santaRepo) CountParticipants(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(roomIDs))
	if len(roomIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		RoomID uuid.UUID
		N      int
	}
	if err := r.db.WithContext(ctx).
		Model(&SantaParticipantModel{}).
		Select("room_id, count(*) AS n").
		Where("room_id IN ?", roomIDs).
		Group("room_id").
		Scan(&rows).Error; err != nil {
		return nil, santaErr("santaRepo.CountParticipants", err)
	}
	for _, row := range rows {
		counts[row.RoomID] = row.N
	}
	return counts, nil
}

func (r *santaRepo) UpdateParticipant(ctx context.Context, p entity.SantaParticipant, notes ...entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&SantaParticipantModel{}).
			Where("id = ?", p.ID).
			Updates(map[string]any{
				"name":         p.Name,
				"wishes":       p.Wishes,
				"wishlist_url": p.WishlistURL,
				"updated_at":   time.Now(),
			})
		if res.Error != nil {
			return santaErr("santaRepo.UpdateParticipant", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.UpdateParticipant: %w", repo.ErrNotFound)
		}
		// «Пожелания обновились» схлопываются: несданное старое стираем, кладём
		// одно новое. Стираем любое pending этого вида — и ждущее повтора, и
		// взятое обработчиком в работу: тот при отметке увидит 0 строк, а Санта
		// всё равно получит свежее сообщение о последней правке.
		for _, n := range notes {
			if n.Kind != entity.SantaNotifyWishesUpdated {
				continue
			}
			if err := tx.Where("participant_id = ? AND kind = ? AND status = ?",
				n.ParticipantID, string(n.Kind), string(entity.SantaNotificationPending)).
				Delete(&SantaNotificationModel{}).Error; err != nil {
				return santaErr("santaRepo.UpdateParticipant collapse", err)
			}
		}
		return insertNotifications(tx, notes...)
	})
}

func (r *santaRepo) DeleteParticipant(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m SantaParticipantModel
		if err := tx.First(&m, "id = ?", id).Error; err != nil {
			return santaErr("santaRepo.DeleteParticipant find", err)
		}
		if err := lockOpenRoom(tx, "santaRepo.DeleteParticipant lock", m.RoomID); err != nil {
			return err
		}
		res := tx.Delete(&SantaParticipantModel{}, "id = ?", id)
		if res.Error != nil {
			return santaErr("santaRepo.DeleteParticipant", res.Error)
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("santaRepo.DeleteParticipant: %w", repo.ErrNotFound)
		}
		return nil
	})
}

func (r *santaRepo) GetAssignment(ctx context.Context, roomID, giverID uuid.UUID) (entity.SantaAssignment, error) {
	var m SantaAssignmentModel
	if err := r.db.WithContext(ctx).First(&m, "room_id = ? AND giver_id = ?", roomID, giverID).Error; err != nil {
		return entity.SantaAssignment{}, santaErr("santaRepo.GetAssignment", err)
	}
	return entity.SantaAssignment{RoomID: m.RoomID, GiverID: m.GiverID, ReceiverID: m.ReceiverID}, nil
}

// santaReadySQL — тот же «готов», что entity.SantaParticipant.Ready(); меняются вместе.
const santaReadySQL = "((channel = 'email' AND email IS NOT NULL AND email_verified_at IS NOT NULL) OR (channel = 'telegram' AND tg_chat_id IS NOT NULL))"

func (r *santaRepo) GetGiver(ctx context.Context, roomID, receiverID uuid.UUID) (entity.SantaAssignment, error) {
	var m SantaAssignmentModel
	if err := r.db.WithContext(ctx).First(&m, "room_id = ? AND receiver_id = ?", roomID, receiverID).Error; err != nil {
		return entity.SantaAssignment{}, santaErr("santaRepo.GetGiver", err)
	}
	return entity.SantaAssignment{RoomID: m.RoomID, GiverID: m.GiverID, ReceiverID: m.ReceiverID}, nil
}

func (r *santaRepo) Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func([]uuid.UUID) ([]entity.SantaAssignment, error), note func(giverID uuid.UUID) entity.SantaNotification) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE: два одновременных нажатия «Жеребьёвка» выстраиваются в
		// очередь, и второе видит уже drawn.
		var room SantaRoomModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&room, "id = ?", roomID).Error; err != nil {
			return santaErr("santaRepo.Draw lock", err)
		}
		if room.Status != string(expected) {
			return repo.ErrStatusMismatch
		}
		ids, err := readyIDs(tx, roomID)
		if err != nil {
			return err
		}
		return drawLocked(tx, roomID, ids, build, note, time.Now())
	})
}

// readyIDs — готовые участники комнаты (канал подтверждён) в порядке вступления.
func readyIDs(tx *gorm.DB, roomID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := tx.Model(&SantaParticipantModel{}).
		Where("room_id = ?", roomID).
		Where(santaReadySQL).
		Order("created_at, id").
		Pluck("id", &ids).Error; err != nil {
		return nil, santaErr("santaRepo.readyIDs", err)
	}
	return ids, nil
}

// drawLocked — жеребьёвка под уже взятой блокировкой комнаты: стирает
// прошлые пары и несданные «кому дарить», пишет новый круг, кладёт
// уведомления и ставит drawn. Ошибка build откатывает транзакцию.
func drawLocked(tx *gorm.DB, roomID uuid.UUID, ids []uuid.UUID, build func([]uuid.UUID) ([]entity.SantaAssignment, error), note func(giverID uuid.UUID) entity.SantaNotification, now time.Time) error {
	if err := tx.Where("room_id = ?", roomID).Delete(&SantaAssignmentModel{}).Error; err != nil {
		return santaErr("santaRepo.Draw clear", err)
	}
	// Несданные «кому дарить» прошлой жеребьёвки больше не правда, а «новое
	// сообщение» — про переписку, которой сейчас не станет.
	members := tx.Model(&SantaParticipantModel{}).Select("id").Where("room_id = ?", roomID)
	if err := tx.Where("status = ? AND kind IN ? AND participant_id IN (?)",
		string(entity.SantaNotificationPending),
		[]string{string(entity.SantaNotifyDrawn), string(entity.SantaNotifyChatMessage)}, members).
		Delete(&SantaNotificationModel{}).Error; err != nil {
		return santaErr("santaRepo.Draw clear notes", err)
	}
	// Переписка привязана к паре: у новых пар старые сообщения всплыть не должны.
	if err := tx.Where("room_id = ?", roomID).Delete(&SantaMessageModel{}).Error; err != nil {
		return santaErr("santaRepo.Draw clear messages", err)
	}
	pairs, err := build(ids)
	if err != nil {
		return err
	}
	models := make([]SantaAssignmentModel, len(pairs))
	notes := make([]entity.SantaNotification, len(pairs))
	for i, p := range pairs {
		models[i] = SantaAssignmentModel{RoomID: p.RoomID, GiverID: p.GiverID, ReceiverID: p.ReceiverID}
		notes[i] = note(p.GiverID)
	}
	if len(models) > 0 {
		if err := tx.Create(&models).Error; err != nil {
			return santaErr("santaRepo.Draw insert", err)
		}
	}
	if err := insertNotifications(tx, notes...); err != nil {
		return err
	}
	if err := tx.Model(&SantaRoomModel{}).Where("id = ?", roomID).Updates(map[string]any{
		"status":         string(entity.SantaRoomDrawn),
		"drawn_at":       now,
		"draw_failed_at": nil,
		"updated_at":     now,
	}).Error; err != nil {
		return santaErr("santaRepo.Draw status", err)
	}
	return nil
}

func (r *santaRepo) DueDrawRooms(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&SantaRoomModel{}).
		Where("status = ? AND draw_at IS NOT NULL AND draw_at <= ?", string(entity.SantaRoomOpen), now).
		Order("draw_at, id").
		Limit(limit).
		Pluck("id", &ids).Error; err != nil {
		return nil, santaErr("santaRepo.DueDrawRooms", err)
	}
	return ids, nil
}

func (r *santaRepo) DrawScheduled(ctx context.Context, roomID uuid.UUID, now time.Time, minReady int, build func([]uuid.UUID) ([]entity.SantaAssignment, error), note func(giverID uuid.UUID) entity.SantaNotification, failNote func(organizerID uuid.UUID) entity.SantaNotification) (repo.ScheduledDrawOutcome, error) {
	outcome := repo.ScheduledDrawSkipped
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// SKIP LOCKED: комнату держит ручная жеребьёвка или второй экземпляр —
		// пропускаем; к следующему тику она уже не open. Условие повторяем под
		// блокировкой: время могли снять или перенести.
		var rooms []SantaRoomModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("id = ? AND status = ? AND draw_at IS NOT NULL AND draw_at <= ?", roomID, string(entity.SantaRoomOpen), now).
			Limit(1).
			Find(&rooms).Error; err != nil {
			return santaErr("santaRepo.DrawScheduled lock", err)
		}
		if len(rooms) == 0 {
			return nil
		}
		ids, err := readyIDs(tx, roomID)
		if err != nil {
			return err
		}
		if len(ids) < minReady {
			// Время снимаем, иначе планировщик пытался бы каждую минуту.
			if err := tx.Model(&SantaRoomModel{}).Where("id = ?", roomID).Updates(map[string]any{
				"draw_at":        nil,
				"draw_failed_at": now,
				"updated_at":     now,
			}).Error; err != nil {
				return santaErr("santaRepo.DrawScheduled fail", err)
			}
			outcome = repo.ScheduledDrawTooFew
			// У аккаунтов нет почты: написать организатору можно, только если
			// он сам участник с подтверждённым каналом.
			var owner []SantaParticipantModel
			if err := tx.Where("room_id = ? AND user_id = ?", roomID, rooms[0].OwnerID).
				Where(santaReadySQL).
				Limit(1).
				Find(&owner).Error; err != nil {
				return santaErr("santaRepo.DrawScheduled owner", err)
			}
			if len(owner) == 0 {
				return nil
			}
			return insertNotifications(tx, failNote(owner[0].ID))
		}
		if err := drawLocked(tx, roomID, ids, build, note, now); err != nil {
			return err
		}
		outcome = repo.ScheduledDrawDone
		return nil
	})
	if err != nil {
		return repo.ScheduledDrawSkipped, err
	}
	return outcome, nil
}
