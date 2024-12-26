package services

import (
	"github.com/google/uuid"
	"main/database"
	"main/model"
)

type PresentService interface {
	GetOne(id uuid.UUID) (error, model.Present)
	GetAll(wishlistId uuid.UUID) (error, []model.Present)
}

type presentService struct{}

func NewPresentService() PresentService {
	return &presentService{}
}

// GetOne получает один элемент желаемого списка по ID
func (w *presentService) GetOne(id uuid.UUID) (error, model.Present) {
	present := model.Present{ID: id}
	result := database.DB.First(&present)
	return result.Error, present
}

func (w *presentService) GetAll(wishlistId uuid.UUID) (error, []model.Present) {
	var presents []model.Present
	result := database.DB.Where("wishlist_id = ?", wishlistId).Find(&presents)

	return result.Error, presents
}
