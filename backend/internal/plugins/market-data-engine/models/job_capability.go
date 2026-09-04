package models

import (
	"time"
	"github.com/google/uuid"
)

// JobCapability represents a capability associated with a job posting
type JobCapability struct {
	ID           uuid.UUID `json:"id"`
	JobPostingID uuid.UUID `json:"job_posting_id"`
	CapabilityID uuid.UUID `json:"capability_id"`
	Level        string    `json:"level"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}