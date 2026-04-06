package persistent

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"main/internal/entity"
)

type templateRepo struct {
	db *gorm.DB
}

func NewTemplateRepo(db *gorm.DB) *templateRepo {
	return &templateRepo{db: db}
}

func (r *templateRepo) Create(ctx context.Context, template entity.Template) error {
	m := toTemplateModel(template)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("templateRepo.Create: %w", err)
	}
	return nil
}

func (r *templateRepo) GetByID(ctx context.Context, id uuid.UUID) (entity.Template, error) {
	var m TemplateModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return entity.Template{}, fmt.Errorf("templateRepo.GetByID: %w", err)
	}
	return toTemplateEntity(m), nil
}

func (r *templateRepo) GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]entity.Template, error) {
	var models []TemplateModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("templateRepo.GetAllByUserID: %w", err)
	}
	result := make([]entity.Template, len(models))
	for i, m := range models {
		result[i] = toTemplateEntity(m)
	}
	return result, nil
}

type templateWithAuthorRow struct {
	TemplateModel
	UserDisplayName string `gorm:"column:user_display_name"`
	LikedByMe       bool   `gorm:"column:liked_by_me"`
}

func (r *templateRepo) GetPublic(ctx context.Context, limit, offset int, userID uuid.UUID) ([]entity.TemplateWithAuthor, error) {
	var rows []templateWithAuthorRow

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			t.*,
			COALESCE(u.display_name, '') AS user_display_name,
			CASE WHEN tl.user_id IS NOT NULL THEN true ELSE false END AS liked_by_me
		FROM templates t
		LEFT JOIN users u ON u.id = t.user_id
		LEFT JOIN template_likes tl ON tl.template_id = t.id AND tl.user_id = ?
		WHERE t.is_public = true
		ORDER BY
			t.likes_count::float8 / POWER(EXTRACT(EPOCH FROM (NOW() - t.created_at)) / 3600.0 + 2, 1.5) DESC,
			t.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset).Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("templateRepo.GetPublic: %w", err)
	}

	result := make([]entity.TemplateWithAuthor, len(rows))
	for i, row := range rows {
		result[i] = entity.TemplateWithAuthor{
			Template:        toTemplateEntity(row.TemplateModel),
			UserDisplayName: row.UserDisplayName,
			LikesCount:      row.LikesCount,
			LikedByMe:       row.LikedByMe,
		}
	}
	return result, nil
}

func (r *templateRepo) Like(ctx context.Context, userID, templateID uuid.UUID) (int, error) {
	var newCount int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			"INSERT INTO template_likes (user_id, template_id, created_at) VALUES (?, ?, NOW()) ON CONFLICT DO NOTHING",
			userID, templateID,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("already liked")
		}
		return tx.Raw(
			"UPDATE templates SET likes_count = likes_count + 1 WHERE id = ? RETURNING likes_count",
			templateID,
		).Scan(&newCount).Error
	})
	return newCount, err
}

func (r *templateRepo) Unlike(ctx context.Context, userID, templateID uuid.UUID) (int, error) {
	var newCount int
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			"DELETE FROM template_likes WHERE user_id = ? AND template_id = ?",
			userID, templateID,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("not liked")
		}
		return tx.Raw(
			"UPDATE templates SET likes_count = GREATEST(likes_count - 1, 0) WHERE id = ? RETURNING likes_count",
			templateID,
		).Scan(&newCount).Error
	})
	return newCount, err
}

func (r *templateRepo) Update(ctx context.Context, template entity.Template) error {
	m := toTemplateModel(template)
	if err := r.db.WithContext(ctx).Save(&m).Error; err != nil {
		return fmt.Errorf("templateRepo.Update: %w", err)
	}
	return nil
}

func (r *templateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&TemplateModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("templateRepo.Delete: %w", err)
	}
	return nil
}

func (r *templateRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&TemplateModel{}).
		Where("user_id = ?", userID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("templateRepo.CountByUserID: %w", err)
	}
	return count, nil
}
