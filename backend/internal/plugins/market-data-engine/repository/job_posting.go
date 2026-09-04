package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobPostingRepository handles storage and retrieval of job postings
type JobPostingRepository struct {
	db *pgxpool.Pool
}

// NewJobPostingRepository creates a new JobPostingRepository
func NewJobPostingRepository(db *pgxpool.Pool) *JobPostingRepository {
	return &JobPostingRepository{db: db}
}

// Insert adds a new job posting and returns its ID
func (r *JobPostingRepository) Insert(ctx context.Context, jobPosting *models.JobPosting) (uuid.UUID, error) {
	query := `
		INSERT INTO job_postings
			(id, job_id, title, company_id, location_id, employment_type_id,
			 work_mode_id, salary_min, salary_max, salary_currency, salary_period,
			 description, apply_url, posted_at, expires_at, created_at, updated_at,
			 matched_skills_count, total_skills_count, match_score, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobPosting.ID,
		jobPosting.JobID,
		jobPosting.Title,
		jobPosting.CompanyID,
		jobPosting.LocationID,
		jobPosting.EmploymentTypeID,
		jobPosting.WorkModeID,
		jobPosting.SalaryMin,
		jobPosting.SalaryMax,
		jobPosting.SalaryCurrency,
		jobPosting.SalaryPeriod,
		jobPosting.Description,
		jobPosting.ApplyURL,
		jobPosting.PostedAt,
		jobPosting.ExpiresAt,
		jobPosting.CreatedAt,
		jobPosting.UpdatedAt,
		jobPosting.MatchedSkillsCount,
		jobPosting.TotalSkillsCount,
		jobPosting.MatchScore,
		jobPosting.Status,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job posting by its ID
func (r *JobPostingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobPosting, error) {
	query := `
		SELECT id, job_id, title, company_id, location_id, employment_type_id,
		       work_mode_id, salary_min, salary_max, salary_currency, salary_period,
		       description, apply_url, posted_at, expires_at, created_at, updated_at,
		       matched_skills_count, total_skills_count, match_score, status
		FROM job_postings
		WHERE id = $1
	`

	var jobPosting models.JobPosting
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobPosting.ID,
		&jobPosting.JobID,
		&jobPosting.Title,
		&jobPosting.CompanyID,
		&jobPosting.LocationID,
		&jobPosting.EmploymentTypeID,
		&jobPosting.WorkModeID,
		&jobPosting.SalaryMin,
		&jobPosting.SalaryMax,
		&jobPosting.SalaryCurrency,
		&jobPosting.SalaryPeriod,
		&jobPosting.Description,
		&jobPosting.ApplyURL,
		&jobPosting.PostedAt,
		&jobPosting.ExpiresAt,
		&jobPosting.CreatedAt,
		&jobPosting.UpdatedAt,
		&jobPosting.MatchedSkillsCount,
		&jobPosting.TotalSkillsCount,
		&jobPosting.MatchScore,
		&jobPosting.Status,
	)

	if err != nil {
		return nil, err
	}

	return &jobPosting, nil
}

// GetByJobID retrieves a job posting by its job ID
func (r *JobPostingRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*models.JobPosting, error) {
	query := `
		SELECT id, job_id, title, company_id, location_id, employment_type_id,
		       work_mode_id, salary_min, salary_max, salary_currency, salary_period,
		       description, apply_url, posted_at, expires_at, created_at, updated_at,
		       matched_skills_count, total_skills_count, match_score, status
		FROM job_postings
		WHERE job_id = $1
	`

	var jobPosting models.JobPosting
	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&jobPosting.ID,
		&jobPosting.JobID,
		&jobPosting.Title,
		&jobPosting.CompanyID,
		&jobPosting.LocationID,
		&jobPosting.EmploymentTypeID,
		&jobPosting.WorkModeID,
		&jobPosting.SalaryMin,
		&jobPosting.SalaryMax,
		&jobPosting.SalaryCurrency,
		&jobPosting.SalaryPeriod,
		&jobPosting.Description,
		&jobPosting.ApplyURL,
		&jobPosting.PostedAt,
		&jobPosting.ExpiresAt,
		&jobPosting.CreatedAt,
		&jobPosting.UpdatedAt,
		&jobPosting.MatchedSkillsCount,
		&jobPosting.TotalSkillsCount,
		&jobPosting.MatchScore,
		&jobPosting.Status,
	)

	if err != nil {
		return nil, err
	}

	return &jobPosting, nil
}

// ListByJobIDWithStatus returns job postings for a given job with optional status filter
func (r *JobPostingRepository) ListByJobIDWithStatus(ctx context.Context, jobID uuid.UUID, status string) ([]*models.JobPosting, error) {
	var rows pgx.Rows
	var err error

	if status == "" {
		rows, err = r.db.Query(ctx, `
			SELECT id, job_id, title, company_id, location_id, employment_type_id,
			       work_mode_id, salary_min, salary_max, salary_currency, salary_period,
			       description, apply_url, posted_at, expires_at, created_at, updated_at,
			       matched_skills_count, total_skills_count, match_score, status
			FROM job_postings
			WHERE job_id = $1
			ORDER BY posted_at DESC
		`, jobID)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT id, job_id, title, company_id, location_id, employment_type_id,
			       work_mode_id, salary_min, salary_max, salary_currency, salary_period,
			       description, apply_url, posted_at, expires_at, created_at, updated_at,
			       matched_skills_count, total_skills_count, match_score, status
			FROM job_postings
			WHERE job_id = $1 AND status = $2
			ORDER BY posted_at DESC
		`, jobID, status)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobPostings []*models.JobPosting
	for rows.Next() {
		var jobPosting models.JobPosting
		if err := rows.Scan(
			&jobPosting.ID,
			&jobPosting.JobID,
			&jobPosting.Title,
			&jobPosting.CompanyID,
			&jobPosting.LocationID,
			&jobPosting.EmploymentTypeID,
			&jobPosting.WorkModeID,
			&jobPosting.SalaryMin,
			&jobPosting.SalaryMax,
			&jobPosting.SalaryCurrency,
			&jobPosting.SalaryPeriod,
			&jobPosting.Description,
			&jobPosting.ApplyURL,
			&jobPosting.PostedAt,
			&jobPosting.ExpiresAt,
			&jobPosting.CreatedAt,
			&jobPosting.UpdatedAt,
			&jobPosting.MatchedSkillsCount,
			&jobPosting.TotalSkillsCount,
			&jobPosting.MatchScore,
			&jobPosting.Status,
		); err != nil {
			return nil, err
		}
		jobPostings = append(jobPostings, &jobPosting)
	}

	return jobPostings, rows.Err()
}

