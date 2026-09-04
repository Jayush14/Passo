package models

import (
	"time"
	"github.com/google/uuid"
)

// FailedExtraction represents a job extraction that failed processing
type FailedExtraction struct {
	ID            uuid.UUID `json:"id"`
	JobID         uuid.UUID `json:"job_id"`
	ExtractionData string   `json:"extraction_data"`
	ErrorMessage  string   `json:"error_message"`
	StackTrace    string   `json:"stack_trace"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}