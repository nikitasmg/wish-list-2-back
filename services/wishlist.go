package services

import (
	"github.com/google/uuid"
	"main/database"
	"main/model"
)

// WishlistService интерфейс для взаимодействия с желаемым списком
type WishlistService interface {
	GetOne(id uuid.UUID) (error, model.Wishlist)
	GetAll(userID *uuid.UUID) (error, []model.Wishlist)
}

// wishlistService реализация интерфейса WishlistService
type wishlistService struct{}

// NewWishlistService создает новый экземпляр WishlistService
func NewWishlistService() WishlistService {
	return &wishlistService{}
}

// GetOne получает один элемент желаемого списка по ID
func (w *wishlistService) GetOne(id uuid.UUID) (error, model.Wishlist) {
	wishlist := model.Wishlist{ID: id}
	result := database.DB.First(&wishlist)
	return result.Error, wishlist
}

func (w *wishlistService) GetAll(userID *uuid.UUID) (error, []model.Wishlist) {
	var Wishlists []model.Wishlist
	result := database.DB.Where("user_id = ?", userID).Find(&Wishlists)
	return result.Error, Wishlists
}
