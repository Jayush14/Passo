package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobDomainRepository handles storage and retrieval of job domains
type JobDomainRepository struct {
	db *pgxpool.Pool
}

// NewJobDomainRepository creates a new JobDomainRepository
func NewJobDomainRepository(db *pgxpool.Pool) *JobDomainRepository {
	return &JobDomainRepository{db: db}
}

// Insert adds a new job domain and returns its ID
func (r *JobDomainRepository) Insert(ctx context.Context, jobDomain *models.JobDomain) (uuid.UUID, error) {
	query := `
		INSERT INTO job_domains
			(id, job_posting_id, domain_id, relevance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobDomain.ID,
		jobDomain.JobPostingID,
		jobDomain.DomainID,
		jobDomain.Relevance,
		jobDomain.CreatedAt,
		jobDomain.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job domain by its ID
func (r *JobDomainRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobDomain, error) {
	query := `
		SELECT id, job_posting_id, domain_id, relevance, created_at, updated_at
		FROM job_domains
		WHERE id = $1
	`

	var jobDomain models.JobDomain
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobDomain.ID,
		&jobDomain.JobPostingID,
		&jobDomain.DomainID,
		&jobDomain.Relevance,
		&jobDomain.CreatedAt,
		&jobDomain.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &jobDomain, nil
}

// ListByJobPostingID returns all domains for a given job posting
func (r *JobDomainRepository) ListByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) ([]*models.JobDomain, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, domain_id, relevance, created_at, updated_at
		FROM job_domains
		WHERE job_posting_id = $1
		ORDER BY relevance DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobDomains []*models.JobDomain
	for rows.Next() {
		var jobDomain models.JobDomain
		if err := rows.Scan(
			&jobDomain.ID,
			&jobDomain.JobPostingID,
			&jobDomain.DomainID,
			&jobDomain.Relevance,
			&jobDomain.CreatedAt,
			&jobDomain.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobDomains = append(jobDomains, &jobDomain)
	}

	return jobDomains, rows.Err()
}

// ListByDomainID returns all job postings that are in a given domain
func (r *JobDomainRepository) ListByDomainID(ctx context.Context, domainID uuid.UUID) ([]*models.JobDomain, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_posting_id, domain_id, relevance, created_at, updated_at
		FROM job_domains
		WHERE domain_id = $1
		ORDER BY job_posting_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobDomains []*models.JobDomain
	for rows.Next() {
		var jobDomain models.JobDomain
		if err := rows.Scan(
			&jobDomain.ID,
			&jobDomain.JobPostingID,
			&jobDomain.DomainID,
			&jobDomain.Relevance,
			&jobDomain.CreatedAt,
			&jobDomain.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobDomains = append(jobDomains, &jobDomain)
	}

	return jobDomains, rows.Err()
}

// Update updates an existing job domain
func (r *JobDomainRepository) Update(ctx context.Context, jobDomain *models.JobDomain) error {
	query := `
		UPDATE job_domains
		SET job_posting_id = $2,
		    domain_id = $3,
		    relevance = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobDomain.ID,
		jobDomain.JobPostingID,
		jobDomain.DomainID,
		jobDomain.Relevance,
		jobDomain.UpdatedAt,
	)

	return err
}

// Delete removes a job domain by its ID
func (r *JobDomainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_domains
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobPostingID removes all job domains for a given job posting
func (r *JobDomainRepository) DeleteByJobPostingID(ctx context.Context, jobPostingID uuid.UUID) error {
	query := `
		DELETE FROM job_domains
		WHERE job_posting_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobPostingID)
	return err
}