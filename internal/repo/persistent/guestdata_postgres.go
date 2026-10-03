package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"main/internal/entity"
)

type guestDataRepo struct {
	db *gorm.DB
}

func NewGuestDataRepo(db *gorm.DB) *guestDataRepo {
	return &guestDataRepo{db: db}
}

// RSVP

// UpsertRSVP — повторная отправка обновляет собственный ответ гостя.
// Дедупликация держится уникальным индексом, а не чтением перед записью:
// два быстрых нажатия «Приду» иначе создали бы два ответа.
func (r *guestDataRepo) UpsertRSVP(ctx context.Context, response entity.RSVPResponse) error {
	m := RSVPResponseModel{
		ID:         response.ID,
		WishlistID: response.WishlistID,
		BlockID:    response.BlockID,
		GuestID:    response.GuestID,
		Name:       response.Name,
		Going:      response.Going,
		PlusOne:    response.PlusOne,
		Kids:       response.Kids,
		Menu:       response.Menu,
		Transfer:   response.Transfer,
		Comment:    response.Comment,
		Answers:    AnswersJSON(response.Answers),
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "block_id"}, {Name: "guest_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "going", "plus_one", "kids", "menu", "transfer", "comment", "answers", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("guestDataRepo.UpsertRSVP: %w", err)
	}
	return nil
}

func (r *guestDataRepo) ListRSVP(ctx context.Context, blockID string) ([]entity.RSVPResponse, error) {
	var models []RSVPResponseModel
	if err := r.db.WithContext(ctx).
		Where("block_id = ?", blockID).
		Order("created_at").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("guestDataRepo.ListRSVP: %w", err)
	}

	out := make([]entity.RSVPResponse, len(models))
	for i, m := range models {
		out[i] = entity.RSVPResponse{
			ID: m.ID, WishlistID: m.WishlistID, BlockID: m.BlockID, GuestID: m.GuestID,
			Name: m.Name, Going: m.Going, PlusOne: m.PlusOne, Kids: m.Kids,
			Menu: m.Menu, Transfer: m.Transfer, Comment: m.Comment, Answers: map[string]string(m.Answers),
			CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		}
	}
	return out, nil
}

// Poll

// ReplacePollChoices — выбор гостя целиком, одной транзакцией: при
// переголосовании старые варианты снимаются, а не копятся рядом с новыми.
func (r *guestDataRepo) ReplacePollChoices(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, optionIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_id = ? AND guest_id = ?", blockID, guestID).
			Delete(&PollChoiceModel{}).Error; err != nil {
			return fmt.Errorf("guestDataRepo.ReplacePollChoices: delete: %w", err)
		}
		for _, id := range optionIDs {
			choice := PollChoiceModel{WishlistID: wishlistID, BlockID: blockID, GuestID: guestID, OptionID: id}
			if err := tx.Create(&choice).Error; err != nil {
				return fmt.Errorf("guestDataRepo.ReplacePollChoices: insert: %w", err)
			}
		}
		return nil
	})
}

// CountPollChoices — сколько голосов у каждого варианта и что выбрал этот гость.
func (r *guestDataRepo) CountPollChoices(ctx context.Context, blockID string, guestID uuid.UUID) (map[string]int, []string, error) {
	var rows []struct {
		OptionID string
		Count    int
	}
	if err := r.db.WithContext(ctx).Model(&PollChoiceModel{}).
		Select("option_id, count(*) as count").
		Where("block_id = ?", blockID).
		Group("option_id").
		Scan(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("guestDataRepo.CountPollChoices: %w", err)
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.OptionID] = row.Count
	}

	var mine []string
	if guestID != uuid.Nil {
		if err := r.db.WithContext(ctx).Model(&PollChoiceModel{}).
			Where("block_id = ? AND guest_id = ?", blockID, guestID).
			Order("created_at").
			Pluck("option_id", &mine).Error; err != nil {
			return nil, nil, fmt.Errorf("guestDataRepo.CountPollChoices: own: %w", err)
		}
	}
	return counts, mine, nil
}

