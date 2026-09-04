package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"passo/backend/internal/plugins/market-data-engine/models"
)

// SkillRepository handles storage and retrieval of skills
type SkillRepository struct {
	db *pgxpool.Pool
}

// NewSkillRepository creates a new SkillRepository
func NewSkillRepository(db *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{db: db}
}

// Insert adds a new skill and returns its ID
func (r *SkillRepository) Insert(ctx context.Context, skill *models.Skill) (uuid.UUID, error) {
	query := `
		INSERT INTO skills
			(id, canonical_name, normalized_name, skill_type_id, category_id, parent_skill_id,
			 description, status, version, taxonomy_version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query,
		skill.ID,
		skill.CanonicalName,
		skill.NormalizedName,
		skill.SkillTypeID,
		skill.CategoryID,
		skill.ParentSkillID,
		skill.Description,
		skill.Status,
		skill.Version,
		skill.TaxonomyVersion,
		skill.CreatedAt,
		skill.UpdatedAt,
	).Scan(&id)

	return id, err
}

// GetByID retrieves a skill by its ID
func (r *SkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Skill, error) {
	query := `
		SELECT id, canonical_name, normalized_name, skill_type_id, category_id, parent_skill_id,
		       description, status, version, taxonomy_version, created_at, updated_at
		FROM skills
		WHERE id = $1
	`

	var skill models.Skill
	err := r.db.QueryRow(ctx, query, id).Scan(
		&skill.ID,
		&skill.CanonicalName,
		&skill.NormalizedName,
		&skill.SkillTypeID,
		&skill.CategoryID,
		&skill.ParentSkillID,
		&skill.Description,
		&skill.Status,
		&skill.Version,
		&skill.TaxonomyVersion,
		&skill.CreatedAt,
		&skill.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &skill, nil
}

// GetByCanonicalName retrieves a skill by its canonical name
func (r *SkillRepository) GetByCanonicalName(ctx context.Context, canonicalName string) (*models.Skill, error) {
	query := `
		SELECT id, canonical_name, normalized_name, skill_type_id, category_id, parent_skill_id,
		       description, status, version, taxonomy_version, created_at, updated_at
		FROM skills
		WHERE canonical_name = $1
	`

	var skill models.Skill
	err := r.db.QueryRow(ctx, query, canonicalName).Scan(
		&skill.ID,
		&skill.CanonicalName,
		&skill.NormalizedName,
		&skill.SkillTypeID,
		&skill.CategoryID,
		&skill.ParentSkillID,
		&skill.Description,
		&skill.Status,
		&skill.Version,
		&skill.TaxonomyVersion,
		&skill.CreatedAt,
		&skill.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &skill, nil
}

// List returns all skills
func (r *SkillRepository) List(ctx context.Context) ([]*models.Skill, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, canonical_name, normalized_name, skill_type_id, category_id, parent_skill_id,
		       description, status, version, taxonomy_version, created_at, updated_at
		FROM skills
		ORDER BY canonical_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []*models.Skill
	for rows.Next() {
		var skill models.Skill
		if err := rows.Scan(
			&skill.ID,
			&skill.CanonicalName,
			&skill.NormalizedName,
			&skill.SkillTypeID,
			&skill.CategoryID,
			&skill.ParentSkillID,
			&skill.Description,
			&skill.Status,
			&skill.Version,
			&skill.TaxonomyVersion,
			&skill.CreatedAt,
			&skill.UpdatedAt,
		); err != nil {
			return nil, err
		}
		skills = append(skills, &skill)
	}

	return skills, rows.Err()
}

// Update updates an existing skill
func (r *SkillRepository) Update(ctx context.Context, skill *models.Skill) error {
	query := `
		UPDATE skills
		SET canonical_name = $2,
		    normalized_name = $3,
		    skill_type_id = $4,
		    category_id = $5,
		    parent_skill_id = $6,
		    description = $7,
		    status = $8,
		    version = $9,
		    taxonomy_version = $10,
		    updated_at = $11
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		skill.ID,
		skill.CanonicalName,
		skill.NormalizedName,
		skill.SkillTypeID,
		skill.CategoryID,
		skill.ParentSkillID,
		skill.Description,
		skill.Status,
		skill.Version,
		skill.TaxonomyVersion,
		skill.UpdatedAt,
	)

	return err
}

// Delete removes a skill by its ID
func (r *SkillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM skills
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	return err
}