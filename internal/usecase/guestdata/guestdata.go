// Package guestdata — данные, которые оставляют гости публичной страницы:
// ответы на приглашение, голоса, треки и поздравления.
//
// Четыре блока в одном пакете, а не в четырёх: у них общий способ опознать
// гостя, общая привязка к блоку и общие правила — кто что видит и сколько раз
// может отправить. Разнести их значило бы четыре раза написать одно и то же.
package guestdata

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
)

type guestDataUseCase struct {
	guestRepo    repo.GuestDataRepo
	wishlistRepo repo.WishlistRepo
}

func New(guestRepo repo.GuestDataRepo, wishlistRepo repo.WishlistRepo) usecase.GuestDataUseCase {
	return &guestDataUseCase{guestRepo: guestRepo, wishlistRepo: wishlistRepo}
}

// assertOwner — сводку ответов и модерацию гостевой книги видит только хозяин
// вишлиста. Проверка здесь, а не в хендлере: её нельзя забыть в новом методе.
func (uc *guestDataUseCase) assertOwner(ctx context.Context, userID, wishlistID uuid.UUID) error {
	w, err := uc.wishlistRepo.GetByID(ctx, wishlistID)
	if err != nil {
		return fmt.Errorf("wishlist not found: %w", err)
	}
	if w.UserID != userID {
		return usecase.ErrForbidden
	}
	return nil
}

// RSVP

func (uc *guestDataUseCase) SubmitRSVP(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input usecase.RSVPInput) (entity.RSVPResponse, error) {
	if err := requireGuest(guestID); err != nil {
		return entity.RSVPResponse{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return entity.RSVPResponse{}, fmt.Errorf("укажите, как вас записать")
	}
	if err := checkLen("имя", name, entity.MaxGuestNameLen); err != nil {
		return entity.RSVPResponse{}, err
	}
	if err := checkLen("комментарий", input.Comment, entity.MaxRSVPCommentLen); err != nil {
		return entity.RSVPResponse{}, err
	}
	if err := checkLen("пожелания по меню", input.Menu, entity.MaxRSVPMenuLen); err != nil {
		return entity.RSVPResponse{}, err
	}
	if input.PlusOne < 0 || input.PlusOne > entity.MaxPlusOne {
		return entity.RSVPResponse{}, fmt.Errorf("с собой можно привести не больше %d человек", entity.MaxPlusOne)
	}
	if input.Kids < 0 || input.Kids > entity.MaxKids {
		return entity.RSVPResponse{}, fmt.Errorf("детей можно указать не больше %d", entity.MaxKids)
	}

	response := entity.RSVPResponse{
		ID:         uuid.New(),
		WishlistID: wishlistID,
		BlockID:    blockID,
		GuestID:    guestID,
		Name:       name,
		Going:      input.Going,
		PlusOne:    input.PlusOne,
		Kids:       input.Kids,
		Menu:       strings.TrimSpace(input.Menu),
		Transfer:   input.Transfer,
		Comment:    strings.TrimSpace(input.Comment),
		Mine:       true,
	}

	// «Не приду» обнуляет спутников: иначе в сводке окажутся плюс-один у того,
	// кто не придёт.
	if !response.Going {
		response.PlusOne = 0
		response.Kids = 0
		response.Transfer = false
	}

	if err := uc.guestRepo.UpsertRSVP(ctx, response); err != nil {
		return entity.RSVPResponse{}, err
	}
	return response, nil
}

func (uc *guestDataUseCase) MyRSVP(ctx context.Context, blockID string, guestID uuid.UUID) (*entity.RSVPResponse, error) {
	if guestID == uuid.Nil {
		return nil, nil
	}

	responses, err := uc.guestRepo.ListRSVP(ctx, blockID)
	if err != nil {
		return nil, err
	}
	for _, r := range responses {
		if r.GuestID == guestID {
			r.Mine = true
			return &r, nil
		}
	}
	return nil, nil
}

func (uc *guestDataUseCase) OwnerRSVPSummary(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) (entity.RSVPSummary, error) {
	if err := uc.assertOwner(ctx, userID, wishlistID); err != nil {
		return entity.RSVPSummary{}, err
	}

	responses, err := uc.guestRepo.ListRSVP(ctx, blockID)
	if err != nil {
		return entity.RSVPSummary{}, err
	}

	summary := entity.RSVPSummary{Responses: responses}
	for _, r := range responses {
		if !r.Going {
			summary.NotGoing++
			continue
		}
		summary.Going++
		summary.PlusOnes += r.PlusOne
		summary.Kids += r.Kids
		if r.Transfer {
			summary.Transfer++
		}
	}
	// Сколько человек придёт на самом деле — то число, ради которого
	// организатор и открывает сводку.
	summary.TotalPeople = summary.Going + summary.PlusOnes + summary.Kids

	return summary, nil
}

// Голосование

func (uc *guestDataUseCase) Vote(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, option int) (entity.PollResults, error) {
	if err := requireGuest(guestID); err != nil {
		return entity.PollResults{}, err
	}
	if option < 0 {
		return entity.PollResults{}, fmt.Errorf("не выбран вариант ответа")
	}

	if err := uc.guestRepo.UpsertPollVote(ctx, wishlistID, blockID, guestID, option); err != nil {
		return entity.PollResults{}, err
	}
	return uc.PollResults(ctx, blockID, guestID)
}

func (uc *guestDataUseCase) PollResults(ctx context.Context, blockID string, guestID uuid.UUID) (entity.PollResults, error) {
	counts, mine, err := uc.guestRepo.CountPollVotes(ctx, blockID, guestID)
	if err != nil {
		return entity.PollResults{}, err
	}

	// Варианты живут в data блока, здесь их нет — отдаём плотный массив по
	// максимальному индексу, фронт сопоставит его со своим списком.
	size := 0
	for index := range counts {
		if index+1 > size {
			size = index + 1
		}
	}

	results := entity.PollResults{Votes: make([]int, size), MyVote: mine}
	for index, count := range counts {
		results.Votes[index] = count
		results.Total += count
	}
	return results, nil
}

// Плейлист

func (uc *guestDataUseCase) SuggestTrack(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, title string) ([]entity.PlaylistTrack, error) {
	if err := requireGuest(guestID); err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("напишите название трека")
	}
	if err := checkLen("название трека", title, entity.MaxTrackTitleLen); err != nil {
		return nil, err
	}

	// Ограничение на гостя, а не на блок: иначе один человек забьёт плейлист
	// целиком, и остальным будет нечего предложить.
	count, err := uc.guestRepo.CountTracksByGuest(ctx, blockID, guestID)
	if err != nil {
		return nil, err
	}
	if count >= entity.MaxTracksPerGuest {
		return nil, fmt.Errorf("можно предложить не больше %d треков", entity.MaxTracksPerGuest)
	}

	if _, err := uc.guestRepo.CreateTrack(ctx, wishlistID, blockID, guestID, title); err != nil {
		return nil, err
	}
	return uc.guestRepo.ListTracks(ctx, blockID, guestID)
}

