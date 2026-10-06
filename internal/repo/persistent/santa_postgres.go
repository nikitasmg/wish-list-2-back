package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
			"title":         room.Title,
			"budget":        room.Budget,
			"exchange_date": room.ExchangeDate,
			"draw_at":       room.DrawAt,
			"message":       room.Message,
			"updated_at":    time.Now(),
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

func (r *santaRepo) UpdateParticipant(ctx context.Context, p entity.SantaParticipant) error {
	res := r.db.WithContext(ctx).Model(&SantaParticipantModel{}).
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
	return nil
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

func (r *santaRepo) Draw(ctx context.Context, roomID uuid.UUID, expected entity.SantaRoomStatus, build func([]uuid.UUID) ([]entity.SantaAssignment, error)) error {
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
		if err := tx.Where("room_id = ?", roomID).Delete(&SantaAssignmentModel{}).Error; err != nil {
			return santaErr("santaRepo.Draw clear", err)
		}
		var ids []uuid.UUID
		if err := tx.Model(&SantaParticipantModel{}).
			Where("room_id = ?", roomID).
			Order("created_at, id").
			Pluck("id", &ids).Error; err != nil {
			return santaErr("santaRepo.Draw participants", err)
		}
		pairs, err := build(ids)
		if err != nil {
			return err
		}
		models := make([]SantaAssignmentModel, len(pairs))
		for i, p := range pairs {
			models[i] = SantaAssignmentModel{RoomID: p.RoomID, GiverID: p.GiverID, ReceiverID: p.ReceiverID}
		}
		if len(models) > 0 {
			if err := tx.Create(&models).Error; err != nil {
				return santaErr("santaRepo.Draw insert", err)
			}
		}
		now := time.Now()
		if err := tx.Model(&SantaRoomModel{}).Where("id = ?", roomID).Updates(map[string]any{
			"status":     string(entity.SantaRoomDrawn),
			"drawn_at":   now,
			"updated_at": now,
		}).Error; err != nil {
			return santaErr("santaRepo.Draw status", err)
		}
		return nil
	})
}
