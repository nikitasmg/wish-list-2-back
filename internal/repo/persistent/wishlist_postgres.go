package persistent

import (
	"context"
	"fmt"
	"time"

	"main/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// UpdateMetadata — запись настроек вишлиста с проверкой версии.
//
// Обновляются только метаданные, и сохранённая строка возвращается целиком
// (RETURNING). Save() всей модели здесь не годится по двум причинам сразу:
// он писал бы ещё и blocks из снимка, прочитанного до правки, — и настройки
// молча затирали бы блоки, сохранённые в это время из конструктора; а клиент
// получал бы в ответ прежний updated_at и уносил в следующий запрос версию,
// которой в базе уже нет.
func (r *wishlistRepo) UpdateMetadata(
	ctx context.Context,
	id uuid.UUID,
	wishlist entity.Wishlist,
	expectedUpdatedAt time.Time,
) (entity.Wishlist, bool, error) {
	m := toWishlistModel(wishlist)

	query := `
		UPDATE wishlists
		SET title = ?, description = ?, cover = ?, settings = ?, location = ?,
		    event_date = ?, occasion = ?, updated_at = ?
		WHERE id = ?`
	args := []interface{}{
		m.Title, m.Description, m.Cover, m.Settings, m.Location,
		m.EventDate, m.Occasion, time.Now(), id,
	}

	if !expectedUpdatedAt.IsZero() {
		query += ` AND updated_at = ?`
		args = append(args, expectedUpdatedAt)
	}
	query += ` RETURNING *`

	// Скан в срез, а не в структуру: ноль строк — это «версия разошлась»,
	// обычный исход, а не ошибка ErrRecordNotFound.
	var saved []WishlistModel
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&saved).Error; err != nil {
		return entity.Wishlist{}, false, fmt.Errorf("wishlistRepo.UpdateMetadata: %w", err)
	}
	if len(saved) == 0 {
		return entity.Wishlist{}, false, nil
	}

	return toWishlistEntity(saved[0]), true, nil
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

// ReservedCountsByUser — сколько подарков занято в каждом вишлисте пользователя.
// Одним запросом с группировкой: карточка в кабинете показывает «4 из 8 заняты»,
// и тянуть ради этой строки все подарки каждого вишлиста незачем.
func (r *wishlistRepo) ReservedCountsByUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uint, error) {
	var rows []struct {
		WishlistID uuid.UUID
		Reserved   uint
	}

	err := r.db.WithContext(ctx).
		Model(&PresentModel{}).
		Select("wishlist_id, count(*) as reserved").
		Where("reserved = ? AND wishlist_id IN (?)", true,
			r.db.Model(&WishlistModel{}).Select("id").Where("user_id = ?", userID)).
		Group("wishlist_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("wishlistRepo.ReservedCountsByUser: %w", err)
	}

	counts := make(map[uuid.UUID]uint, len(rows))
	for _, row := range rows {
		counts[row.WishlistID] = row.Reserved
	}
	return counts, nil
}

// RegisterView — засчитывает просмотр публичной страницы. Дедупликация живёт
// в уникальном ключе таблицы wishlist_views: повторная загрузка страницы тем же
// гостем попадает в ON CONFLICT DO NOTHING и счётчик не трогает.
func (r *wishlistRepo) RegisterView(ctx context.Context, wishlistID, guestID uuid.UUID) error {
	view := WishlistViewModel{WishlistID: wishlistID, GuestID: guestID}

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&view)
	if result.Error != nil {
		return fmt.Errorf("wishlistRepo.RegisterView: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil // этот гость уже был здесь
	}

	if err := r.db.WithContext(ctx).Model(&WishlistModel{}).
		Where("id = ?", wishlistID).
		UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
		return fmt.Errorf("wishlistRepo.RegisterView: increment: %w", err)
	}
	return nil
}

// UpdateBlocks — запись блоков с проверкой версии. Условие по updated_at
// заменяет чтение-перед-записью: если из другой вкладки уже сохранились другие
// блоки, строка под условие не подойдёт и мы вернём false вместо молчаливой
// перезаписи чужой правки.
func (r *wishlistRepo) UpdateBlocks(ctx context.Context, id uuid.UUID, blocks []entity.Block, blocksVersion int, expectedUpdatedAt time.Time) (bool, error) {
	updates := map[string]interface{}{
		"blocks":         toBlocksJSON(blocks),
		"blocks_version": blocksVersion,
	}

	query := r.db.WithContext(ctx).Model(&WishlistModel{}).Where("id = ?", id)
	if !expectedUpdatedAt.IsZero() {
		// Postgres хранит микросекунды, клиент присылает их же в RFC3339 —
		// сравнение точное, поэтому округлять ничего не нужно.
		query = query.Where("updated_at = ?", expectedUpdatedAt)
	}

	result := query.Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("wishlistRepo.UpdateBlocks: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *wishlistRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&WishlistModel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
