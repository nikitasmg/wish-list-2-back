package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	tgverifier "github.com/electrofocus/telegram-auth-verifier"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"main/internal/entity"
	"main/internal/repo"
	"main/internal/usecase"
	"main/pkg/hasher"
)

type Claims struct {
	Username string    `json:"username"`
	Id       uuid.UUID `json:"id"`
	jwt.StandardClaims
}

type userUseCase struct {
	userRepo  repo.UserRepo
	hasher    hasher.PasswordHasher
	jwtSecret string
	botToken  string
}

func New(userRepo repo.UserRepo, h hasher.PasswordHasher, jwtSecret, botToken string) usecase.UserUseCase {
	return &userUseCase{
		userRepo:  userRepo,
		hasher:    h,
		jwtSecret: jwtSecret,
		botToken:  botToken,
	}
}

func (uc *userUseCase) Register(ctx context.Context, username, password string) (usecase.AuthResult, error) {
	_, err := uc.userRepo.GetByUsername(ctx, username)
	if err == nil {
		return usecase.AuthResult{}, errors.New("пользователь с таким username уже существует")
	}

	hashed, err := uc.hasher.Hash(password)
	if err != nil {
		return usecase.AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user := entity.User{
		ID:       uuid.New(),
		Username: username,
		Password: hashed,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return usecase.AuthResult{}, fmt.Errorf("create user: %w", err)
	}

	token, err := uc.generateToken(user)
	if err != nil {
		return usecase.AuthResult{}, fmt.Errorf("generate token: %w", err)
	}

	return usecase.AuthResult{Token: token, User: user}, nil
}

func (uc *userUseCase) Login(ctx context.Context, username, password string) (usecase.AuthResult, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return usecase.AuthResult{}, errors.New("неверный логин или пароль")
	}

	if err := uc.hasher.Compare(user.Password, password); err != nil {
		return usecase.AuthResult{}, errors.New("неверный логин или пароль")
	}

	token, err := uc.generateToken(user)
	if err != nil {
		return usecase.AuthResult{}, fmt.Errorf("generate token: %w", err)
	}

	return usecase.AuthResult{Token: token, User: user}, nil
}

func (uc *userUseCase) AuthenticateTelegram(ctx context.Context, input usecase.TelegramAuthInput) (usecase.AuthResult, error) {
	if uc.botToken == "" {
		return usecase.AuthResult{}, errors.New("BOT_TOKEN is not set")
	}

	creds := tgverifier.Credentials{
		ID:        input.ID,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		PhotoURL:  input.PhotoURL,
		Username:  input.Username,
		AuthDate:  input.AuthDate,
		Hash:      input.Hash,
	}

	if err := creds.Verify([]byte(uc.botToken)); err != nil {
		return usecase.AuthResult{}, errors.New("authentication failed")
	}

	userIDStr := strconv.FormatInt(input.ID, 10)

	existingUser, err := uc.userRepo.GetByUsername(ctx, userIDStr)
	var user entity.User
	if err != nil {
		// Пользователь не найден — создаём нового
		user = entity.User{
			ID:       uuid.New(),
			Username: userIDStr,
			Password: userIDStr,
		}
		if err := uc.userRepo.Create(ctx, user); err != nil {
			return usecase.AuthResult{}, fmt.Errorf("create telegram user: %w", err)
		}
	} else {
		user = existingUser
	}

	token, err := uc.generateToken(user)
	if err != nil {
		return usecase.AuthResult{}, fmt.Errorf("generate token: %w", err)
	}

	return usecase.AuthResult{Token: token, User: user}, nil
}

func (uc *userUseCase) GetMe(ctx context.Context, userID uuid.UUID) (entity.User, error) {
	return uc.userRepo.GetByID(ctx, userID)
}

func (uc *userUseCase) generateToken(user entity.User) (string, error) {
	claims := &Claims{
		Username: user.Username,
		Id:       user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwtSecret))
}
