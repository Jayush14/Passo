package models

import (
	"github.com/google/uuid"
)

// SkillCategory represents a category of skills
type SkillCategory struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code"`
}