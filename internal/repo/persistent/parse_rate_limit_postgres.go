package persistent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"main/internal/repo"
)

type parseRateLimitRepo struct {
	db *gorm.DB
}

func NewParseRateLimitRepo(db *gorm.DB) repo.ParseRateLimitRepo {
	return &parseRateLimitRepo{db: db}
}

func (r *parseRateLimitRepo) IncrementAndCheck(ctx context.Context, userID uuid.UUID, windowStart time.Time) (int, error) {
	var count int
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO parse_rate_limits (user_id, window_start, count)
		VALUES (?, ?, 1)
		ON CONFLICT (user_id) DO UPDATE SET
		  count        = CASE WHEN parse_rate_limits.window_start = ?
		                      THEN parse_rate_limits.count + 1
		                      ELSE 1 END,
		  window_start = ?
		RETURNING count
	`, userID, windowStart, windowStart, windowStart).Scan(&count).Error
	return count, err
}
