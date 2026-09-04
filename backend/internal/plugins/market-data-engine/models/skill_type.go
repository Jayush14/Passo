package models

import (
	"github.com/google/uuid"
)

// SkillType represents a type of skill
type SkillType struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code"`
}