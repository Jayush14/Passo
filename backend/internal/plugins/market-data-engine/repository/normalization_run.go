package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// NormalizationRunRepository handles storage and retrieval of normalization runs
type NormalizationRunRepository struct {
	db *pgxpool.Pool
}

// NewNormalizationRunRepository creates a new NormalizationRunRepository
func NewNormalizationRunRepository(db *pgxpool.Pool) *NormalizationRunRepository {
	return &NormalizationRunRepository{db: db}
}

// Insert adds a new normalization run and returns its ID
func (r *NormalizationRunRepository) Insert(ctx context.Context, run *models.NormalizationRun) (uuid.UUID, error) {
	query := `
		INSERT INTO normalization_runs
			(id, run_type, status, started_at, completed_at, records_processed, records_updated, records_failed, error_message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		run.ID,
		run.RunType,
		run.Status,
		run.StartedAt,
		run.CompletedAt,
		run.RecordsProcessed,
		run.RecordsUpdated,
		run.RecordsFailed,
		run.ErrorMessage,
		run.CreatedAt,
		run.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a normalization run by its ID
func (r *NormalizationRunRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.NormalizationRun, error) {
	query := `
		SELECT id, run_type, status, started_at, completed_at, records_processed, records_updated, records_failed, error_message, created_at, updated_at
		FROM normalization_runs
		WHERE id = $1
	`

	var run models.NormalizationRun
	err := r.db.QueryRow(ctx, query, id).Scan(
		&run.ID,
		&run.RunType,
		&run.Status,
		&run.StartedAt,
		&run.CompletedAt,
		&run.RecordsProcessed,
		&run.RecordsUpdated,
		&run.RecordsFailed,
		&run.ErrorMessage,
		&run.CreatedAt,
		&run.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &run, nil
}

// List returns all normalization runs with pagination
func (r *NormalizationRunRepository) List(ctx context.Context, limit, offset int) ([]*models.NormalizationRun, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, run_type, status, started_at, completed_at, records_processed, records_updated, records_failed, error_message, created_at, updated_at
		FROM normalization_runs
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*models.NormalizationRun
	for rows.Next() {
		var run models.NormalizationRun
		if err := rows.Scan(
			&run.ID,
			&run.RunType,
			&run.Status,
			&run.StartedAt,
			&run.CompletedAt,
			&run.RecordsProcessed,
			&run.RecordsUpdated,
			&run.RecordsFailed,
			&run.ErrorMessage,
			&run.CreatedAt,
			&run.UpdatedAt,
		); err != nil {
			return nil, err
		}
		runs = append(runs, &run)
	}

	return runs, rows.Err()
}

// Update updates an existing normalization run
func (r *NormalizationRunRepository) Update(ctx context.Context, run *models.NormalizationRun) error {
	query := `
		UPDATE normalization_runs
		SET run_type = $2,
		    status = $3,
		    started_at = $4,
		    completed_at = $5,
		    records_processed = $6,
		    records_updated = $7,
		    records_failed = $8,
		    error_message = $9,
		    updated_at = $10
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		run.ID,
		run.RunType,
		run.Status,
		run.StartedAt,
		run.CompletedAt,
		run.RecordsProcessed,
		run.RecordsUpdated,
		run.RecordsFailed,
		run.ErrorMessage,
		run.UpdatedAt,
	)

	return err
}

// Delete removes a normalization run by its ID
func (r *NormalizationRunRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM normalization_runs
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// UpdateStatus updates the status of a normalization run
func (r *NormalizationRunRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, completedAt *string, errorMessage *string) error {
	query := `
		UPDATE normalization_runs
		SET status = $2,
		    completed_at = $3,
		    error_message = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		id,
		status,
		completedAt,
		errorMessage,
	)
	return err
}

// UpdateProgress updates the progress of a normalization run
func (r *NormalizationRunRepository) UpdateProgress(ctx context.Context, id uuid.UUID, recordsProcessed, recordsUpdated, recordsFailed int) error {
	query := `
		UPDATE normalization_runs
		SET records_processed = $2,
		    records_updated = $3,
		    records_failed = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		id,
		recordsProcessed,
		recordsUpdated,
		recordsFailed,
	)
	return err
}