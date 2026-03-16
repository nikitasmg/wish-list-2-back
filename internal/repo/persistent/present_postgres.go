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
	if err := r.db.WithContext(ctx).Where("wishlist_id = ?", wishlistID).Find(&models).Error; err != nil {
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

func (r *presentRepo) CountByWishlistID(ctx context.Context, wishlistID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PresentModel{}).Where("wishlist_id = ?", wishlistID).Count(&count).Error
	return count, err
}
