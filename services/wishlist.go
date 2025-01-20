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
	IncreasePresentsCount(id uuid.UUID) (error, model.Wishlist)
	DecreasePresentsCount(id uuid.UUID) (error, model.Wishlist)
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
	var wishlists []model.Wishlist
	result := database.DB.Where("user_id = ?", userID).Find(&wishlists)
	return result.Error, wishlists
}

func (w *wishlistService) IncreasePresentsCount(id uuid.UUID) (error, model.Wishlist) {
	err, wishlist := w.GetOne(id)
	if err != nil {
		return err, model.Wishlist{}
	}
	wishlist.PresentsCount += 1
	result := database.DB.Save(&wishlist)
	return result.Error, wishlist
}

func (w *wishlistService) DecreasePresentsCount(id uuid.UUID) (error, model.Wishlist) {
	err, wishlist := w.GetOne(id)
	if err != nil {
		return err, model.Wishlist{}
	}
	wishlist.PresentsCount -= 1
	result := database.DB.Save(&wishlist)
	return result.Error, wishlist
}
