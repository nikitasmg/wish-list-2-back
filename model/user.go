package model

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"unique;not null" json:"username" validate:"required"`
	Password  string     `gorm:"not null" json:"password" validate:"required"`
	Wishlists []Wishlist `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"` // связь с Wishlists
}

type Claims struct {
	Username string    `json:"username"`
	Id       uuid.UUID `json:"id"`
	jwt.StandardClaims
}
