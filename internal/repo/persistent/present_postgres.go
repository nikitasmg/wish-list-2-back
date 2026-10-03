package persistent

import (
	"context"
	"fmt"

	"main/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type presentRepo struct {
	db *gorm.DB
}

func NewPresentRepo(db *gorm.DB) *presentRepo {
	return &presentRepo{db: db}
}

func (r *presentRepo) Create(ctx context.Context, present entity.Present) error {
	m := toPresentModel(present)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("presentRepo.Create: %w", err)
	}
	return nil
}

func (r *presentRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Present, error) {
	var m PresentModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.Present{}, fmt.Errorf("presentRepo.GetByID: %w", err)
	}
	return toPresentEntity(m), nil
}

func (r *presentRepo) GetAllByWishlistID(ctx context.Context, wishlistID uuid.UUID) ([]entity.Present, error) {
	var models []PresentModel
	// Главная мечта всегда первая, дальше — порядок, выставленный руками.
	// created_at — для подарков, у которых порядок ещё не задавали (все нули).
	if err := r.db.WithContext(ctx).Where("wishlist_id = ?", wishlistID).
		Order("is_main DESC, sort_order ASC, created_at ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("presentRepo.GetAllByWishlistID: %w", err)
	}
	presents := make([]entity.Present, len(models))
	for i, m := range models {
		presents[i] = toPresentEntity(m)
	}
	return presents, nil
}

func (r *presentRepo) Update(ctx context.Context, present entity.Present) error {
	m := toPresentModel(present)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("presentRepo.Update: %w", err)
	}
	return nil
}

func (r *presentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&PresentModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("presentRepo.Delete: %w", err)
	}
	return nil
}

// Reserve — атомарная бронь: одним условным UPDATE, без чтения перед записью.
// Два гостя, нажавших «Забронировать» одновременно, попадают в один и тот же
// WHERE reserved = false, но выиграет ровно один — второй получит 0 строк.
func (r *presentRepo) Reserve(ctx context.Context, id, guestID uuid.UUID, name string) (bool, error) {
	guest := guestID.String()
	result := r.db.WithContext(ctx).Model(&PresentModel{}).
		Where("id = ? AND reserved = ? AND gifted = ?", id, false, false).
		Updates(map[string]interface{}{
			"reserved":          true,
			"reserved_by_guest": &guest,
			"reserved_by_name":  name,
		})
	if result.Error != nil {
		return false, fmt.Errorf("presentRepo.Reserve: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Release — снятие брони только тем гостем, который её поставил. Проверка живёт
// в WHERE, а не в коде выше: иначе между чтением и записью успевает вклиниться
// чужой запрос.
func (r *presentRepo) Release(ctx context.Context, id, guestID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).Model(&PresentModel{}).
		Where("id = ? AND reserved = ? AND reserved_by_guest = ?", id, true, guestID.String()).
		Updates(map[string]interface{}{
			"reserved":          false,
			"reserved_by_guest": nil,
			"reserved_by_name":  "",
		})
	if result.Error != nil {
		return false, fmt.Errorf("presentRepo.Release: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// ClearMain — у вишлиста одна главная мечта.
func (r *presentRepo) ClearMain(ctx context.Context, wishlistID, exceptID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&PresentModel{}).
		Where("wishlist_id = ? AND id <> ? AND is_main = ?", wishlistID, exceptID, true).
		Update("is_main", false).Error
	if err != nil {
		return fmt.Errorf("presentRepo.ClearMain: %w", err)
	}
	return nil
}

// Reorder — порядок одной транзакцией: оборванная на середине перестановка
// оставила бы два подарка с одинаковым номером.
func (r *presentRepo) Reorder(ctx context.Context, wishlistID uuid.UUID, ids []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			// wishlist_id в условии: чужой id из запроса просто не найдётся.
			if err := tx.Model(&PresentModel{}).
				Where("id = ? AND wishlist_id = ?", id, wishlistID).
				Update("sort_order", i).Error; err != nil {
				return fmt.Errorf("presentRepo.Reorder: %w", err)
			}
		}
		return nil
	})
}

func (r *presentRepo) SetGifted(ctx context.Context, id uuid.UUID, gifted bool) error {
	if err := r.db.WithContext(ctx).Model(&PresentModel{}).Where("id = ?", id).
		Update("gifted", gifted).Error; err != nil {
		return fmt.Errorf("presentRepo.SetGifted: %w", err)
	}
	return nil
}

func (r *presentRepo) CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PresentModel{}).Where("wishlist_id = ?", wishlistID).Count(&count).Error
	return count, err
}
