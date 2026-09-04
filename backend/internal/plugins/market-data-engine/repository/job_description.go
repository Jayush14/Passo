package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobDescriptionRepository handles storage and retrieval of job descriptions
type JobDescriptionRepository struct {
	db *pgxpool.Pool
}

// NewJobDescriptionRepository creates a new JobDescriptionRepository
func NewJobDescriptionRepository(db *pgxpool.Pool) *JobDescriptionRepository {
	return &JobDescriptionRepository{db: db}
}

// Insert adds a new job description and returns its ID
func (r *JobDescriptionRepository) Insert(ctx context.Context, jobDescription *models.JobDescription) (uuid.UUID, error) {
	query := `
		INSERT INTO job_descriptions
			(id, job_id, title, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobDescription.ID,
		jobDescription.JobID,
		jobDescription.Title,
		jobDescription.Description,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job description by its ID
func (r *JobDescriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobDescription, error) {
	query := `
		SELECT id, job_id, title, description
		FROM job_descriptions
		WHERE id = $1
	`

	var jobDescription models.JobDescription
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobDescription.ID,
		&jobDescription.JobID,
		&jobDescription.Title,
		&jobDescription.Description,
	)

	if err != nil {
		return nil, err
	}

	return &jobDescription, nil
}

// GetByJobID retrieves a job description by its job ID
func (r *JobDescriptionRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*models.JobDescription, error) {
	query := `
		SELECT id, job_id, title, description
		FROM job_descriptions
		WHERE job_id = $1
	`

	var jobDescription models.JobDescription
	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&jobDescription.ID,
		&jobDescription.JobID,
		&jobDescription.Title,
		&jobDescription.Description,
	)

	if err != nil {
		return nil, err
	}

	return &jobDescription, nil
}

// ListByJobID returns all job descriptions for a given job
func (r *JobDescriptionRepository) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobDescription, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, title, description
		FROM job_descriptions
		WHERE job_id = $1
		ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobDescriptions []*models.JobDescription
	for rows.Next() {
		var jobDescription models.JobDescription
		if err := rows.Scan(
			&jobDescription.ID,
			&jobDescription.JobID,
			&jobDescription.Title,
			&jobDescription.Description,
		); err != nil {
			return nil, err
		}
		jobDescriptions = append(jobDescriptions, &jobDescription)
	}

	return jobDescriptions, rows.Err()
}

// List returns all job descriptions with pagination
func (r *JobDescriptionRepository) List(ctx context.Context, limit, offset int) ([]*models.JobDescription, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, title, description
		FROM job_descriptions
		ORDER BY job_id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobDescriptions []*models.JobDescription
	for rows.Next() {
		var jobDescription models.JobDescription
		if err := rows.Scan(
			&jobDescription.ID,
			&jobDescription.JobID,
			&jobDescription.Title,
			&jobDescription.Description,
		); err != nil {
			return nil, err
		}
		jobDescriptions = append(jobDescriptions, &jobDescription)
	}

	return jobDescriptions, rows.Err()
}

// Update updates an existing job description
func (r *JobDescriptionRepository) Update(ctx context.Context, jobDescription *models.JobDescription) error {
	query := `
		UPDATE job_descriptions
		SET job_id = $2,
		    title = $3,
		    description = $4
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobDescription.ID,
		jobDescription.JobID,
		jobDescription.Title,
		jobDescription.Description,
	)

	return err
}

// Delete removes a job description by its ID
func (r *JobDescriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_descriptions
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}