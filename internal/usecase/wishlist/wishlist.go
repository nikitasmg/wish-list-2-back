package wishlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/internal/usecase/systemtemplate"
	"main/pkg/imagefile"
	minioPkg "main/pkg/minio"
	"main/pkg/shortid"
)

type wishlistUseCase struct {
	wishlistRepo repo.WishlistRepo
	fileStorage  minioPkg.FileStorage
}

func New(wishlistRepo repo.WishlistRepo, fileStorage minioPkg.FileStorage) usecase.WishlistUseCase {
	return &wishlistUseCase{
		wishlistRepo: wishlistRepo,
		fileStorage:  fileStorage,
	}
}

func (uc *wishlistUseCase) Create(ctx context.Context, userID uuid.UUID, input usecase.CreateWishlistInput) (entity.Wishlist, error) {
	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}
	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}
	if err := validateCustomScheme(input.ColorScheme, input.CustomScheme); err != nil {
		return entity.Wishlist{}, err
	}

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Wishlist{}, err
	}

	sid, err := uc.generateUniqueShortID(ctx)
	if err != nil {
		return entity.Wishlist{}, err
	}

	w := entity.Wishlist{
		ID:          uuid.New(),
		UserID:      userID,
		ShortID:     sid,
		Title:       input.Title,
		Description: input.Description,
		Cover:       coverURL,
		Settings: entity.Settings{
			ColorScheme:          input.ColorScheme,
			ShowGiftAvailability: input.ShowGiftAvailability,
			PresentsLayout:       input.PresentsLayout,
			CustomScheme:         input.CustomScheme,
		},
		Location: entity.Location{
			Name: input.LocationName,
			Link: input.LocationLink,
			Time: input.LocationTime,
		},
		EventDate:     input.EventDate,
		Occasion:      input.Occasion,
		BlocksVersion: entity.BlocksVersionCurrent,
		PresentsCount: 0,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) CreateConstructor(ctx context.Context, userID uuid.UUID, input usecase.CreateConstructorInput) (entity.Wishlist, error) {
	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}
	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}
	if err := validateNewBlocks(input.Blocks); err != nil {
		return entity.Wishlist{}, err
	}
	if err := validateCustomScheme(input.ColorScheme, input.CustomScheme); err != nil {
		return entity.Wishlist{}, err
	}

	sid, err := uc.generateUniqueShortID(ctx)
	if err != nil {
		return entity.Wishlist{}, err
	}

	w := entity.Wishlist{
		ID:          uuid.New(),
		UserID:      userID,
		ShortID:     sid,
		Title:       input.Title,
		Description: input.Description,
		Cover:       input.CoverURL,
		Settings: entity.Settings{
			ColorScheme:          input.ColorScheme,
			ShowGiftAvailability: input.ShowGiftAvailability,
			PresentsLayout:       input.PresentsLayout,
			CustomScheme:         input.CustomScheme,
		},
		Location: entity.Location{
			Name: input.LocationName,
			Link: input.LocationLink,
			Time: input.LocationTime,
		},
		EventDate:     input.EventDate,
		Occasion:      input.Occasion,
		Blocks:        ensureBlockIDs(input.Blocks),
		BlocksVersion: entity.BlocksVersionCurrent,
		PresentsCount: 0,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create constructor wishlist: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) CreateFromSystemTemplate(ctx context.Context, userID uuid.UUID, input usecase.CreateFromSystemTemplateInput) (entity.Wishlist, error) {
	tpl, err := systemtemplate.Get(input.TemplateID)
	if err != nil {
		return entity.Wishlist{}, err
	}

	count, err := uc.wishlistRepo.CountByUserID(ctx, userID)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("count wishlists: %w", err)
	}
	if count >= usecase.MaxWishlistsPerUser {
		return entity.Wishlist{}, errors.New("достигнут лимит вишлистов (20)")
	}
	title := input.Title
	if title == "" {
		title = tpl.SampleTitle
	}

	if err := validateWishlistFields(title, "", "", "", ""); err != nil {
		return entity.Wishlist{}, err
	}

	// Копия: блоки шаблона общие для всех пользователей, и правка одного
	// вишлиста не должна задеть тех, кто создаётся следом.
	blocks := make([]entity.Block, len(tpl.Blocks))
	copy(blocks, tpl.Blocks)

	// Название с обложки должно совпадать с названием вишлиста — иначе человек
	// вводит «Маше — 30!», а на странице остаётся «Ане — 28» из шаблона.
	for i := range blocks {
		if blocks[i].Type == "cover" {
			blocks[i].Title = title
		}
	}

	sid, err := uc.generateUniqueShortID(ctx)
	if err != nil {
		return entity.Wishlist{}, err
	}

	w := entity.Wishlist{
		ID:      uuid.New(),
		UserID:  userID,
		ShortID: sid,
		Title:   title,
		Settings: entity.Settings{
			ColorScheme:          tpl.ColorScheme,
			ShowGiftAvailability: true,
			PresentsLayout:       "list",
		},
		EventDate:     input.EventDate,
		Occasion:      tpl.Occasion,
		Blocks:        ensureBlockIDs(blocks),
		BlocksVersion: entity.BlocksVersionCurrent,
	}

	if err := uc.wishlistRepo.Create(ctx, w); err != nil {
		return entity.Wishlist{}, fmt.Errorf("create wishlist from template: %w", err)
	}

	return w, nil
}

