package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// FailedExtractionRepository handles storage and retrieval of failed extractions
type FailedExtractionRepository struct {
	db *pgxpool.Pool
}

// NewFailedExtractionRepository creates a new FailedExtractionRepository
func NewFailedExtractionRepository(db *pgxpool.Pool) *FailedExtractionRepository {
	return &FailedExtractionRepository{db: db}
}

// Insert adds a new failed extraction and returns its ID
func (r *FailedExtractionRepository) Insert(ctx context.Context, failedExtraction *models.FailedExtraction) (uuid.UUID, error) {
	query := `
		INSERT INTO failed_extractions
			(id, job_id, extraction_data, error_message, stack_trace, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		failedExtraction.ID,
		failedExtraction.JobID,
		failedExtraction.ExtractionData,
		failedExtraction.ErrorMessage,
		failedExtraction.StackTrace,
		failedExtraction.CreatedAt,
		failedExtraction.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a failed extraction by its ID
func (r *FailedExtractionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FailedExtraction, error) {
	query := `
		SELECT id, job_id, extraction_data, error_message, stack_trace, created_at, updated_at
		FROM failed_extractions
		WHERE id = $1
	`

	var failedExtraction models.FailedExtraction
	err := r.db.QueryRow(ctx, query, id).Scan(
		&failedExtraction.ID,
		&failedExtraction.JobID,
		&failedExtraction.ExtractionData,
		&failedExtraction.ErrorMessage,
		&failedExtraction.StackTrace,
		&failedExtraction.CreatedAt,
		&failedExtraction.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &failedExtraction, nil
}

// GetByJobID retrieves failed extractions for a given job
func (r *FailedExtractionRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.FailedExtraction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, extraction_data, error_message, stack_trace, created_at, updated_at
		FROM failed_extractions
		WHERE job_id = $1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failedExtractions []*models.FailedExtraction
	for rows.Next() {
		var failedExtraction models.FailedExtraction
		if err := rows.Scan(
			&failedExtraction.ID,
			&failedExtraction.JobID,
			&failedExtraction.ExtractionData,
			&failedExtraction.ErrorMessage,
			&failedExtraction.StackTrace,
			&failedExtraction.CreatedAt,
			&failedExtraction.UpdatedAt,
		); err != nil {
			return nil, err
		}
		failedExtractions = append(failedExtractions, &failedExtraction)
	}

	return failedExtractions, rows.Err()
}

// List returns all failed extractions with pagination
func (r *FailedExtractionRepository) List(ctx context.Context, limit, offset int) ([]*models.FailedExtraction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, extraction_data, error_message, stack_trace, created_at, updated_at
		FROM failed_extractions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failedExtractions []*models.FailedExtraction
	for rows.Next() {
		var failedExtraction models.FailedExtraction
		if err := rows.Scan(
			&failedExtraction.ID,
			&failedExtraction.JobID,
			&failedExtraction.ExtractionData,
			&failedExtraction.ErrorMessage,
			&failedExtraction.StackTrace,
			&failedExtraction.CreatedAt,
			&failedExtraction.UpdatedAt,
		); err != nil {
			return nil, err
		}
		failedExtractions = append(failedExtractions, &failedExtraction)
	}

	return failedExtractions, rows.Err()
}

// Update updates an existing failed extraction
func (r *FailedExtractionRepository) Update(ctx context.Context, failedExtraction *models.FailedExtraction) error {
	query := `
		UPDATE failed_extractions
		SET job_id = $2,
		    extraction_data = $3,
		    error_message = $4,
		    stack_trace = $5,
		    updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		failedExtraction.ID,
		failedExtraction.JobID,
		failedExtraction.ExtractionData,
		failedExtraction.ErrorMessage,
		failedExtraction.StackTrace,
		failedExtraction.UpdatedAt,
	)

	return err
}

// Delete removes a failed extraction by its ID
func (r *FailedExtractionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM failed_extractions
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobID removes all failed extractions for a given job
func (r *FailedExtractionRepository) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	query := `
		DELETE FROM failed_extractions
		WHERE job_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobID)
	return err
}