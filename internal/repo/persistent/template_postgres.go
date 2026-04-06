package persistent

import (
	"context"
	"fmt"
	"time"

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
}

func (r *templateRepo) GetPublic(ctx context.Context, limit int, cursor time.Time) ([]entity.TemplateWithAuthor, error) {
	var rows []templateWithAuthorRow

	q := r.db.WithContext(ctx).
		Table("templates").
		Select("templates.*, users.display_name AS user_display_name").
		Joins("LEFT JOIN users ON users.id = templates.user_id").
		Where("templates.is_public = ?", true).
		Order("templates.created_at DESC").
		Limit(limit)

	if !cursor.IsZero() {
		q = q.Where("templates.created_at < ?", cursor)
	}

	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("templateRepo.GetPublic: %w", err)
	}

	result := make([]entity.TemplateWithAuthor, len(rows))
	for i, row := range rows {
		result[i] = entity.TemplateWithAuthor{
			Template:        toTemplateEntity(row.TemplateModel),
			UserDisplayName: row.UserDisplayName,
		}
	}
	return result, nil
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
