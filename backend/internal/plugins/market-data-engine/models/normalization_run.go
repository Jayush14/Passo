package models

import (
	"time"
	"github.com/google/uuid"
)

// NormalizationRun represents a run of the normalization process
type NormalizationRun struct {
	ID              uuid.UUID  `json:"id"`
	RunType         string     `json:"run_type"` // e.g., "full", "incremental"
	Status          string     `json:"status"`   // running, completed, failed
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	RecordsProcessed int        `json:"records_processed"`
	RecordsUpdated  int        `json:"records_updated"`
	RecordsFailed   int        `json:"records_failed"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}