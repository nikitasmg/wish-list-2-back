package persistent

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"main/internal/entity"
	"main/internal/repo"
)

type presentMetaRepo struct {
	db *gorm.DB
}

func NewPresentMetaRepo(db *gorm.DB) repo.PresentMetaRepo {
	return &presentMetaRepo{db: db}
}

func (r *presentMetaRepo) Upsert(ctx context.Context, meta entity.PresentMeta) error {
	model := PresentMetaModel{
		PresentID:   meta.PresentID,
		Source:      meta.Source,
		OriginalURL: meta.OriginalURL,
		Category:    meta.Category,
		Brand:       meta.Brand,
		ParsedAt:    meta.ParsedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "present_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"source", "original_url", "category", "brand", "parsed_at"}),
	}).Create(&model).Error
}
