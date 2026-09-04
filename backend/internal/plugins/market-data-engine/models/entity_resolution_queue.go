package models

import (
	"time"
	"github.com/google/uuid"
)

// EntityResolutionQueueItem represents an item waiting for entity resolution
type EntityResolutionQueueItem struct {
	ID          uuid.UUID `json:"id"`
	EntityType  string    `json:"entity_type"`
	EntityID    uuid.UUID `json:"entity_id"`
	FieldName   string    `json:"field_name"`
	FieldValue  string    `json:"field_value"`
	Priority    int       `json:"priority"`
	Attempts    int       `json:"attempts"`
	Status      string    `json:"status"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}