package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobCapabilityRepository handles storage and retrieval of job capabilities
type JobCapabilityRepository struct {
	db *pgxpool.Pool
}

// NewJobCapabilityRepository creates a new JobCapabilityRepository
func NewJobCapabilityRepository(db *pgxpool.Pool) *JobCapabilityRepository {
	return &JobCapabilityRepository{db: db}
}

// Insert adds a new job capability and returns its ID
func (r *JobCapabilityRepository) Insert(ctx context.Context, jobCapability *models.JobCapability) (uuid.UUID, error) {
	query := `
		INSERT INTO job_capabilities
			(id, job_posting_id, capability_id, level, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobCapability.ID,
		jobCapability.JobPostingID,
		jobCapability.CapabilityID,
		jobCapability.Level,
		jobCapability.CreatedAt,
		jobCapability.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job capability by its ID
func (r *JobCapabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobCapability, error) {
	query := `
		SELECT id, job_posting_id, capability_id, level, created_at, updated_at
		FROM job_capabilities
		WHERE id = $1
	`

	var jobCapability models.JobCapability
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobCapability.ID,
		&jobCapability.JobPostingID,
		&jobCapability.CapabilityID,
		&jobCapability.Level,
		&jobCapability.CreatedAt,
		&jobCapability.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobCapability, nil
}

// ListByJobPostingID returns all capabilities for a given job posting
func (r *JobCapabilityRepository) ListByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) ([]*models.JobCapability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, capability_id, level, created_at, updated_at
		FROM job_capabilities
		WHERE job_posting_id = $1
		ORDER BY level DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobCapabilities []*models.JobCapability
	for rows.Next() {
		var jobCapability models.JobCapability
		if err := rows.Scan(
			&jobCapability.ID,
			&jobCapability.JobPostingID,
			&jobCapability.CapabilityID,
			&jobCapability.Level,
			&jobCapability.CreatedAt,
			&jobCapability.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobCapabilities = append(jobCapabilities, &jobCapability)
	}

	return jobCapabilities, rows.Err()
}

// ListByCapabilityID returns all job postings that require a given capability
func (r *JobCapabilityRepository) ListByCapabilityID(ctx context.Context, capabilityID uuid.UUID) ([]*models.JobCapability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, capability_id, level, created_at, updated_at
		FROM job_capabilities
		WHERE capability_id = $1
		ORDER BY job_posting_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobCapabilities []*models.JobCapability
	for rows.Next() {
		var jobCapability models.JobCapability
		if err := rows.Scan(
			&jobCapability.ID,
			&jobCapability.JobPostingID,
			&jobCapability.CapabilityID,
			&jobCapability.Level,
			&jobCapability.CreatedAt,
			&jobCapability.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobCapabilities = append(jobCapabilities, &jobCapability)
	}

	return jobCapabilities, rows.Err()
}

// Update updates an existing job capability
func (r *JobCapabilityRepository) Update(ctx context.Context, jobCapability *models.JobCapability) error {
	query := `
		UPDATE job_capabilities
		SET job_posting_id = $2,
		    capability_id = $3,
		    level = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobCapability.ID,
		jobCapability.JobPostingID,
		jobCapability.CapabilityID,
		jobCapability.Level,
		jobCapability.UpdatedAt,
	)

	return err
}

// Delete removes a job capability by its ID
func (r *JobCapabilityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_capabilities
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobPostingID removes all job capabilities for a given job posting
func (r *JobCapabilityRepository) DeleteByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) error {
	query := `
		DELETE FROM job_capabilities
		WHERE job_posting_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobPostingID)
	return err
}