func (uc *wishlistUseCase) GetByID(ctx context.Context, id uuid.UUID) (entity.Wishlist, error) {
	return uc.wishlistRepo.GetByID(ctx, id)
}

func (uc *wishlistUseCase) GetByShortID(ctx context.Context, shortID string, guestID uuid.UUID) (entity.Wishlist, error) {
	w, err := uc.wishlistRepo.GetByShortID(ctx, shortID)
	if err != nil {
		return entity.Wishlist{}, err
	}

	if guestID != uuid.Nil {
		// Просмотр не должен ронять страницу: счётчик в кабинете — не та вещь,
		// ради которой гость заслуживает пятисотку.
		if err := uc.wishlistRepo.RegisterView(ctx, w.ID, guestID); err != nil {
			log.Printf("register view for wishlist %s: %v", w.ID, err)
		}
	}

	w.Blocks = sanitizeBlocksForGuest(w.Blocks, time.Now())
	return w, nil
}

func (uc *wishlistUseCase) GetAllByUser(ctx context.Context, userID uuid.UUID) ([]entity.Wishlist, error) {
	wishlists, err := uc.wishlistRepo.GetAllByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	reserved, err := uc.wishlistRepo.ReservedCountsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reserved counts: %w", err)
	}
	for i := range wishlists {
		wishlists[i].ReservedCount = reserved[wishlists[i].ID]
	}

	return wishlists, nil
}

// sanitizeBlocksForGuest готовит блоки к публичной выдаче.
//
// Скрытые блоки уходят целиком, а у нераскрытых секретов остаётся только тип и
// дата раскрытия: прятать их на фронте бесполезно — содержимое всё равно видно
// в ответе API, и весь смысл «секрета до даты» пропадает.
func sanitizeBlocksForGuest(blocks []entity.Block, now time.Time) []entity.Block {
	if blocks == nil {
		return nil
	}

	visible := make([]entity.Block, 0, len(blocks))
	for _, b := range blocks {
		if b.Hidden {
			continue
		}
		if b.IsSecret(now) {
			b = entity.Block{
				// ID остаётся: он не выдаёт содержимого, а фронту нужен, чтобы
				// отличить один таймер от другого.
				ID:       b.ID,
				Type:     b.Type,
				Row:      b.Row,
				Col:      b.Col,
				ColSpan:  b.ColSpan,
				RevealAt: b.RevealAt,
				Data:     json.RawMessage("{}"),
			}
		}
		visible = append(visible, b)
	}
	return visible
}

// assertOwner — вишлист существует и принадлежит этому пользователю.
//
// JWT говорит, кто пришёл, но не чей вишлист он открыл: без этой проверки
// любой залогиненный человек мог бы править и удалять чужие страницы, зная
// только UUID из публичной ссылки.
func (uc *wishlistUseCase) assertOwner(ctx context.Context, userID, id uuid.UUID) (entity.Wishlist, error) {
	w, err := uc.wishlistRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
	}
	if w.UserID != userID {
		return entity.Wishlist{}, usecase.ErrForbidden
	}
	return w, nil
}

