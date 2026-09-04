package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// EntityResolutionQueueRepository handles storage and retrieval of entity resolution queue items
type EntityResolutionQueueRepository struct {
	db *pgxpool.Pool
}

// NewEntityResolutionQueueRepository creates a new EntityResolutionQueueRepository
func NewEntityResolutionQueueRepository(db *pgxpool.Pool) *EntityResolutionQueueRepository {
	return &EntityResolutionQueueRepository{db: db}
}

// Insert adds a new entity resolution queue item and returns its ID
func (r *EntityResolutionQueueRepository) Insert(ctx context.Context, item *models.EntityResolutionQueueItem) (uuid.UUID, error) {
	query := `
		INSERT INTO entity_resolution_queue
			(id, entity_type, entity_id, field_name, field_value, priority, attempts, status, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		item.ID,
		item.EntityType,
		item.EntityID,
		item.FieldName,
		item.FieldValue,
		item.Priority,
		item.Attempts,
		item.Status,
		item.ErrorMessage,
		item.CreatedAt,
		item.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves an entity resolution queue item by its ID
func (r *EntityResolutionQueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EntityResolutionQueueItem, error) {
	query := `
		SELECT id, entity_type, entity_id, field_name, field_value, priority, attempts, status, error_message, created_at, updated_at
		FROM entity_resolution_queue
		WHERE id = $1
	`

	var item models.EntityResolutionQueueItem
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.EntityType,
		&item.EntityID,
		&item.FieldName,
		&item.FieldValue,
		&item.Priority,
		&item.Attempts,
		&item.Status,
		&item.ErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &item, nil
}

// GetPendingItems retrieves pending entity resolution queue items
func (r *EntityResolutionQueueRepository) GetPendingItems(ctx context.Context, limit int) ([]*models.EntityResolutionQueueItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, entity_type, entity_id, field_name, field_value, priority, attempts, status, error_message, created_at, updated_at
		FROM entity_resolution_queue
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.EntityResolutionQueueItem
	for rows.Next() {
		var item models.EntityResolutionQueueItem
		if err := rows.Scan(
			&item.ID,
			&item.EntityType,
			&item.EntityID,
			&item.FieldName,
			&item.FieldValue,
			&item.Priority,
			&item.Attempts,
			&item.Status,
			&item.ErrorMessage,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, rows.Err()
}

// Update updates an existing entity resolution queue item
func (r *EntityResolutionQueueRepository) Update(ctx context.Context, item *models.EntityResolutionQueueItem) error {
	query := `
		UPDATE entity_resolution_queue
		SET entity_type = $2,
		    entity_id = $3,
		    field_name = $4,
		    field_value = $5,
		    priority = $6,
		    attempts = $7,
		    status = $8,
		    error_message = $9,
		    updated_at = $10
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.EntityType,
		item.EntityID,
		item.FieldName,
		item.FieldValue,
		item.Priority,
		item.Attempts,
		item.Status,
		item.ErrorMessage,
		item.UpdatedAt,
	)

	return err
}

// Delete removes an entity resolution queue item by its ID
func (r *EntityResolutionQueueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM entity_resolution_queue
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByIDAndEntityType removes entity resolution queue items by ID and entity type
func (r *EntityResolutionQueueRepository) DeleteByIDAndEntityType(ctx context.Context, id uuid.UUID, entityType string) error {
	query := `
		DELETE FROM entity_resolution_queue
		WHERE id = $1 AND entity_type = $2
	`

	_, err := r.db.Exec(ctx, query, id, entityType)
	return err
}