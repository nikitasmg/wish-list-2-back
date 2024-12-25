package model

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID  `gorm:"primaryKey" json:"id"`
	Username  string     `gorm:"unique;not null" json:"username" validate:"required"`
	Password  string     `gorm:"not null" json:"password" validate:"required"`
	WishLists []WishList `gorm:"foreignKey:UserID" json:"wish_lists"` // связь с wishLists
}

type Claims struct {
	Username string    `json:"username"`
	Id       uuid.UUID `json:"id"`
	jwt.StandardClaims
}