func (uc *guestDataUseCase) Tracks(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	return uc.guestRepo.ListTracks(ctx, blockID, guestID)
}

func (uc *guestDataUseCase) ToggleTrackVote(ctx context.Context, trackID, guestID uuid.UUID) ([]entity.PlaylistTrack, error) {
	if err := requireGuest(guestID); err != nil {
		return nil, err
	}

	_, blockID, err := uc.guestRepo.TrackBlockID(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("трек не найден: %w", err)
	}
	if _, err := uc.guestRepo.ToggleTrackVote(ctx, trackID, guestID); err != nil {
		return nil, err
	}
	return uc.guestRepo.ListTracks(ctx, blockID, guestID)
}

// Гостевая книга

func (uc *guestDataUseCase) AddGuestbookEntry(ctx context.Context, wishlistID uuid.UUID, blockID string, guestID uuid.UUID, input usecase.GuestbookInput) (entity.GuestbookEntry, error) {
	if err := requireGuest(guestID); err != nil {
		return entity.GuestbookEntry{}, err
	}

	name := strings.TrimSpace(input.Name)
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return entity.GuestbookEntry{}, fmt.Errorf("напишите поздравление")
	}
	if err := checkLen("имя", name, entity.MaxGuestNameLen); err != nil {
		return entity.GuestbookEntry{}, err
	}
	if err := checkLen("поздравление", text, entity.MaxGuestbookTextLen); err != nil {
		return entity.GuestbookEntry{}, err
	}

	count, err := uc.guestRepo.CountGuestbookByGuest(ctx, blockID, guestID)
	if err != nil {
		return entity.GuestbookEntry{}, err
	}
	if count >= entity.MaxGuestbookPerGuest {
		return entity.GuestbookEntry{}, fmt.Errorf("можно оставить не больше %d поздравлений", entity.MaxGuestbookPerGuest)
	}

	entry := entity.GuestbookEntry{
		ID:       uuid.New(),
		Name:     name,
		Text:     text,
		PhotoURL: input.PhotoURL,
		Mine:     true,
	}
	if err := uc.guestRepo.CreateGuestbookEntry(ctx, entry, wishlistID, blockID, guestID); err != nil {
		return entity.GuestbookEntry{}, err
	}
	return entry, nil
}

func (uc *guestDataUseCase) Guestbook(ctx context.Context, blockID string, guestID uuid.UUID) ([]entity.GuestbookEntry, error) {
	return uc.guestRepo.ListGuestbook(ctx, blockID, guestID, false)
}

func (uc *guestDataUseCase) OwnerGuestbook(ctx context.Context, userID, wishlistID uuid.UUID, blockID string) ([]entity.GuestbookEntry, error) {
	if err := uc.assertOwner(ctx, userID, wishlistID); err != nil {
		return nil, err
	}
	return uc.guestRepo.ListGuestbook(ctx, blockID, uuid.Nil, true)
}

func (uc *guestDataUseCase) OwnerSetGuestbookHidden(ctx context.Context, userID, entryID uuid.UUID, hidden bool) error {
	wishlistID, err := uc.guestRepo.GuestbookEntryWishlist(ctx, entryID)
	if err != nil {
		return fmt.Errorf("запись не найдена: %w", err)
	}
	if err := uc.assertOwner(ctx, userID, wishlistID); err != nil {
		return err
	}
	return uc.guestRepo.SetGuestbookHidden(ctx, entryID, hidden)
}

// requireGuest — без куки непонятно, чей это ответ и можно ли его потом
// поправить, поэтому лучше честно попросить включить куки.
func requireGuest(guestID uuid.UUID) error {
	if guestID == uuid.Nil {
		return fmt.Errorf("не удалось определить гостя — включите куки и обновите страницу")
	}
	return nil
}

// checkLen считает символы, а не байты: тексты пишут по-русски.
func checkLen(field, value string, max int) error {
	if utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s длиннее %d символов", field, max)
	}
	return nil
}
