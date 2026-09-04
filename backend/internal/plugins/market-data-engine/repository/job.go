package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobRepository handles storage and retrieval of jobs
type JobRepository struct {
	db *pgxpool.Pool
}

// NewJobRepository creates a new JobRepository
func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

// Insert adds a new job and returns its ID
func (r *JobRepository) Insert(ctx context.Context, job *models.Job) (uuid.UUID, error) {
	query := `
		INSERT INTO jobs
			(id, external_id, source, source_url, raw_content, dedup_key,
			 language, retrieved_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		job.ID,
		job.ExternalID,
		job.Source,
		job.SourceURL,
		job.RawContent,
		job.DedupKey,
		job.Language,
		job.RetrievedAt,
		job.CreatedAt,
		job.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job by its ID
func (r *JobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	query := `
		SELECT id, external_id, source, source_url, raw_content, dedup_key,
		       language, retrieved_at, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	var job models.Job
	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.ExternalID,
		&job.Source,
		&job.SourceURL,
		&job.RawContent,
		&job.DedupKey,
		&job.Language,
		&job.RetrievedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// GetByExternalID retrieves a job by its external ID
func (r *JobRepository) GetByExternalID(ctx context.Context, externalID string) (*models.Job, error) {
	query := `
		SELECT id, external_id, source, source_url, raw_content, dedup_key,
		       language, retrieved_at, created_at, updated_at
		FROM jobs
		WHERE external_id = $1
	`

	var job models.Job
	err := r.db.QueryRow(ctx, query, externalID).Scan(
		&job.ID,
		&job.ExternalID,
		&job.Source,
		&job.SourceURL,
		&job.RawContent,
		&job.DedupKey,
		&job.Language,
		&job.RetrievedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// GetByDedupKey retrieves a job by its deduplication key
func (r *JobRepository) GetByDedupKey(ctx context.Context, dedupKey string) (*models.Job, error) {
	query := `
		SELECT id, external_id, source, source_url, raw_content, dedup_key,
		       language, retrieved_at, created_at, updated_at
		FROM jobs
		WHERE dedup_key = $1
	`

	var job models.Job
	err := r.db.QueryRow(ctx, query, dedupKey).Scan(
		&job.ID,
		&job.ExternalID,
		&job.Source,
		&job.SourceURL,
		&job.RawContent,
		&job.DedupKey,
		&job.Language,
		&job.RetrievedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// List returns all jobs with pagination
func (r *JobRepository) List(ctx context.Context, limit, offset int) ([]*models.Job, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, external_id, source, source_url, raw_content, dedup_key,
		       language, retrieved_at, created_at, updated_at
		FROM jobs
		ORDER BY retrieved_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		var job models.Job
		if err := rows.Scan(
			&job.ID,
			&job.ExternalID,
			&job.Source,
			&job.SourceURL,
			&job.RawContent,
			&job.DedupKey,
			&job.Language,
			&job.RetrievedAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, &job)
	}

	return jobs, rows.Err()
}

// Update updates an existing job
func (r *JobRepository) Update(ctx context.Context, job *models.Job) error {
	query := `
		UPDATE jobs
		SET external_id = $2,
		    source = $3,
		    source_url = $4,
		    raw_content = $5,
		    dedup_key = $6,
		    language = $7,
		    retrieved_at = $8,
		    updated_at = $9
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		job.ID,
		job.ExternalID,
		job.Source,
		job.SourceURL,
		job.RawContent,
		job.DedupKey,
		job.Language,
		job.RetrievedAt,
		job.UpdatedAt,
	)

	return err
}

// Delete removes a job by its ID
func (r *JobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM jobs
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// UpdateMDEProcessingStatus updates the MDE processing status of a job
func (r *JobRepository) UpdateMDEProcessingStatus(ctx context.Context, jobID uuid.UUID, processed bool, processedAt *string, errorMessage *string) error {
	query := `
		UPDATE jobs
		SET mde_processed = $2,
		    mde_processed_at = $3,
		    mde_error_message = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobID,
		processed,
		processedAt,
		errorMessage,
	)
	return err
}