func (uc *wishlistUseCase) Update(ctx context.Context, userID, id uuid.UUID, input usecase.CreateWishlistInput, expectedUpdatedAt time.Time) (entity.Wishlist, error) {
	w, err := uc.assertOwner(ctx, userID, id)
	if err != nil {
		return entity.Wishlist{}, err
	}

	if err := validateWishlistFields(input.Title, input.Description, input.LocationName, input.LocationLink, input.CoverURL); err != nil {
		return entity.Wishlist{}, err
	}
	if err := validateCustomScheme(input.ColorScheme, input.CustomScheme); err != nil {
		return entity.Wishlist{}, err
	}

	w.Title = input.Title
	w.Description = input.Description
	w.Settings = entity.Settings{
		ColorScheme:          input.ColorScheme,
		ShowGiftAvailability: input.ShowGiftAvailability,
		PresentsLayout:       input.PresentsLayout,
		CustomScheme:         input.CustomScheme,
	}
	w.Location = entity.Location{
		Name: input.LocationName,
		Link: input.LocationLink,
		Time: input.LocationTime,
	}
	w.EventDate = input.EventDate
	w.Occasion = input.Occasion

	coverURL, err := uc.resolveCover(input.CoverData, input.CoverName, input.CoverURL)
	if err != nil {
		return entity.Wishlist{}, err
	}
	w.Cover = coverURL

	saved, updated, err := uc.wishlistRepo.UpdateMetadata(ctx, id, w, expectedUpdatedAt)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("update wishlist: %w", err)
	}
	if !updated {
		// Версия разошлась. Отдаём актуальный вишлист вместе с ошибкой: клиенту
		// есть что показать и с чем слить свою правку.
		current, err := uc.wishlistRepo.GetByID(ctx, id)
		if err != nil {
			return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
		}
		return current, usecase.ErrVersionConflict
	}

	return saved, nil
}

func (uc *wishlistUseCase) UpdateBlocks(ctx context.Context, userID, id uuid.UUID, blocks []entity.Block, expectedUpdatedAt time.Time) (entity.Wishlist, error) {
	if _, err := uc.assertOwner(ctx, userID, id); err != nil {
		return entity.Wishlist{}, err
	}

	// Legacy-типы здесь допускаются намеренно: вишлист формата v1 можно открыть
	// и сохранить, не пересобирая его целиком. Запрещено только создавать из них
	// новые — см. CreateConstructor.
	if err := validateBlocks(blocks); err != nil {
		return entity.Wishlist{}, err
	}
	blocks = ensureBlockIDs(blocks)

	updated, err := uc.wishlistRepo.UpdateBlocks(ctx, id, blocks, entity.BlocksVersionCurrent, expectedUpdatedAt)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("update blocks: %w", err)
	}

	w, err := uc.wishlistRepo.GetByID(ctx, id)
	if err != nil {
		return entity.Wishlist{}, fmt.Errorf("wishlist not found: %w", err)
	}

	if !updated {
		// Отдаём актуальный вишлист вместе с ошибкой: клиенту есть что показать
		// и с чем слить свою правку.
		return w, usecase.ErrVersionConflict
	}

	return w, nil
}

func (uc *wishlistUseCase) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if _, err := uc.assertOwner(ctx, userID, id); err != nil {
		return err
	}
	return uc.wishlistRepo.Delete(ctx, id)
}

// ensureBlockIDs выдаёт блокам стабильные идентификаторы.
//
// Позиция для этого не годится: ответы гостей, голоса и треки привязаны к
// конкретному блоку, а позиция меняется при каждой перестановке — после
// переноса блока вверх голоса уехали бы к соседу.
func ensureBlockIDs(blocks []entity.Block) []entity.Block {
	for i := range blocks {
		if blocks[i].ID == "" {
			blocks[i].ID = uuid.NewString()
		}
	}
	return blocks
}

// resolveCover — возвращает URL обложки: загружает файл в MinIO или возвращает URL as-is
func (uc *wishlistUseCase) resolveCover(data []byte, name, url string) (string, error) {
	if len(data) > 0 {
		if err := imagefile.Validate(data); err != nil {
			return "", err
		}
		uploaded, err := uc.fileStorage.Upload(name, data)
		if err != nil {
			return "", fmt.Errorf("upload cover: %w", err)
		}
		return uploaded, nil
	}
	return url, nil
}

