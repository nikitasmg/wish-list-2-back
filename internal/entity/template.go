package entity

import (
	"time"

	"github.com/google/uuid"
)

type Template struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Name      string    `json:"name"`
	Settings  Settings  `json:"settings"`
	Blocks    []Block   `json:"blocks"`
	IsPublic  bool      `json:"isPublic"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TemplateWithAuthor — template with author name and like data (for public gallery)
type TemplateWithAuthor struct {
	Template
	UserDisplayName string `json:"userDisplayName"`
	LikesCount      int    `json:"likesCount"`
	LikedByMe       bool   `json:"likedByMe"`
}
