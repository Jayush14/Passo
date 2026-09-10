package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// JobSkillRepository handles storage and retrieval of job skills
type JobSkillRepository struct {
	db *pgxpool.Pool
}

// NewJobSkillRepository creates a new JobSkillRepository
func NewJobSkillRepository(db *pgxpool.Pool) *JobSkillRepository {
	return &JobSkillRepository{db: db}
}

// Insert adds a new job skill and returns its ID
func (r *JobSkillRepository) Insert(ctx context.Context, jobSkill *models.JobSkill) (uuid.UUID, error) {
	query := `
		INSERT INTO job_skills
			(id, job_id, skill_id, requirement_type, requirement_weight, source_section, confidence, evidence, raw_skill_name, normalization_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		jobSkill.ID,
		jobSkill.JobID,
		jobSkill.SkillID,
		jobSkill.RequirementType,
		jobSkill.RequirementWeight,
		"", // source_section - not in model, using empty string
		jobSkill.Confidence,
		jobSkill.Evidence,
		jobSkill.RawSkillName,
		jobSkill.NormalizationVersion,
		jobSkill.CreatedAt,
		jobSkill.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a job skill by its ID
func (r *JobSkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobSkill, error) {
	query := `
		SELECT id, job_id, skill_id, requirement_type, requirement_weight, source_section, confidence, evidence, raw_skill_name, normalization_version, created_at, updated_at
		FROM job_skills
		WHERE id = $1
	`

	var jobSkill models.JobSkill
	var sourceSection string
	var normalizationVersion uuid.UUID
	err := r.db.QueryRow(ctx, query, id).Scan(
		&jobSkill.ID,
		&jobSkill.JobID,
		&jobSkill.SkillID,
		&jobSkill.RequirementType,
		&jobSkill.RequirementWeight,
		&sourceSection,
		&jobSkill.Confidence,
		&jobSkill.Evidence,
		&jobSkill.RawSkillName,
		&normalizationVersion,
		&jobSkill.CreatedAt,
		&jobSkill.UpdatedAt,
	)
	jobSkill.NormalizationVersion = normalizationVersion

	if err != nil {
		return nil, err
	}

	return &jobSkill, nil
}

// ListByJobID returns all skills for a given job
func (r *JobSkillRepository) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, skill_id, requirement_type, requirement_weight, source_section, confidence, evidence, raw_skill_name, normalization_version, created_at, updated_at
		FROM job_skills
		WHERE job_id = $1
		ORDER BY requirement_weight DESC, confidence DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSkills []*models.JobSkill
	for rows.Next() {
		var jobSkill models.JobSkill
		var sourceSection string
		var normalizationVersion uuid.UUID
		if err := rows.Scan(
			&jobSkill.ID,
			&jobSkill.JobID,
			&jobSkill.SkillID,
			&jobSkill.RequirementType,
			&jobSkill.RequirementWeight,
			&sourceSection,
			&jobSkill.Confidence,
			&jobSkill.Evidence,
			&jobSkill.RawSkillName,
			&normalizationVersion,
			&jobSkill.CreatedAt,
			&jobSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSkill.NormalizationVersion = normalizationVersion
		jobSkills = append(jobSkills, &jobSkill)
	}

	return jobSkills, rows.Err()
}

// ListBySkillID returns all jobs that require a given skill
func (r *JobSkillRepository) ListBySkillID(ctx context.Context, skillID uuid.UUID) ([]*models.JobSkill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, job_id, skill_id, requirement_type, requirement_weight, source_section, confidence, evidence, raw_skill_name, normalization_version, created_at, updated_at
		FROM job_skills
		WHERE skill_id = $1
		ORDER BY job_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobSkills []*models.JobSkill
	for rows.Next() {
		var jobSkill models.JobSkill
		var sourceSection string
		var normalizationVersion uuid.UUID
		if err := rows.Scan(
			&jobSkill.ID,
			&jobSkill.JobID,
			&jobSkill.SkillID,
			&jobSkill.RequirementType,
			&jobSkill.RequirementWeight,
			&sourceSection,
			&jobSkill.Confidence,
			&jobSkill.Evidence,
			&jobSkill.RawSkillName,
			&normalizationVersion,
			&jobSkill.CreatedAt,
			&jobSkill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobSkill.NormalizationVersion = normalizationVersion
		jobSkills = append(jobSkills, &jobSkill)
	}

	return jobSkills, rows.Err()
}

// Update updates an existing job skill
func (r *JobSkillRepository) Update(ctx context.Context, jobSkill *models.JobSkill) error {
	query := `
		UPDATE job_skills
		SET job_id = $2,
		    skill_id = $3,
		    requirement_type = $4,
		    requirement_weight = $5,
		    source_section = $6,
		    confidence = $7,
		    evidence = $8,
		    raw_skill_name = $9,
		    normalization_version = $10,
		    updated_at = $11
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		jobSkill.ID,
		jobSkill.JobID,
		jobSkill.SkillID,
		jobSkill.RequirementType,
		jobSkill.RequirementWeight,
		"", // source_section - not in model, using empty string
		jobSkill.Confidence,
		jobSkill.Evidence,
		jobSkill.RawSkillName,
		jobSkill.NormalizationVersion,
		jobSkill.UpdatedAt,
	)

	return err
}

// Delete removes a job skill by its ID
func (r *JobSkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM job_skills
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteByJobID removes all job skills for a given job
func (r *JobSkillRepository) DeleteByJobID(ctx context.Context, jobID uuid.UUID) error {
	query := `
		DELETE FROM job_skills
		WHERE job_id = $1
	`

	_, err := r.db.Exec(ctx, query, jobID)
	return err
}