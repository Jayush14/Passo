package models

import (
	"time"
	"github.com/google/uuid"
)

// JobCapability represents a capability associated with a job
type JobCapability struct {
	ID           uuid.UUID `json:"id"`
	JobID        uuid.UUID `json:"job_id"`
	CapabilityID *uuid.UUID `json:"capability_id,omitempty"` // Can be null for Phase 2 placeholder
	Importance   float64   `json:"importance"`
	Evidence     string    `json:"evidence,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}