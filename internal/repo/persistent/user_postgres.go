package persistent

import (
	"context"
	"fmt"

	"main/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *userRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user entity.User) error {
	m := toUserModel(user)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (entity.User, error) {
	var m UserModel
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&m).Error; err != nil {
		return entity.User{}, fmt.Errorf("userRepo.GetByUsername: %w", err)
	}
	return toUserEntity(m), nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.User, error) {
	var m UserModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.User{}, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return toUserEntity(m), nil
}

func (r *userRepo) Update(ctx context.Context, user entity.User) error {
	m := toUserModel(user)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	return nil
}