// List returns all job postings with pagination
func (r *JobPostingRepository) List(ctx context.Context, limit, offset int) ([]*models.JobPosting, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, title, company_id, location_id, employment_type_id,
		       work_mode_id, salary_min, salary_max, salary_currency, salary_period,
		       description, apply_url, posted_at, expires_at, created_at, updated_at,
		       matched_skills_count, total_skills_count, match_score, status
		FROM job_postings
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobPostings []*models.JobPosting
	for rows.Next() {
		var jobPosting models.JobPosting
		if err := rows.Scan(
			&jobPosting.ID,
			&jobPosting.JobID,
			&jobPosting.Title,
			&jobPosting.CompanyID,
			&jobPosting.LocationID,
			&jobPosting.EmploymentTypeID,
			&jobPosting.WorkModeID,
			&jobPosting.SalaryMin,
			&jobPosting.SalaryMax,
			&jobPosting.SalaryCurrency,
			&jobPosting.SalaryPeriod,
			&jobPosting.Description,
			&jobPosting.ApplyURL,
			&jobPosting.PostedAt,
			&jobPosting.ExpiresAt,
			&jobPosting.CreatedAt,
			&jobPosting.UpdatedAt,
			&jobPosting.MatchedSkillsCount,
			&jobPosting.TotalSkillsCount,
			&jobPosting.MatchScore,
			&jobPosting.Status,
		); err != nil {
			return nil, err
		}
		jobPostings = append(jobPostings, &jobPosting)
	}

	return jobPostings, rows.Err()
}

// Update updates an existing job posting
func (r *JobPostingRepository) Update(ctx context.Context, jobPosting *models.JobPosting) error {
	query := `
		UPDATE job_postings
		SET job_id = $2,
		    title = $3,
		    company_id = $4,
		    location_id = $5,
		    employment_type_id = $6,
		    work_mode_id = $7,
		    salary_min = $8,
		    salary_max = $9,
		    salary_currency = $10,
		    salary_period = $11,
		    description = $12,
		    apply_url = $13,
		    posted_at = $14,
		    expires_at = $15,
		    updated_at = $16,
		    matched_skills_count = $17,
		    total_skills_count = $18,
		    match_score = $19,
		    status = $20
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobPosting.ID,
		jobPosting.JobID,
		jobPosting.Title,
		jobPosting.CompanyID,
		jobPosting.LocationID,
		jobPosting.EmploymentTypeID,
		jobPosting.WorkModeID,
		jobPosting.SalaryMin,
		jobPosting.SalaryMax,
		jobPosting.SalaryCurrency,
		jobPosting.SalaryPeriod,
		jobPosting.Description,
		jobPosting.ApplyURL,
		jobPosting.PostedAt,
		jobPosting.ExpiresAt,
		jobPosting.CreatedAt,
		jobPosting.UpdatedAt,
		jobPosting.MatchedSkillsCount,
		jobPosting.TotalSkillsCount,
		jobPosting.MatchScore,
		jobPosting.Status,
	)

	return err
}

// Delete removes a job posting by its ID
func (r *JobPostingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_postings
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}