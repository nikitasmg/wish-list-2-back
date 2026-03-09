package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/entity"
	"main/internal/usecase"
	userUC "main/internal/usecase/user"
	mockrepo "main/mock/repo"
	"main/pkg/hasher"
)

const testJWTSecret = "test-secret-key"

func newUserUC(ur *mockrepo.MockUserRepo) usecase.UserUseCase {
	return userUC.New(ur, hasher.New(), testJWTSecret, "")
}

func TestRegister_DuplicateUsername(t *testing.T) {
	ur := &mockrepo.MockUserRepo{}
	uc := newUserUC(ur)

	ur.On("GetByUsername", mock.Anything, "alice").Return(entity.User{Username: "alice"}, nil)

	_, err := uc.Register(context.Background(), "alice", "password123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "уже существует")
}

func TestLogin_WrongPassword(t *testing.T) {
	ur := &mockrepo.MockUserRepo{}
	uc := newUserUC(ur)

	h := hasher.New()
	hashed, _ := h.Hash("correctpassword")
	ur.On("GetByUsername", mock.Anything, "bob").Return(entity.User{
		ID:       uuid.New(),
		Username: "bob",
		Password: hashed,
	}, nil)

	_, err := uc.Login(context.Background(), "bob", "wrongpassword")
	require.Error(t, err)
	assert.Equal(t, "неверный логин или пароль", err.Error())
}

func TestLogin_Success(t *testing.T) {
	ur := &mockrepo.MockUserRepo{}
	uc := newUserUC(ur)

	h := hasher.New()
	hashed, _ := h.Hash("password123")
	userID := uuid.New()
	ur.On("GetByUsername", mock.Anything, "alice").Return(entity.User{
		ID:       userID,
		Username: "alice",
		Password: hashed,
	}, nil)

	result, err := uc.Login(context.Background(), "alice", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)

	// Token must contain the user ID
	token, parseErr := jwt.Parse(result.Token, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, parseErr)
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, userID.String(), claims["id"].(string))
}

func TestRegister_NotFound_ThenCreates(t *testing.T) {
	ur := &mockrepo.MockUserRepo{}
	uc := newUserUC(ur)

	ur.On("GetByUsername", mock.Anything, "newuser").Return(entity.User{}, errors.New("not found"))
	ur.On("Create", mock.Anything, mock.Anything).Return(nil)

	result, err := uc.Register(context.Background(), "newuser", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	ur.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}
