//go:build integration

package persistent_test

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"main/internal/entity"
	"main/internal/repo/persistent"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&persistent.UserModel{},
		&persistent.WishlistModel{},
		&persistent.PresentModel{},
	)
	require.NoError(t, err)

	return db
}

func TestUserRepo_CreateAndGet(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewUserRepo(db)

	user := entity.User{
		ID:       uuid.New(),
		Username: "alice",
		Password: "hashed",
	}

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)

	got, err := repo.GetByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.Equal(t, user.Username, got.Username)
}

func TestUserRepo_GetByUsername_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewUserRepo(db)

	_, err := repo.GetByUsername(context.Background(), "nobody")
	require.Error(t, err)
}

func TestWishlistRepo_CreateAndGetByID(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewWishlistRepo(db)

	sid := "abc-def-ghi"
	w := entity.Wishlist{
		ID:      uuid.New(),
		Title:   "My Wishlist",
		UserID:  uuid.New(),
		ShortID: sid,
		Settings: entity.Settings{
			ColorScheme:          "dark",
			ShowGiftAvailability: true,
		},
	}

	err := repo.Create(context.Background(), w)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), w.ID)
	require.NoError(t, err)
	assert.Equal(t, w.Title, got.Title)
	assert.Equal(t, sid, got.ShortID)
	assert.True(t, got.Settings.ShowGiftAvailability)
}

func TestWishlistRepo_GetByShortID_Found(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewWishlistRepo(db)

	sid := "xyz-123-abc"
	w := entity.Wishlist{ID: uuid.New(), Title: "Short", UserID: uuid.New(), ShortID: sid}
	require.NoError(t, repo.Create(context.Background(), w))

	got, err := repo.GetByShortID(context.Background(), sid)
	require.NoError(t, err)
	assert.Equal(t, w.ID, got.ID)
}

func TestWishlistRepo_GetByShortID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewWishlistRepo(db)

	_, err := repo.GetByShortID(context.Background(), "no-such-id")
	require.Error(t, err)
}

func TestWishlistRepo_IncrementPresentsCount(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewWishlistRepo(db)

	w := entity.Wishlist{ID: uuid.New(), Title: "Atomic", UserID: uuid.New()}
	require.NoError(t, repo.Create(context.Background(), w))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = repo.IncrementPresentsCount(context.Background(), w.ID)
		}()
	}
	wg.Wait()

	got, err := repo.GetByID(context.Background(), w.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(10), got.PresentsCount)
}

func TestWishlistRepo_DecrementPresentsCount_NoNegative(t *testing.T) {
	db := setupDB(t)
	repo := persistent.NewWishlistRepo(db)

	w := entity.Wishlist{ID: uuid.New(), Title: "NoNeg", UserID: uuid.New(), PresentsCount: 0}
	require.NoError(t, repo.Create(context.Background(), w))

	// Decrement at 0 should not go negative
	err := repo.DecrementPresentsCount(context.Background(), w.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(context.Background(), w.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(0), got.PresentsCount)
}

func TestPresentRepo_CreateAndGetAll(t *testing.T) {
	db := setupDB(t)
	wishlistRepo := persistent.NewWishlistRepo(db)
	presentRepo := persistent.NewPresentRepo(db)

	wid := uuid.New()
	w := entity.Wishlist{ID: wid, Title: "Gifts", UserID: uuid.New()}
	require.NoError(t, wishlistRepo.Create(context.Background(), w))

	p := entity.Present{
		ID:         uuid.New(),
		Title:      "Book",
		WishlistID: wid,
	}
	require.NoError(t, presentRepo.Create(context.Background(), p))

	presents, err := presentRepo.GetAllByWishlistID(context.Background(), wid)
	require.NoError(t, err)
	require.Len(t, presents, 1)
	assert.Equal(t, p.Title, presents[0].Title)
}

func TestPresentRepo_Delete(t *testing.T) {
	db := setupDB(t)
	wishlistRepo := persistent.NewWishlistRepo(db)
	presentRepo := persistent.NewPresentRepo(db)

	wid := uuid.New()
	w := entity.Wishlist{ID: wid, Title: "Gifts", UserID: uuid.New()}
	require.NoError(t, wishlistRepo.Create(context.Background(), w))

	pid := uuid.New()
	p := entity.Present{ID: pid, Title: "Book", WishlistID: wid}
	require.NoError(t, presentRepo.Create(context.Background(), p))

	err := presentRepo.Delete(context.Background(), pid)
	require.NoError(t, err)

	_, err = presentRepo.GetByID(context.Background(), pid)
	require.Error(t, err)
}