func (r *guestDataRepo) CreatePollOption(ctx context.Context, option entity.PollGuestOption) error {
	m := PollGuestOptionModel{
		ID: option.ID, WishlistID: option.WishlistID, BlockID: option.BlockID,
		GuestID: option.GuestID, Text: option.Text,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("guestDataRepo.CreatePollOption: %w", err)
	}
	return nil
}

func (r *guestDataRepo) CountPollOptionsByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&PollGuestOptionModel{}).
		Where("block_id = ? AND guest_id = ?", blockID, guestID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("guestDataRepo.CountPollOptionsByGuest: %w", err)
	}
	return count, nil
}

func (r *guestDataRepo) ListPollOptions(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.PollOption, error) {
	query := r.db.WithContext(ctx).Where("block_id = ?", blockID)
	if !includeHidden {
		query = query.Where("hidden = ?", false)
	}
	var models []PollGuestOptionModel
	if err := query.Order("created_at").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("guestDataRepo.ListPollOptions: %w", err)
	}
	out := make([]entity.PollOption, len(models))
	for i, m := range models {
		out[i] = entity.PollOption{ID: m.ID.String(), Text: m.Text, Mine: m.GuestID == guestID, Hidden: m.Hidden}
	}
	return out, nil
}

func (r *guestDataRepo) PollOptionWishlist(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error) {
	var m PollGuestOptionModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", optionID).Error; err != nil {
		return uuid.Nil, fmt.Errorf("guestDataRepo.PollOptionWishlist: %w", err)
	}
	return m.WishlistID, nil
}

func (r *guestDataRepo) SetPollOptionHidden(ctx context.Context, optionID uuid.UUID, hidden bool) error {
	if err := r.db.WithContext(ctx).Model(&PollGuestOptionModel{}).
		Where("id = ?", optionID).Update("hidden", hidden).Error; err != nil {
		return fmt.Errorf("guestDataRepo.SetPollOptionHidden: %w", err)
	}
	return nil
}

// Playlist

func (r *guestDataRepo) CountTracksByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&PlaylistTrackModel{}).
		Where("block_id = ? AND guest_id = ?", blockID, guestID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("guestDataRepo.CountTracksByGuest: %w", err)
	}
	return count, nil
}

func (r *guestDataRepo) CreateTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) (uuid.UUID, error) {
	m := PlaylistTrackModel{
		ID: uuid.New(), WishlistID: wishlistID, BlockID: blockID, GuestID: guestID, Title: title,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return uuid.Nil, fmt.Errorf("guestDataRepo.CreateTrack: %w", err)
	}
	return m.ID, nil
}

// ListTracks — треки с числом голосов, сразу отсортированные так, как их
// показывает страница: считать и сортировать на стороне Go означало бы тянуть
// все голоса в память.
func (r *guestDataRepo) ListTracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	var rows []struct {
		ID        uuid.UUID
		Title     string
		GuestID   uuid.UUID
		CreatedAt time.Time
		Votes     int
		VotedByMe bool
	}

	err := r.db.WithContext(ctx).Model(&PlaylistTrackModel{}).
		Select(`playlist_tracks.id,
			playlist_tracks.title,
			playlist_tracks.guest_id,
			playlist_tracks.created_at,
			count(playlist_votes.track_id) as votes,
			bool_or(playlist_votes.guest_id = ?) as voted_by_me`, guestID).
		Joins("left join playlist_votes on playlist_votes.track_id = playlist_tracks.id").
		Where("playlist_tracks.block_id = ?", blockID).
		Group("playlist_tracks.id").
		Order("votes desc, playlist_tracks.created_at").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("guestDataRepo.ListTracks: %w", err)
	}

	out := make([]entity.PlaylistTrack, len(rows))
	for i, row := range rows {
		out[i] = entity.PlaylistTrack{
			ID:        row.ID,
			Title:     row.Title,
			Votes:     row.Votes,
			VotedByMe: row.VotedByMe,
			Mine:      row.GuestID == guestID,
			CreatedAt: row.CreatedAt,
		}
	}
	return out, nil
}