// generateUniqueShortID — генерирует уникальный короткий ID с повторной попыткой при коллизии
func (uc *wishlistUseCase) generateUniqueShortID(ctx context.Context) (string, error) {
	for i := 0; i < 5; i++ {
		sid, err := shortid.Generate()
		if err != nil {
			return "", fmt.Errorf("generate short id: %w", err)
		}
		_, err = uc.wishlistRepo.GetByShortID(ctx, sid)
		if err != nil {
			// не найден — значит уникальный
			return sid, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique short id after 5 attempts")
}

func validateWishlistFields(title, description, locationName, locationLink, coverURL string) error {
	if len([]rune(title)) > usecase.MaxTitleLen {
		return fmt.Errorf("title exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len([]rune(description)) > usecase.MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", usecase.MaxDescriptionLen)
	}
	if len([]rune(locationName)) > usecase.MaxTitleLen {
		return fmt.Errorf("location name exceeds maximum length of %d characters", usecase.MaxTitleLen)
	}
	if len(locationLink) > usecase.MaxURLLen {
		return fmt.Errorf("location link exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	if len(coverURL) > usecase.MaxURLLen {
		return fmt.Errorf("cover URL exceeds maximum URL length of %d", usecase.MaxURLLen)
	}
	return nil
}

// validateBlocks — проверяет типы блоков
func validateBlocks(blocks []entity.Block) error {
	if len(blocks) > usecase.MaxBlocksPerWishlist {
		return fmt.Errorf("too many blocks: max %d", usecase.MaxBlocksPerWishlist)
	}
	for i, b := range blocks {
		if !entity.ValidBlockTypes[b.Type] {
			return fmt.Errorf("block[%d]: unknown type %q", i, b.Type)
		}
		if b.ColSpan > 2 {
			return fmt.Errorf("block[%d]: colSpan %d exceeds maximum of 2", i, b.ColSpan)
		}
		if b.Row < 0 {
			return fmt.Errorf("block[%d]: row must be >= 0", i)
		}
		if b.Col < 0 || b.Col > 1 {
			return fmt.Errorf("block[%d]: col must be 0 or 1", i)
		}
		if len(b.Data) > usecase.MaxBlockDataSize {
			return fmt.Errorf("block[%d]: data too large (max %d bytes)", i, usecase.MaxBlockDataSize)
		}
		if err := validateBlockData(i, b.Type, b.Data); err != nil {
			return err
		}
	}
	return nil
}

// validateBlockData checks type-specific content constraints.
// b.Data is guaranteed non-nil at this point (nil-substituted to "{}" in HTTP layer).
func validateBlockData(idx int, blockType string, data json.RawMessage) error {
	switch blockType {
	case "text", "quote":
		var d struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil // malformed but not oversized — let DB store it, frontend owns schema
		}
		if len([]rune(d.Content)) > usecase.MaxBlockTextField {
			return fmt.Errorf("block[%d]: content exceeds maximum length of %d characters", idx, usecase.MaxBlockTextField)
		}

	case "checklist":
		var d struct {
			Items []struct {
				Text string `json:"text"`
			} `json:"items"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Items) > 100 {
			return fmt.Errorf("block[%d]: checklist exceeds maximum of 100 items", idx)
		}
		for j, item := range d.Items {
			if len([]rune(item.Text)) > 500 {
				return fmt.Errorf("block[%d]: checklist item[%d] text exceeds 500 characters", idx, j)
			}
		}

	case "image", "text_image", "video":
		var d struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.URL) > usecase.MaxURLLen {
			return fmt.Errorf("block[%d]: url exceeds maximum length of %d", idx, usecase.MaxURLLen)
		}

	case "gallery":
		var d struct {
			Images []string `json:"images"`
		}
		if err := json.Unmarshal(data, &d); err != nil {
			return nil
		}
		if len(d.Images) > 50 {
			return fmt.Errorf("block[%d]: gallery exceeds maximum of 50 images", idx)
		}
		for j, u := range d.Images {
			if len(u) > usecase.MaxURLLen {
				return fmt.Errorf("block[%d]: gallery image[%d] URL exceeds maximum length", idx, j)
			}
		}
	}
	return nil
}

// validateNewBlocks — то же плюс запрет на типы формата v1. Применяется только
// при создании: старый вишлист должен оставаться сохраняемым.
func validateNewBlocks(blocks []entity.Block) error {
	if err := validateBlocks(blocks); err != nil {
		return err
	}
	for i, b := range blocks {
		if entity.LegacyBlockTypes[b.Type] {
			return fmt.Errorf("block[%d]: тип %q больше не используется — его заменили list и media", i, b.Type)
		}
	}
	return nil
}

// validateCustomScheme — «своя схема» приходит прямо из формы, поэтому база и
// акцент проверяются: иначе в настройки уедет произвольная строка, которую
// фронт потом подставит в CSS.
func validateCustomScheme(colorScheme string, scheme *entity.CustomScheme) error {
	if scheme == nil {
		return nil
	}
	if colorScheme != CustomColorScheme {
		return fmt.Errorf("customScheme задаётся только при colorScheme = %q", CustomColorScheme)
	}
	if scheme.Base != "dark" && scheme.Base != "light" {
		return fmt.Errorf("customScheme.base должен быть dark или light")
	}
	if !hexColor.MatchString(scheme.Accent) {
		return fmt.Errorf("customScheme.accent должен быть цветом вида #RRGGBB")
	}
	return nil
}

// CustomColorScheme — значение colorScheme, при котором цвета берутся из
// customScheme, а не из готового набора схем.
const CustomColorScheme = "custom"

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
