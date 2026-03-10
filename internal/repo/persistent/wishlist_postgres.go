package persistent

import (
	"context"
	"fmt"

	"main/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type wishlistRepo struct {
	db *gorm.DB
}

func NewWishlistRepo(db *gorm.DB) *wishlistRepo {
	return &wishlistRepo{db: db}
}

func (r *wishlistRepo) Create(ctx context.Context, wishlist entity.Wishlist) error {
	m := toWishlistModel(wishlist)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("wishlistRepo.Create: %w", err)
	}
	return nil
}

func (r *wishlistRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	var m WishlistModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlistRepo.GetByID: %w", err)
	}
	return toWishlistEntity(m), nil
}

func (r *wishlistRepo) GetByShortID(ctx context.Context, shortID string) (entity.Wishlist, error) {
	var m WishlistModel
	if err := r.db.WithContext(ctx).First(&m, "short_id = ?", shortID).Error; err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlistRepo.GetByShortID: %w", err)
	}
	return toWishlistEntity(m), nil
}

func (r *wishlistRepo) GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	var models []WishlistModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("wishlistRepo.GetAllByUserID: %w", err)
	}
	wishlists := make([]entity.Wishlist, len(models))
	for i, m := range models {
		wishlists[i] = toWishlistEntity(m)
	}
	return wishlists, nil
}

func (r *wishlistRepo) Update(ctx context.Context, wishlist entity.Wishlist) error {
	m := toWishlistModel(wishlist)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("wishlistRepo.Update: %w", err)
	}
	return nil
}

func (r *wishlistRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&WishlistModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("wishlistRepo.Delete: %w", err)
	}
	return nil
}

// IncrementPresentsCount — атомарное обновление, исключает race condition
func (r *wishlistRepo) IncrementPresentsCount(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&WishlistModel{}).
		Where("id = ?", id).
		UpdateColumn("presents_count", gorm.Expr("presents_count + 1"))
	if result.Error != nil {
		return fmt.Errorf("wishlistRepo.IncrementPresentsCount: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("wishlistRepo.IncrementPresentsCount: wishlist not found")
	}
	return nil
}

// DecrementPresentsCount — атомарное обновление, исключает race condition
func (r *wishlistRepo) DecrementPresentsCount(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Model(&WishlistModel{}).
		Where("id = ? AND presents_count > 0", id).
		UpdateColumn("presents_count", gorm.Expr("presents_count - 1"))
	if result.Error != nil {
		return fmt.Errorf("wishlistRepo.DecrementPresentsCount: %w", result.Error)
	}
	return nil
}