func (r *guestDataRepo) TrackBlockID(ctx context.Context, trackID uuid.UUID) (uuid.UUID, string, error) {
	var m PlaylistTrackModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", trackID).Error; err != nil {
		return uuid.Nil, "", fmt.Errorf("guestDataRepo.TrackBlockID: %w", err)
	}
	return m.WishlistID, m.BlockID, nil
}

// ToggleTrackVote — повторное нажатие снимает свой голос. Возвращает, стоит ли
// голос после операции.
func (r *guestDataRepo) ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("track_id = ? AND guest_id = ?", trackID, guestID).
		Delete(&PlaylistVoteModel{})
	if result.Error != nil {
		return false, fmt.Errorf("guestDataRepo.ToggleTrackVote: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		return false, nil
	}

	vote := PlaylistVoteModel{TrackID: trackID, GuestID: guestID}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&vote).Error; err != nil {
		return false, fmt.Errorf("guestDataRepo.ToggleTrackVote: %w", err)
	}
	return true, nil
}

// Guestbook

func (r *guestDataRepo) CountGuestbookByGuest(ctx context.Context, blockID string, guestID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&GuestbookEntryModel{}).
		Where("block_id = ? AND guest_id = ?", blockID, guestID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("guestDataRepo.CountGuestbookByGuest: %w", err)
	}
	return count, nil
}

func (r *guestDataRepo) CreateGuestbookEntry(ctx context.Context, entry entity.GuestbookEntry, wishlistID uuid.UUID, blockID string, guestID uuid.UUID) error {
	m := GuestbookEntryModel{
		ID: entry.ID, WishlistID: wishlistID, BlockID: blockID, GuestID: guestID,
		Name: entry.Name, Text: entry.Text, PhotoURL: entry.PhotoURL,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("guestDataRepo.CreateGuestbookEntry: %w", err)
	}
	return nil
}

// ListGuestbook — записи блока. includeHidden=true только для владельца:
// гостю скрытые записи не отдаются вовсе, иначе «скрыть» ничего не скрывает.
func (r *guestDataRepo) ListGuestbook(ctx context.Context, blockID string, guestID uuid.UUID, includeHidden bool) ([]entity.GuestbookEntry, error) {
	query := r.db.WithContext(ctx).Model(&GuestbookEntryModel{}).Where("block_id = ?", blockID)
	if !includeHidden {
		query = query.Where("hidden = ?", false)
	}

	var models []GuestbookEntryModel
	if err := query.Order("created_at desc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("guestDataRepo.ListGuestbook: %w", err)
	}

	out := make([]entity.GuestbookEntry, len(models))
	for i, m := range models {
		out[i] = entity.GuestbookEntry{
			ID: m.ID, Name: m.Name, Text: m.Text, PhotoURL: m.PhotoURL,
			Hidden: m.Hidden, Mine: m.GuestID == guestID, CreatedAt: m.CreatedAt,
		}
	}
	return out, nil
}

func (r *guestDataRepo) GuestbookEntryWishlist(ctx context.Context, entryID uuid.UUID) (uuid.UUID, error) {
	var m GuestbookEntryModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", entryID).Error; err != nil {
		return uuid.Nil, fmt.Errorf("guestDataRepo.GuestbookEntryWishlist: %w", err)
	}
	return m.WishlistID, nil
}

func (r *guestDataRepo) SetGuestbookHidden(ctx context.Context, entryID uuid.UUID, hidden bool) error {
	result := r.db.WithContext(ctx).Model(&GuestbookEntryModel{}).
		Where("id = ?", entryID).
		Update("hidden", hidden)
	if result.Error != nil {
		return fmt.Errorf("guestDataRepo.SetGuestbookHidden: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("guestDataRepo.SetGuestbookHidden: запись не найдена")
	}
	return nil
}
