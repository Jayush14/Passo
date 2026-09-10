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
			(id, job_id, capability_id, importance, evidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id uuid.UUID
	var capabilityID *uuid.UUID
	if jobCapability.CapabilityID != nil && *jobCapability.CapabilityID != uuid.Nil {
		capabilityID = jobCapability.CapabilityID
	}
	err := r.db.QueryRow(ctx, query,
		jobCapability.ID,
		jobCapability.JobID,
		capabilityID,
		jobCapability.Importance,
		jobCapability.Evidence,
		jobCapability.CreatedAt,
		jobCapability.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job capability by its ID
func (r *JobCapabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobCapability, error) {
	query := `
		SELECT id, job_id, capability_id, importance, evidence, created_at, updated_at
		FROM job_capabilities
		WHERE id = $1
	`

	var jobCapability models.JobCapability
	var capabilityID *uuid.UUID
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobCapability.ID,
		&jobCapability.JobID,
		&capabilityID,
		&jobCapability.Importance,
		&jobCapability.Evidence,
		&jobCapability.CreatedAt,
		&jobCapability.UpdatedAt,
	)
	if capabilityID != nil {
		jobCapability.CapabilityID = capabilityID
	} else {
		jobCapability.CapabilityID = nil
	}

	if err != nil {
		return nil, err
	}

	return &jobCapability, nil
}

// ListByJobID returns all capabilities for a given job
func (r *JobCapabilityRepository) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobCapability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, capability_id, importance, evidence, created_at, updated_at
		FROM job_capabilities
		WHERE job_id = $1
		ORDER BY importance DESC
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
			&jobCapability.JobID,
			&jobCapability.CapabilityID,
			&jobCapability.Importance,
			&jobCapability.Evidence,
			&jobCapability.CreatedAt,
			&jobCapability.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobCapabilities = append(jobCapabilities, &jobCapability)
	}

	return jobCapabilities, rows.Err()
}

// ListByCapabilityID returns all jobs that require a given capability
func (r *JobCapabilityRepository) ListByCapabilityID(ctx context.Context, capabilityID uuid.UUID) ([]*models.JobCapability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, capability_id, importance, evidence, created_at, updated_at
		FROM job_capabilities
		WHERE capability_id = $1
		ORDER BY job_id
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
			&jobCapability.JobID,
			&jobCapability.CapabilityID,
			&jobCapability.Importance,
			&jobCapability.Evidence,
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
		SET job_id = $2,
		    capability_id = $3,
		    importance = $4,
		    evidence = $5,
		    updated_at = $6
		WHERE id = $1
	`

	var capabilityID *uuid.UUID
	if jobCapability.CapabilityID != nil && *jobCapability.CapabilityID != uuid.Nil {
		capabilityID = jobCapability.CapabilityID
	}
	_, err := r.db.Exec(ctx, query,
		jobCapability.ID,
		jobCapability.JobID,
		capabilityID,
		jobCapability.Importance,
		jobCapability.Evidence,
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