package entity

import (
	"time"

	"github.com/google/uuid"
)

type Template struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Settings  Settings // reuse existing type
	Blocks    []Block  // data of each block = "{}" (stripped)
	IsPublic  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TemplateWithAuthor — template with author name (for public gallery)
type TemplateWithAuthor struct {
	Template
	UserDisplayName string
}
